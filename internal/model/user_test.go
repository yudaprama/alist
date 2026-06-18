package model

import (
	"testing"

	"github.com/alist-org/alist/v3/internal/errs"
)

func TestJoinPath_BasePathContainment(t *testing.T) {
	tests := []struct {
		name      string
		user      *User
		reqPath   string
		want      string
		wantErr   error
	}{
		{
			name:    "admin (BasePath=/) absolute path allowed",
			user:    &User{BasePath: "/", Permission: 0},
			reqPath: "/any/path",
			want:    "/any/path",
		},
		{
			name:    "admin root returns BasePath",
			user:    &User{BasePath: "/", Permission: 0},
			reqPath: "/",
			want:    "/",
		},
		{
			name:    "kratos user absolute path inside BasePath allowed",
			user:    &User{BasePath: "/abc123-id", Permission: 0},
			reqPath: "/abc123-id/sub/file.txt",
			want:    "/abc123-id/sub/file.txt",
		},
		{
			name:    "kratos user path without leading slash (normalized) treated as absolute and denied",
			user:    &User{BasePath: "/abc123-id", Permission: 0},
			reqPath: "my-folder",
			wantErr: errs.PermissionDenied,
		},
		{
			name:    "kratos user absolute path OUTSIDE BasePath denied",
			user:    &User{BasePath: "/efg123-id", Permission: 0},
			reqPath: "/abc123-id/secret",
			wantErr: errs.PermissionDenied,
		},
		{
			name:    "kratos user root returns own BasePath",
			user:    &User{BasePath: "/efg123-id", Permission: 0},
			reqPath: "/",
			want:    "/efg123-id",
		},
		{
			name:    "relative path traversal rejected",
			user:    &User{BasePath: "/abc123-id", Permission: 0},
			reqPath: "../etc/passwd",
			wantErr: errs.RelativePath,
		},
		{
			name:    "deny cross-user when target is sibling prefix",
			user:    &User{BasePath: "/efg", Permission: 0},
			reqPath: "/efg-evil/file",
			wantErr: errs.PermissionDenied,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := tt.user.JoinPath(tt.reqPath)
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
