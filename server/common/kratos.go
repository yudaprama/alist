package common

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/Xhofe/go-cache"
	"github.com/alist-org/alist/v3/internal/conf"
	"github.com/alist-org/alist/v3/internal/model"
	"github.com/alist-org/alist/v3/internal/op"
	"github.com/alist-org/alist/v3/internal/setting"
	"github.com/alist-org/alist/v3/pkg/utils/random"
	kratos "github.com/ory/kratos-client-go/v26"
	"gorm.io/gorm"
)

type KratosIdentity = kratos.Identity
type KratosSession = kratos.Session

// Validator is the minimal surface needed to verify a Kratos session.
// Exists so the validator can be swapped in tests.
type KratosValidator interface {
	ValidateSession(ctx context.Context, sessionToken string) (*KratosSession, error)
}

type kratosValidator struct {
	client  *kratos.APIClient
	timeout time.Duration
}

func (v *kratosValidator) ValidateSession(ctx context.Context, sessionToken string) (*KratosSession, error) {
	if sessionToken == "" {
		return nil, errors.New("empty kratos session token")
	}
	callCtx, cancel := context.WithTimeout(ctx, v.timeout)
	defer cancel()

	session, _, err := v.client.FrontendAPI.
		ToSession(callCtx).
		XSessionToken(sessionToken).
		Execute()
	if err != nil {
		return nil, err
	}
	if session == nil {
		return nil, errors.New("kratos returned nil session")
	}
	if session.Active != nil && !*session.Active {
		return nil, errors.New("kratos session is not active")
	}
	if session.Identity == nil || session.Identity.GetId() == "" {
		return nil, errors.New("kratos session has no identity")
	}
	return session, nil
}

var (
	validatorMu      sync.RWMutex
	validatorCache   = map[string]KratosValidator{}
	validatorFactory = newKratosValidator
)

func newKratosValidator(cfg kratosValidatorConfig) KratosValidator {
	c := kratos.NewConfiguration()
	c.Servers = kratos.ServerConfigurations{{URL: cfg.BrowserURL}}
	timeout := time.Duration(cfg.RequestTimeout) * time.Second
	if timeout <= 0 {
		timeout = 5 * time.Second
	}
	c.HTTPClient = &http.Client{Timeout: timeout}
	return &kratosValidator{
		client:  kratos.NewAPIClient(c),
		timeout: timeout,
	}
}

type kratosValidatorConfig struct {
	BrowserURL     string
	RequestTimeout int
}

func resolveKratosConfig() kratosValidatorConfig {
	return kratosValidatorConfig{
		BrowserURL:     setting.GetStr(conf.KratosPublicUrl),
		RequestTimeout: 5,
	}
}

func getKratosValidator() (KratosValidator, error) {
	if !setting.GetBool(conf.KratosEnabled) {
		return nil, errors.New("kratos auth is not enabled")
	}
	cfg := resolveKratosConfig()
	if cfg.BrowserURL == "" {
		return nil, errors.New("kratos public url is not configured")
	}

	validatorMu.RLock()
	if v, ok := validatorCache[cfg.BrowserURL]; ok {
		validatorMu.RUnlock()
		return v, nil
	}
	validatorMu.RUnlock()

	validatorMu.Lock()
	defer validatorMu.Unlock()
	if v, ok := validatorCache[cfg.BrowserURL]; ok {
		return v, nil
	}
	v := validatorFactory(cfg)
	validatorCache[cfg.BrowserURL] = v
	return v, nil
}

var kratosSessionCache = cache.NewMemCache[string](cache.WithShards[string](16))

// ValidateKratosSession validates a Kratos session token via the
// official Kratos SDK (FrontendAPI.ToSession).
func ValidateKratosSession(token string) (*KratosSession, error) {
	if token == "" {
		return nil, errors.New("empty kratos token")
	}

	if cached, ok := kratosSessionCache.Get(token); ok {
		var session KratosSession
		if err := json.Unmarshal([]byte(cached), &session); err == nil {
			if session.Active != nil && !*session.Active {
				return nil, errors.New("kratos session is not active")
			}
			return &session, nil
		}
	}

	v, err := getKratosValidator()
	if err != nil {
		return nil, err
	}

	session, err := v.ValidateSession(context.Background(), token)
	if err != nil {
		return nil, err
	}

	if buf, err := json.Marshal(session); err == nil {
		kratosSessionCache.Set(token, string(buf), cache.WithEx[string](time.Minute))
	}

	return session, nil
}

// InvalidateKratosSession removes a cached session so the next request
// hits Kratos directly (used on logout).
func InvalidateKratosSession(token string) {
	if token == "" {
		return
	}
	kratosSessionCache.Del(token)
}

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
func GetOrCreateKratosUser(session *KratosSession) (*model.User, error) {
	if session == nil || session.Identity == nil || session.Identity.GetId() == "" {
		return nil, errors.New("invalid kratos session")
	}

	ssoID := "kratos:" + session.Identity.GetId()

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

	traitsRaw := session.Identity.GetTraits()
	var traitsMap map[string]interface{}
	if t, ok := traitsRaw.(map[string]interface{}); ok {
		traitsMap = t
	}
	username := traitToString(traitsMap, "username", "email", "name")
	if username == "" {
		username = session.Identity.GetId()
	}

	basePath := setting.GetStr(conf.KratosDefaultDir)
	if basePath == "" || basePath == "/" {
		basePath = "/" + session.Identity.GetId()
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

	// Auto-create user directory under storage root (lazy init)
	// Matches main.go default: ~/data/alist-files/<identity_id>/
	storagePath := os.Getenv("ALIST_STORAGE_PATH")
	if storagePath != "" {
		userDir := filepath.Join(storagePath, session.Identity.GetId())
		_ = os.MkdirAll(userDir, 0755) // best-effort, non-fatal
	}

	return user, nil
}
