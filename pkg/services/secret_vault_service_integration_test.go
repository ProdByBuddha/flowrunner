package services

import (
	"flag"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/tcmartin/flowrunner/pkg/storage"
)

var realDynamoDBSecrets = flag.Bool("real-dynamodb-secrets", false, "Run secret vault tests against real DynamoDB (requires local DynamoDB)")

func TestSecretVaultService_AllStorageBackends(t *testing.T) {
	// Test all storage backends
	backends := map[string]func() storage.SecretStore{
		"memory": func() storage.SecretStore {
			return storage.NewMemorySecretStore()
		},
		"postgres": func() storage.SecretStore {
			provider, err := storage.NewPostgreSQLProvider(storage.PostgreSQLProviderConfig{
				Host:     "localhost",
				Port:     5432,
				Database: "flowrunner_test",
				User:     "postgres",
				Password: "postgres",
				SSLMode:  "disable",
			})
			if err != nil {
				t.Skipf("Skipping PostgreSQL test: %v", err)
				return nil
			}

			if err := provider.Initialize(); err != nil {
				t.Skipf("Skipping PostgreSQL test: failed to initialize: %v", err)
				return nil
			}

			return provider.GetSecretStore()
		},
		"dynamodb": func() storage.SecretStore {
			if !*realDynamoDBSecrets {
				// Use mock DynamoDB for fast tests
				provider := storage.NewDynamoDBProviderWithClient(
					storage.NewMockDynamoDBAPI(),
					"test_",
				)

				if err := provider.Initialize(); err != nil {
					t.Skipf("Skipping DynamoDB mock test: failed to initialize: %v", err)
					return nil
				}

				return provider.GetSecretStore()
			} else {
				// Use real DynamoDB for integration tests
				provider, err := storage.NewDynamoDBProvider(storage.DynamoDBProviderConfig{
					Region:      "us-west-2",
					Endpoint:    "http://localhost:8000",
					TablePrefix: "secret_vault_test_",
				})
				if err != nil {
					t.Skipf("Skipping real DynamoDB test: %v", err)
					return nil
				}

				if err := provider.Initialize(); err != nil {
					t.Skipf("Skipping real DynamoDB test: failed to initialize: %v", err)
					return nil
				}

				return provider.GetSecretStore()
			}
		},
	}

	for backendName, createStore := range backends {
		t.Run(backendName, func(t *testing.T) {
			store := createStore()
			if store == nil {
				return // Skip test
			}

			// Generate encryption key
			encryptionKey, err := GenerateEncryptionKey()
			require.NoError(t, err)

			// Create secret vault service
			vault, err := NewSecretVaultService(store, encryptionKey)
			require.NoError(t, err)

			// Test basic operations
			testBasicSecretOperations(t, vault)

			// Test account isolation
			testAccountIsolation(t, vault)

			// Test encryption consistency
			testEncryptionConsistency(t, vault, store)
		})
	}
}

func testBasicSecretOperations(t *testing.T, vault *SecretVaultService) {
	accountID := "test-account"

	// Test Set and Get
	err := vault.Set(accountID, "api_key", "secret-api-key-123")
	assert.NoError(t, err)

	value, err := vault.Get(accountID, "api_key")
	assert.NoError(t, err)
	assert.Equal(t, "secret-api-key-123", value)

	// Test List
	keys, err := vault.List(accountID)
	assert.NoError(t, err)
	assert.Contains(t, keys, "api_key")

	// Test Update
	err = vault.Set(accountID, "api_key", "updated-secret-key-456")
	assert.NoError(t, err)

	value, err = vault.Get(accountID, "api_key")
	assert.NoError(t, err)
	assert.Equal(t, "updated-secret-key-456", value)

	// Test Delete
	err = vault.Delete(accountID, "api_key")
	assert.NoError(t, err)

	_, err = vault.Get(accountID, "api_key")
	assert.Error(t, err)

	// Test List after delete
	keys, err = vault.List(accountID)
	assert.NoError(t, err)
	assert.NotContains(t, keys, "api_key")
}

