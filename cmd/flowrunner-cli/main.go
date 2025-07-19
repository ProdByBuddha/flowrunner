// Package main provides a CLI for interacting with the flowrunner server.
package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"
)

var (
	// Global flags
	serverURL  string
	username   string
	password   string
	token      string
	configPath string
)

// Config represents the CLI configuration
type Config struct {
	ServerURL string `json:"server_url"`
	Username  string `json:"username"`
	Token     string `json:"token"`
	JWTToken  string `json:"jwt_token"`
}

func main() {
	// Root command
	rootCmd := &cobra.Command{
		Use:   "flowrunner-cli",
		Short: "FlowRunner CLI",
		Long:  "Command-line interface for interacting with the FlowRunner server",
		PersistentPreRun: func(cmd *cobra.Command, args []string) {
			// Load config if not explicitly provided
			if serverURL == "" || (username == "" && token == "") {
				loadConfig()
			}
		},
	}

	// Add global flags
	rootCmd.PersistentFlags().StringVar(&serverURL, "server", "", "Server URL")
	rootCmd.PersistentFlags().StringVar(&username, "username", "", "Username")
	rootCmd.PersistentFlags().StringVar(&password, "password", "", "Password")
	rootCmd.PersistentFlags().StringVar(&token, "token", "", "API token")
	rootCmd.PersistentFlags().StringVar(&configPath, "config", "", "Path to config file")

	// Add login command
	rootCmd.AddCommand(loginCmd)

	// Account commands
	accountCmd := &cobra.Command{
		Use:   "account",
		Short: "Account management",
	}

	accountCreateCmd := &cobra.Command{
		Use:   "create",
		Short: "Create a new account",
		Run:   createAccount,
	}

	accountInfoCmd := &cobra.Command{
		Use:   "info",
		Short: "Get account information",
		Run:   getAccountInfo,
	}

	accountCmd.AddCommand(accountCreateCmd, accountInfoCmd)

	// Flow commands
	flowCmd := &cobra.Command{
		Use:   "flow",
		Short: "Flow management",
	}

	flowListCmd := &cobra.Command{
		Use:   "list",
		Short: "List flows",
		Run:   listFlows,
	}

	flowCreateCmd := &cobra.Command{
		Use:   "create [name] [file]",
		Short: "Create a new flow",
		Args:  cobra.ExactArgs(2),
		Run:   createFlow,
	}

	flowGetCmd := &cobra.Command{
		Use:   "get [id]",
		Short: "Get a flow",
		Args:  cobra.ExactArgs(1),
		Run:   getFlow,
	}

	flowUpdateCmd := &cobra.Command{
		Use:   "update [id] [file]",
		Short: "Update a flow",
		Args:  cobra.ExactArgs(2),
		Run:   updateFlow,
	}

	flowDeleteCmd := &cobra.Command{
		Use:   "delete [id]",
		Short: "Delete a flow",
		Args:  cobra.ExactArgs(1),
		Run:   deleteFlow,
	}

	flowCmd.AddCommand(flowListCmd, flowCreateCmd, flowGetCmd, flowUpdateCmd, flowDeleteCmd)

	// Secret commands
	secretCmd := &cobra.Command{
		Use:   "secret",
		Short: "Secret management",
	}

	secretListCmd := &cobra.Command{
		Use:   "list",
		Short: "List secret keys",
		Run:   listSecrets,
	}

	secretGetCmd := &cobra.Command{
		Use:   "get [key]",
		Short: "Get a secret value",
		Args:  cobra.ExactArgs(1),
		Run:   getSecret,
	}

	secretSetCmd := &cobra.Command{
		Use:   "set [key] [value]",
		Short: "Set a secret value",
		Args:  cobra.ExactArgs(2),
		Run:   setSecret,
	}

	secretDeleteCmd := &cobra.Command{
		Use:   "delete [key]",
		Short: "Delete a secret",
		Args:  cobra.ExactArgs(1),
		Run:   deleteSecret,
	}

	secretCmd.AddCommand(secretListCmd, secretGetCmd, secretSetCmd, secretDeleteCmd)

	// Add commands to root
	rootCmd.AddCommand(accountCmd, flowCmd, secretCmd)

	// Execute
	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}

