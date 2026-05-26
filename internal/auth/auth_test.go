package auth

import (
	"testing"
	"time"

	"github.com/google/uuid"
)

func TestMakeAndValidateJWT(t *testing.T) {
	userID := uuid.New()
	secret := "my-secret"

	token, err := MakeJWT(userID, secret, time.Hour)
	if err != nil {
		t.Fatalf("failed to create jwt: %v", err)
	}

	gotID, err := ValidateJWT(token, secret)
	if err != nil {
		t.Fatalf("failed to validate jwt: %v", err)
	}

	if gotID != userID {
		t.Errorf("expected userID %v, got %v", userID, gotID)
	}
}

func TestExpiredJWT(t *testing.T) {
	userID := uuid.New()
	secret := "my-secret"

	token, err := MakeJWT(userID, secret, -time.Hour)
	if err != nil {
		t.Fatalf("failed to create jwt: %v", err)
	}

	_, err = ValidateJWT(token, secret)

	if err == nil {
		t.Fatal("expected error for expired token, got nil")
	}
}

func TestJWTWrongSecret(t *testing.T) {
	userID := uuid.New()

	token, err := MakeJWT(userID, "correct-secret", time.Hour)
	if err != nil {
		t.Fatalf("failed to create jwt: %v", err)
	}

	_, err = ValidateJWT(token, "wrong-secret")

	if err == nil {
		t.Fatal("expected error for wrong secret, got nil")
	}
}