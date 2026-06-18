package common

import (
	"context"
	"sync"
	"time"

	log "github.com/sirupsen/logrus"
)

// Provisioner creates a per-user folder in one storage backend when a new
// Kratos identity is auto-registered in AList. Implementations must be
// idempotent — safe to call multiple times for the same identity.
type Provisioner interface {
	// MountPath returns the AList mount path that this provisioner manages.
	// Example: "/", "/cloud", "/s3".
	MountPath() string
	// Provision creates the user folder if it doesn't already exist.
	// Returns nil if the folder already exists (idempotent).
	Provision(ctx context.Context, identityID string) error
}

var (
	provisionersMu sync.RWMutex
	provisioners   []Provisioner
)

// RegisterProvisioner registers a provisioner. Called at startup (after
// alistAdminSetup mounts the corresponding storage) or via init() for
// built-in providers.
func RegisterProvisioner(p Provisioner) {
	if p == nil {
		return
	}
	provisionersMu.Lock()
	defer provisionersMu.Unlock()
	provisioners = append(provisioners, p)
}

// ProvisionUserFolders runs every registered provisioner concurrently.
// Called after a new AList user is created in GetOrCreateKratosUser.
//
// Each provisioner runs in its own goroutine with a 30s timeout. Errors
// are logged but never returned — provisioning is best-effort and must
// not block user login or fail AList user creation.
func ProvisionUserFolders(parent context.Context, identityID string) {
	if identityID == "" {
		return
	}

	provisionersMu.RLock()
	snapshot := make([]Provisioner, len(provisioners))
	copy(snapshot, provisioners)
	provisionersMu.RUnlock()

	for _, p := range snapshot {
		go func(p Provisioner) {
			ctx, cancel := context.WithTimeout(parent, 30*time.Second)
			defer cancel()
			if err := p.Provision(ctx, identityID); err != nil {
				log.WithFields(log.Fields{
					"provider":    p.MountPath(),
					"identity_id": identityID,
				}).Warnf("auto-provision failed: %v", err)
				return
			}
			log.WithFields(log.Fields{
				"provider":    p.MountPath(),
				"identity_id": identityID,
			}).Debug("auto-provision ok")
		}(p)
	}
}

// RegisteredProvisioners returns a snapshot of currently-registered
// provisioners. Used by tests and admin tooling.
func RegisteredProvisioners() []Provisioner {
	provisionersMu.RLock()
	defer provisionersMu.RUnlock()
	out := make([]Provisioner, len(provisioners))
	copy(out, provisioners)
	return out
}

// ResetProvisioners clears the provisioner registry. Test-only.
func ResetProvisioners() {
	provisionersMu.Lock()
	provisioners = nil
	provisionersMu.Unlock()
}