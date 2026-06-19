---
# This is the icon of the page
icon: iconfont icon-state
# This control sidebar order
order: 14
# A page can have multiple categories
category:
  - Guide
# A page can have multiple tags
tag:
  - Advanced
  - Guide
# this page is sticky in article list
sticky: true
# this page will appear in starred articles
star: true
---

# Storage routing

AList exposes a single virtual filesystem rooted at `/`. Every API request — list, get, upload, copy, move, remove — carries a **mount path** that must be mapped to a concrete storage instance and a path inside that storage. This page describes how that mapping works, with upload (`PUT /api/fs/put`, `PUT /api/fs/form`) as the running example. The same routing logic is reused by every other filesystem operation.

## From HTTP request to driver call

```
PUT /api/fs/put   (header: File-Path: /onedrive/photos/cat.jpg)
  │
  ├─ middlewares.FsUp              server/middlewares/fsup.go
  │     • resolves the user's base path
  │     • looks up the nearest meta for the parent directory
  │     • checks write permission           (does NOT pick a storage)
  │
  ├─ handles.FsStream / FsForm    server/handles/fsup.go
  │     • reads File-Path, As-Task, Overwrite, hashes, etc.
  │     • builds a stream.FileStream
  │     • calls fs.PutAsTask  OR  fs.PutDirectly
  │
  ├─ internal/fs/put.go           putAsTask / putDirectly
  │     • calls op.GetStorageAndActualPath(dstDirPath)
  │     • bails out early if the storage declares NoUpload
  │
  ├─ internal/op/path.go          GetStorageAndActualPath        ← the resolver
  │     • GetBalancedStorage(rawPath)        (longest-prefix match)
  │     • strips the mount path to get the actual path
  │
  └─ internal/op/fs.go            op.Put
        • type-asserts storage.(driver.Put) / driver.PutResult
        • calls storage.Put(ctx, parentDir, file, up)
              └─ e.g. drivers/onedriver/driver.go  Put(...)
```

The `internal/fs` package's only job is to convert a **mount path** into a **(storage, actual path)** pair and hand it off to `internal/op`. Its package comment makes this explicit:

> the param named path of functions in this package is a mount path.
> So, the purpose of this package is to convert mount path to actual path
> then pass the actual path to the op package

## The resolver: `GetStorageAndActualPath`

`internal/op/path.go`:

```go
func GetStorageAndActualPath(rawPath string) (storage driver.Driver, actualPath string, err error) {
    rawPath = utils.FixAndCleanPath(rawPath)
    storage = GetBalancedStorage(rawPath)
    if storage == nil {
        err = errs.NewErr(errs.StorageNotFound, "rawPath: %s", rawPath)
        return
    }
    mountPath := utils.GetActualMountPath(storage.GetStorage().MountPath)
    actualPath = utils.FixAndCleanPath(strings.TrimPrefix(rawPath, mountPath))
    return
}
```

Two steps:

1. Find the storage whose mount path is the best prefix of `rawPath`.
2. Strip that mount path off `rawPath` to get the path the driver will receive.

For an upload to `/onedrive/photos/cat.jpg`, with a storage mounted at `/onedrive`, the driver receives `photos/cat.jpg`.

## Longest-prefix match: `getStoragesByPath`

`internal/op/storage.go`:

```go
// getStoragesByPath get storage by longest match path, contains balance storage.
// for example, there is /a/b,/a/c,/a/d/e,/a/d/e.balance
// getStoragesByPath(/a/d/e/f) => /a/d/e,/a/d/e.balance
func getStoragesByPath(path string) []driver.Driver {
    storages := make([]driver.Driver, 0)
    curSlashCount := 0
    storagesMap.Range(func(mountPath string, value driver.Driver) bool {
        mountPath = utils.GetActualMountPath(mountPath)
        if utils.IsSubPath(mountPath, path) {
            slashCount := strings.Count(utils.PathAddSeparatorSuffix(mountPath), "/")
            if slashCount > curSlashCount {
                storages = storages[:0]
                curSlashCount = slashCount
            }
            if slashCount == curSlashCount {
                storages = append(storages, value)
            }
        }
        return true
    })
    sort.Slice(storages, func(i, j int) bool {
        return storages[i].GetStorage().MountPath < storages[j].GetStorage().MountPath
    })
    return storages
}
```

