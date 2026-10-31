package auth

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"github.com/google/uuid"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

type Claims struct {
	jwt.RegisteredClaims
	UserID    int32    `json:"user_id"`
	Roles     []string `json:"roles,omitempty"`
	SessionID string   `json:"session_id"`
	Purpose   string   `json:"token_use"`
}

func GenerateAccessToken(userID int32, roles []string, sessionID string) (string, time.Time, error) {
	exp := time.Now().Add(AccessTokenDuration)
	claims := Claims{
		RegisteredClaims: jwt.RegisteredClaims{
			ID:        uuid.New().String(),
			ExpiresAt: jwt.NewNumericDate(exp),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			Issuer:    "gym-jinni",
			Subject:   fmt.Sprintf("%d", userID),
		},
		UserID:    userID,
		Roles:     roles,
		SessionID: sessionID,
		Purpose:   "access",
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	signed, err := token.SignedString(JWTSecret())
	if err != nil {
		return "", time.Time{}, fmt.Errorf("sign access token: %w", err)
	}
	return signed, exp, nil
}

func GenerateRefreshToken(userID int32, sessionID string) (string, time.Time, error) {
	return GenerateRefreshTokenUntil(userID, sessionID, time.Now().Add(RefreshTokenDuration))
}

// GenerateRefreshTokenUntil keeps rotation bounded by the original session expiry.
func GenerateRefreshTokenUntil(userID int32, sessionID string, exp time.Time) (string, time.Time, error) {
	claims := Claims{
		RegisteredClaims: jwt.RegisteredClaims{
			ID:        uuid.New().String(),
			ExpiresAt: jwt.NewNumericDate(exp),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			Issuer:    "gym-jinni",
			Subject:   fmt.Sprintf("%d", userID),
		},
		UserID:    userID,
		SessionID: sessionID,
		Purpose:   "refresh",
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	signed, err := token.SignedString(JWTSecret())
	if err != nil {
		return "", time.Time{}, fmt.Errorf("sign refresh token: %w", err)
	}
	return signed, exp, nil
}

func ValidateToken(tokenString string, purpose string) (*Claims, error) {
	token, err := jwt.ParseWithClaims(tokenString, &Claims{}, func(t *jwt.Token) (interface{}, error) {
		if t.Method != jwt.SigningMethodHS256 {
			return nil, fmt.Errorf("unexpected signing method: %v", t.Header["alg"])
		}
		return JWTSecret(), nil
	}, jwt.WithValidMethods([]string{"HS256"}), jwt.WithIssuer("gym-jinni"), jwt.WithExpirationRequired())
	if err != nil {
		return nil, fmt.Errorf("parse token: %w", err)
	}
	claims, ok := token.Claims.(*Claims)
	if !ok || !token.Valid || claims.UserID <= 0 || claims.SessionID == "" || claims.Purpose != purpose || claims.Subject != fmt.Sprint(claims.UserID) {
		return nil, fmt.Errorf("invalid token claims")
	}
	return claims, nil
}

// RefreshTokenHash is the only persisted representation of a refresh credential.
func RefreshTokenHash(token string) string {
	sum := sha256.Sum256([]byte(token))
	return hex.EncodeToString(sum[:])
}
