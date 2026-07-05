// Package service holds the business logic layer: metric ingestion, notifications
// and authentication helpers. The auth helpers are thin, real wrappers (bcrypt /
// jwt-v5); ingestion and notification are P2/P7 stubs.
package service

import (
	"context"
	"strconv"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"go.uber.org/zap"
	"golang.org/x/crypto/bcrypt"

	"github.com/lwshen/go-server-monitor/internal/models"
	"github.com/lwshen/go-server-monitor/internal/store"
)

// jwtTTL is the admin JWT validity window (REQ-RES-05 / REQ-SEC-02): 7 days.
const jwtTTL = 7 * 24 * time.Hour

// HashPassword bcrypt-hashes a plaintext password (REQ-SEC-03).
func HashPassword(password string) (string, error) {
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return "", err
	}
	return string(hash), nil
}

// CheckPassword verifies a plaintext password against a bcrypt hash.
func CheckPassword(hash, password string) bool {
	return bcrypt.CompareHashAndPassword([]byte(hash), []byte(password)) == nil
}

// IssueJWT signs an HS256 admin token valid for 7 days (REQ-RES-05).
func IssueJWT(secret, subject string) (string, error) {
	now := time.Now()
	claims := jwt.RegisteredClaims{
		Subject:   subject,
		IssuedAt:  jwt.NewNumericDate(now),
		ExpiresAt: jwt.NewNumericDate(now.Add(jwtTTL)),
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(secret))
}

// ParseJWT validates a token string and returns its registered claims.
func ParseJWT(secret, tokenString string) (*jwt.RegisteredClaims, error) {
	claims := &jwt.RegisteredClaims{}
	_, err := jwt.ParseWithClaims(tokenString, claims, func(t *jwt.Token) (interface{}, error) {
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, jwt.ErrTokenSignatureInvalid
		}
		return []byte(secret), nil
	})
	if err != nil {
		return nil, err
	}
	return claims, nil
}

// BootstrapAdmin seeds the admin credentials into the settings table on first
// start (REQ-RES-05): if no admin_password_hash exists yet and an ADMIN_PASSWORD
// was provided, it stores the username and the bcrypt hash. Idempotent — a
// no-op once configured. With no password set, admin login stays unavailable.
func BootstrapAdmin(ctx context.Context, st store.Store, username, password string, log *zap.Logger) error {
	existing, err := st.GetSetting(ctx, models.SettingAdminPasswordHash)
	if err != nil {
		return err
	}
	if existing != "" {
		return nil
	}
	if password == "" {
		log.Warn("ADMIN_PASSWORD 未设置：管理后台登录不可用（设置后重启以启用）")
		return nil
	}
	hash, err := HashPassword(password)
	if err != nil {
		return err
	}
	if username == "" {
		username = "admin"
	}
	if err := st.SetSetting(ctx, models.SettingAdminUsername, username); err != nil {
		return err
	}
	if err := st.SetSetting(ctx, models.SettingAdminPasswordHash, hash); err != nil {
		return err
	}
	log.Info("管理员账号已初始化", zap.String("username", username))
	return nil
}

// BootstrapSettings seeds operational settings from env/config values on first
// start (only when absent), after which the DB settings table is authoritative
// and they are editable via the admin API. Idempotent.
func BootstrapSettings(ctx context.Context, st store.Store, retentionDays, offlineFactor int, log *zap.Logger) error {
	seed := func(key string, val int) error {
		cur, err := st.GetSetting(ctx, key)
		if err != nil {
			return err
		}
		if cur != "" {
			return nil // already set — DB is authoritative
		}
		return st.SetSetting(ctx, key, strconv.Itoa(val))
	}
	if err := seed(models.SettingRetentionDays, retentionDays); err != nil {
		return err
	}
	if err := seed(models.SettingOfflineFactor, offlineFactor); err != nil {
		return err
	}
	return nil
}
