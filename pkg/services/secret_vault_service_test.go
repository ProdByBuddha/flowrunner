package services

import (
	"encoding/hex"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/tcmartin/flowrunner/pkg/storage"
)

func TestNewSecretVaultService(t *testing.T) {
	store := storage.NewMemorySecretStore()

	t.Run("valid key", func(t *testing.T) {
		key := make([]byte, 32)
		vault, err := NewSecretVaultService(store, key)
		
		require.NoError(t, err)
		assert.NotNil(t, vault)
		assert.Equal(t, key, vault.encryptionKey)
	})

	t.Run("invalid key length", func(t *testing.T) {
		invalidKey := make([]byte, 16) // Too short
		vault, err := NewSecretVaultService(store, invalidKey)
		
		assert.Error(t, err)
		assert.Nil(t, vault)
		assert.Contains(t, err.Error(), "32 bytes")
	})
}

func TestSecretVaultService_Set(t *testing.T) {
	store := storage.NewMemorySecretStore()
	key := make([]byte, 32)
	vault, err := NewSecretVaultService(store, key)
	require.NoError(t, err)

	accountID := "test-account"
	secretKey := "test-secret"
	secretValue := "test-value"

	t.Run("successful set", func(t *testing.T) {
		err := vault.Set(accountID, secretKey, secretValue)
		
		assert.NoError(t, err)

		// Verify secret was stored (encrypted)
		storedSecret, err := store.GetSecret(accountID, secretKey)
		require.NoError(t, err)
		assert.Equal(t, accountID, storedSecret.AccountID)
		assert.Equal(t, secretKey, storedSecret.Key)
		assert.NotEqual(t, secretValue, storedSecret.Value) // Should be encrypted
		assert.True(t, len(storedSecret.Value) > 0)
	})

	t.Run("empty account ID", func(t *testing.T) {
		err := vault.Set("", secretKey, secretValue)
		
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "account ID is required")
	})

	t.Run("empty secret key", func(t *testing.T) {
		err := vault.Set(accountID, "", secretValue)
		
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "secret key is required")
	})

	t.Run("update existing secret preserves creation time", func(t *testing.T) {
		// First set
		err := vault.Set(accountID, "update-test", "value1")
		require.NoError(t, err)

		storedSecret1, err := store.GetSecret(accountID, "update-test")
		require.NoError(t, err)
		originalCreatedAt := storedSecret1.CreatedAt

		// Wait a bit
		time.Sleep(10 * time.Millisecond)

		// Update
		err = vault.Set(accountID, "update-test", "value2")
		require.NoError(t, err)

		storedSecret2, err := store.GetSecret(accountID, "update-test")
		require.NoError(t, err)

		// Creation time should be preserved, update time should be newer
		assert.Equal(t, originalCreatedAt.Unix(), storedSecret2.CreatedAt.Unix())
		assert.True(t, storedSecret2.UpdatedAt.After(originalCreatedAt))
	})
}

func TestSecretVaultService_Get(t *testing.T) {
	store := storage.NewMemorySecretStore()
	key := make([]byte, 32)
	vault, err := NewSecretVaultService(store, key)
	require.NoError(t, err)

	accountID := "test-account"
	secretKey := "test-secret"
	secretValue := "test-value"

	// Set a secret first
	err = vault.Set(accountID, secretKey, secretValue)
	require.NoError(t, err)

	t.Run("successful get", func(t *testing.T) {
		retrievedValue, err := vault.Get(accountID, secretKey)
		
		assert.NoError(t, err)
		assert.Equal(t, secretValue, retrievedValue)
	})

	t.Run("empty account ID", func(t *testing.T) {
		_, err := vault.Get("", secretKey)
		
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "account ID is required")
	})

	t.Run("empty secret key", func(t *testing.T) {
		_, err := vault.Get(accountID, "")
		
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "secret key is required")
	})

	t.Run("non-existent secret", func(t *testing.T) {
		_, err := vault.Get(accountID, "non-existent")
		
		assert.Error(t, err)
		assert.Equal(t, storage.ErrSecretNotFound, err)
	})

	t.Run("non-existent account", func(t *testing.T) {
		_, err := vault.Get("non-existent-account", secretKey)
		
		assert.Error(t, err)
		assert.Equal(t, storage.ErrSecretNotFound, err)
	})
}