func testAccountIsolation(t *testing.T, vault *SecretVaultService) {
	account1 := "account-1"
	account2 := "account-2"

	// Add secrets to both accounts
	err := vault.Set(account1, "secret1", "value1")
	assert.NoError(t, err)
	err = vault.Set(account2, "secret2", "value2")
	assert.NoError(t, err)

	// Verify account1 can't see account2's secrets
	_, err = vault.Get(account1, "secret2")
	assert.Error(t, err)

	// Verify account2 can't see account1's secrets
	_, err = vault.Get(account2, "secret1")
	assert.Error(t, err)

	// Verify list operations are isolated
	keys1, err := vault.List(account1)
	assert.NoError(t, err)
	assert.Contains(t, keys1, "secret1")
	assert.NotContains(t, keys1, "secret2")

	keys2, err := vault.List(account2)
	assert.NoError(t, err)
	assert.Contains(t, keys2, "secret2")
	assert.NotContains(t, keys2, "secret1")
}

func testEncryptionConsistency(t *testing.T, vault *SecretVaultService, store storage.SecretStore) {
	accountID := "test-account"
	secretKey := "test-secret"
	secretValue := "test-value"

	// Store a secret
	err := vault.Set(accountID, secretKey, secretValue)
	require.NoError(t, err)

	// Get the raw secret from the store
	rawSecret, err := store.GetSecret(accountID, secretKey)
	require.NoError(t, err)

	// The stored value should be encrypted (different from original)
	assert.NotEqual(t, secretValue, rawSecret.Value)
	assert.NotEmpty(t, rawSecret.Value)

	// Getting through vault should return decrypted value
	decryptedValue, err := vault.Get(accountID, secretKey)
	require.NoError(t, err)
	assert.Equal(t, secretValue, decryptedValue)

	// Multiple encryptions should produce different ciphertexts
	err = vault.Set(accountID, "key1", secretValue)
	require.NoError(t, err)
	err = vault.Set(accountID, "key2", secretValue)
	require.NoError(t, err)

	secret1, err := store.GetSecret(accountID, "key1")
	require.NoError(t, err)
	secret2, err := store.GetSecret(accountID, "key2")
	require.NoError(t, err)

	// Same plaintext should produce different ciphertexts due to random nonce
	assert.NotEqual(t, secret1.Value, secret2.Value)

	// But both should decrypt to the same value
	value1, err := vault.Get(accountID, "key1")
	require.NoError(t, err)
	value2, err := vault.Get(accountID, "key2")
	require.NoError(t, err)
	assert.Equal(t, value1, value2)
	assert.Equal(t, secretValue, value1)
}

func TestSecretVaultService_EdgeCases_AllBackends(t *testing.T) {
	backends := map[string]func() storage.SecretStore{
		"memory": func() storage.SecretStore {
			return storage.NewMemorySecretStore()
		},
		"dynamodb_mock": func() storage.SecretStore {
			provider := storage.NewDynamoDBProviderWithClient(
				storage.NewMockDynamoDBAPI(),
				"test_",
			)
			require.NoError(t, provider.Initialize())
			return provider.GetSecretStore()
		},
	}

	for backendName, createStore := range backends {
		t.Run(backendName, func(t *testing.T) {
			store := createStore()
			encryptionKey, err := GenerateEncryptionKey()
			require.NoError(t, err)

			vault, err := NewSecretVaultService(store, encryptionKey)
			require.NoError(t, err)

			testSecretVaultEdgeCases(t, vault)
		})
	}
}

