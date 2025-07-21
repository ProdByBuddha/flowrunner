package main

import (
	"fmt"
	"log"

	"github.com/tcmartin/flowrunner/pkg/scripting"
	"github.com/tcmartin/flowrunner/pkg/services"
	"github.com/tcmartin/flowrunner/pkg/storage"
)

func main() {
	// Create an in-memory secret store
	secretStore := storage.NewMemorySecretStore()

	// Create a secret vault service
	encKey := make([]byte, 32) // In a real app, use a secure key
	secretVault, err := services.NewSecretVaultService(secretStore, encKey)
	if err != nil {
		log.Fatalf("Failed to create secret vault: %v", err)
	}

	// Set some secrets
	accountID := "example-account"
	err = secretVault.Set(accountID, "API_KEY", "my-secret-api-key")
	if err != nil {
		log.Fatalf("Failed to set secret: %v", err)
	}

	err = secretVault.Set(accountID, "DB_PASSWORD", "super-secret-db-password")
	if err != nil {
		log.Fatalf("Failed to set secret: %v", err)
	}

	// Create a secret-aware expression evaluator
	evaluator := scripting.NewSecretAwareExpressionEvaluator(secretVault)

	// Define a context with the account ID
	context := map[string]any{
		"accountID": accountID,
		"username":  "john.doe",
	}

	// Evaluate expressions that use secrets
	examples := []string{
		"${secrets.API_KEY}",
		"${\"Bearer \" + secrets.API_KEY}",
		"${username + \" is using \" + secrets.API_KEY}",
		"${JSON.stringify({auth: secrets.API_KEY, db: secrets.DB_PASSWORD})}",
	}

	for _, expr := range examples {
		result, err := evaluator.Evaluate(expr, context)
		if err != nil {
			fmt.Printf("Error evaluating %q: %v\n", expr, err)
		} else {
			fmt.Printf("%q => %v\n", expr, result)
		}
	}

	// Evaluate an object with secrets
	obj := map[string]any{
		"headers": map[string]any{
			"Authorization": "${\"Bearer \" + secrets.API_KEY}",
		},
		"database": map[string]any{
			"username": "${username}",
			"password": "${secrets.DB_PASSWORD}",
		},
	}

	result, err := evaluator.EvaluateInObject(obj, context)
	if err != nil {
		fmt.Printf("Error evaluating object: %v\n", err)
	} else {
		fmt.Printf("Object result: %v\n", result)
	}
}