func TestSecretVaultService_Delete(t *testing.T) {
	store := storage.NewMemorySecretStore()
	key := make([]byte, 32)
	vault, err := NewSecretVaultService(store, key)
	require.NoError(t, err)

	accountID := "test-account"
	secretKey := "test-secret"
	secretValue := "test-value"

	// Set a secret first
	err = vault.Set(accountID, secretKey, secretValue)
	require.NoError(t, err)

	t.Run("successful delete", func(t *testing.T) {
		err := vault.Delete(accountID, secretKey)
		
		assert.NoError(t, err)

		// Verify secret was deleted
		_, err = vault.Get(accountID, secretKey)
		assert.Error(t, err)
		assert.Equal(t, storage.ErrSecretNotFound, err)
	})

	t.Run("empty account ID", func(t *testing.T) {
		err := vault.Delete("", secretKey)
		
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "account ID is required")
	})

	t.Run("empty secret key", func(t *testing.T) {
		err := vault.Delete(accountID, "")
		
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "secret key is required")
	})

	t.Run("delete non-existent secret", func(t *testing.T) {
		err := vault.Delete(accountID, "non-existent")
		
		assert.Error(t, err)
		assert.Equal(t, storage.ErrSecretNotFound, err)
	})
}

func TestSecretVaultService_List(t *testing.T) {
	store := storage.NewMemorySecretStore()
	key := make([]byte, 32)
	vault, err := NewSecretVaultService(store, key)
	require.NoError(t, err)

	accountID := "test-account"

	t.Run("empty list", func(t *testing.T) {
		keys, err := vault.List(accountID)
		
		assert.NoError(t, err)
		assert.Empty(t, keys)
	})

	t.Run("list with secrets", func(t *testing.T) {
		// Add some secrets
		err := vault.Set(accountID, "secret1", "value1")
		require.NoError(t, err)
		err = vault.Set(accountID, "secret2", "value2")
		require.NoError(t, err)
		err = vault.Set(accountID, "secret3", "value3")
		require.NoError(t, err)

		keys, err := vault.List(accountID)
		
		assert.NoError(t, err)
		assert.Len(t, keys, 3)
		assert.Contains(t, keys, "secret1")
		assert.Contains(t, keys, "secret2")
		assert.Contains(t, keys, "secret3")
	})

	t.Run("empty account ID", func(t *testing.T) {
		_, err := vault.List("")
		
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "account ID is required")
	})

	t.Run("account isolation", func(t *testing.T) {
		otherAccountID := "other-account"
		
		// Add secret to other account
		err := vault.Set(otherAccountID, "other-secret", "other-value")
		require.NoError(t, err)

		// List should only return secrets for the requested account
		keys, err := vault.List(accountID)
		assert.NoError(t, err)
		assert.NotContains(t, keys, "other-secret")

		otherKeys, err := vault.List(otherAccountID)
		assert.NoError(t, err)
		assert.Contains(t, otherKeys, "other-secret")
		assert.Len(t, otherKeys, 1)
	})
}

func TestSecretVaultService_ListWithMetadata(t *testing.T) {
	store := storage.NewMemorySecretStore()
	key := make([]byte, 32)
	vault, err := NewSecretVaultService(store, key)
	require.NoError(t, err)

	accountID := "test-account"

	t.Run("list with metadata", func(t *testing.T) {
		// Add some secrets
		err := vault.Set(accountID, "secret1", "value1")
		require.NoError(t, err)
		err = vault.Set(accountID, "secret2", "value2")
		require.NoError(t, err)

		secrets, err := vault.ListWithMetadata(accountID)
		
		assert.NoError(t, err)
		assert.Len(t, secrets, 2)

		for _, secret := range secrets {
			assert.Equal(t, accountID, secret.AccountID)
			assert.Empty(t, secret.Value) // Values should be cleared
			assert.NotZero(t, secret.CreatedAt)
			assert.NotZero(t, secret.UpdatedAt)
		}
	})

	t.Run("empty account ID", func(t *testing.T) {
		_, err := vault.ListWithMetadata("")
		
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "account ID is required")
	})
}

