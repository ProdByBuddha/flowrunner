package services

import (
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/tcmartin/flowrunner/pkg/auth"
)

// JWTService handles JWT token generation and validation
type JWTService struct {
	secret          string
	tokenExpiration time.Duration
}

// NewJWTService creates a new JWT service
func NewJWTService(secret string, expirationHours int) *JWTService {
	return &JWTService{
		secret:          secret,
		tokenExpiration: time.Duration(expirationHours) * time.Hour,
	}
}

// Claims represents the JWT claims
type Claims struct {
	AccountID string `json:"account_id"`
	Username  string `json:"username"`
	jwt.RegisteredClaims
}

// GenerateToken generates a JWT token for an account
func (s *JWTService) GenerateToken(account auth.Account) (string, error) {
	// Create the claims
	claims := Claims{
		AccountID: account.ID,
		Username:  account.Username,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(s.tokenExpiration)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			NotBefore: jwt.NewNumericDate(time.Now()),
			Issuer:    "flowrunner",
			Subject:   account.ID,
		},
	}

	// Create the token
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)

	// Sign the token
	tokenString, err := token.SignedString([]byte(s.secret))
	if err != nil {
		return "", fmt.Errorf("failed to sign token: %w", err)
	}

	return tokenString, nil
}

// ValidateToken validates a JWT token and returns the account ID
func (s *JWTService) ValidateToken(tokenString string) (string, error) {
	// Parse the token
	token, err := jwt.ParseWithClaims(tokenString, &Claims{}, func(token *jwt.Token) (interface{}, error) {
		// Validate the signing method
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}

		return []byte(s.secret), nil
	})

	if err != nil {
		return "", fmt.Errorf("invalid token: %w", err)
	}

	// Get the claims
	if claims, ok := token.Claims.(*Claims); ok && token.Valid {
		return claims.AccountID, nil
	}

	return "", fmt.Errorf("invalid token claims")
}