// loadConfig loads the CLI configuration
func loadConfig() {
	// If a config path is specified, use it
	if configPath == "" {
		home, err := os.UserHomeDir()
		if err == nil {
			configPath = filepath.Join(home, ".flowrunner", "cli-config.json")
		}
	}

	// If the config file doesn't exist, return
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		return
	}

	// Read the config file
	data, err := os.ReadFile(configPath)
	if err != nil {
		fmt.Printf("Warning: Failed to read config file: %v\n", err)
		return
	}

	// Parse the config
	var config Config
	if err := json.Unmarshal(data, &config); err != nil {
		fmt.Printf("Warning: Failed to parse config file: %v\n", err)
		return
	}

	// Set values if not explicitly provided
	if serverURL == "" {
		serverURL = config.ServerURL
	}
	if username == "" && token == "" {
		username = config.Username
		token = config.Token

		// Prefer JWT token if available
		if config.JWTToken != "" {
			token = config.JWTToken
		}
	}
}

// saveConfig saves the CLI configuration
func saveConfig(config Config) error {
	// If no config path is specified, use the default
	if configPath == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			return fmt.Errorf("failed to get user home directory: %w", err)
		}
		configDir := filepath.Join(home, ".flowrunner")
		if err := os.MkdirAll(configDir, 0755); err != nil {
			return fmt.Errorf("failed to create config directory: %w", err)
		}
		configPath = filepath.Join(configDir, "cli-config.json")
	}

	// Marshal the config
	data, err := json.MarshalIndent(config, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal config: %w", err)
	}

	// Write the config file
	if err := os.WriteFile(configPath, data, 0644); err != nil {
		return fmt.Errorf("failed to write config file: %w", err)
	}

	return nil
}

// createAccount creates a new account
func createAccount(cmd *cobra.Command, args []string) {
	if serverURL == "" {
		fmt.Println("Error: Server URL is required")
		os.Exit(1)
	}

	if username == "" {
		fmt.Print("Username: ")
		fmt.Scanln(&username)
	}

	if password == "" {
		fmt.Print("Password: ")
		fmt.Scanln(&password)
	}

	// Create request body
	reqBody, err := json.Marshal(map[string]string{
		"username": username,
		"password": password,
	})
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		os.Exit(1)
	}

	// Send request
	resp, err := http.Post(
		fmt.Sprintf("%s/api/v1/accounts", serverURL),
		"application/json",
		bytes.NewBuffer(reqBody),
	)
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		os.Exit(1)
	}
	defer resp.Body.Close()

	// Read response
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		os.Exit(1)
	}

	// Check response status
	if resp.StatusCode != http.StatusCreated {
		fmt.Printf("Error: %s\n", body)
		os.Exit(1)
	}

	fmt.Println("Account created successfully")

	// Save config
	config := Config{
		ServerURL: serverURL,
		Username:  username,
		Token:     token,
		JWTToken:  "", // JWT token is set by login command
	}
	if err := saveConfig(config); err != nil {
		fmt.Printf("Warning: Failed to save config: %v\n", err)
	}
}

// getAccountInfo gets information about the current account
func getAccountInfo(cmd *cobra.Command, args []string) {
	if serverURL == "" {
		fmt.Println("Error: Server URL is required")
		os.Exit(1)
	}

	// Create request
	req, err := http.NewRequest(http.MethodGet, fmt.Sprintf("%s/api/v1/accounts/me", serverURL), nil)
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		os.Exit(1)
	}

	// Add authentication
	if token != "" {
		req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", token))
	} else if username != "" && password != "" {
		req.SetBasicAuth(username, password)
	} else {
		fmt.Println("Error: Authentication required")
		os.Exit(1)
	}

	// Send request
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		os.Exit(1)
	}
	defer resp.Body.Close()

	// Read response
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		os.Exit(1)
	}

	// Check response status
	if resp.StatusCode != http.StatusOK {
		fmt.Printf("Error: %s\n", body)
		os.Exit(1)
	}

	// Pretty print JSON
	var prettyJSON bytes.Buffer
	if err := json.Indent(&prettyJSON, body, "", "  "); err != nil {
		fmt.Printf("Error: %v\n", err)
		os.Exit(1)
	}

	fmt.Println(prettyJSON.String())
}

