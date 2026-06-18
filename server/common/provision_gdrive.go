package common

import (
	"context"
	"errors"
	"fmt"
	"os"
	"strings"

	"github.com/alist-org/alist/v3/internal/op"
)

// GDriveProvisioner creates a per-user folder in a Google Drive storage.
// The drive must already be mounted via the AList admin API with
// root_folder_id set to the parent folder that will hold per-user
// directories (e.g. the "alist_users/" folder).
//
// The provisioner creates "<identityID>/" directly under the mount root,
// so the folder structure looks like:
//
//	storage mount /cloud  (root_folder_id = "alist_users_parent_id")
//	/cloud/identity_A/
//	/cloud/identity_B/
type GDriveProvisioner struct {
	Mount string // AList mount path, e.g. "/cloud"
}

func (g *GDriveProvisioner) MountPath() string { return g.Mount }

func (g *GDriveProvisioner) Provision(ctx context.Context, identityID string) error {
	if g == nil || g.Mount == "" {
		return errors.New("gdrive provisioner: Mount not set")
	}
	if identityID == "" {
		return errors.New("gdrive provisioner: empty identityID")
	}

	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
	}

	storage, err := op.GetStorageByMountPath(g.Mount)
	if err != nil {
		// Storage not mounted — expected when GDrive is not configured.
		return nil
	}

	folderName := sanitizeDrive(identityID)
	if folderName == "" {
		return fmt.Errorf("gdrive provisioner: identity %q yields empty folder name", identityID)
	}

	// op.MakeDir creates folders recursively and supports all drivers
	// that implement driver.Mkdir, including GoogleDrive.
	return op.MakeDir(ctx, storage, "/"+folderName)
}

// sanitizeDrive strips characters that Google Drive rejects and truncates
// to a safe length. Folder names are restricted to 255 chars.
func sanitizeDrive(s string) string {
	s = strings.TrimSpace(s)
	if s == "" {
		return ""
	}
	replacer := strings.NewReplacer("\x00", "", "/", "_", "\\", "_", ":", "_")
	s = replacer.Replace(s)
	if len(s) > 128 {
		s = s[:128]
	}
	return s
}

// init auto-registers the GDrive provisioner when the GDRIVE_REFRESH_TOKEN
// environment variable is set (indicating Google Drive is configured).
func init() {
	if os.Getenv("GDRIVE_REFRESH_TOKEN") == "" {
		return
	}
	mount := os.Getenv("GDRIVE_MOUNT_PATH")
	if mount == "" {
		mount = "/cloud"
	}
	RegisterProvisioner(&GDriveProvisioner{Mount: mount})
}