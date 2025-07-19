package services

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/tcmartin/flowrunner/pkg/auth"
	"github.com/tcmartin/flowrunner/pkg/storage"
)

// ExtendedSecretVaultService implements the auth.ExtendedSecretVault interface
type ExtendedSecretVaultService struct {
	*SecretVaultService
}

// NewExtendedSecretVaultService creates a new extended secret vault service
func NewExtendedSecretVaultService(store storage.SecretStore, encryptionKey []byte) (*ExtendedSecretVaultService, error) {
	baseService, err := NewSecretVaultService(store, encryptionKey)
	if err != nil {
		return nil, err
	}
	
	return &ExtendedSecretVaultService{
		SecretVaultService: baseService,
	}, nil
}

// SetStructured stores a structured secret
func (s *ExtendedSecretVaultService) SetStructured(accountID string, secret auth.StructuredSecret) error {
	if accountID == "" {
		return fmt.Errorf("account ID is required")
	}
	if secret.Key == "" {
		return fmt.Errorf("secret key is required")
	}

	// Validate the secret data against schema if provided
	if secret.Schema != nil {
		if err := s.validateSecretData(secret.Value, *secret.Schema); err != nil {
			return fmt.Errorf("secret validation failed: %w", err)
		}
	}

	// Encrypt the value
	encryptedValue, err := s.encrypt(secret.Value)
	if err != nil {
		return fmt.Errorf("failed to encrypt secret: %w", err)
	}

	// Create metadata JSON
	metadataJSON, err := json.Marshal(secret.Metadata)
	if err != nil {
		return fmt.Errorf("failed to marshal metadata: %w", err)
	}

	// Create schema JSON if provided
	var schemaJSON string
	if secret.Schema != nil {
		schemaBytes, err := json.Marshal(secret.Schema)
		if err != nil {
			return fmt.Errorf("failed to marshal schema: %w", err)
		}
		schemaJSON = string(schemaBytes)
	}

	// Store as a compound secret with metadata
	compoundSecret := auth.Secret{
		AccountID: accountID,
		Key:       secret.Key,
		Value:     s.createCompoundValue(encryptedValue, string(metadataJSON), schemaJSON),
		CreatedAt: secret.CreatedAt,
		UpdatedAt: secret.UpdatedAt,
	}

	// Check if secret already exists to preserve creation time
	existingSecret, err := s.store.GetSecret(accountID, secret.Key)
	if err == nil {
		compoundSecret.CreatedAt = existingSecret.CreatedAt
	} else {
		compoundSecret.CreatedAt = time.Now()
	}
	compoundSecret.UpdatedAt = time.Now()

	return s.store.SaveSecret(compoundSecret)
}

// GetStructured retrieves a structured secret
func (s *ExtendedSecretVaultService) GetStructured(accountID string, key string) (auth.StructuredSecret, error) {
	if accountID == "" {
		return auth.StructuredSecret{}, fmt.Errorf("account ID is required")
	}
	if key == "" {
		return auth.StructuredSecret{}, fmt.Errorf("secret key is required")
	}

	// Retrieve the secret
	secret, err := s.store.GetSecret(accountID, key)
	if err != nil {
		return auth.StructuredSecret{}, err
	}

	// Parse compound value
	encryptedValue, metadataJSON, schemaJSON, err := s.parseCompoundValue(secret.Value)
	if err != nil {
		// Fallback to simple secret format
		decryptedValue, err := s.decrypt(secret.Value)
		if err != nil {
			return auth.StructuredSecret{}, fmt.Errorf("failed to decrypt secret: %w", err)
		}
		
		return auth.StructuredSecret{
			AccountID: accountID,
			Key:       key,
			CreatedAt: secret.CreatedAt,
			UpdatedAt: secret.UpdatedAt,
			Metadata: auth.SecretMetadata{
				Type:    auth.SecretTypeGeneral,
				Version: 1,
			},
			Value: decryptedValue,
		}, nil
	}

	// Decrypt the value
	decryptedValue, err := s.decrypt(encryptedValue)
	if err != nil {
		return auth.StructuredSecret{}, fmt.Errorf("failed to decrypt secret: %w", err)
	}

	// Parse metadata
	var metadata auth.SecretMetadata
	if metadataJSON != "" {
		if err := json.Unmarshal([]byte(metadataJSON), &metadata); err != nil {
			return auth.StructuredSecret{}, fmt.Errorf("failed to unmarshal metadata: %w", err)
		}
	}

	// Parse schema
	var schema *auth.SecretSchema
	if schemaJSON != "" {
		schema = &auth.SecretSchema{}
		if err := json.Unmarshal([]byte(schemaJSON), schema); err != nil {
			return auth.StructuredSecret{}, fmt.Errorf("failed to unmarshal schema: %w", err)
		}
	}

	// Mark as used
	s.MarkUsed(accountID, key)

	return auth.StructuredSecret{
		AccountID: accountID,
		Key:       key,
		CreatedAt: secret.CreatedAt,
		UpdatedAt: secret.UpdatedAt,
		Metadata:  metadata,
		Value:     decryptedValue,
		Schema:    schema,
	}, nil
}

