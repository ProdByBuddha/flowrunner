package services

import (
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/crypto/bcrypt"

	"github.com/tcmartin/flowrunner/pkg/auth"
	"github.com/tcmartin/flowrunner/pkg/storage"
)

func TestNewAccountService(t *testing.T) {
	store := storage.NewMemoryAccountStore()
	service := NewAccountService(store)
	
	assert.NotNil(t, service)
	assert.Equal(t, store, service.store)
}

func TestAccountService_CreateAccount(t *testing.T) {
	store := storage.NewMemoryAccountStore()
	service := NewAccountService(store)

	t.Run("successful creation", func(t *testing.T) {
		username := "testuser"
		password := "testpassword"

		accountID, err := service.CreateAccount(username, password)
		
		require.NoError(t, err)
		assert.NotEmpty(t, accountID)

		// Verify account was saved
		account, err := service.GetAccount(accountID)
		require.NoError(t, err)
		assert.Equal(t, accountID, account.ID)
		assert.Equal(t, username, account.Username)
		assert.NotEmpty(t, account.PasswordHash)
		assert.NotEmpty(t, account.APIToken)
		assert.False(t, account.CreatedAt.IsZero())
		assert.False(t, account.UpdatedAt.IsZero())

		// Verify password was hashed
		assert.NotEqual(t, password, account.PasswordHash)
		err = bcrypt.CompareHashAndPassword([]byte(account.PasswordHash), []byte(password))
		assert.NoError(t, err)
	})

	t.Run("empty username", func(t *testing.T) {
		accountID, err := service.CreateAccount("", "password")
		
		assert.Error(t, err)
		assert.Empty(t, accountID)
		assert.Contains(t, err.Error(), "username and password are required")
	})

	t.Run("empty password", func(t *testing.T) {
		accountID, err := service.CreateAccount("username", "")
		
		assert.Error(t, err)
		assert.Empty(t, accountID)
		assert.Contains(t, err.Error(), "username and password are required")
	})

	t.Run("duplicate username", func(t *testing.T) {
		username := "duplicate"
		password := "password"

		// Create first account
		accountID1, err := service.CreateAccount(username, password)
		require.NoError(t, err)
		assert.NotEmpty(t, accountID1)

		// Try to create second account with same username
		accountID2, err := service.CreateAccount(username, password)
		assert.Error(t, err)
		assert.Empty(t, accountID2)
		assert.Contains(t, err.Error(), "username already exists")
	})
}

func TestAccountService_Authenticate(t *testing.T) {
	store := storage.NewMemoryAccountStore()
	service := NewAccountService(store)

	// Create test account
	username := "testuser"
	password := "testpassword"
	accountID, err := service.CreateAccount(username, password)
	require.NoError(t, err)

	t.Run("successful authentication", func(t *testing.T) {
		authAccountID, err := service.Authenticate(username, password)
		
		require.NoError(t, err)
		assert.Equal(t, accountID, authAccountID)
	})

	t.Run("wrong password", func(t *testing.T) {
		authAccountID, err := service.Authenticate(username, "wrongpassword")
		
		assert.Error(t, err)
		assert.Empty(t, authAccountID)
		assert.Contains(t, err.Error(), "authentication failed")
	})

	t.Run("non-existent username", func(t *testing.T) {
		authAccountID, err := service.Authenticate("nonexistent", password)
		
		assert.Error(t, err)
		assert.Empty(t, authAccountID)
		assert.Contains(t, err.Error(), "authentication failed")
	})

	t.Run("empty username", func(t *testing.T) {
		authAccountID, err := service.Authenticate("", password)
		
		assert.Error(t, err)
		assert.Empty(t, authAccountID)
		assert.Contains(t, err.Error(), "username and password are required")
	})

	t.Run("empty password", func(t *testing.T) {
		authAccountID, err := service.Authenticate(username, "")
		
		assert.Error(t, err)
		assert.Empty(t, authAccountID)
		assert.Contains(t, err.Error(), "username and password are required")
	})
}

