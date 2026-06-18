package utils

import (
	"testing"

	"github.com/alist-org/alist/v3/internal/errs"
)

func TestEncodePath(t *testing.T) {
	t.Log(EncodePath("http://localhost:5244/d/123#.png"))
}

func TestFixAndCleanPath(t *testing.T) {
	datas := map[string]string{
		"":                          "/",
		".././":                     "/",
		"../../.../":                "/...",
		"x//\\y/":                   "/x/y",
		".././.x/.y/.//..x../..y..": "/.x/.y/..x../..y..",
	}
	for key, value := range datas {
		if FixAndCleanPath(key) != value {
			t.Logf("raw %s fix fail", key)
		}
	}
}

func TestValidateNameComponent(t *testing.T) {
	validNames := []string{
		"file.txt",
		"abc",
		"file_name-1",
	}
	for _, name := range validNames {
		if err := ValidateNameComponent(name); err != nil {
			t.Fatalf("expected valid name %q, got error: %v", name, err)
		}
	}

	invalidNames := []string{
		"",
		".",
		"..",
		"a/b",
		`a\b`,
		"a..b",
		string([]byte{'a', 0, 'b'}),
	}
	for _, name := range invalidNames {
		if err := ValidateNameComponent(name); err == nil {
			t.Fatalf("expected invalid name %q to be rejected", name)
		}
	}
}

func TestJoinUnderBase(t *testing.T) {
	base := "/lanzou-y/shared/test1"
	out, err := JoinUnderBase(base, "file.txt")
	if err != nil {
		t.Fatalf("expected join success, got error: %v", err)
	}
	if out != "/lanzou-y/shared/test1/file.txt" {
		t.Fatalf("unexpected join result: %s", out)
	}

	if _, err := JoinUnderBase(base, "../admin/screts.txt"); err == nil {
		t.Fatalf("expected traversal to be rejected")
	}
	if _, err := JoinUnderBase(base, "sub/child"); err == nil {
		t.Fatalf("expected nested path to be rejected")
	}
}

func TestJoinBasePath(t *testing.T) {
	tests := []struct {
		name     string
		basePath string
		reqPath  string
		want     string
		wantErr  error
	}{
		{
			name:     "truly relative single segment resolves under basePath",
			basePath: "/abc123-id",
			reqPath:  "my-folder",
			want:     "/abc123-id/my-folder",
		},
		{
			name:     "truly relative nested resolves under basePath",
			basePath: "/abc123-id",
			reqPath:  "sub/child/file.txt",
			want:     "/abc123-id/sub/child/file.txt",
		},
		{
			name:     "absolute path inside basePath passes through",
			basePath: "/abc123-id",
			reqPath:  "/abc123-id/file.txt",
			want:     "/abc123-id/file.txt",
		},
		{
			name:     "absolute path outside basePath passes through (containment enforced by caller)",
			basePath: "/abc123-id",
			reqPath:  "/xyz/file.txt",
			want:     "/xyz/file.txt",
		},
		{
			name:     "root absolute path passes through",
			basePath: "/abc123-id",
			reqPath:  "/",
			want:     "/",
		},
		{
			name:     "relative traversal rejected",
			basePath: "/abc123-id",
			reqPath:  "../etc/passwd",
			wantErr:  errs.RelativePath,
		},
		{
			name:     "mid-path traversal rejected",
			basePath: "/abc123-id",
			reqPath:  "/a/b/../c",
			wantErr:  errs.RelativePath,
		},
		{
			name:     "admin basePath with absolute path",
			basePath: "/",
			reqPath:  "/any/path",
			want:     "/any/path",
		},
		{
			name:     "relative path with backslash normalized",
			basePath: "/abc123-id",
			reqPath:  `my\folder`,
			want:     "/abc123-id/my/folder",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := JoinBasePath(tt.basePath, tt.reqPath)
			if tt.wantErr != nil {
				if err == nil {
					t.Fatalf("expected error %v, got nil (path=%q)", tt.wantErr, got)
				}
				if err != tt.wantErr {
					t.Fatalf("expected error %v, got %v", tt.wantErr, err)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got != tt.want {
				t.Fatalf("expected %q, got %q", tt.want, got)
			}
		})
	}
}
