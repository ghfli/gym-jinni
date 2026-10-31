package auth

import (
	"github.com/golang-jwt/jwt/v5"
	"testing"
	"time"
)

func TestTokenPurposeAndSession(t *testing.T) {
	t.Setenv("JWT_SECRET", "unit-test-signing-secret-at-least-32-bytes")
	access, _, err := GenerateAccessToken(7, nil, "session-one")
	if err != nil {
		t.Fatal(err)
	}
	refresh, _, err := GenerateRefreshToken(7, "session-one")
	if err != nil {
		t.Fatal(err)
	}
	for _, test := range []struct {
		name, token, purpose string
		valid                bool
	}{{"access", access, "access", true}, {"refresh", refresh, "refresh", true}, {"access_as_refresh", access, "refresh", false}, {"refresh_as_access", refresh, "access", false}} {
		t.Run(test.name, func(t *testing.T) {
			claims, err := ValidateToken(test.token, test.purpose)
			if (err == nil) != test.valid {
				t.Fatalf("valid=%t error=%v", test.valid, err)
			}
			if err == nil && (claims.SessionID != "session-one" || claims.UserID != 7) {
				t.Fatal("session identity lost")
			}
		})
	}
}
func TestRejectInvalidClaims(t *testing.T) {
	t.Setenv("JWT_SECRET", "unit-test-signing-secret-at-least-32-bytes")
	for _, name := range []string{"missing_expiry", "expired", "wrong_issuer", "missing_session", "wrong_subject", "wrong_algorithm"} {
		t.Run(name, func(t *testing.T) {
			c := Claims{RegisteredClaims: jwt.RegisteredClaims{Issuer: "gym-jinni", Subject: "7", ExpiresAt: jwt.NewNumericDate(time.Now().Add(time.Minute))}, UserID: 7, SessionID: "session", Purpose: "access"}
			method := jwt.SigningMethodHS256
			switch name {
			case "missing_expiry":
				c.ExpiresAt = nil
			case "expired":
				c.ExpiresAt = jwt.NewNumericDate(time.Now().Add(-time.Minute))
			case "wrong_issuer":
				c.Issuer = "other"
			case "missing_session":
				c.SessionID = ""
			case "wrong_subject":
				c.Subject = "8"
			case "wrong_algorithm":
				method = jwt.SigningMethodHS384
			}
			token, err := jwt.NewWithClaims(method, c).SignedString(JWTSecret())
			if err != nil {
				t.Fatal(err)
			}
			if _, err := ValidateToken(token, "access"); err == nil {
				t.Fatal("invalid claims accepted")
			}
		})
	}
}

func TestSigningConfiguration(t *testing.T) {
	for _, test := range []struct {
		name, env, key string
		valid          bool
	}{{"missing", "", "", false}, {"default", "production", "gym-jinni-dev-secret-change-in-prod", false}, {"short", "production", "short", false}, {"explicit_development", "development", "", true}, {"deployment", "production", "a-dedicated-deployment-secret-32-bytes", true}} {
		t.Run(test.name, func(t *testing.T) {
			t.Setenv("APP_ENV", test.env)
			t.Setenv("JWT_SECRET", test.key)
			if err := ValidateConfiguration(); (err == nil) != test.valid {
				t.Fatalf("configuration valid=%t error=%v", test.valid, err)
			}
		})
	}
}

func TestRefreshCredentialsAreUnique(t *testing.T) {
	t.Setenv("JWT_SECRET", "unit-test-signing-secret-at-least-32-bytes")
	first, _, err := GenerateRefreshToken(7, "same-session")
	if err != nil {
		t.Fatal(err)
	}
	second, _, err := GenerateRefreshToken(7, "same-session")
	if err != nil {
		t.Fatal(err)
	}
	if first == second {
		t.Fatal("rotation reused a refresh credential")
	}
	if len(RefreshTokenHash(first)) != 64 || RefreshTokenHash(first) == RefreshTokenHash(second) {
		t.Fatal("invalid refresh credential hash")
	}
}