The algorithm:

- Walk every entry in the in-memory `storagesMap` (keyed by mount path).
- Keep only entries whose mount path is a sub-path of the request path.
- Among the candidates, keep the ones with the **most path separators** — i.e. the longest, most specific mount path wins.
- Sort the survivors by mount path so the result is deterministic for a given input.

This is why nested mounts work. With mounts `/a`, `/a/b`, and `/a/b/c`, a request for `/a/b/c/file.txt` is served by `/a/b/c`, never by `/a` or `/a/b`.

## Load balancing: `GetBalancedStorage`

`internal/op/storage.go`:

```go
var balanceMap generic_sync.MapOf[string, int]

func GetBalancedStorage(path string) driver.Driver {
    path = utils.FixAndCleanPath(path)
    storages := getStoragesByPath(path)
    storageNum := len(storages)
    switch storageNum {
    case 0:
        return nil
    case 1:
        return storages[0]
    default:
        virtualPath := utils.GetActualMountPath(storages[0].GetStorage().MountPath)
        i, _ := balanceMap.LoadOrStore(virtualPath, 0)
        i = (i + 1) % storageNum
        balanceMap.Store(virtualPath, i)
        return storages[i]
    }
}
```

When several storages resolve to the same effective mount path (see [Load balancing](./balance.md) — add siblings with the `mountpath.balanceN` convention), `GetBalancedStorage` round-robins between them using a per-virtual-path counter in `balanceMap`. Otherwise the single winner is returned directly. `nil` means no storage matched and the caller surfaces `StorageNotFound`.

## In-memory storage registry: `storagesMap`

`internal/op/storage.go`:

```go
var storagesMap generic_sync.MapOf[string, driver.Driver]
```

A concurrent map keyed by mount path. It is populated by `CreateStorage` / `LoadStorage` when an admin adds or enables a storage account. Each entry holds a fully-initialised `driver.Driver` instance bound to that account's configuration (credentials, root folder, etc.). The routing functions above only read this map; they never inspect the underlying database.

## Driver registry: `driverMap`

`internal/op/driver.go`:

```go
type DriverConstructor func() driver.Driver

var driverMap = map[string]DriverConstructor{}

func RegisterDriver(driver DriverConstructor) {
    tempDriver := driver()
    tempConfig := tempDriver.Config()
    registerDriverItems(tempConfig, tempDriver.GetAddition())
    driverMap[tempConfig.Name] = driver
}
```

This is a separate registry, keyed by **driver name** (`"Local"`, `"OneDrive"`, `"S3"`, `"WebDAV"`, …). Each driver package under `drivers/*` calls `RegisterDriver` from its `init()`. `drivers/all.go` exists solely to pull every driver package in via blank imports so their `init()` functions run at startup.

When an admin creates a storage, the configured driver name is looked up here, the constructor is invoked to produce an instance, the instance is configured with the account-specific additions, and the result is placed in `storagesMap` under the configured mount path.

The upload capability itself is expressed as an interface — only drivers that implement `driver.Put` (or `driver.PutResult`) accept uploads:

`internal/driver/driver.go`:

```go
type Put interface {
    Put(ctx context.Context, dstDir model.Obj, file model.FileStreamer, up UpdateProgress) error
}
```

`op.Put` type-asserts the resolved storage against this interface and returns `errs.NotImplement` if the driver does not support writing.

## TL;DR

1. Every filesystem operation is path-based; there is no "provider id" field on requests.
2. `op.GetStorageAndActualPath` does the routing in two steps: longest-prefix match on mount path, then strip the mount path.
3. Ties at the same mount-path depth are either a single winner or a round-robin load-balancing set.
4. The driver itself is chosen at configuration time (admin picks a driver name per storage account); at request time only the storage *instance* is selected, never the driver type.
5. Upload capability is opt-in via the `driver.Put` interface — `op.Put` returns `NotImplement` for read-only drivers.
