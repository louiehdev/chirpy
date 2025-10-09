package auth_test

import (
	"strings"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"github.com/louiehdev/chirpy/internal/auth"
)

func TestMakeAndValidateJWT(t *testing.T) {
	secret := "supersecret"
	userID := uuid.New()

	t.Run("valid token roundtrip", func(t *testing.T) {
		tokenString, err := auth.MakeJWT(userID, secret, time.Hour)
		if err != nil {
			t.Fatalf("MakeJWT returned error: %v", err)
		}
		if tokenString == "" {
			t.Fatal("expected non-empty token string")
		}

		validatedID, err := auth.ValidateJWT(tokenString, secret)
		if err != nil {
			t.Fatalf("ValidateJWT returned error: %v", err)
		}
		if validatedID != userID {
			t.Fatalf("expected userID %v, got %v", userID, validatedID)
		}
	})

	t.Run("expired token", func(t *testing.T) {
		tokenString, err := auth.MakeJWT(userID, secret, -time.Minute)
		if err != nil {
			t.Fatalf("MakeJWT returned error: %v", err)
		}

		_, err = auth.ValidateJWT(tokenString, secret)
		if err == nil {
			t.Fatal("expected error for expired token, got nil")
		}
		if !strings.Contains(err.Error(), "token is expired") {
			t.Errorf("expected expiration error, got %v", err)
		}
	})

	t.Run("invalid signature", func(t *testing.T) {
		tokenString, err := auth.MakeJWT(userID, secret, time.Hour)
		if err != nil {
			t.Fatalf("MakeJWT returned error: %v", err)
		}

		_, err = auth.ValidateJWT(tokenString, "wrongsecret")
		if err == nil {
			t.Fatal("expected error for invalid signature, got nil")
		}
	})

	t.Run("malformed token", func(t *testing.T) {
		_, err := auth.ValidateJWT("not.a.jwt", secret)
		if err == nil {
			t.Fatal("expected error for malformed token, got nil")
		}
	})
}

func TestMakeJWTFields(t *testing.T) {
	secret := "anothersecret"
	userID := uuid.New()
	expiresIn := 2 * time.Hour

	tokenString, err := auth.MakeJWT(userID, secret, expiresIn)
	if err != nil {
		t.Fatalf("MakeJWT returned error: %v", err)
	}

	token, err := jwt.ParseWithClaims(tokenString, &jwt.RegisteredClaims{}, func(token *jwt.Token) (any, error) {
		return []byte(secret), nil
	})
	if err != nil {
		t.Fatalf("failed to parse token: %v", err)
	}

	claims, ok := token.Claims.(*jwt.RegisteredClaims)
	if !ok {
		t.Fatal("expected RegisteredClaims type")
	}

	if claims.Issuer != "chirpy" {
		t.Errorf("expected issuer 'chirpy', got %q", claims.Issuer)
	}
	if claims.Subject != userID.String() {
		t.Errorf("expected subject %q, got %q", userID.String(), claims.Subject)
	}

	expectedExpiry := time.Now().Add(expiresIn)
	diff := time.Until(claims.ExpiresAt.Time)
	if diff < 0 || diff > expiresIn {
		t.Errorf("expiresAt not within expected range: got %v, expected around %v", claims.ExpiresAt.Time, expectedExpiry)
	}
}
