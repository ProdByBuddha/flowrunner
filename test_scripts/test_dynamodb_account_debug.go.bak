package main

import (
	"fmt"
	"log"
	"time"

	"github.com/tcmartin/flowrunner/pkg/auth"
	"github.com/tcmartin/flowrunner/pkg/storage"
	"golang.org/x/crypto/bcrypt"
)

func main() {
	// Test DynamoDB account store
	config := storage.DynamoDBProviderConfig{
		Region:      "us-west-2",
		TablePrefix: "flowrunner_debug_",
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

	accountStore := provider.GetAccountStore()

	// Create a test account with proper bcrypt hash
	password := "testpassword123"
	passwordHash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		log.Fatalf("Failed to hash password: %v", err)
	}

	account := auth.Account{
		ID:           "test-debug-account-id",
		Username:     "debuguser",
		PasswordHash: string(passwordHash),
		APIToken:     "debug-api-token-12345",
		CreatedAt:    time.Now(),
		UpdatedAt:    time.Now(),
	}

	fmt.Printf("Original account:\n")
	fmt.Printf("  ID: %s\n", account.ID)
	fmt.Printf("  Username: %s\n", account.Username)
	fmt.Printf("  PasswordHash length: %d\n", len(account.PasswordHash))
	fmt.Printf("  PasswordHash starts with: %s\n", account.PasswordHash[:10])
	fmt.Printf("  APIToken: %s\n", account.APIToken)

	// Save the account
	fmt.Println("\nSaving account...")
	if err := accountStore.SaveAccount(account); err != nil {
		log.Fatalf("Failed to save account: %v", err)
	}

	// Retrieve the account by username
	fmt.Println("Retrieving account by username...")
	retrievedAccount, err := accountStore.GetAccountByUsername("debuguser")
	if err != nil {
		log.Fatalf("Failed to get account by username: %v", err)
	}

	fmt.Printf("\nRetrieved account:\n")
	fmt.Printf("  ID: %s\n", retrievedAccount.ID)
	fmt.Printf("  Username: %s\n", retrievedAccount.Username)
	fmt.Printf("  PasswordHash length: %d\n", len(retrievedAccount.PasswordHash))
	if len(retrievedAccount.PasswordHash) > 10 {
		fmt.Printf("  PasswordHash starts with: %s\n", retrievedAccount.PasswordHash[:10])
	} else {
		fmt.Printf("  PasswordHash (full): %s\n", retrievedAccount.PasswordHash)
	}
	fmt.Printf("  APIToken: %s\n", retrievedAccount.APIToken)

	// Test password verification
	fmt.Println("\nTesting password verification...")
	err = bcrypt.CompareHashAndPassword([]byte(retrievedAccount.PasswordHash), []byte(password))
	if err != nil {
		fmt.Printf("Password verification failed: %v\n", err)
	} else {
		fmt.Println("Password verification successful!")
	}

	// Compare original vs retrieved
	fmt.Println("\nComparison:")
	fmt.Printf("  Original hash == Retrieved hash: %t\n", account.PasswordHash == retrievedAccount.PasswordHash)
	fmt.Printf("  Original ID == Retrieved ID: %t\n", account.ID == retrievedAccount.ID)
	fmt.Printf("  Original Username == Retrieved Username: %t\n", account.Username == retrievedAccount.Username)

	fmt.Println("\nTest completed!")
}