// listFlows lists all flows
func listFlows(cmd *cobra.Command, args []string) {
	if serverURL == "" {
		fmt.Println("Error: Server URL is required")
		os.Exit(1)
	}

	// Create request
	req, err := http.NewRequest(http.MethodGet, fmt.Sprintf("%s/api/v1/flows", serverURL), nil)
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		os.Exit(1)
	}

	// Add authentication
	if token != "" {
		req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", token))
	} else if username != "" && password != "" {
		req.SetBasicAuth(username, password)
	} else {
		fmt.Println("Error: Authentication required")
		os.Exit(1)
	}

	// Send request
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		os.Exit(1)
	}
	defer resp.Body.Close()

	// Read response
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		os.Exit(1)
	}

	// Check response status
	if resp.StatusCode != http.StatusOK {
		fmt.Printf("Error: %s\n", body)
		os.Exit(1)
	}

	// Parse response
	var flows []map[string]interface{}
	if err := json.Unmarshal(body, &flows); err != nil {
		fmt.Printf("Error: %v\n", err)
		os.Exit(1)
	}

	// Print flows
	if len(flows) == 0 {
		fmt.Println("No flows found")
		return
	}

	fmt.Println("ID\t\tName\t\tVersion\t\tCreated")
	fmt.Println("--\t\t----\t\t-------\t\t-------")
	for _, flow := range flows {
		fmt.Printf("%s\t%s\t\t%s\t\t%s\n",
			flow["id"],
			flow["name"],
			flow["version"],
			flow["created_at"],
		)
	}
}

// createFlow creates a new flow
func createFlow(cmd *cobra.Command, args []string) {
	if serverURL == "" {
		fmt.Println("Error: Server URL is required")
		os.Exit(1)
	}

	name := args[0]
	filePath := args[1]

	// Read file
	content, err := os.ReadFile(filePath)
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		os.Exit(1)
	}

	// Create request body
	reqBody, err := json.Marshal(map[string]string{
		"name":    name,
		"content": string(content),
	})
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		os.Exit(1)
	}

	// Create request
	req, err := http.NewRequest(
		http.MethodPost,
		fmt.Sprintf("%s/api/v1/flows", serverURL),
		bytes.NewBuffer(reqBody),
	)
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		os.Exit(1)
	}
	req.Header.Set("Content-Type", "application/json")

	// Add authentication
	if token != "" {
		req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", token))
	} else if username != "" && password != "" {
		req.SetBasicAuth(username, password)
	} else {
		fmt.Println("Error: Authentication required")
		os.Exit(1)
	}

	// Send request
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		os.Exit(1)
	}
	defer resp.Body.Close()

	// Read response
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		os.Exit(1)
	}

	// Check response status
	if resp.StatusCode != http.StatusCreated {
		fmt.Printf("Error: %s\n", body)
		os.Exit(1)
	}

	// Parse response
	var result map[string]string
	if err := json.Unmarshal(body, &result); err != nil {
		fmt.Printf("Error: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("Flow created with ID: %s\n", result["id"])
}

// getFlow gets a flow by ID
func getFlow(cmd *cobra.Command, args []string) {
	if serverURL == "" {
		fmt.Println("Error: Server URL is required")
		os.Exit(1)
	}

	flowID := args[0]

	// Create request
	req, err := http.NewRequest(
		http.MethodGet,
		fmt.Sprintf("%s/api/v1/flows/%s", serverURL, flowID),
		nil,
	)
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		os.Exit(1)
	}

	// Add authentication
	if token != "" {
		req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", token))
	} else if username != "" && password != "" {
		req.SetBasicAuth(username, password)
	} else {
		fmt.Println("Error: Authentication required")
		os.Exit(1)
	}

	// Send request
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		os.Exit(1)
	}
	defer resp.Body.Close()

	// Read response
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		os.Exit(1)
	}

	// Check response status
	if resp.StatusCode != http.StatusOK {
		fmt.Printf("Error: %s\n", body)
		os.Exit(1)
	}

	// Print YAML content
	fmt.Println(string(body))
}

// updateFlow updates a flow
func updateFlow(cmd *cobra.Command, args []string) {
	if serverURL == "" {
		fmt.Println("Error: Server URL is required")
		os.Exit(1)
	}

	flowID := args[0]
	filePath := args[1]

	// Read file
	content, err := os.ReadFile(filePath)
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		os.Exit(1)
	}

	// Create request body
	reqBody, err := json.Marshal(map[string]string{
		"content": string(content),
	})
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		os.Exit(1)
	}

	// Create request
	req, err := http.NewRequest(
		http.MethodPut,
		fmt.Sprintf("%s/api/v1/flows/%s", serverURL, flowID),
		bytes.NewBuffer(reqBody),
	)
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		os.Exit(1)
	}
	req.Header.Set("Content-Type", "application/json")

	// Add authentication
	if token != "" {
		req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", token))
	} else if username != "" && password != "" {
		req.SetBasicAuth(username, password)
	} else {
		fmt.Println("Error: Authentication required")
		os.Exit(1)
	}

	// Send request
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		os.Exit(1)
	}
	defer resp.Body.Close()

	// Check response status
	if resp.StatusCode != http.StatusNoContent {
		body, _ := io.ReadAll(resp.Body)
		fmt.Printf("Error: %s\n", body)
		os.Exit(1)
	}

	fmt.Println("Flow updated successfully")
}

