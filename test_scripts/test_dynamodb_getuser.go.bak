package main

import (
	"fmt"
	"log"
	"time"

	"github.com/tcmartin/flowrunner/pkg/services"
	"github.com/tcmartin/flowrunner/pkg/storage"
	"golang.org/x/crypto/bcrypt"
)

func main() {
	// Test DynamoDB GetAccountByUsername specifically
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

	// Wait for indexes to become active
	time.Sleep(5 * time.Second)

	accountStore := provider.GetAccountStore()
	accountService := services.NewAccountService(accountStore)

	// Create account using account service (proper password hashing)
	username := fmt.Sprintf("getuser-%d", time.Now().Unix())
	password := "testpassword"

	fmt.Printf("Creating account with username: %s\n", username)
	accountID, err := accountService.CreateAccount(username, password)
	if err != nil {
		log.Fatalf("Failed to create account: %v", err)
	}
	fmt.Printf("Account created with ID: %s\n", accountID)

	// Wait for eventual consistency
	time.Sleep(3 * time.Second)

	// Test GetAccountByUsername directly
	fmt.Println("Testing GetAccountByUsername...")
	account, err := accountStore.GetAccountByUsername(username)
	if err != nil {
		log.Printf("GetAccountByUsername failed: %v", err)

		// List all accounts to see what's there
		fmt.Println("Listing all accounts...")
		accounts, listErr := accountStore.ListAccounts()
		if listErr != nil {
			log.Printf("Failed to list accounts: %v", listErr)
		} else {
			fmt.Printf("Found %d accounts:\n", len(accounts))
			for i, acc := range accounts {
				fmt.Printf("  %d. ID=%s, Username=%s\n", i+1, acc.ID, acc.Username)
			}
		}
		return
	}

	fmt.Printf("Found account: ID=%s, Username=%s\n", account.ID, account.Username)

	// Test password verification
	fmt.Println("Testing password verification...")
	err = bcrypt.CompareHashAndPassword([]byte(account.PasswordHash), []byte(password))
	if err != nil {
		log.Printf("Password verification failed: %v", err)
	} else {
		fmt.Println("Password verification successful!")
	}

	// Test authentication through service
	fmt.Println("Testing authentication through service...")
	authID, err := accountService.Authenticate(username, password)
	if err != nil {
		log.Printf("Authentication failed: %v", err)
	} else {
		fmt.Printf("Authentication successful! Account ID: %s\n", authID)
	}

	fmt.Println("Test completed!")
}
