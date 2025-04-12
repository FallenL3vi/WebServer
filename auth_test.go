package main

import (
	"github.com/FallenL3vi/WebServer/internal/auth"
	"testing"
	"time"
	"github.com/google/uuid"
)

func TestHash(t *testing.T) {
	password := "BOBY123"

	_, err := auth.HashPassword(password)

	if err != nil {
		t.Errorf("Failed hashing\n")
	}
}

func TestCheckHash(t *testing.T) {
	password := "BOBY123"
	hash, err := auth.HashPassword(password)

	err = auth.CheckPasswordHash(hash, "BOBY123")

	if err != nil {
		t.Errorf("Failed password hash check\n")
	}
}


func TestValidJWT(t *testing.T) {
	userID := uuid.New()
	validToken, _ := auth.MakeJWT(userID, "secret", time.Hour)

	tests := []struct {
		name string
		tokenString string
		tokenSecret string
		wantUserID uuid.UUID
		wantErr bool
	}{
		{
			name:        "Valid token",
			tokenString: validToken,
			tokenSecret: "secret",
			wantUserID:  userID,
			wantErr:     false,
		},
		{
			name:        "Invalid token",
			tokenString: "invalid.token.string",
			tokenSecret: "secret",
			wantUserID:  uuid.Nil,
			wantErr:     true,
		},
		{
			name:        "Wrong secret",
			tokenString: validToken,
			tokenSecret: "wrong_secret",
			wantUserID:  uuid.Nil,
			wantErr:     true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotUserID, err := auth.ValidateJWT(tt.tokenString, tt.tokenSecret)
			if(err != nil) != tt.wantErr {
				t.Errorf("ValidateJWT() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if gotUserID != tt.wantUserID {
				t.Errorf("ValidateJWT() gotUserID = %v, want %v", gotUserID, tt.wantUserID)
			}
		}) 
	}
}