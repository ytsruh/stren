// Package utils provides shared utility functions used across the application.
package utils

import (
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

const CookieName = "auth_token"

// Claims represents the JWT claims for an authenticated user.
type Claims struct {
	UserID  int64  `json:"user_id"`
	Email   string `json:"email"`
	Name    string `json:"name"`
	IsAdmin bool   `json:"is_admin"`
	jwt.RegisteredClaims
}

// JWTService handles signing and verifying JWT tokens.
type JWTService struct {
	secret []byte
}

// NewJWTService creates a new JWTService with the given secret.
func NewJWTService(secret string) *JWTService {
	return &JWTService{secret: []byte(secret)}
}

// GenerateToken creates a new JWT token for the given user details.
// Tokens expire after 7 days.
func (s *JWTService) GenerateToken(userID int64, email, name string, isAdmin bool) (string, error) {
	claims := Claims{
		UserID:  userID,
		Email:   email,
		Name:    name,
		IsAdmin: isAdmin,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(7 * 24 * time.Hour)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(s.secret)
}

// VerifyToken parses and validates a JWT token string.
// It returns the claims if the token is valid.
func (s *JWTService) VerifyToken(tokenString string) (*Claims, error) {
	token, err := jwt.ParseWithClaims(tokenString, &Claims{}, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return s.secret, nil
	})
	if err != nil {
		return nil, fmt.Errorf("invalid token: %w", err)
	}

	claims, ok := token.Claims.(*Claims)
	if !ok || !token.Valid {
		return nil, fmt.Errorf("invalid token claims")
	}

	return claims, nil
}
