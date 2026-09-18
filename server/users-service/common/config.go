package common

import "os"

var (
	Issuer = getenv("JWT_ISSUER", "VideoConf")
	// No default on purpose: main fails fast when JWT_SECRET is unset.
	JwtSecretPassword = os.Getenv("JWT_SECRET")
	MgDBName          = getenv("DB_NAME", "vidchat")
	MgAddress         = getenv("DB_HOST", "127.0.0.1") + ":" + getenv("DB_PORT", "27017")
	MgUsername        = getenv("DB_USERNAME", "root")
	MgPassword        = getenv("DB_PASSWORD", "rootpassword")
	UsersCol          = "users"
)

func getenv(key, fallback string) string {
	value := os.Getenv(key)
	if len(value) == 0 {
		return fallback
	}
	return value
}
