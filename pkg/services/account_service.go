package services

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"time"

	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"

	"github.com/tcmartin/flowrunner/pkg/auth"
	"github.com/tcmartin/flowrunner/pkg/storage"
)

// AccountService implements the auth.AccountService interface
type AccountService struct {
	store storage.AccountStore
}

// NewAccountService creates a new account service with the given storage backend
func NewAccountService(store storage.AccountStore) *AccountService {
	return &AccountService{
		store: store,
	}
}

// Authenticate verifies credentials and returns an account ID
func (s *AccountService) Authenticate(username, password string) (string, error) {
	if username == "" || password == "" {
		return "", fmt.Errorf("username and password are required")
	}

	// Get account by username
	account, err := s.store.GetAccountByUsername(username)
	if err != nil {
		return "", fmt.Errorf("authentication failed")
	}

	// Verify password
	err = bcrypt.CompareHashAndPassword([]byte(account.PasswordHash), []byte(password))
	if err != nil {
		return "", fmt.Errorf("authentication failed")
	}

	return account.ID, nil
}

// ValidateToken verifies a bearer token and returns an account ID
func (s *AccountService) ValidateToken(token string) (string, error) {
	if token == "" {
		return "", fmt.Errorf("token is required")
	}

	// Get account by token
	account, err := s.store.GetAccountByToken(token)
	if err != nil {
		return "", fmt.Errorf("invalid token")
	}

	return account.ID, nil
}

// CreateAccount creates a new account
func (s *AccountService) CreateAccount(username, password string) (string, error) {
	if username == "" || password == "" {
		return "", fmt.Errorf("username and password are required")
	}

	// Check if username already exists
	_, err := s.store.GetAccountByUsername(username)
	if err == nil {
		return "", fmt.Errorf("username already exists")
	}
	if err != storage.ErrAccountNotFound {
		return "", fmt.Errorf("failed to check username availability: %w", err)
	}

	// Hash password
	passwordHash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return "", fmt.Errorf("failed to hash password: %w", err)
	}

	// Generate API token
	apiToken, err := generateAPIToken()
	if err != nil {
		return "", fmt.Errorf("failed to generate API token: %w", err)
	}

	// Create account
	accountID := uuid.New().String()
	now := time.Now()
	account := auth.Account{
		ID:           accountID,
		Username:     username,
		PasswordHash: string(passwordHash),
		APIToken:     apiToken,
		CreatedAt:    now,
		UpdatedAt:    now,
	}

	// Save account
	err = s.store.SaveAccount(account)
	if err != nil {
		return "", fmt.Errorf("failed to save account: %w", err)
	}

	return accountID, nil
}

// DeleteAccount removes an account
func (s *AccountService) DeleteAccount(accountID string) error {
	if accountID == "" {
		return fmt.Errorf("account ID is required")
	}

	err := s.store.DeleteAccount(accountID)
	if err != nil {
		return fmt.Errorf("failed to delete account: %w", err)
	}

	return nil
}

// GetAccount retrieves account information
func (s *AccountService) GetAccount(accountID string) (auth.Account, error) {
	if accountID == "" {
		return auth.Account{}, fmt.Errorf("account ID is required")
	}

	account, err := s.store.GetAccount(accountID)
	if err != nil {
		return auth.Account{}, fmt.Errorf("failed to get account: %w", err)
	}

	return account, nil
}

// ListAccounts returns all accounts (admin only)
func (s *AccountService) ListAccounts() ([]auth.Account, error) {
	accounts, err := s.store.ListAccounts()
	if err != nil {
		return nil, fmt.Errorf("failed to list accounts: %w", err)
	}

	return accounts, nil
}

// generateAPIToken generates a secure random API token
func generateAPIToken() (string, error) {
	bytes := make([]byte, 32) // 256 bits
	_, err := rand.Read(bytes)
	if err != nil {
		return "", err
	}
	return hex.EncodeToString(bytes), nil
}