// GetField retrieves a specific field from a structured secret
func (s *ExtendedSecretVaultService) GetField(accountID string, key string, field string) (interface{}, error) {
	secret, err := s.GetStructured(accountID, key)
	if err != nil {
		return nil, err
	}

	// Parse the value as JSON
	var data map[string]interface{}
	if err := json.Unmarshal([]byte(secret.Value), &data); err != nil {
		// If not JSON, treat as simple value
		if field == "value" || field == "" {
			return secret.Value, nil
		}
		return nil, fmt.Errorf("secret is not structured, field '%s' not found", field)
	}

	// Support dot notation for nested fields
	return s.getNestedField(data, field)
}

// SetOAuth stores OAuth credentials
func (s *ExtendedSecretVaultService) SetOAuth(accountID string, key string, oauth auth.OAuthSecret, metadata auth.SecretMetadata) error {
	// Set type
	metadata.Type = auth.SecretTypeOAuth
	if metadata.Version == 0 {
		metadata.Version = 1
	}

	// Convert to JSON
	valueJSON, err := auth.ToJSON(oauth)
	if err != nil {
		return fmt.Errorf("failed to marshal OAuth secret: %w", err)
	}

	schema := auth.GetOAuthSecretSchema()
	
	structured := auth.StructuredSecret{
		AccountID: accountID,
		Key:       key,
		Metadata:  metadata,
		Value:     valueJSON,
		Schema:    &schema,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	return s.SetStructured(accountID, structured)
}

// SetAPIKey stores API key credentials
func (s *ExtendedSecretVaultService) SetAPIKey(accountID string, key string, apiKey auth.APIKeySecret, metadata auth.SecretMetadata) error {
	metadata.Type = auth.SecretTypeAPIKey
	if metadata.Version == 0 {
		metadata.Version = 1
	}

	valueJSON, err := auth.ToJSON(apiKey)
	if err != nil {
		return fmt.Errorf("failed to marshal API key secret: %w", err)
	}

	schema := auth.GetAPIKeySecretSchema()

	structured := auth.StructuredSecret{
		AccountID: accountID,
		Key:       key,
		Metadata:  metadata,
		Value:     valueJSON,
		Schema:    &schema,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	return s.SetStructured(accountID, structured)
}

// SetDatabase stores database credentials
func (s *ExtendedSecretVaultService) SetDatabase(accountID string, key string, db auth.DatabaseSecret, metadata auth.SecretMetadata) error {
	metadata.Type = auth.SecretTypeDatabase
	if metadata.Version == 0 {
		metadata.Version = 1
	}

	valueJSON, err := auth.ToJSON(db)
	if err != nil {
		return fmt.Errorf("failed to marshal database secret: %w", err)
	}

	schema := auth.GetDatabaseSecretSchema()

	structured := auth.StructuredSecret{
		AccountID: accountID,
		Key:       key,
		Metadata:  metadata,
		Value:     valueJSON,
		Schema:    &schema,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	return s.SetStructured(accountID, structured)
}

// SetJWT stores JWT token
func (s *ExtendedSecretVaultService) SetJWT(accountID string, key string, jwt auth.JWTSecret, metadata auth.SecretMetadata) error {
	metadata.Type = auth.SecretTypeJWT
	if metadata.Version == 0 {
		metadata.Version = 1
	}

	valueJSON, err := auth.ToJSON(jwt)
	if err != nil {
		return fmt.Errorf("failed to marshal JWT secret: %w", err)
	}

	structured := auth.StructuredSecret{
		AccountID: accountID,
		Key:       key,
		Metadata:  metadata,
		Value:     valueJSON,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	return s.SetStructured(accountID, structured)
}

// SetCustom stores arbitrary structured data
func (s *ExtendedSecretVaultService) SetCustom(accountID string, key string, data map[string]interface{}, metadata auth.SecretMetadata) error {
	metadata.Type = auth.SecretTypeCustom
	if metadata.Version == 0 {
		metadata.Version = 1
	}

	valueJSON, err := auth.ToJSON(data)
	if err != nil {
		return fmt.Errorf("failed to marshal custom secret: %w", err)
	}

	structured := auth.StructuredSecret{
		AccountID: accountID,
		Key:       key,
		Metadata:  metadata,
		Value:     valueJSON,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	return s.SetStructured(accountID, structured)
}

// ListByType returns secrets filtered by type
func (s *ExtendedSecretVaultService) ListByType(accountID string, secretType auth.SecretType) ([]auth.StructuredSecret, error) {
	query := auth.SecretQuery{Type: secretType}
	return s.Search(accountID, query)
}

// ListByTags returns secrets filtered by tags
func (s *ExtendedSecretVaultService) ListByTags(accountID string, tags []string) ([]auth.StructuredSecret, error) {
	query := auth.SecretQuery{Tags: tags}
	return s.Search(accountID, query)
}

// Search searches secrets by metadata
func (s *ExtendedSecretVaultService) Search(accountID string, query auth.SecretQuery) ([]auth.StructuredSecret, error) {
	if accountID == "" {
		return nil, fmt.Errorf("account ID is required")
	}

	// Get all secrets for the account
	secrets, err := s.store.ListSecrets(accountID)
	if err != nil {
		return nil, err
	}

	var results []auth.StructuredSecret
	count := 0

	for _, secret := range secrets {
		// Skip if offset not reached
		if query.Offset > 0 && count < query.Offset {
			count++
			continue
		}

		// Stop if limit reached
		if query.Limit > 0 && len(results) >= query.Limit {
			break
		}

		// Parse the secret
		structured, err := s.GetStructured(accountID, secret.Key)
		if err != nil {
			continue // Skip invalid secrets
		}

		// Apply filters
		if !s.matchesQuery(structured, query) {
			continue
		}

		// Clear the value for security
		structured.Value = ""
		results = append(results, structured)
		count++
	}

	return results, nil
}

// UpdateMetadata updates only the metadata of a secret
func (s *ExtendedSecretVaultService) UpdateMetadata(accountID string, key string, metadata auth.SecretMetadata) error {
	// Get existing secret
	existing, err := s.GetStructured(accountID, key)
	if err != nil {
		return err
	}

	// Update metadata
	existing.Metadata = metadata
	existing.UpdatedAt = time.Now()

	return s.SetStructured(accountID, existing)
}

// MarkUsed updates the last used timestamp
func (s *ExtendedSecretVaultService) MarkUsed(accountID string, key string) error {
	// Get existing secret
	existing, err := s.GetStructured(accountID, key)
	if err != nil {
		return err
	}

	// Update last used
	now := time.Now()
	existing.Metadata.LastUsed = &now
	existing.UpdatedAt = now

	return s.SetStructured(accountID, existing)
}

// GetExpiring returns secrets that expire within the given duration
func (s *ExtendedSecretVaultService) GetExpiring(accountID string, within time.Duration) ([]auth.StructuredSecret, error) {
	query := auth.SecretQuery{ExpiringWithin: &within}
	return s.Search(accountID, query)
}

// Helper methods

// createCompoundValue creates a compound value containing encrypted data, metadata, and schema
func (s *ExtendedSecretVaultService) createCompoundValue(encryptedValue, metadataJSON, schemaJSON string) string {
	compound := map[string]string{
		"value":    encryptedValue,
		"metadata": metadataJSON,
		"schema":   schemaJSON,
	}
	
	bytes, _ := json.Marshal(compound)
	return string(bytes)
}

// parseCompoundValue parses a compound value into its components
func (s *ExtendedSecretVaultService) parseCompoundValue(compoundValue string) (encryptedValue, metadataJSON, schemaJSON string, err error) {
	var compound map[string]string
	if err := json.Unmarshal([]byte(compoundValue), &compound); err != nil {
		return "", "", "", err
	}

	return compound["value"], compound["metadata"], compound["schema"], nil
}

// getNestedField retrieves a nested field using dot notation
func (s *ExtendedSecretVaultService) getNestedField(data map[string]interface{}, field string) (interface{}, error) {
	if field == "" {
		return data, nil
	}

	parts := strings.Split(field, ".")
	current := data

	for i, part := range parts {
		value, exists := current[part]
		if !exists {
			return nil, fmt.Errorf("field '%s' not found", field)
		}

		// If this is the last part, return the value
		if i == len(parts)-1 {
			return value, nil
		}

		// Otherwise, continue navigating
		if nextMap, ok := value.(map[string]interface{}); ok {
			current = nextMap
		} else {
			return nil, fmt.Errorf("cannot navigate to field '%s': intermediate value is not an object", field)
		}
	}

	return current, nil
}

// matchesQuery checks if a secret matches the given query
func (s *ExtendedSecretVaultService) matchesQuery(secret auth.StructuredSecret, query auth.SecretQuery) bool {
	// Filter by type
	if query.Type != "" && secret.Metadata.Type != query.Type {
		return false
	}

	// Filter by tags (AND operation)
	if len(query.Tags) > 0 {
		secretTags := make(map[string]bool)
		for _, tag := range secret.Metadata.Tags {
			secretTags[tag] = true
		}
		
		for _, requiredTag := range query.Tags {
			if !secretTags[requiredTag] {
				return false
			}
		}
	}

	// Filter by description
	if query.Description != "" {
		if !strings.Contains(strings.ToLower(secret.Metadata.Description), strings.ToLower(query.Description)) {
			return false
		}
	}

	// Filter by expiring within duration
	if query.ExpiringWithin != nil && secret.Metadata.ExpiresAt != nil {
		if time.Until(*secret.Metadata.ExpiresAt) > *query.ExpiringWithin {
			return false
		}
	}

	// Filter by last used before
	if query.LastUsedBefore != nil && secret.Metadata.LastUsed != nil {
		if secret.Metadata.LastUsed.After(*query.LastUsedBefore) {
			return false
		}
	}

	return true
}

// validateSecretData validates secret data against a schema
func (s *ExtendedSecretVaultService) validateSecretData(valueJSON string, schema auth.SecretSchema) error {
	var data map[string]interface{}
	if err := json.Unmarshal([]byte(valueJSON), &data); err != nil {
		return fmt.Errorf("invalid JSON: %w", err)
	}

	// Check required fields
	for _, required := range schema.Required {
		if _, exists := data[required]; !exists {
			return fmt.Errorf("required field '%s' is missing", required)
		}
	}

	// TODO: Add more detailed validation based on field definitions
	// This could include type checking, pattern matching, etc.

	return nil
}
