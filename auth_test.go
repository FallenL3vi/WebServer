package main

import (
	"github.com/FallenL3vi/WebServer/internal/auth"
	"testing"
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

	err = auth.CheckPasswordHash(hash, "BOBY1234")

	if err != nil {
		t.Errorf("Failed password hash check\n")
	}
}
