package storage

import (
	"os"
	"testing"

	"github.com/joho/godotenv"
	"github.com/stretchr/testify/assert"
)

func init() {
	// Load .env file from project root
	_ = godotenv.Load("../../.env")
}

func TestNewProvider(t *testing.T) {
	// Test memory provider
	memoryConfig := ProviderConfig{
		Type: MemoryProviderType,
	}

	memoryProvider, err := NewProvider(memoryConfig)
	assert.NoError(t, err)
	assert.NotNil(t, memoryProvider)
	assert.IsType(t, &MemoryProvider{}, memoryProvider)

	// Test DynamoDB provider with missing config
	dynamoDBConfigMissing := ProviderConfig{
		Type: DynamoDBProviderType,
	}

	_, err = NewProvider(dynamoDBConfigMissing)
	assert.Error(t, err)

	// Test DynamoDB provider with config
	dynamoDBConfig := ProviderConfig{
		Type: DynamoDBProviderType,
		DynamoDB: &DynamoDBProviderConfig{
			Region:      "us-east-1",
			TablePrefix: "test_",
			AccessKey:   os.Getenv("AWS_ACCESS_KEY_ID"),
			SecretKey:   os.Getenv("AWS_SECRET_ACCESS_KEY"),
		},
	}

	// Skip if AWS credentials are not set
	if dynamoDBConfig.DynamoDB.AccessKey != "" && dynamoDBConfig.DynamoDB.SecretKey != "" {
		dynamoProvider, err := NewProvider(dynamoDBConfig)
		assert.NoError(t, err)
		assert.NotNil(t, dynamoProvider)
		assert.IsType(t, &DynamoDBProvider{}, dynamoProvider)
	}

	// Test PostgreSQL provider with missing config
	postgresConfigMissing := ProviderConfig{
		Type: PostgreSQLProviderType,
	}

	_, err = NewProvider(postgresConfigMissing)
	assert.Error(t, err)

	// Test PostgreSQL provider with config
	postgresConfig := ProviderConfig{
		Type: PostgreSQLProviderType,
		PostgreSQL: &PostgreSQLProviderConfig{
			Host:     os.Getenv("POSTGRES_HOST"),
			User:     os.Getenv("POSTGRES_USER"),
			Password: os.Getenv("POSTGRES_PASSWORD"),
			Database: os.Getenv("POSTGRES_DB"),
			SSLMode:  "disable",
		},
	}

	// Skip if PostgreSQL credentials are not set
	if postgresConfig.PostgreSQL.Host != "" && postgresConfig.PostgreSQL.User != "" {
		postgresProvider, err := NewProvider(postgresConfig)
		assert.NoError(t, err)
		assert.NotNil(t, postgresProvider)
		assert.IsType(t, &PostgreSQLProvider{}, postgresProvider)
	}

	// Test unknown provider
	unknownConfig := ProviderConfig{
		Type: "unknown",
	}

	_, err = NewProvider(unknownConfig)
	assert.Error(t, err)
}
