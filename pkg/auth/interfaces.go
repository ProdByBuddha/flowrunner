// Package auth provides authentication and authorization functionality.
package auth

import (
	"time"
)

// AccountService manages accounts and authentication
type AccountService interface {
	// Authenticate verifies credentials and returns an account ID
	Authenticate(username, password string) (string, error)

	// ValidateToken verifies a bearer token and returns an account ID
	ValidateToken(token string) (string, error)

	// CreateAccount creates a new account
	CreateAccount(username, password string) (string, error)

	// DeleteAccount removes an account
	DeleteAccount(accountID string) error

	// GetAccount retrieves account information
	GetAccount(accountID string) (Account, error)

	// ListAccounts returns all accounts (admin only)
	ListAccounts() ([]Account, error)
}

// Account represents a tenant in the system
type Account struct {
	// ID of the account
	ID string `json:"id"`

	// Username for the account
	Username string `json:"username"`

	// PasswordHash is the hashed password (not exposed via API)
	PasswordHash string `json:"-"`

	// APIToken for authentication
	APIToken string `json:"-"`

	// CreatedAt is when the account was created
	CreatedAt time.Time `json:"created_at"`

	// UpdatedAt is when the account was last updated
	UpdatedAt time.Time `json:"updated_at"`
}

// SecretVault manages per-account secrets
type SecretVault interface {
	// Set stores an encrypted secret for an account
	Set(accountID string, key string, value string) error

	// Get retrieves and decrypts a secret for an account
	Get(accountID string, key string) (string, error)

	// Delete removes a secret
	Delete(accountID string, key string) error

	// List returns all secret keys for an account (without values)
	List(accountID string) ([]string, error)

	// RotateEncryptionKey changes the encryption key for all secrets
	RotateEncryptionKey(oldKey, newKey []byte) error
}

// Secret represents a stored credential
type Secret struct {
	// AccountID is the ID of the account that owns the secret
	AccountID string `json:"-"`

	// Key is the name of the secret
	Key string `json:"key"`

	// Value is the encrypted secret value (not exposed via API)
	Value string `json:"-"`

	// CreatedAt is when the secret was created
	CreatedAt time.Time `json:"created_at"`

	// UpdatedAt is when the secret was last updated
	UpdatedAt time.Time `json:"updated_at"`
}
