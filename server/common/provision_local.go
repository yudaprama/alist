package common

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// LocalProvisioner creates a per-user directory on the local filesystem.
// It mirrors the existing alistAdminSetup behaviour where each Kratos
// identity gets a directory under ALIST_STORAGE_PATH.
type LocalProvisioner struct {
	BasePath string // ALIST_STORAGE_PATH, e.g. "/data/alist-files"
}

func (l *LocalProvisioner) MountPath() string { return "/" }

func (l *LocalProvisioner) Provision(ctx context.Context, identityID string) error {
	if l == nil || l.BasePath == "" {
		return errors.New("local provisioner: BasePath not set")
	}
	if identityID == "" {
		return errors.New("local provisioner: empty identityID")
	}

	folderName := sanitizeLocal(identityID)
	if folderName == "" {
		return fmt.Errorf("local provisioner: identity %q yields empty folder name", identityID)
	}

	dir := filepath.Join(l.BasePath, folderName)

	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
	}

	if err := os.MkdirAll(dir, 0o755); err != nil {
		return fmt.Errorf("mkdir %q: %w", dir, err)
	}

	fi, err := os.Stat(dir)
	if err != nil {
		return fmt.Errorf("stat %q: %w", dir, err)
	}
	if !fi.IsDir() {
		return fmt.Errorf("%q exists but is not a directory", dir)
	}
	return nil
}

// sanitizeLocal strips characters that are illegal in directory names
// on common filesystems and truncates to a safe length.
func sanitizeLocal(s string) string {
	s = strings.TrimSpace(s)
	if s == "" {
		return ""
	}
	replacer := strings.NewReplacer("\x00", "", "/", "_", "\\", "_")
	s = replacer.Replace(s)
	lower := strings.ToLower(s)
	if lower == "." || lower == ".." {
		return ""
	}
	if len(s) > 128 {
		s = s[:128]
	}
	return s
}

// init auto-registers the Local provisioner when the ALIST_STORAGE_PATH
// environment variable is set. planoctl sets this before starting AList.
func init() {
	storagePath := os.Getenv("ALIST_STORAGE_PATH")
	if storagePath == "" {
		// Fallback consistent with planoctl main.go
		home, _ := os.UserHomeDir()
		if home != "" {
			storagePath = filepath.Join(home, "data", "alist-files")
		}
	}
	if storagePath != "" {
		RegisterProvisioner(&LocalProvisioner{BasePath: storagePath})
	}
}