func TestAccountService_ValidateToken(t *testing.T) {
	store := storage.NewMemoryAccountStore()
	service := NewAccountService(store)

	// Create test account
	username := "testuser"
	password := "testpassword"
	accountID, err := service.CreateAccount(username, password)
	require.NoError(t, err)

	// Get the API token
	account, err := service.GetAccount(accountID)
	require.NoError(t, err)
	apiToken := account.APIToken

	t.Run("valid token", func(t *testing.T) {
		validatedAccountID, err := service.ValidateToken(apiToken)
		
		require.NoError(t, err)
		assert.Equal(t, accountID, validatedAccountID)
	})

	t.Run("invalid token", func(t *testing.T) {
		validatedAccountID, err := service.ValidateToken("invalid-token")
		
		assert.Error(t, err)
		assert.Empty(t, validatedAccountID)
		assert.Contains(t, err.Error(), "invalid token")
	})

	t.Run("empty token", func(t *testing.T) {
		validatedAccountID, err := service.ValidateToken("")
		
		assert.Error(t, err)
		assert.Empty(t, validatedAccountID)
		assert.Contains(t, err.Error(), "token is required")
	})
}

func TestAccountService_GetAccount(t *testing.T) {
	store := storage.NewMemoryAccountStore()
	service := NewAccountService(store)

	// Create test account
	username := "testuser"
	password := "testpassword"
	accountID, err := service.CreateAccount(username, password)
	require.NoError(t, err)

	t.Run("existing account", func(t *testing.T) {
		account, err := service.GetAccount(accountID)
		
		require.NoError(t, err)
		assert.Equal(t, accountID, account.ID)
		assert.Equal(t, username, account.Username)
		assert.NotEmpty(t, account.PasswordHash)
		assert.NotEmpty(t, account.APIToken)
	})

	t.Run("non-existent account", func(t *testing.T) {
		account, err := service.GetAccount("non-existent")
		
		assert.Error(t, err)
		assert.Equal(t, auth.Account{}, account)
		assert.Contains(t, err.Error(), "failed to get account")
	})

	t.Run("empty account ID", func(t *testing.T) {
		account, err := service.GetAccount("")
		
		assert.Error(t, err)
		assert.Equal(t, auth.Account{}, account)
		assert.Contains(t, err.Error(), "account ID is required")
	})
}

func TestAccountService_DeleteAccount(t *testing.T) {
	store := storage.NewMemoryAccountStore()
	service := NewAccountService(store)

	// Create test account
	username := "testuser"
	password := "testpassword"
	accountID, err := service.CreateAccount(username, password)
	require.NoError(t, err)

	t.Run("existing account", func(t *testing.T) {
		err := service.DeleteAccount(accountID)
		
		require.NoError(t, err)

		// Verify account was deleted
		_, err = service.GetAccount(accountID)
		assert.Error(t, err)
	})

	t.Run("non-existent account", func(t *testing.T) {
		err := service.DeleteAccount("non-existent")
		
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "failed to delete account")
	})

	t.Run("empty account ID", func(t *testing.T) {
		err := service.DeleteAccount("")
		
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "account ID is required")
	})
}

func TestAccountService_ListAccounts(t *testing.T) {
	store := storage.NewMemoryAccountStore()
	service := NewAccountService(store)

	t.Run("empty store", func(t *testing.T) {
		accounts, err := service.ListAccounts()
		
		require.NoError(t, err)
		assert.Empty(t, accounts)
	})

	t.Run("with accounts", func(t *testing.T) {
		// Create test accounts
		accountID1, err := service.CreateAccount("user1", "password1")
		require.NoError(t, err)
		
		accountID2, err := service.CreateAccount("user2", "password2")
		require.NoError(t, err)

		accounts, err := service.ListAccounts()
		
		require.NoError(t, err)
		assert.Len(t, accounts, 2)

		// Find accounts by ID
		var account1, account2 *auth.Account
		for i := range accounts {
			if accounts[i].ID == accountID1 {
				account1 = &accounts[i]
			} else if accounts[i].ID == accountID2 {
				account2 = &accounts[i]
			}
		}

		require.NotNil(t, account1)
		require.NotNil(t, account2)
		assert.Equal(t, "user1", account1.Username)
		assert.Equal(t, "user2", account2.Username)
	})
}

