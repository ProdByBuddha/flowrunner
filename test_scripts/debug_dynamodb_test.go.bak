package main

import (
	"fmt"
	"time"

	"github.com/aws/aws-sdk-go/service/dynamodb/dynamodbattribute"
	"github.com/tcmartin/flowrunner/pkg/auth"
	"github.com/tcmartin/flowrunner/pkg/storage"
)

func main() {
	// Create mock DynamoDB and provider
	mockClient := storage.NewMockDynamoDBAPI()
	provider := storage.NewDynamoDBProviderWithClient(mockClient, "test_")

	// Initialize the provider (creates tables)
	if err := provider.Initialize(); err != nil {
		panic(fmt.Sprintf("Failed to initialize provider: %v", err))
	}

	// Get the secret store
	secretStore := provider.GetSecretStore()

	// Create a test secret
	secret := auth.Secret{
		AccountID: "test-account",
		Key:       "test-key",
		Value:     "test-value",
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	fmt.Printf("Original secret: %+v\n", secret)

	// Marshal the secret to see what DynamoDB sees
	av, err := dynamodbattribute.MarshalMap(secret)
	if err != nil {
		panic(fmt.Sprintf("Failed to marshal secret: %v", err))
	}
	fmt.Printf("Marshaled attributes: %+v\n", av)

	// Save the secret
	fmt.Println("Saving secret...")
	if err := secretStore.SaveSecret(secret); err != nil {
		panic(fmt.Sprintf("Failed to save secret: %v", err))
	}

	// Try to retrieve the secret
	fmt.Println("Retrieving secret...")
	retrievedSecret, err := secretStore.GetSecret("test-account", "test-key")
	if err != nil {
		fmt.Printf("Failed to retrieve secret: %v\n", err)
	} else {
		fmt.Printf("Retrieved secret: %+v\n", retrievedSecret)
	}
}
