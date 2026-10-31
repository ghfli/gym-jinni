package auth

import (
	"fmt"
	"os"
	"time"
)

var (
	AccessTokenDuration  = 15 * time.Minute
	RefreshTokenDuration = 24 * time.Hour * 7
)

func JWTSecret() []byte {
	secret := os.Getenv("JWT_SECRET")
	if secret == "" {
		secret = "gym-jinni-dev-secret-change-in-prod"
	}
	return []byte(secret)
}

// Development fallback is opt-in; deployments must supply their own signing key.
func ValidateConfiguration() error {
	if os.Getenv("APP_ENV") == "development" {
		return nil
	}
	key := os.Getenv("JWT_SECRET")
	if key == "" || key == "gym-jinni-dev-secret-change-in-prod" || len(key) < 32 {
		return fmt.Errorf("set JWT_SECRET to a non-development key of at least 32 bytes, or explicitly set APP_ENV=development for local use")
	}
	return nil
}
