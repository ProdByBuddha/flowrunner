//go:build ignore

package main

import (
	"fmt"
	"log"

	"github.com/tcmartin/flowrunner/pkg/scripting"
	"github.com/tcmartin/flowrunner/pkg/services"
	"github.com/tcmartin/flowrunner/pkg/storage"
)

func main() {
	// Create in-memory storage
	storageProvider := storage.NewMemoryProvider()
	if err := storageProvider.Initialize(); err != nil {
		log.Fatal(err)
	}

	// Create secret vault
	encryptionKey, err := services.GenerateEncryptionKey()
	if err != nil {
		log.Fatal(err)
	}

	secretVault, err := services.NewExtendedSecretVaultService(storageProvider.GetSecretStore(), encryptionKey)
	if err != nil {
		log.Fatal(err)
	}

	// Store a test secret
	accountID := "test-account"
	err = secretVault.Set(accountID, "OPENAI_API_KEY", "test-api-key-value")
	if err != nil {
		log.Fatal(err)
	}

	// Test secret resolution
	evaluator := scripting.NewSecretAwareExpressionEvaluator(secretVault)

	context := map[string]interface{}{
		"accountID": accountID,
	}

	// Test the expression
	result, err := evaluator.Evaluate("${secrets.OPENAI_API_KEY}", context)
	if err != nil {
		log.Printf("Error evaluating expression: %v", err)
	} else {
		log.Printf("Result: %v", result)
	}

	// Test direct secret access
	secretValue, err := secretVault.Get(accountID, "OPENAI_API_KEY")
	if err != nil {
		log.Printf("Error getting secret directly: %v", err)
	} else {
		log.Printf("Direct secret access: %v", secretValue)
	}

	// Test list secrets
	keys, err := secretVault.List(accountID)
	if err != nil {
		log.Printf("Error listing secrets: %v", err)
	} else {
		log.Printf("Available secret keys: %v", keys)
	}
}
