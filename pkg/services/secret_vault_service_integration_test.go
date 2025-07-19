package services

import (
	"flag"
	"fmt"
	"os"
	"strconv"
	"testing"

	"github.com/joho/godotenv"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/tcmartin/flowrunner/pkg/storage"
)

var realDynamoDBSecrets = flag.Bool("real-dynamodb-secrets", false, "Run secret vault tests against real DynamoDB (requires local DynamoDB)")
var realPostgreSQLSecrets = flag.Bool("real-postgresql-secrets", false, "Run secret vault tests against real PostgreSQL (requires PostgreSQL server)")

func TestSecretVaultService_AllStorageBackends(t *testing.T) {
	// Test Memory backend
	t.Run("memory", func(t *testing.T) {
		store := storage.NewMemorySecretStore()

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

	// Test PostgreSQL backend
	t.Run("postgres", func(t *testing.T) {
		if !*realPostgreSQLSecrets {
			t.Skip("Skipping PostgreSQL test: use -real-postgresql-secrets flag to enable")
		}

		// Load environment variables
		err := godotenv.Load("../../.env")
		if err != nil {
			t.Logf("Warning: could not load .env file: %v", err)
		}

		// Get PostgreSQL configuration from environment variables
		host := os.Getenv("FLOWRUNNER_POSTGRES_HOST")
		user := os.Getenv("FLOWRUNNER_POSTGRES_USER")
		password := os.Getenv("FLOWRUNNER_POSTGRES_PASSWORD")
		database := os.Getenv("FLOWRUNNER_POSTGRES_DATABASE")
		portStr := os.Getenv("FLOWRUNNER_POSTGRES_PORT")
		sslMode := os.Getenv("FLOWRUNNER_POSTGRES_SSL_MODE")

		// Set defaults if not specified
		if host == "" {
			host = "localhost"
		}
		if user == "" {
			user = "postgres"
		}
		if password == "" {
			password = "postgres"
		}
		if database == "" {
			database = "flowrunner_test"
		}
		if portStr == "" {
			portStr = "5432"
		}
		if sslMode == "" {
			sslMode = "disable"
		}

		port, err := strconv.Atoi(portStr)
		if err != nil {
			t.Fatalf("Invalid port number: %s", portStr)
		}

		t.Logf("PostgreSQL test config: host=%s, port=%d, user=%s, database=%s, sslmode=%s", 
			host, port, user, database, sslMode)

		provider, err := storage.NewPostgreSQLProvider(storage.PostgreSQLProviderConfig{
			Host:     host,
			Port:     port,
			Database: database,
			User:     user,
			Password: password,
			SSLMode:  sslMode,
		})
		if err != nil {
			t.Skipf("Skipping PostgreSQL test: %v", err)
		}

		if err := provider.Initialize(); err != nil {
			t.Skipf("Skipping PostgreSQL test: failed to initialize: %v", err)
		}

		store := provider.GetSecretStore()

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

	// Test DynamoDB backend
	t.Run("dynamodb", func(t *testing.T) {
		var store storage.SecretStore

		if !*realDynamoDBSecrets {
			// Use mock DynamoDB for fast tests
			provider := storage.NewDynamoDBProviderWithClient(
				storage.NewMockDynamoDBAPI(),
				"test_",
			)

			if err := provider.Initialize(); err != nil {
				t.Skipf("Skipping DynamoDB mock test: failed to initialize: %v", err)
			}

			store = provider.GetSecretStore()
		} else {
			// Load environment variables
			err := godotenv.Load("../../.env")
			if err != nil {
				t.Logf("Warning: could not load .env file: %v", err)
			}

			// Get DynamoDB configuration from environment variables
			region := os.Getenv("FLOWRUNNER_DYNAMODB_REGION")
			endpoint := os.Getenv("FLOWRUNNER_DYNAMODB_ENDPOINT")
			tablePrefix := os.Getenv("FLOWRUNNER_DYNAMODB_TABLE_PREFIX")

			// Set defaults if not specified
			if region == "" {
				region = "us-west-2"
			}
			if endpoint == "" {
				endpoint = "http://localhost:8000"
			}
			if tablePrefix == "" {
				tablePrefix = "secret_vault_test_"
			}

			t.Logf("DynamoDB test config: region=%s, endpoint=%s, tablePrefix=%s", 
				region, endpoint, tablePrefix)

			// Use real DynamoDB for integration tests
			provider, err := storage.NewDynamoDBProvider(storage.DynamoDBProviderConfig{
				Region:      region,
				Endpoint:    endpoint,
				TablePrefix: tablePrefix,
			})
			if err != nil {
				t.Skipf("Skipping real DynamoDB test: %v", err)
			}

			if err := provider.Initialize(); err != nil {
				t.Skipf("Skipping real DynamoDB test: failed to initialize: %v", err)
			}

			store = provider.GetSecretStore()
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

func testBasicSecretOperations(t *testing.T, vault *SecretVaultService) {
	accountID := "test-account-basic"

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
	account1 := "account-isolation-1"
	account2 := "account-isolation-2"

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
	accountID := "test-account-encryption"
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
	// Test Memory backend edge cases
	t.Run("memory", func(t *testing.T) {
		store := storage.NewMemorySecretStore()
		encryptionKey, err := GenerateEncryptionKey()
		require.NoError(t, err)

		vault, err := NewSecretVaultService(store, encryptionKey)
		require.NoError(t, err)

		testSecretVaultEdgeCases(t, vault)
	})

	// Test DynamoDB mock backend edge cases
	t.Run("dynamodb_mock", func(t *testing.T) {
		provider := storage.NewDynamoDBProviderWithClient(
			storage.NewMockDynamoDBAPI(),
			"test_",
		)
		require.NoError(t, provider.Initialize())
		store := provider.GetSecretStore()

		encryptionKey, err := GenerateEncryptionKey()
		require.NoError(t, err)

		vault, err := NewSecretVaultService(store, encryptionKey)
		require.NoError(t, err)

		testSecretVaultEdgeCases(t, vault)
	})

	// Test PostgreSQL backend edge cases
	t.Run("postgres", func(t *testing.T) {
		if !*realPostgreSQLSecrets {
			t.Skip("Skipping PostgreSQL edge cases test: use -real-postgresql-secrets flag to enable")
		}

		// Load environment variables
		err := godotenv.Load("../../.env")
		if err != nil {
			t.Logf("Warning: could not load .env file: %v", err)
		}

		// Get PostgreSQL configuration from environment variables
		host := os.Getenv("FLOWRUNNER_POSTGRES_HOST")
		user := os.Getenv("FLOWRUNNER_POSTGRES_USER")
		password := os.Getenv("FLOWRUNNER_POSTGRES_PASSWORD")
		database := os.Getenv("FLOWRUNNER_POSTGRES_DATABASE")
		portStr := os.Getenv("FLOWRUNNER_POSTGRES_PORT")
		sslMode := os.Getenv("FLOWRUNNER_POSTGRES_SSL_MODE")

		// Set defaults if not specified
		if host == "" {
			host = "localhost"
		}
		if user == "" {
			user = "postgres"
		}
		if password == "" {
			password = "postgres"
		}
		if database == "" {
			database = "flowrunner_test"
		}
		if portStr == "" {
			portStr = "5432"
		}
		if sslMode == "" {
			sslMode = "disable"
		}

		port, err := strconv.Atoi(portStr)
		if err != nil {
			t.Fatalf("Invalid port number: %s", portStr)
		}

		provider, err := storage.NewPostgreSQLProvider(storage.PostgreSQLProviderConfig{
			Host:     host,
			Port:     port,
			Database: database,
			User:     user,
			Password: password,
			SSLMode:  sslMode,
		})
		if err != nil {
			t.Skipf("Skipping PostgreSQL edge cases test: %v", err)
		}

		if err := provider.Initialize(); err != nil {
			t.Skipf("Skipping PostgreSQL edge cases test: failed to initialize: %v", err)
		}

		store := provider.GetSecretStore()

		encryptionKey, err := GenerateEncryptionKey()
		require.NoError(t, err)

		vault, err := NewSecretVaultService(store, encryptionKey)
		require.NoError(t, err)

		testSecretVaultEdgeCases(t, vault)
	})
}

func testSecretVaultEdgeCases(t *testing.T, vault *SecretVaultService) {
	accountID := "test-account-edge-cases"

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
