package common

import (
	"context"
	"errors"

	"github.com/alist-org/alist/v3/internal/conf"
	"github.com/alist-org/alist/v3/internal/model"
	"github.com/alist-org/alist/v3/internal/op"
	"github.com/alist-org/alist/v3/internal/setting"
	"github.com/alist-org/alist/v3/pkg/utils/random"
	"gorm.io/gorm"
)

func traitToString(traits map[string]interface{}, keys ...string) string {
	for _, k := range keys {
		if v, ok := traits[k]; ok {
			if s, ok := v.(string); ok && s != "" {
				return s
			}
		}
	}
	return ""
}

// GetOrCreateKratosUser finds the AList user bound to a Kratos identity,
// or creates one if auto-register is enabled. The user is isolated under
// /<identity_id> so they can only access their own files.
func GetOrCreateKratosUser(identityID string, traits map[string]interface{}) (*model.User, error) {
	if identityID == "" {
		return nil, errors.New("invalid kratos identity")
	}

	ssoID := "kratos:" + identityID

	user, err := op.GetUserBySsoID(ssoID)
	if err == nil {
		return user, nil
	}
	if !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, err
	}

	if !setting.GetBool(conf.KratosAutoRegister) {
		return nil, gorm.ErrRecordNotFound
	}

	username := traitToString(traits, "username", "email", "name")
	if username == "" {
		username = identityID
	}

	basePath := setting.GetStr(conf.KratosDefaultDir)
	if basePath == "" || basePath == "/" {
		basePath = "/" + identityID
	}

	defaultRole := int(setting.GetInt(conf.KratosDefaultRole, 2))
	if defaultRole == 0 {
		defaultRole = op.GetDefaultRoleID()
	}

	user = &model.User{
		Username:   username,
		Password:   random.String(16),
		BasePath:   basePath,
		Role:       model.Roles{defaultRole},
		Permission: int32(setting.GetInt(conf.SSODefaultPermission, 0)),
		Disabled:   false,
		SsoID:      ssoID,
	}

	if err := op.CreateUser(user); err != nil {
		// Race condition: another request created the user; fetch it
		if u, getErr := op.GetUserBySsoID(ssoID); getErr == nil {
			return u, nil
		}
		return nil, err
	}

	// Auto-provision per-user folders across all mounted storages
	// (Local, GoogleDrive, S3, etc.). Each provisioner runs async and
	// is best-effort — failures must not block user creation.
	ProvisionUserFolders(context.Background(), identityID)

	return user, nil
}
