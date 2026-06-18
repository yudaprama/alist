package common

import (
	"context"
	"os"
	"path/filepath"
	"testing"
)

func TestLocalProvisioner_Provision(t *testing.T) {
	dir := t.TempDir()
	p := &LocalProvisioner{BasePath: dir}

	// First call: should create the directory
	if err := p.Provision(context.Background(), "test-user"); err != nil {
		t.Fatalf("first provision: %v", err)
	}
	// Verify directory exists
	if _, err := os.Stat(filepath.Join(dir, "test-user")); err != nil {
		t.Fatalf("stat: %v", err)
	}

	// Second call: should be idempotent
	if err := p.Provision(context.Background(), "test-user"); err != nil {
		t.Fatalf("second provision (idempotent): %v", err)
	}
}

func TestLocalProvisioner_EmptyIdentity(t *testing.T) {
	p := &LocalProvisioner{BasePath: "/tmp"}
	if err := p.Provision(context.Background(), ""); err == nil {
		t.Fatal("expected error for empty identity")
	}
}

func TestLocalProvisioner_NilReceiver(t *testing.T) {
	var p *LocalProvisioner
	if err := p.Provision(context.Background(), "x"); err == nil {
		t.Fatal("expected error for nil receiver")
	}
}

func TestSanitizeLocal(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"user-abc", "user-abc"},
		{"abc/def", "abc_def"},
		{"abc\x00def", "abcdef"},
		{"../etc", ".._etc"},
		{".", ""},
		{"..", ""},
	}
	for _, tc := range tests {
		got := sanitizeLocal(tc.input)
		if got != tc.want {
			t.Errorf("sanitizeLocal(%q) = %q; want %q", tc.input, got, tc.want)
		}
	}
}

func TestSanitizeDrive(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"user-abc", "user-abc"},
		{"abc/def", "abc_def"},
		{"abc\\def", "abc_def"},
		{"abc\x00def", "abcdef"},
	}
	for _, tc := range tests {
		got := sanitizeDrive(tc.input)
		if got != tc.want {
			t.Errorf("sanitizeDrive(%q) = %q; want %q", tc.input, got, tc.want)
		}
	}
}

func TestRegisterProvisioner(t *testing.T) {
	ResetProvisioners()
	defer ResetProvisioners()

	if len(RegisteredProvisioners()) != 0 {
		t.Fatal("expected empty registry")
	}

	p := &LocalProvisioner{BasePath: "/tmp"}
	RegisterProvisioner(p)
	if len(RegisteredProvisioners()) != 1 {
		t.Fatal("expected 1 provisioner")
	}

	// Nil should be no-op
	RegisterProvisioner(nil)
	if len(RegisteredProvisioners()) != 1 {
		t.Fatal("nil should not be registered")
	}
}

func TestProvisionUserFolders_EmptyIdentity(t *testing.T) {
	ResetProvisioners()
	defer ResetProvisioners()

	ProvisionUserFolders(context.Background(), "")
	// Should not panic
}

func TestProvisionUserFolders_NoProvisioners(t *testing.T) {
	ResetProvisioners()
	defer ResetProvisioners()

	ProvisionUserFolders(context.Background(), "test-user")
	// Should not panic from empty registry
}
