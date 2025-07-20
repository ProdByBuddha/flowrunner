package main

import (
	"fmt"
	"log"

	"github.com/tcmartin/flowrunner/pkg/storage"
)

func main() {
	// Test DynamoDB provider initialization
	config := storage.DynamoDBProviderConfig{
		Region:      "us-west-2",
		TablePrefix: "flowrunner_test_",
		Endpoint:    "http://localhost:8000",
	}

	fmt.Println("Creating DynamoDB provider...")
	provider, err := storage.NewDynamoDBProvider(config)
	if err != nil {
		log.Fatalf("Failed to create DynamoDB provider: %v", err)
	}

	fmt.Println("Initializing DynamoDB provider...")
	if err := provider.Initialize(); err != nil {
		log.Fatalf("Failed to initialize DynamoDB provider: %v", err)
	}

	fmt.Println("DynamoDB provider initialized successfully!")

	// Test account store
	accountStore := provider.GetAccountStore()
	fmt.Printf("Account store: %T\n", accountStore)

	fmt.Println("Test completed successfully!")
}
