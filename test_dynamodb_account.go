package main

import (
	"fmt"
	"log"
	"time"

	"github.com/google/uuid"
	"github.com/tcmartin/flowrunner/pkg/auth"
	"github.com/tcmartin/flowrunner/pkg/storage"
	"golang.org/x/crypto/bcrypt"
)

func main() {
	// Test DynamoDB account store
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

	// Test account creation
	accountStore := provider.GetAccountStore()

	// Create a test account
	passwordHash, err := bcrypt.GenerateFromPassword([]byte("testpassword"), bcrypt.DefaultCost)
	if err != nil {
		log.Fatalf("Failed to hash password: %v", err)
	}

	accountID := uuid.New().String()
	now := time.Now()

	account := auth.Account{
		ID:           accountID,
		Username:     "testuser",
		PasswordHash: string(passwordHash),
		APIToken:     "test-api-token-" + accountID,
		CreatedAt:    now,
		UpdatedAt:    now,
	}

	fmt.Printf("Creating account with ID: %s\n", accountID)
	fmt.Printf("Account data: %+v\n", account)

	if err := accountStore.SaveAccount(account); err != nil {
		log.Fatalf("Failed to save account: %v", err)
	}

	fmt.Println("Account saved successfully!")

	// Try to retrieve the account
	retrievedAccount, err := accountStore.GetAccount(accountID)
	if err != nil {
		log.Fatalf("Failed to retrieve account: %v", err)
	}

	fmt.Printf("Retrieved account: %+v\n", retrievedAccount)
	fmt.Println("Test completed successfully!")
}
