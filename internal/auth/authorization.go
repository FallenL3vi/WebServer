package auth

import (
	"golang.org/x/crypto/bcrypt"
	"fmt"
)

func HashPassword(password string) (string, error) {
 hashed, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)

 if err != nil {
	return "", err
 }

 return string(hashed), nil
}

func CheckPasswordHash(hash, password string) error {
	fmt.Printf(hash)
	fmt.Printf("\n")
	fmt.Printf(password)
	err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(password))

	return err
}