func TestAccountService_APITokenGeneration(t *testing.T) {
	store := storage.NewMemoryAccountStore()
	service := NewAccountService(store)

	// Create multiple accounts
	var tokens []string
	for i := 0; i < 5; i++ {
		accountID, err := service.CreateAccount(
			"user"+string(rune('0'+i)), 
			"password"+string(rune('0'+i)),
		)
		require.NoError(t, err)

		account, err := service.GetAccount(accountID)
		require.NoError(t, err)
		
		tokens = append(tokens, account.APIToken)
	}

	// Verify all tokens are unique
	for i := 0; i < len(tokens); i++ {
		for j := i + 1; j < len(tokens); j++ {
			assert.NotEqual(t, tokens[i], tokens[j], "API tokens should be unique")
		}
	}

	// Verify token format (should be hex string)
	for _, token := range tokens {
		assert.Len(t, token, 64) // 32 bytes * 2 (hex encoding)
		assert.True(t, isHexString(token), "Token should be hex string: %s", token)
	}
}

func TestAccountService_PasswordSecurity(t *testing.T) {
	store := storage.NewMemoryAccountStore()
	service := NewAccountService(store)

	password := "mysecretpassword"
	accountID, err := service.CreateAccount("testuser", password)
	require.NoError(t, err)

	account, err := service.GetAccount(accountID)
	require.NoError(t, err)

	// Verify password is hashed (not stored in plain text)
	assert.NotEqual(t, password, account.PasswordHash)
	
	// Verify hash starts with bcrypt identifier
	assert.True(t, strings.HasPrefix(account.PasswordHash, "$2a$") || 
		strings.HasPrefix(account.PasswordHash, "$2b$") || 
		strings.HasPrefix(account.PasswordHash, "$2y$"),
		"Password hash should be bcrypt format")

	// Verify hash can be used to validate password
	err = bcrypt.CompareHashAndPassword([]byte(account.PasswordHash), []byte(password))
	assert.NoError(t, err)

	// Verify wrong password fails
	err = bcrypt.CompareHashAndPassword([]byte(account.PasswordHash), []byte("wrongpassword"))
	assert.Error(t, err)
}

func TestAccountService_Timestamps(t *testing.T) {
	store := storage.NewMemoryAccountStore()
	service := NewAccountService(store)

	before := time.Now()
	accountID, err := service.CreateAccount("testuser", "password")
	require.NoError(t, err)
	after := time.Now()

	account, err := service.GetAccount(accountID)
	require.NoError(t, err)

	// Verify timestamps are set
	assert.False(t, account.CreatedAt.IsZero())
	assert.False(t, account.UpdatedAt.IsZero())

	// Verify timestamps are reasonable
	assert.True(t, account.CreatedAt.After(before) || account.CreatedAt.Equal(before))
	assert.True(t, account.CreatedAt.Before(after) || account.CreatedAt.Equal(after))
	assert.True(t, account.UpdatedAt.After(before) || account.UpdatedAt.Equal(before))
	assert.True(t, account.UpdatedAt.Before(after) || account.UpdatedAt.Equal(after))

	// For new accounts, CreatedAt and UpdatedAt should be the same
	assert.Equal(t, account.CreatedAt, account.UpdatedAt)
}

// Test interface compliance
func TestAccountService_ImplementsInterface(t *testing.T) {
	store := storage.NewMemoryAccountStore()
	service := NewAccountService(store)
	
	// Verify it implements the interface
	var _ auth.AccountService = service
}

// Helper function to check if a string is valid hex
func isHexString(s string) bool {
	for _, c := range s {
		if !((c >= '0' && c <= '9') || (c >= 'a' && c <= 'f') || (c >= 'A' && c <= 'F')) {
			return false
		}
	}
	return true
}
