package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"

	"github.com/spf13/cobra"
)

// LoginRequest represents a login request
type LoginRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

// LoginResponse represents a login response
type LoginResponse struct {
	Token     string `json:"token"`
	AccountID string `json:"account_id"`
	Username  string `json:"username"`
}

// loginCmd represents the login command
var loginCmd = &cobra.Command{
	Use:   "login",
	Short: "Login to the server",
	Run:   login,
}

// login logs in to the server
func login(cmd *cobra.Command, args []string) {
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
	reqBody, err := json.Marshal(LoginRequest{
		Username: username,
		Password: password,
	})
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		os.Exit(1)
	}

	// Send request
	resp, err := http.Post(
		fmt.Sprintf("%s/api/v1/login", serverURL),
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
	if resp.StatusCode != http.StatusOK {
		fmt.Printf("Error: %s\n", body)
		os.Exit(1)
	}

	// Parse response
	var loginResp LoginResponse
	if err := json.Unmarshal(body, &loginResp); err != nil {
		fmt.Printf("Error: %v\n", err)
		os.Exit(1)
	}

	// Save token to config
	token = loginResp.Token
	config := Config{
		ServerURL: serverURL,
		Username:  username,
		JWTToken:  token,
	}

	// Save config
	if err := saveConfig(config); err != nil {
		fmt.Printf("Warning: Failed to save config: %v\n", err)
	}

	fmt.Println("Login successful")
}
