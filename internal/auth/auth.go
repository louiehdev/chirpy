package auth

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/alexedwards/argon2id"
	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

func HashPassword(password string) (string, error) {
	hash, err := argon2id.CreateHash(password, argon2id.DefaultParams)
	if err != nil {
		log.Fatal(err)
	}
	return hash, nil
}

func CheckPasswordHash(password, hash string) (bool, error) {
	match, err := argon2id.ComparePasswordAndHash(password, hash)
	if err != nil {
		log.Fatal(err)
	}
	return match, nil
}

func MakeJWT(userID uuid.UUID, tokenSecret string, expiresIn time.Duration) (string, error) {
	token := jwt.NewWithClaims(
		jwt.SigningMethodHS256,
		jwt.RegisteredClaims{
			Issuer:    "chirpy",
			Subject:   userID.String(),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(expiresIn))})
	signedString, err := token.SignedString([]byte(tokenSecret))
	if err != nil {
		return "", err
	}
	return signedString, nil
}

func ValidateJWT(tokenString, tokenSecret string) (uuid.UUID, error) {
	claims := &jwt.RegisteredClaims{}
	token, err := jwt.ParseWithClaims(tokenString, claims, func(token *jwt.Token) (any, error) { return []byte(tokenSecret), nil })
	if err != nil {
		return uuid.Nil, err
	}
	if expTime, err := token.Claims.GetExpirationTime(); err != nil || time.Now().After(expTime.Time) {
		return uuid.Nil, err
	}
	userID, err := token.Claims.GetSubject()
	if err != nil {
		return uuid.Nil, err
	}
	userUUID, err := uuid.Parse(userID)
	if err != nil {
		return uuid.Nil, err
	}
	return userUUID, nil
}

func GetBearerToken(headers http.Header) (string, error) {
	tokenString := headers.Get("Authorization")
	if len(tokenString) == 0 {
		return "", fmt.Errorf("no token string found")
	}

	return strings.TrimPrefix(tokenString, "Bearer "), nil
}

func GetAPIKey(headers http.Header) (string, error) {
	keyString := headers.Get("Authorization")
	if len(keyString) == 0 {
		return "", fmt.Errorf("no token string found")
	}

	return strings.TrimPrefix(keyString, "ApiKey "), nil
}

func MakeRefreshToken() string {
	key := make([]byte, 32)
	rand.Read(key)
	token := hex.EncodeToString(key)

	return token
}
