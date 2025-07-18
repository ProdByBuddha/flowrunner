package storage

import (
	"fmt"
)

// ProviderType represents the type of storage provider
type ProviderType string

const (
	// MemoryProviderType is an in-memory storage provider
	MemoryProviderType ProviderType = "memory"

	// DynamoDBProviderType is a DynamoDB storage provider
	DynamoDBProviderType ProviderType = "dynamodb"

	// PostgreSQLProviderType is a PostgreSQL storage provider
	PostgreSQLProviderType ProviderType = "postgresql"
)

// ProviderConfig contains configuration for storage providers
type ProviderConfig struct {
	// Type is the type of storage provider to create
	Type ProviderType

	// DynamoDB contains configuration for the DynamoDB provider
	DynamoDB *DynamoDBProviderConfig

	// PostgreSQL contains configuration for the PostgreSQL provider
	PostgreSQL *PostgreSQLProviderConfig
}

// NewProvider creates a new storage provider based on the configuration
func NewProvider(config ProviderConfig) (StorageProvider, error) {
	switch config.Type {
	case MemoryProviderType:
		return NewMemoryProvider(), nil

	case DynamoDBProviderType:
		if config.DynamoDB == nil {
			return nil, fmt.Errorf("DynamoDB configuration is required for DynamoDB provider")
		}
		return NewDynamoDBProvider(*config.DynamoDB)

	case PostgreSQLProviderType:
		if config.PostgreSQL == nil {
			return nil, fmt.Errorf("PostgreSQL configuration is required for PostgreSQL provider")
		}
		return NewPostgreSQLProvider(*config.PostgreSQL)

	default:
		return nil, fmt.Errorf("unknown provider type: %s", config.Type)
	}
}
