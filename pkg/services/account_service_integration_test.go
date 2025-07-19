package services

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/tcmartin/flowrunner/pkg/auth"
	"github.com/tcmartin/flowrunner/pkg/storage"
)

func TestAccountService_MultipleStorageBackends(t *testing.T) {
	testCases := []struct {
		name        string
		createStore func() storage.AccountStore
	}{
		{
			name: "Memory Storage",
			createStore: func() storage.AccountStore {
				return storage.NewMemoryAccountStore()
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			store := tc.createStore()
			service := NewAccountService(store)

			// Test complete workflow
			username := "testuser"
			password := "testpassword"

			// Create account
			accountID, err := service.CreateAccount(username, password)
			require.NoError(t, err)
			assert.NotEmpty(t, accountID)

			// Authenticate
			authAccountID, err := service.Authenticate(username, password)
			require.NoError(t, err)
			assert.Equal(t, accountID, authAccountID)

			// Get account
			account, err := service.GetAccount(accountID)
			require.NoError(t, err)
			assert.Equal(t, username, account.Username)

			// Validate token
			validatedID, err := service.ValidateToken(account.APIToken)
			require.NoError(t, err)
			assert.Equal(t, accountID, validatedID)

			// List accounts
			accounts, err := service.ListAccounts()
			require.NoError(t, err)
			assert.Len(t, accounts, 1)
			assert.Equal(t, accountID, accounts[0].ID)

			// Delete account
			err = service.DeleteAccount(accountID)
			require.NoError(t, err)

			// Verify account is deleted
			_, err = service.GetAccount(accountID)
			assert.Error(t, err)
		})
	}
}

func TestAccountService_InterfaceCompliance(t *testing.T) {
	// Test that our implementation satisfies the interface
	store := storage.NewMemoryAccountStore()
	service := NewAccountService(store)

	// This should compile if the interface is correctly implemented
	var accountService auth.AccountService = service

	// Test basic functionality
	accountID, err := accountService.CreateAccount("test", "password")
	require.NoError(t, err)
	assert.NotEmpty(t, accountID)

	// Test authentication
	authID, err := accountService.Authenticate("test", "password")
	require.NoError(t, err)
	assert.Equal(t, accountID, authID)

	// Test token validation
	account, err := accountService.GetAccount(accountID)
	require.NoError(t, err)

	validatedID, err := accountService.ValidateToken(account.APIToken)
	require.NoError(t, err)
	assert.Equal(t, accountID, validatedID)

	// Test listing
	accounts, err := accountService.ListAccounts()
	require.NoError(t, err)
	assert.Len(t, accounts, 1)

	// Test deletion
	err = accountService.DeleteAccount(accountID)
	require.NoError(t, err)
}

// TestAccountService_ConcurrentAccess tests concurrent operations on the account service
func TestAccountService_ConcurrentAccess(t *testing.T) {
	store := storage.NewMemoryAccountStore()
	service := NewAccountService(store)

	// Create accounts concurrently
	const numAccounts = 10
	accountChan := make(chan string, numAccounts)
	errorChan := make(chan error, numAccounts)

	for i := 0; i < numAccounts; i++ {
		go func(i int) {
			username := fmt.Sprintf("user%d", i)
			password := fmt.Sprintf("password%d", i)
			
			accountID, err := service.CreateAccount(username, password)
			if err != nil {
				errorChan <- err
				return
			}
			accountChan <- accountID
		}(i)
	}

	// Collect results
	var accountIDs []string
	for i := 0; i < numAccounts; i++ {
		select {
		case accountID := <-accountChan:
			accountIDs = append(accountIDs, accountID)
		case err := <-errorChan:
			t.Fatalf("Error creating account: %v", err)
		}
	}

	// Verify all accounts were created
	assert.Len(t, accountIDs, numAccounts)

	// Verify all account IDs are unique
	seen := make(map[string]bool)
	for _, id := range accountIDs {
		assert.False(t, seen[id], "Duplicate account ID: %s", id)
		seen[id] = true
	}

	// List all accounts
	accounts, err := service.ListAccounts()
	require.NoError(t, err)
	assert.Len(t, accounts, numAccounts)
}
