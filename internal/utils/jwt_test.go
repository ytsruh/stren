package utils

import (
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

func TestJWTService_GenerateAndVerify(t *testing.T) {
	service := NewJWTService("test-secret")

	token, err := service.GenerateToken("user-123", "alice@example.com", "Alice", true)
	if err != nil {
		t.Fatalf("GenerateToken failed: %v", err)
	}
	if token == "" {
		t.Fatal("expected non-empty token")
	}

	claims, err := service.VerifyToken(token)
	if err != nil {
		t.Fatalf("VerifyToken failed: %v", err)
	}
	if claims.UserID != "user-123" {
		t.Fatalf("expected user_id 'user-123', got %q", claims.UserID)
	}
	if claims.Email != "alice@example.com" {
		t.Fatalf("expected email 'alice@example.com', got %q", claims.Email)
	}
	if claims.Name != "Alice" {
		t.Fatalf("expected name 'Alice', got %q", claims.Name)
	}
	if !claims.IsAdmin {
		t.Fatal("expected is_admin to be true")
	}
}

func TestJWTService_VerifyToken_Expired(t *testing.T) {
	service := NewJWTService("test-secret")

	claims := Claims{
		UserID:  "user-456",
		Email:   "test@example.com",
		Name:    "Test",
		IsAdmin: false,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(-1 * time.Hour)),
			IssuedAt:  jwt.NewNumericDate(time.Now().Add(-2 * time.Hour)),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenString, err := token.SignedString([]byte("test-secret"))
	if err != nil {
		t.Fatalf("failed to sign expired token: %v", err)
	}

	_, err = service.VerifyToken(tokenString)
	if err == nil {
		t.Fatal("expected error for expired token, got nil")
	}
}

func TestJWTService_VerifyToken_InvalidSecret(t *testing.T) {
	service1 := NewJWTService("secret-one")
	service2 := NewJWTService("secret-two")

	token, err := service1.GenerateToken("user-789", "test@example.com", "Test", false)
	if err != nil {
		t.Fatalf("GenerateToken failed: %v", err)
	}

	_, err = service2.VerifyToken(token)
	if err == nil {
		t.Fatal("expected error for token signed with different secret")
	}
}

func TestJWTService_VerifyToken_Malformed(t *testing.T) {
	service := NewJWTService("test-secret")

	_, err := service.VerifyToken("not-a-valid-token")
	if err == nil {
		t.Fatal("expected error for malformed token")
	}
}