// deleteFlow deletes a flow
func deleteFlow(cmd *cobra.Command, args []string) {
	if serverURL == "" {
		fmt.Println("Error: Server URL is required")
		os.Exit(1)
	}

	flowID := args[0]

	// Confirm deletion
	fmt.Printf("Are you sure you want to delete flow %s? (y/N): ", flowID)
	var confirm string
	fmt.Scanln(&confirm)
	if strings.ToLower(confirm) != "y" {
		fmt.Println("Deletion cancelled")
		return
	}

	// Create request
	req, err := http.NewRequest(
		http.MethodDelete,
		fmt.Sprintf("%s/api/v1/flows/%s", serverURL, flowID),
		nil,
	)
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		os.Exit(1)
	}

	// Add authentication
	if token != "" {
		req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", token))
	} else if username != "" && password != "" {
		req.SetBasicAuth(username, password)
	} else {
		fmt.Println("Error: Authentication required")
		os.Exit(1)
	}

	// Send request
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		os.Exit(1)
	}
	defer resp.Body.Close()

	// Check response status
	if resp.StatusCode != http.StatusNoContent {
		body, _ := io.ReadAll(resp.Body)
		fmt.Printf("Error: %s\n", body)
		os.Exit(1)
	}

	fmt.Println("Flow deleted successfully")
}

// Secret management functions

// listSecrets lists all secret keys for the authenticated account
func listSecrets(cmd *cobra.Command, args []string) {
	if serverURL == "" {
		fmt.Println("Error: Server URL is required")
		os.Exit(1)
	}

	// Get account info to get account ID
	accountID, err := getAccountID()
	if err != nil {
		fmt.Printf("Error getting account ID: %v\n", err)
		os.Exit(1)
	}

	// Create request
	req, err := http.NewRequest(
		http.MethodGet,
		fmt.Sprintf("%s/api/v1/accounts/%s/secrets/keys", serverURL, accountID),
		nil,
	)
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		os.Exit(1)
	}

	// Add authentication
	if token != "" {
		req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", token))
	} else if username != "" && password != "" {
		req.SetBasicAuth(username, password)
	} else {
		fmt.Println("Error: Authentication required")
		os.Exit(1)
	}

	// Send request
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		os.Exit(1)
	}
	defer resp.Body.Close()

	// Read response
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		os.Exit(1)
	}

	// Check response status
	if resp.StatusCode != http.StatusOK {
		fmt.Printf("Error: %s\n", body)
		os.Exit(1)
	}

	// Parse and display response
	var response map[string]interface{}
	if err := json.Unmarshal(body, &response); err != nil {
		fmt.Printf("Error parsing response: %v\n", err)
		os.Exit(1)
	}

	keys := response["keys"].([]interface{})
	total := response["total"].(float64)

	fmt.Printf("Found %d secret(s):\n", int(total))
	for _, key := range keys {
		fmt.Printf("  %s\n", key)
	}
}

