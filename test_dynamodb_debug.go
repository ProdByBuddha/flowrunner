package main

import (
	"fmt"
	"log"
	"time"

	"github.com/tcmartin/flowrunner/pkg/storage"
)

func main() {
	// Test DynamoDB account operations with debugging
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

	// Wait a bit for indexes to become active
	fmt.Println("Waiting for indexes to become active...")
	time.Sleep(5 * time.Second)

	// Test account store directly
	accountStore := provider.GetAccountStore()

	// Create account
	username := fmt.Sprintf("debuguser-%d", time.Now().Unix())

	fmt.Printf("Creating account with username: %s\n", username)

	// We need to create the account through the account service to get proper password hashing
	// But let's first check if we can retrieve accounts

	// List all accounts first
	fmt.Println("Listing all accounts...")
	accounts, err := accountStore.ListAccounts()
	if err != nil {
		log.Printf("Failed to list accounts: %v", err)
	} else {
		fmt.Printf("Found %d accounts\n", len(accounts))
		for i, account := range accounts {
			fmt.Printf("Account %d: ID=%s, Username=%s\n", i+1, account.ID, account.Username)
		}
	}

	fmt.Println("Test completed!")
}
