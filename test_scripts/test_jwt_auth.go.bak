package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/tcmartin/flowrunner/pkg/auth"
	"github.com/tcmartin/flowrunner/pkg/middleware"
	"github.com/tcmartin/flowrunner/pkg/services"
	"github.com/tcmartin/flowrunner/pkg/storage"
)

// Test server configuration
const (
	serverPort = 8095
	serverHost = "localhost"
	jwtSecret  = "test-jwt-secret-key-for-authentication-testing"
)

// Request/response structures
type LoginRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

type LoginResponse struct {
	Token     string `json:"token"`
	AccountID string `json:"account_id"`
	Username  string `json:"username"`
}

type CreateAccountRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

type CreateAccountResponse struct {
	ID        string    `json:"id"`
	Username  string    `json:"username"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// Test server
type TestServer struct {
	server         *http.Server
	accountService *services.AccountService
	jwtService     *services.JWTService
}

// NewTestServer creates a new test server
func NewTestServer() *TestServer {
	// Create a memory account store
	accountStore := storage.NewMemoryAccountStore()

	// Create an account service with JWT support
	accountService := services.NewAccountService(accountStore)
	accountService = accountService.WithJWTService(jwtSecret, 24)

	// Create a JWT service
	jwtService := services.NewJWTService(jwtSecret, 24)

	// Create a router
	router := http.NewServeMux()

	// Create authentication middleware
	authMiddleware := middleware.NewAuthMiddleware(accountService)

	// Add routes
	router.HandleFunc("/api/v1/accounts", handleCreateAccount(accountService))
	router.HandleFunc("/api/v1/login", handleLogin(accountService))
	router.HandleFunc("/api/v1/protected", authMiddlewareFunc(authMiddleware, handleProtected()))

	// Create server
	server := &http.Server{
		Addr:    fmt.Sprintf("%s:%d", serverHost, serverPort),
		Handler: router,
	}

	return &TestServer{
		server:         server,
		accountService: accountService,
		jwtService:     jwtService,
	}
}

// Start starts the test server
func (s *TestServer) Start() {
	go func() {
		log.Printf("Starting test server on %s:%d", serverHost, serverPort)
		if err := s.server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Server error: %v", err)
		}
	}()

	// Wait for server to start
	time.Sleep(100 * time.Millisecond)
}

// Stop stops the test server
func (s *TestServer) Stop() {
	s.server.Close()
}

// handleCreateAccount handles account creation
func handleCreateAccount(accountService auth.AccountService) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}

		var req CreateAccountRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "Invalid request body", http.StatusBadRequest)
			return
		}

		accountID, err := accountService.CreateAccount(req.Username, req.Password)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		account, err := accountService.GetAccount(accountID)
		if err != nil {
			http.Error(w, "Failed to retrieve account", http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(account)
	}
}

// handleLogin handles user login
func handleLogin(accountService auth.AccountService) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}

		var req LoginRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "Invalid request body", http.StatusBadRequest)
			return
		}

		// Authenticate the user
		accountID, err := accountService.Authenticate(req.Username, req.Password)
		if err != nil {
			http.Error(w, "Authentication failed", http.StatusUnauthorized)
			return
		}

		// Get the account
		account, err := accountService.GetAccount(accountID)
		if err != nil {
			http.Error(w, "Failed to retrieve account", http.StatusInternalServerError)
			return
		}

		// Generate a JWT token
		svcWithJWT, ok := accountService.(*services.AccountService)
		if !ok {
			http.Error(w, "JWT authentication not supported", http.StatusInternalServerError)
			return
		}

		token, err := svcWithJWT.GenerateJWT(accountID)
		if err != nil {
			http.Error(w, "Failed to generate token", http.StatusInternalServerError)
			return
		}

		// Return the token
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(LoginResponse{
			Token:     token,
			AccountID: accountID,
			Username:  account.Username,
		})
	}
}

// handleProtected handles a protected endpoint
func handleProtected() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Get account ID from context
		accountID, ok := r.Context().Value(middleware.AccountIDKey).(string)
		if !ok {
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}

		// Return success
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{
			"message":    "Protected resource accessed successfully",
			"account_id": accountID,
		})
	}
}

// authMiddlewareFunc wraps the authentication middleware for use with http.HandlerFunc
func authMiddlewareFunc(authMiddleware *middleware.AuthMiddleware, next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		handler := authMiddleware.Authenticate(next)
		handler.ServeHTTP(w, r)
	}
}

// parseJWT parses a JWT token and returns the claims
func parseJWT(tokenString string) (jwt.MapClaims, error) {
	token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
		// Validate the signing method
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return []byte(jwtSecret), nil
	})

	if err != nil {
		return nil, err
	}

	if claims, ok := token.Claims.(jwt.MapClaims); ok && token.Valid {
		return claims, nil
	}

	return nil, fmt.Errorf("invalid token")
}

func main() {
	// Start the test server
	server := NewTestServer()
	server.Start()
	defer server.Stop()

	// Base URL for API requests
	baseURL := fmt.Sprintf("http://%s:%d", serverHost, serverPort)

	// Step 1: Create a test account
	fmt.Println("Step 1: Creating test account...")
	createReq := CreateAccountRequest{
		Username: "testuser",
		Password: "testpassword",
	}

	createReqBody, _ := json.Marshal(createReq)
	createResp, err := http.Post(
		fmt.Sprintf("%s/api/v1/accounts", baseURL),
		"application/json",
		bytes.NewBuffer(createReqBody),
	)
	if err != nil {
		log.Fatalf("Failed to create account: %v", err)
	}
	defer createResp.Body.Close()

	if createResp.StatusCode != http.StatusCreated {
		body, _ := io.ReadAll(createResp.Body)
		log.Fatalf("Failed to create account: %s", body)
	}

	var createRespBody CreateAccountResponse
	if err := json.NewDecoder(createResp.Body).Decode(&createRespBody); err != nil {
		log.Fatalf("Failed to parse create account response: %v", err)
	}

	fmt.Printf("Account created: %s (%s)\n", createRespBody.Username, createRespBody.ID)

	// Step 2: Login and get JWT token
	fmt.Println("\nStep 2: Logging in to get JWT token...")
	loginReq := LoginRequest{
		Username: "testuser",
		Password: "testpassword",
	}

	loginReqBody, _ := json.Marshal(loginReq)
	loginResp, err := http.Post(
		fmt.Sprintf("%s/api/v1/login", baseURL),
		"application/json",
		bytes.NewBuffer(loginReqBody),
	)
	if err != nil {
		log.Fatalf("Failed to login: %v", err)
	}
	defer loginResp.Body.Close()

	if loginResp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(loginResp.Body)
		log.Fatalf("Failed to login: %s", body)
	}

	var loginRespBody LoginResponse
	if err := json.NewDecoder(loginResp.Body).Decode(&loginRespBody); err != nil {
		log.Fatalf("Failed to parse login response: %v", err)
	}

	fmt.Printf("Login successful. JWT token received: %s\n", loginRespBody.Token)

	// Step 3: Parse and verify the JWT token
	fmt.Println("\nStep 3: Parsing and verifying JWT token...")
	claims, err := parseJWT(loginRespBody.Token)
	if err != nil {
		log.Fatalf("Failed to parse JWT token: %v", err)
	}

	fmt.Println("JWT token verified successfully!")
	fmt.Printf("Token claims: %+v\n", claims)

	// Step 4: Access protected resource with JWT token
	fmt.Println("\nStep 4: Accessing protected resource with JWT token...")
	protectedReq, _ := http.NewRequest("GET", fmt.Sprintf("%s/api/v1/protected", baseURL), nil)
	protectedReq.Header.Set("Authorization", fmt.Sprintf("Bearer %s", loginRespBody.Token))

	client := &http.Client{}
	protectedResp, err := client.Do(protectedReq)
	if err != nil {
		log.Fatalf("Failed to access protected resource: %v", err)
	}
	defer protectedResp.Body.Close()

	if protectedResp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(protectedResp.Body)
		log.Fatalf("Failed to access protected resource: %s", body)
	}

	var protectedRespBody map[string]string
	if err := json.NewDecoder(protectedResp.Body).Decode(&protectedRespBody); err != nil {
		log.Fatalf("Failed to parse protected resource response: %v", err)
	}

	fmt.Printf("Protected resource accessed successfully: %s\n", protectedRespBody["message"])
	fmt.Printf("Account ID from protected resource: %s\n", protectedRespBody["account_id"])

	// Step 5: Try to access protected resource with invalid token
	fmt.Println("\nStep 5: Trying to access protected resource with invalid token...")
	invalidReq, _ := http.NewRequest("GET", fmt.Sprintf("%s/api/v1/protected", baseURL), nil)
	invalidReq.Header.Set("Authorization", "Bearer invalid-token")

	invalidResp, err := client.Do(invalidReq)
	if err != nil {
		log.Fatalf("Failed to make request with invalid token: %v", err)
	}
	defer invalidResp.Body.Close()

	if invalidResp.StatusCode == http.StatusUnauthorized {
		fmt.Println("Access denied with invalid token, as expected!")
	} else {
		body, _ := io.ReadAll(invalidResp.Body)
		log.Fatalf("Unexpected response with invalid token: %d %s", invalidResp.StatusCode, body)
	}

	fmt.Println("\nAll tests passed successfully!")
	os.Exit(0)
}