// getSecret retrieves and displays a secret value
func getSecret(cmd *cobra.Command, args []string) {
	if serverURL == "" {
		fmt.Println("Error: Server URL is required")
		os.Exit(1)
	}

	key := args[0]

	// Get account info to get account ID
	accountID, err := getAccountID()
	if err != nil {
		fmt.Printf("Error getting account ID: %v\n", err)
		os.Exit(1)
	}

	// Create request
	req, err := http.NewRequest(
		http.MethodGet,
		fmt.Sprintf("%s/api/v1/accounts/%s/secrets/%s", serverURL, accountID, key),
		nil,
	)
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		os.Exit(1)
	}

	// Add authentication
	if token != "" {
		req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", token))
	} else if username != "" && password != "" {
		req.SetBasicAuth(username, password)
	} else {
		fmt.Println("Error: Authentication required")
		os.Exit(1)
	}

	// Send request
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		os.Exit(1)
	}
	defer resp.Body.Close()

	// Read response
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		os.Exit(1)
	}

	// Check response status
	if resp.StatusCode != http.StatusOK {
		fmt.Printf("Error: %s\n", body)
		os.Exit(1)
	}

	// Parse and display response
	var response map[string]interface{}
	if err := json.Unmarshal(body, &response); err != nil {
		fmt.Printf("Error parsing response: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("%s\n", response["value"])
}

// setSecret creates or updates a secret
func setSecret(cmd *cobra.Command, args []string) {
	if serverURL == "" {
		fmt.Println("Error: Server URL is required")
		os.Exit(1)
	}

	key := args[0]
	value := args[1]

	// Get account info to get account ID
	accountID, err := getAccountID()
	if err != nil {
		fmt.Printf("Error getting account ID: %v\n", err)
		os.Exit(1)
	}

	// Create request body
	requestBody := map[string]string{
		"value": value,
	}
	jsonBody, err := json.Marshal(requestBody)
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		os.Exit(1)
	}

	// Create request
	req, err := http.NewRequest(
		http.MethodPost,
		fmt.Sprintf("%s/api/v1/accounts/%s/secrets/%s", serverURL, accountID, key),
		bytes.NewBuffer(jsonBody),
	)
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		os.Exit(1)
	}

	req.Header.Set("Content-Type", "application/json")

	// Add authentication
	if token != "" {
		req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", token))
	} else if username != "" && password != "" {
		req.SetBasicAuth(username, password)
	} else {
		fmt.Println("Error: Authentication required")
		os.Exit(1)
	}

	// Send request
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		os.Exit(1)
	}
	defer resp.Body.Close()

	// Read response
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		os.Exit(1)
	}

	// Check response status
	if resp.StatusCode != http.StatusCreated {
		fmt.Printf("Error: %s\n", body)
		os.Exit(1)
	}

	fmt.Printf("Secret '%s' set successfully\n", key)
}

// deleteSecret removes a secret
func deleteSecret(cmd *cobra.Command, args []string) {
	if serverURL == "" {
		fmt.Println("Error: Server URL is required")
		os.Exit(1)
	}

	key := args[0]

	// Confirm deletion
	fmt.Printf("Are you sure you want to delete secret '%s'? (y/N): ", key)
	var confirm string
	fmt.Scanln(&confirm)
	if strings.ToLower(confirm) != "y" {
		fmt.Println("Deletion cancelled")
		return
	}

	// Get account info to get account ID
	accountID, err := getAccountID()
	if err != nil {
		fmt.Printf("Error getting account ID: %v\n", err)
		os.Exit(1)
	}

	// Create request
	req, err := http.NewRequest(
		http.MethodDelete,
		fmt.Sprintf("%s/api/v1/accounts/%s/secrets/%s", serverURL, accountID, key),
		nil,
	)
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		os.Exit(1)
	}

	// Add authentication
	if token != "" {
		req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", token))
	} else if username != "" && password != "" {
		req.SetBasicAuth(username, password)
	} else {
		fmt.Println("Error: Authentication required")
		os.Exit(1)
	}

	// Send request
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		os.Exit(1)
	}
	defer resp.Body.Close()

	// Check response status
	if resp.StatusCode != http.StatusNoContent {
		body, _ := io.ReadAll(resp.Body)
		fmt.Printf("Error: %s\n", body)
		os.Exit(1)
	}

	fmt.Printf("Secret '%s' deleted successfully\n", key)
}

// getAccountID retrieves the current account ID from the server
func getAccountID() (string, error) {
	// Create request to get account info
	req, err := http.NewRequest(
		http.MethodGet,
		fmt.Sprintf("%s/api/v1/accounts/me", serverURL),
		nil,
	)
	if err != nil {
		return "", err
	}

	// Add authentication
	if token != "" {
		req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", token))
	} else if username != "" && password != "" {
		req.SetBasicAuth(username, password)
	} else {
		return "", fmt.Errorf("authentication required")
	}

	// Send request
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("failed to get account info: status %d", resp.StatusCode)
	}

	// Parse response
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	var account map[string]interface{}
	if err := json.Unmarshal(body, &account); err != nil {
		return "", err
	}

	accountID, ok := account["id"].(string)
	if !ok {
		return "", fmt.Errorf("invalid account response format")
	}

	return accountID, nil
}
