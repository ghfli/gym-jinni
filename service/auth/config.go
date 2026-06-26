package auth

import (
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