func TestSecretVaultService_Encryption(t *testing.T) {
	store := storage.NewMemorySecretStore()
	key := make([]byte, 32)
	vault, err := NewSecretVaultService(store, key)
	require.NoError(t, err)

	accountID := "test-account"
	secretKey := "test-secret"

	t.Run("encryption produces different output", func(t *testing.T) {
		value1 := "same-value"
		value2 := "same-value"

		err := vault.Set(accountID, "key1", value1)
		require.NoError(t, err)
		err = vault.Set(accountID, "key2", value2)
		require.NoError(t, err)

		secret1, err := store.GetSecret(accountID, "key1")
		require.NoError(t, err)
		secret2, err := store.GetSecret(accountID, "key2")
		require.NoError(t, err)

		// Even with same input, encrypted values should be different (due to random nonce)
		assert.NotEqual(t, secret1.Value, secret2.Value)
	})

	t.Run("decryption consistency", func(t *testing.T) {
		originalValue := "test-value-for-consistency"
		
		err := vault.Set(accountID, secretKey, originalValue)
		require.NoError(t, err)

		// Get multiple times to ensure consistency
		for i := 0; i < 5; i++ {
			retrievedValue, err := vault.Get(accountID, secretKey)
			require.NoError(t, err)
			assert.Equal(t, originalValue, retrievedValue)
		}
	})

	t.Run("different keys produce different encryption", func(t *testing.T) {
		key1 := make([]byte, 32)
		key2 := make([]byte, 32)
		key2[0] = 1 // Make it different

		vault1, err := NewSecretVaultService(storage.NewMemorySecretStore(), key1)
		require.NoError(t, err)
		vault2, err := NewSecretVaultService(storage.NewMemorySecretStore(), key2)
		require.NoError(t, err)

		value := "same-value"
		
		// Encrypt with both vaults
		encrypted1, err := vault1.encrypt(value)
		require.NoError(t, err)
		encrypted2, err := vault2.encrypt(value)
		require.NoError(t, err)

		// Should be different
		assert.NotEqual(t, encrypted1, encrypted2)

		// Decrypt with wrong vault should fail
		_, err = vault1.decrypt(encrypted2)
		assert.Error(t, err)
		_, err = vault2.decrypt(encrypted1)
		assert.Error(t, err)
	})
}

func TestSecretVaultService_RotateEncryptionKey(t *testing.T) {
	store := storage.NewMemorySecretStore()
	oldKey := make([]byte, 32)
	vault, err := NewSecretVaultService(store, oldKey)
	require.NoError(t, err)

	t.Run("invalid new key length", func(t *testing.T) {
		invalidNewKey := make([]byte, 16)
		err := vault.RotateEncryptionKey(oldKey, invalidNewKey)
		
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "32 bytes")
	})

	t.Run("successful key rotation", func(t *testing.T) {
		newKey := make([]byte, 32)
		newKey[0] = 1 // Make it different
		
		err := vault.RotateEncryptionKey(oldKey, newKey)
		
		assert.NoError(t, err)
		assert.Equal(t, newKey, vault.encryptionKey)
	})
}

func TestSecretVaultService_RotateEncryptionKeyForAccounts(t *testing.T) {
	store := storage.NewMemorySecretStore()
	oldKey := make([]byte, 32)
	vault, err := NewSecretVaultService(store, oldKey)
	require.NoError(t, err)

	accountID := "test-account"

	// Add some secrets with the old key
	err = vault.Set(accountID, "secret1", "value1")
	require.NoError(t, err)
	err = vault.Set(accountID, "secret2", "value2")
	require.NoError(t, err)

	t.Run("invalid new key length", func(t *testing.T) {
		invalidNewKey := make([]byte, 16)
		err := vault.RotateEncryptionKeyForAccounts(oldKey, invalidNewKey, []string{accountID})
		
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "32 bytes")
	})

	t.Run("invalid old key length", func(t *testing.T) {
		invalidOldKey := make([]byte, 16)
		newKey := make([]byte, 32)
		newKey[0] = 1
		err := vault.RotateEncryptionKeyForAccounts(invalidOldKey, newKey, []string{accountID})
		
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "32 bytes")
	})

	t.Run("successful key rotation for accounts", func(t *testing.T) {
		newKey := make([]byte, 32)
		newKey[0] = 1 // Make it different
		
		err := vault.RotateEncryptionKeyForAccounts(oldKey, newKey, []string{accountID})
		
		assert.NoError(t, err)
		assert.Equal(t, newKey, vault.encryptionKey)

		// Verify secrets can still be retrieved with new key
		value1, err := vault.Get(accountID, "secret1")
		assert.NoError(t, err)
		assert.Equal(t, "value1", value1)

		value2, err := vault.Get(accountID, "secret2")
		assert.NoError(t, err)
		assert.Equal(t, "value2", value2)
	})

	t.Run("key rotation with non-existent account", func(t *testing.T) {
		newKey := make([]byte, 32)
		newKey[0] = 2
		
		err := vault.RotateEncryptionKeyForAccounts(oldKey, newKey, []string{"non-existent-account"})
		
		// Should succeed even with non-existent account (no secrets to rotate)
		assert.NoError(t, err)
	})
}

