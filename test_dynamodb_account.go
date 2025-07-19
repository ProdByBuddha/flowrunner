package main

import (
	"fmt"
	"log"
	"time"

	"github.com/tcmartin/flowrunner/pkg/services"
	"github.com/tcmartin/flowrunner/pkg/storage"
)

func main() {
	// Test DynamoDB account operations
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

	// Test account service
	accountService := services.NewAccountService(provider.GetAccountStore())
	accountService = accountService.WithJWTService("test-jwt-secret", 24)

	// Create account
	username := fmt.Sprintf("testuser-%d", time.Now().Unix())
	password := "testpassword"

	fmt.Printf("Creating account with username: %s\n", username)
	accountID, err := accountService.CreateAccount(username, password)
	if err != nil {
		log.Fatalf("Failed to create account: %v", err)
	}

	fmt.Printf("Account created with ID: %s\n", accountID)

	// Wait a bit for eventual consistency
	time.Sleep(2 * time.Second)

	// Test authentication
	fmt.Println("Testing authentication...")
	authAccountID, err := accountService.Authenticate(username, password)
	if err != nil {
		log.Fatalf("Failed to authenticate: %v", err)
	}

	fmt.Printf("Authentication successful! Account ID: %s\n", authAccountID)

	// Test JWT generation
	fmt.Println("Testing JWT generation...")
	token, err := accountService.GenerateJWT(accountID)
	if err != nil {
		log.Fatalf("Failed to generate JWT: %v", err)
	}

	fmt.Printf("JWT generated: %s\n", token[:50]+"...")

	fmt.Println("All tests passed successfully!")
}
