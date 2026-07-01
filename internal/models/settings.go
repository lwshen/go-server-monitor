package models

// Well-known settings keys (REQ-RES-01).
const (
	SettingAdminUsername     = "admin_username"
	SettingAdminPasswordHash = "admin_password_hash"
	SettingSchemaVersion     = "schema_version"
)

// SecretSettingKeys are never returned in plaintext by GET /api/admin/settings;
// they surface as a boolean "<key>_set" marker instead (REQ-RES-01).
var SecretSettingKeys = map[string]bool{
	"api_secret":          true,
	"jwt_secret":          true,
	"admin_password_hash": true,
	"captcha_secret":      true,
	"telegram_bot_token":  true,
}

// WriteProtectedSettingKeys cannot be set via POST /api/admin/settings — they are
// bootstrapped from .env or managed by the system (REQ-RES-01).
var WriteProtectedSettingKeys = map[string]bool{
	"api_secret":          true,
	"jwt_secret":          true,
	"admin_password_hash": true,
	SettingSchemaVersion:  true,
}
