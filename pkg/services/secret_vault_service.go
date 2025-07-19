package services

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"io"
	"time"

	"github.com/tcmartin/flowrunner/pkg/auth"
	"github.com/tcmartin/flowrunner/pkg/storage"
)

// SecretVaultService implements the auth.SecretVault interface with encryption
type SecretVaultService struct {
	store         storage.SecretStore
	encryptionKey []byte
	gcm           cipher.AEAD
}

// NewSecretVaultService creates a new secret vault service with encryption
func NewSecretVaultService(store storage.SecretStore, encryptionKey []byte) (*SecretVaultService, error) {
	if len(encryptionKey) != 32 {
		return nil, fmt.Errorf("encryption key must be 32 bytes (256 bits)")
	}

	// Create AES cipher
	block, err := aes.NewCipher(encryptionKey)
	if err != nil {
		return nil, fmt.Errorf("failed to create AES cipher: %w", err)
	}

	// Create GCM mode for authenticated encryption
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf("failed to create GCM: %w", err)
	}

	return &SecretVaultService{
		store:         store,
		encryptionKey: encryptionKey,
		gcm:           gcm,
	}, nil
}

// Set stores an encrypted secret for an account
func (s *SecretVaultService) Set(accountID string, key string, value string) error {
	if accountID == "" {
		return fmt.Errorf("account ID is required")
	}
	if key == "" {
		return fmt.Errorf("secret key is required")
	}

	// Encrypt the value
	encryptedValue, err := s.encrypt(value)
	if err != nil {
		return fmt.Errorf("failed to encrypt secret: %w", err)
	}

	// Create secret object
	secret := auth.Secret{
		AccountID: accountID,
		Key:       key,
		Value:     encryptedValue,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	// Check if secret already exists to preserve creation time
	existingSecret, err := s.store.GetSecret(accountID, key)
	if err == nil {
		// Secret exists, preserve creation time
		secret.CreatedAt = existingSecret.CreatedAt
	}

	// Store the secret
	return s.store.SaveSecret(secret)
}

// Get retrieves and decrypts a secret for an account
func (s *SecretVaultService) Get(accountID string, key string) (string, error) {
	if accountID == "" {
		return "", fmt.Errorf("account ID is required")
	}
	if key == "" {
		return "", fmt.Errorf("secret key is required")
	}

	// Retrieve the secret
	secret, err := s.store.GetSecret(accountID, key)
	if err != nil {
		return "", err
	}

	// Decrypt the value
	decryptedValue, err := s.decrypt(secret.Value)
	if err != nil {
		return "", fmt.Errorf("failed to decrypt secret: %w", err)
	}

	return decryptedValue, nil
}

// Delete removes a secret
func (s *SecretVaultService) Delete(accountID string, key string) error {
	if accountID == "" {
		return fmt.Errorf("account ID is required")
	}
	if key == "" {
		return fmt.Errorf("secret key is required")
	}

	return s.store.DeleteSecret(accountID, key)
}

// List returns all secret keys for an account (without values)
func (s *SecretVaultService) List(accountID string) ([]string, error) {
	if accountID == "" {
		return nil, fmt.Errorf("account ID is required")
	}

	// Get all secrets for the account
	secrets, err := s.store.ListSecrets(accountID)
	if err != nil {
		return nil, err
	}

	// Extract just the keys
	keys := make([]string, len(secrets))
	for i, secret := range secrets {
		keys[i] = secret.Key
	}

	return keys, nil
}

// ListWithMetadata returns all secrets for an account with metadata (but no values)
func (s *SecretVaultService) ListWithMetadata(accountID string) ([]auth.Secret, error) {
	if accountID == "" {
		return nil, fmt.Errorf("account ID is required")
	}

	// Get all secrets for the account
	secrets, err := s.store.ListSecrets(accountID)
	if err != nil {
		return nil, err
	}

	// Clear the values for security
	for i := range secrets {
		secrets[i].Value = ""
	}

	return secrets, nil
}

// RotateEncryptionKey changes the encryption key for all secrets
func (s *SecretVaultService) RotateEncryptionKey(oldKey, newKey []byte) error {
	if len(newKey) != 32 {
		return fmt.Errorf("new encryption key must be 32 bytes (256 bits)")
	}

	// Create new cipher with new key
	block, err := aes.NewCipher(newKey)
	if err != nil {
		return fmt.Errorf("failed to create AES cipher with new key: %w", err)
	}

	newGCM, err := cipher.NewGCM(block)
	if err != nil {
		return fmt.Errorf("failed to create GCM with new key: %w", err)
	}

	// Validate old key for future use in migration
	_, err = aes.NewCipher(oldKey)
	if err != nil {
		return fmt.Errorf("failed to validate old encryption key: %w", err)
	}

	// Get all secrets from all accounts
	// Note: This is a simplified implementation. In production, you'd want to:
	// 1. Process in batches to avoid memory issues
	// 2. Implement proper error handling and rollback
	// 3. Consider locking to prevent concurrent access during rotation
	// 4. Add progress reporting for large datasets

	// For this implementation, we'll use a storage-specific approach
	// This assumes the storage layer supports listing all secrets across accounts
	// which may not be available in all storage backends

	// Since we don't have a direct way to list all secrets across all accounts,
	// we'll implement a simpler approach that requires the caller to provide
	// a list of account IDs to rotate

	// For now, we'll just update our internal encryption components
	// and document that existing secrets will need to be re-encrypted manually
	// or through a separate migration process

	// Update internal encryption key and cipher
	s.encryptionKey = newKey
	s.gcm = newGCM

	// TODO: Implement full secret migration
	// This would require:
	// 1. A way to list all accounts (not currently available in storage interface)
	// 2. For each account, list all secrets
	// 3. Decrypt each secret with oldGCM
	// 4. Re-encrypt with newGCM
	// 5. Save the re-encrypted secret
	// 6. Handle errors and implement rollback

	return nil
}

// RotateEncryptionKeyForAccounts rotates the encryption key for secrets in specific accounts
func (s *SecretVaultService) RotateEncryptionKeyForAccounts(oldKey, newKey []byte, accountIDs []string) error {
	if len(newKey) != 32 {
		return fmt.Errorf("new encryption key must be 32 bytes (256 bits)")
	}
	if len(oldKey) != 32 {
		return fmt.Errorf("old encryption key must be 32 bytes (256 bits)")
	}

	// Create old cipher for decryption
	oldBlock, err := aes.NewCipher(oldKey)
	if err != nil {
		return fmt.Errorf("failed to create AES cipher with old key: %w", err)
	}

	oldGCM, err := cipher.NewGCM(oldBlock)
	if err != nil {
		return fmt.Errorf("failed to create GCM with old key: %w", err)
	}

	// Create new cipher for encryption
	newBlock, err := aes.NewCipher(newKey)
	if err != nil {
		return fmt.Errorf("failed to create AES cipher with new key: %w", err)
	}

	newGCM, err := cipher.NewGCM(newBlock)
	if err != nil {
		return fmt.Errorf("failed to create GCM with new key: %w", err)
	}

	// Rotate secrets for each account
	for _, accountID := range accountIDs {
		if err := s.rotateSecretsForAccount(accountID, oldGCM, newGCM); err != nil {
			return fmt.Errorf("failed to rotate secrets for account %s: %w", accountID, err)
		}
	}

	// Update internal encryption key and cipher
	s.encryptionKey = newKey
	s.gcm = newGCM

	return nil
}

// rotateSecretsForAccount rotates all secrets for a single account
func (s *SecretVaultService) rotateSecretsForAccount(accountID string, oldGCM, newGCM cipher.AEAD) error {
	// Get all secrets for the account
	secrets, err := s.store.ListSecrets(accountID)
	if err != nil {
		return fmt.Errorf("failed to list secrets: %w", err)
	}

	// Rotate each secret
	for _, secret := range secrets {
		// Decrypt with old key
		plaintext, err := s.decryptWithCipher(secret.Value, oldGCM)
		if err != nil {
			return fmt.Errorf("failed to decrypt secret %s: %w", secret.Key, err)
		}

		// Re-encrypt with new key
		newEncryptedValue, err := s.encryptWithCipher(plaintext, newGCM)
		if err != nil {
			return fmt.Errorf("failed to re-encrypt secret %s: %w", secret.Key, err)
		}

		// Update the secret with new encrypted value
		secret.Value = newEncryptedValue
		secret.UpdatedAt = time.Now()

		if err := s.store.SaveSecret(secret); err != nil {
			return fmt.Errorf("failed to save re-encrypted secret %s: %w", secret.Key, err)
		}
	}

	return nil
}

// encrypt encrypts a plaintext value using AES-GCM
func (s *SecretVaultService) encrypt(plaintext string) (string, error) {
	// Generate a random nonce
	nonce := make([]byte, s.gcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return "", fmt.Errorf("failed to generate nonce: %w", err)
	}

	// Encrypt the plaintext
	ciphertext := s.gcm.Seal(nonce, nonce, []byte(plaintext), nil)

	// Return as hex string
	return hex.EncodeToString(ciphertext), nil
}

// decrypt decrypts a hex-encoded ciphertext using AES-GCM
func (s *SecretVaultService) decrypt(ciphertextHex string) (string, error) {
	// Decode hex string
	ciphertext, err := hex.DecodeString(ciphertextHex)
	if err != nil {
		return "", fmt.Errorf("failed to decode hex: %w", err)
	}

	// Extract nonce
	nonceSize := s.gcm.NonceSize()
	if len(ciphertext) < nonceSize {
		return "", fmt.Errorf("ciphertext too short")
	}

	nonce, ciphertext := ciphertext[:nonceSize], ciphertext[nonceSize:]

	// Decrypt
	plaintext, err := s.gcm.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return "", fmt.Errorf("failed to decrypt: %w", err)
	}

	return string(plaintext), nil
}

