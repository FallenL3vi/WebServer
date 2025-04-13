package auth

import (
	"golang.org/x/crypto/bcrypt"
	"github.com/google/uuid"
	"time"
	"errors"
	"fmt"
	"strings"
	"github.com/golang-jwt/jwt/v5"
	"net/http"
	"crypto/rand"
	"encoding/hex"
)

type TokenType string
const (
	TokenTypeAccess TokenType = "chirpy-access"
)

func HashPassword(password string) (string, error) {
 hashed, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)

 if err != nil {
	return "", err
 }

 return string(hashed), nil
}

func CheckPasswordHash(hash, password string) error {
	err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(password))

	return err
}

func MakeJWT(userID uuid.UUID, tokenSecret string, expiresIn time.Duration) (string, error) {
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.RegisteredClaims{
		ExpiresAt: jwt.NewNumericDate(time.Now().UTC().Add(expiresIn)),
		IssuedAt: jwt.NewNumericDate(time.Now().UTC()),
		Issuer: string(TokenTypeAccess),
		Subject: userID.String(),
	})

	return token.SignedString([]byte(tokenSecret))
}

func ValidateJWT(tokenString, tokenSecret string) (uuid.UUID, error) {
	claims := jwt.RegisteredClaims{}
	token, err := jwt.ParseWithClaims(tokenString, &claims, func(token *jwt.Token) (interface{}, error) {
		return []byte(tokenSecret), nil
	},)

	if err != nil {
		return uuid.Nil, err
	}

	userClaimsID, err := token.Claims.GetSubject()

	if err != nil {
		return uuid.Nil, err
	}

	issuer, err := token.Claims.GetIssuer()

	if err != nil {
		return uuid.Nil ,err
	}

	if issuer != string(TokenTypeAccess) {
		return uuid.Nil, errors.New("invalid issuer")
	}

	id, err := uuid.Parse(userClaimsID)

	if err != nil {
		return uuid.Nil, fmt.Errorf("invalid user ID: %w", err)
	}

	return id, nil
}

func GetBearerToken(headers http.Header) (string, error) {
	header := headers.Get("Authorization")

	if header == "" {
		return "", errors.New("Missing header")
	}
	token := strings.TrimSpace(strings.TrimPrefix(header,"Bearer"))

	if token == "" {
		return "", errors.New("Missing token")
	}

	fmt.Println(token)
	return token, nil

}

func MakeRefreshToken() (string, error) {

	key := make([]byte, 32)
	rand.Read(key)

	encodedStr := hex.EncodeToString(key)

	return encodedStr, nil
}