func TestGenerateEncryptionKey(t *testing.T) {
	t.Run("generates valid key", func(t *testing.T) {
		key, err := GenerateEncryptionKey()
		
		assert.NoError(t, err)
		assert.Len(t, key, 32)
	})

	t.Run("generates different keys", func(t *testing.T) {
		key1, err := GenerateEncryptionKey()
		require.NoError(t, err)
		key2, err := GenerateEncryptionKey()
		require.NoError(t, err)

		assert.NotEqual(t, key1, key2)
	})
}

func TestEncryptionKeyFromHex(t *testing.T) {
	t.Run("valid hex key", func(t *testing.T) {
		originalKey := make([]byte, 32)
		hexKey := hex.EncodeToString(originalKey)
		
		key, err := EncryptionKeyFromHex(hexKey)
		
		assert.NoError(t, err)
		assert.Equal(t, originalKey, key)
	})

	t.Run("invalid hex", func(t *testing.T) {
		_, err := EncryptionKeyFromHex("invalid-hex")
		
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "decode hex")
	})

	t.Run("wrong length", func(t *testing.T) {
		shortKey := make([]byte, 16)
		hexKey := hex.EncodeToString(shortKey)
		
		_, err := EncryptionKeyFromHex(hexKey)
		
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "32 bytes")
	})
}

func TestEncryptionKeyToHex(t *testing.T) {
	key := make([]byte, 32)
	key[0] = 0xAB
	key[1] = 0xCD
	
	hexKey := EncryptionKeyToHex(key)
	
	assert.True(t, strings.HasPrefix(hexKey, "abcd"))
	assert.Len(t, hexKey, 64) // 32 bytes * 2 hex chars per byte
}

func TestSecretVaultService_EdgeCases(t *testing.T) {
	store := storage.NewMemorySecretStore()
	key := make([]byte, 32)
	vault, err := NewSecretVaultService(store, key)
	require.NoError(t, err)

	accountID := "test-account"

	t.Run("empty secret value", func(t *testing.T) {
		err := vault.Set(accountID, "empty-secret", "")
		assert.NoError(t, err)

		value, err := vault.Get(accountID, "empty-secret")
		assert.NoError(t, err)
		assert.Equal(t, "", value)
	})

	t.Run("unicode secret value", func(t *testing.T) {
		unicodeValue := "Hello 世界! 🔐"
		err := vault.Set(accountID, "unicode-secret", unicodeValue)
		assert.NoError(t, err)

		value, err := vault.Get(accountID, "unicode-secret")
		assert.NoError(t, err)
		assert.Equal(t, unicodeValue, value)
	})

	t.Run("large secret value", func(t *testing.T) {
		largeValue := strings.Repeat("A", 10000)
		err := vault.Set(accountID, "large-secret", largeValue)
		assert.NoError(t, err)

		value, err := vault.Get(accountID, "large-secret")
		assert.NoError(t, err)
		assert.Equal(t, largeValue, value)
	})

	t.Run("special characters in key", func(t *testing.T) {
		specialKey := "key-with.special_chars@domain.com"
		err := vault.Set(accountID, specialKey, "value")
		assert.NoError(t, err)

		value, err := vault.Get(accountID, specialKey)
		assert.NoError(t, err)
		assert.Equal(t, "value", value)

		keys, err := vault.List(accountID)
		assert.NoError(t, err)
		assert.Contains(t, keys, specialKey)
	})
}