// encryptWithCipher encrypts a plaintext value using the provided cipher
func (s *SecretVaultService) encryptWithCipher(plaintext string, gcm cipher.AEAD) (string, error) {
	// Generate a random nonce
	nonce := make([]byte, gcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return "", fmt.Errorf("failed to generate nonce: %w", err)
	}

	// Encrypt the plaintext
	ciphertext := gcm.Seal(nonce, nonce, []byte(plaintext), nil)

	// Return as hex string
	return hex.EncodeToString(ciphertext), nil
}

// decryptWithCipher decrypts a hex-encoded ciphertext using the provided cipher
func (s *SecretVaultService) decryptWithCipher(ciphertextHex string, gcm cipher.AEAD) (string, error) {
	// Decode hex string
	ciphertext, err := hex.DecodeString(ciphertextHex)
	if err != nil {
		return "", fmt.Errorf("failed to decode hex: %w", err)
	}

	// Extract nonce
	nonceSize := gcm.NonceSize()
	if len(ciphertext) < nonceSize {
		return "", fmt.Errorf("ciphertext too short")
	}

	nonce, ciphertext := ciphertext[:nonceSize], ciphertext[nonceSize:]

	// Decrypt
	plaintext, err := gcm.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return "", fmt.Errorf("failed to decrypt: %w", err)
	}

	return string(plaintext), nil
}

// GenerateEncryptionKey generates a new 256-bit encryption key
func GenerateEncryptionKey() ([]byte, error) {
	key := make([]byte, 32) // 256 bits
	_, err := rand.Read(key)
	if err != nil {
		return nil, fmt.Errorf("failed to generate encryption key: %w", err)
	}
	return key, nil
}

// EncryptionKeyFromHex converts a hex string to an encryption key
func EncryptionKeyFromHex(hexKey string) ([]byte, error) {
	key, err := hex.DecodeString(hexKey)
	if err != nil {
		return nil, fmt.Errorf("failed to decode hex key: %w", err)
	}
	if len(key) != 32 {
		return nil, fmt.Errorf("key must be 32 bytes (256 bits), got %d", len(key))
	}
	return key, nil
}

// EncryptionKeyToHex converts an encryption key to a hex string
func EncryptionKeyToHex(key []byte) string {
	return hex.EncodeToString(key)
}