func testSecretVaultEdgeCases(t *testing.T, vault *SecretVaultService) {
	accountID := "test-account"

	// Test empty values
	err := vault.Set(accountID, "empty-secret", "")
	assert.NoError(t, err)

	value, err := vault.Get(accountID, "empty-secret")
	assert.NoError(t, err)
	assert.Equal(t, "", value)

	// Test unicode values
	unicodeValue := "Hello 世界! 🔐"
	err = vault.Set(accountID, "unicode-secret", unicodeValue)
	assert.NoError(t, err)

	value, err = vault.Get(accountID, "unicode-secret")
	assert.NoError(t, err)
	assert.Equal(t, unicodeValue, value)

	// Test large values
	largeValue := make([]byte, 1024*10) // 10KB
	for i := range largeValue {
		largeValue[i] = byte(i % 256)
	}
	largeValueStr := string(largeValue)

	err = vault.Set(accountID, "large-secret", largeValueStr)
	assert.NoError(t, err)

	value, err = vault.Get(accountID, "large-secret")
	assert.NoError(t, err)
	assert.Equal(t, largeValueStr, value)

	// Test special characters in keys
	specialKey := "key-with.special_chars@domain.com:8080/path?query=value#fragment"
	err = vault.Set(accountID, specialKey, "value")
	assert.NoError(t, err)

	value, err = vault.Get(accountID, specialKey)
	assert.NoError(t, err)
	assert.Equal(t, "value", value)

	// Verify it appears in listings
	keys, err := vault.List(accountID)
	assert.NoError(t, err)
	assert.Contains(t, keys, specialKey)
}

func TestSecretVaultService_MultipleConcurrentOperations(t *testing.T) {
	store := storage.NewMemorySecretStore()
	encryptionKey, err := GenerateEncryptionKey()
	require.NoError(t, err)

	vault, err := NewSecretVaultService(store, encryptionKey)
	require.NoError(t, err)

	accountID := "test-account"
	numGoroutines := 10
	numOperations := 100

	// Test concurrent writes
	t.Run("concurrent_writes", func(t *testing.T) {
		done := make(chan bool, numGoroutines)

		for i := 0; i < numGoroutines; i++ {
			go func(id int) {
				for j := 0; j < numOperations; j++ {
					key := fmt.Sprintf("key-%d-%d", id, j)
					value := fmt.Sprintf("value-%d-%d", id, j)
					err := vault.Set(accountID, key, value)
					assert.NoError(t, err)
				}
				done <- true
			}(i)
		}

		// Wait for all goroutines to complete
		for i := 0; i < numGoroutines; i++ {
			<-done
		}

		// Verify all keys were written
		keys, err := vault.List(accountID)
		assert.NoError(t, err)
		assert.Len(t, keys, numGoroutines*numOperations)
	})

	// Test concurrent reads
	t.Run("concurrent_reads", func(t *testing.T) {
		// First, write some test data
		testKey := "concurrent-read-test"
		testValue := "concurrent-read-value"
		err := vault.Set(accountID, testKey, testValue)
		require.NoError(t, err)

		done := make(chan bool, numGoroutines)

		for i := 0; i < numGoroutines; i++ {
			go func() {
				for j := 0; j < numOperations; j++ {
					value, err := vault.Get(accountID, testKey)
					assert.NoError(t, err)
					assert.Equal(t, testValue, value)
				}
				done <- true
			}()
		}

		// Wait for all goroutines to complete
		for i := 0; i < numGoroutines; i++ {
			<-done
		}
	})

	// Test mixed operations
	t.Run("mixed_operations", func(t *testing.T) {
		done := make(chan bool, numGoroutines)

		for i := 0; i < numGoroutines; i++ {
			go func(id int) {
				for j := 0; j < numOperations; j++ {
					switch j % 3 {
					case 0: // Write
						key := fmt.Sprintf("mixed-key-%d-%d", id, j)
						value := fmt.Sprintf("mixed-value-%d-%d", id, j)
						err := vault.Set(accountID, key, value)
						assert.NoError(t, err)
					case 1: // Read
						key := fmt.Sprintf("mixed-key-%d-%d", id, j-1)
						if j > 0 {
							_, err := vault.Get(accountID, key)
							// Don't assert on error as key might not exist yet
							_ = err
						}
					case 2: // List
						_, err := vault.List(accountID)
						assert.NoError(t, err)
					}
				}
				done <- true
			}(i)
		}

		// Wait for all goroutines to complete
		for i := 0; i < numGoroutines; i++ {
			<-done
		}
	})
}
