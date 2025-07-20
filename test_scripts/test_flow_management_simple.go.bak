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
)

// Test server configuration
const (
	serverPort = 8097
	serverHost = "localhost"
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

type CreateFlowRequest struct {
	Name    string `json:"name"`
	Content string `json:"content"`
}

type CreateFlowResponse struct {
	ID string `json:"id"`
}

type FlowInfo struct {
	ID          string    `json:"id"`
	Name        string    `json:"name"`
	Description string    `json:"description"`
	Version     string    `json:"version"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

// Simple HTTP server for testing
type TestServer struct {
	server *http.Server
	flows  map[string]map[string]string // accountID -> flowID -> content
	mux    *http.ServeMux
}

// NewTestServer creates a new test server
func NewTestServer() *TestServer {
	mux := http.NewServeMux()
	server := &http.Server{
		Addr:    fmt.Sprintf("%s:%d", serverHost, serverPort),
		Handler: mux,
	}

	ts := &TestServer{
		server: server,
		flows:  make(map[string]map[string]string),
		mux:    mux,
	}

	// Set up routes
	ts.setupRoutes()

	return ts
}

// setupRoutes configures the API routes
func (s *TestServer) setupRoutes() {
	// Public routes
	s.mux.HandleFunc("/api/v1/accounts", s.handleCreateAccount)
	s.mux.HandleFunc("/api/v1/login", s.handleLogin)

	// Protected routes
	s.mux.HandleFunc("/api/v1/flows", s.handleFlows)
	s.mux.HandleFunc("/api/v1/flows/", s.handleFlowByID)
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
func (s *TestServer) handleCreateAccount(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req CreateAccountRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// Simple account creation
	accountID := "test-account-id"

	// Create flows map for this account
	s.flows[accountID] = make(map[string]string)

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"id":         accountID,
		"username":   req.Username,
		"created_at": time.Now(),
		"updated_at": time.Now(),
	})
}

// handleLogin handles user login
func (s *TestServer) handleLogin(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req LoginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// Simple token generation
	accountID := "test-account-id"
	token := "test-jwt-token"

	// Return the token
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(LoginResponse{
		Token:     token,
		AccountID: accountID,
		Username:  req.Username,
	})
}

// handleFlows handles flow listing and creation
func (s *TestServer) handleFlows(w http.ResponseWriter, r *http.Request) {
	// Simple authentication check
	authHeader := r.Header.Get("Authorization")
	if authHeader != "Bearer test-jwt-token" {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	accountID := "test-account-id"

	switch r.Method {
	case http.MethodGet:
		// List flows
		flows := []FlowInfo{}
		for flowID, _ := range s.flows[accountID] {
			flows = append(flows, FlowInfo{
				ID:          flowID,
				Name:        "Test Flow",
				Description: "A test flow",
				Version:     "1.0.0",
				CreatedAt:   time.Now(),
				UpdatedAt:   time.Now(),
			})
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(flows)

	case http.MethodPost:
		// Create flow
		var req CreateFlowRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "Invalid request body", http.StatusBadRequest)
			return
		}

		// Generate flow ID
		flowID := fmt.Sprintf("flow-%d", time.Now().UnixNano())

		// Store flow
		s.flows[accountID][flowID] = req.Content

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(CreateFlowResponse{ID: flowID})

	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

// handleFlowByID handles flow retrieval, update, and deletion
func (s *TestServer) handleFlowByID(w http.ResponseWriter, r *http.Request) {
	// Simple authentication check
	authHeader := r.Header.Get("Authorization")
	if authHeader != "Bearer test-jwt-token" {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	accountID := "test-account-id"

	// Extract flow ID from URL
	path := r.URL.Path
	if len(path) <= len("/api/v1/flows/") {
		http.Error(w, "Invalid flow ID", http.StatusBadRequest)
		return
	}

	flowID := path[len("/api/v1/flows/"):]

	switch r.Method {
	case http.MethodGet:
		// Get flow
		content, ok := s.flows[accountID][flowID]
		if !ok {
			http.Error(w, "Flow not found", http.StatusNotFound)
			return
		}

		w.Header().Set("Content-Type", "application/yaml")
		w.Write([]byte(content))

	case http.MethodPut:
		// Update flow
		var req struct {
			Content string `json:"content"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "Invalid request body", http.StatusBadRequest)
			return
		}

		// Check if flow exists
		if _, ok := s.flows[accountID][flowID]; !ok {
			http.Error(w, "Flow not found", http.StatusNotFound)
			return
		}

		// Update flow
		s.flows[accountID][flowID] = req.Content

		w.WriteHeader(http.StatusNoContent)

	case http.MethodDelete:
		// Delete flow
		if _, ok := s.flows[accountID][flowID]; !ok {
			http.Error(w, "Flow not found", http.StatusNotFound)
			return
		}

		// Delete flow
		delete(s.flows[accountID], flowID)

		w.WriteHeader(http.StatusNoContent)

	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

// Helper function to make authenticated requests
func makeAuthenticatedRequest(method, url, token string, body interface{}) (*http.Response, error) {
	var bodyReader io.Reader
	if body != nil {
		bodyBytes, err := json.Marshal(body)
		if err != nil {
			return nil, err
		}
		bodyReader = bytes.NewBuffer(bodyBytes)
	}

	req, err := http.NewRequest(method, url, bodyReader)
	if err != nil {
		return nil, err
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", token))

	client := &http.Client{}
	return client.Do(req)
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
		Username: "flowuser",
		Password: "flowpassword",
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

	fmt.Println("Account created successfully")

	// Step 2: Login and get JWT token
	fmt.Println("\nStep 2: Logging in to get JWT token...")
	loginReq := LoginRequest{
		Username: "flowuser",
		Password: "flowpassword",
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
	token := loginRespBody.Token

	// Step 3: Create a flow
	fmt.Println("\nStep 3: Creating a flow...")
	flowContent := `
metadata:
  name: Test Flow
  description: A test flow for API testing
  version: 1.0.0
nodes:
  start:
    type: test
    params:
      foo: bar
`

	createFlowReq := CreateFlowRequest{
		Name:    "Test Flow",
		Content: flowContent,
	}

	createFlowResp, err := makeAuthenticatedRequest(
		http.MethodPost,
		fmt.Sprintf("%s/api/v1/flows", baseURL),
		token,
		createFlowReq,
	)
	if err != nil {
		log.Fatalf("Failed to create flow: %v", err)
	}
	defer createFlowResp.Body.Close()

	if createFlowResp.StatusCode != http.StatusCreated {
		body, _ := io.ReadAll(createFlowResp.Body)
		log.Fatalf("Failed to create flow: %s", body)
	}

	var createFlowRespBody CreateFlowResponse
	if err := json.NewDecoder(createFlowResp.Body).Decode(&createFlowRespBody); err != nil {
		log.Fatalf("Failed to parse create flow response: %v", err)
	}

	flowID := createFlowRespBody.ID
	fmt.Printf("Flow created with ID: %s\n", flowID)

	// Step 4: List flows
	fmt.Println("\nStep 4: Listing flows...")
	listFlowsResp, err := makeAuthenticatedRequest(
		http.MethodGet,
		fmt.Sprintf("%s/api/v1/flows", baseURL),
		token,
		nil,
	)
	if err != nil {
		log.Fatalf("Failed to list flows: %v", err)
	}
	defer listFlowsResp.Body.Close()

	if listFlowsResp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(listFlowsResp.Body)
		log.Fatalf("Failed to list flows: %s", body)
	}

	var flows []FlowInfo
	if err := json.NewDecoder(listFlowsResp.Body).Decode(&flows); err != nil {
		log.Fatalf("Failed to parse list flows response: %v", err)
	}

	fmt.Printf("Found %d flows\n", len(flows))

	// Step 5: Get flow by ID
	fmt.Println("\nStep 5: Getting flow by ID...")
	getFlowResp, err := makeAuthenticatedRequest(
		http.MethodGet,
		fmt.Sprintf("%s/api/v1/flows/%s", baseURL, flowID),
		token,
		nil,
	)
	if err != nil {
		log.Fatalf("Failed to get flow: %v", err)
	}
	defer getFlowResp.Body.Close()

	if getFlowResp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(getFlowResp.Body)
		log.Fatalf("Failed to get flow: %s", body)
	}

	flowContentBytes, err := io.ReadAll(getFlowResp.Body)
	if err != nil {
		log.Fatalf("Failed to read flow content: %v", err)
	}

	fmt.Printf("Retrieved flow content successfully (%d bytes)\n", len(flowContentBytes))

	// Step 6: Update flow content
	fmt.Println("\nStep 6: Updating flow content...")
	updatedFlowContent := `
metadata:
  name: Updated Test Flow
  description: An updated test flow for API testing
  version: 1.1.0
nodes:
  start:
    type: test
    params:
      foo: updated
      bar: baz
`

	updateFlowReq := struct {
		Content string `json:"content"`
	}{
		Content: updatedFlowContent,
	}

	updateFlowResp, err := makeAuthenticatedRequest(
		http.MethodPut,
		fmt.Sprintf("%s/api/v1/flows/%s", baseURL, flowID),
		token,
		updateFlowReq,
	)
	if err != nil {
		log.Fatalf("Failed to update flow: %v", err)
	}
	defer updateFlowResp.Body.Close()

	if updateFlowResp.StatusCode != http.StatusNoContent {
		body, _ := io.ReadAll(updateFlowResp.Body)
		log.Fatalf("Failed to update flow: %s", body)
	}

	fmt.Println("Flow content updated successfully")

	// Step 7: Get updated flow
	fmt.Println("\nStep 7: Getting updated flow...")
	getUpdatedFlowResp, err := makeAuthenticatedRequest(
		http.MethodGet,
		fmt.Sprintf("%s/api/v1/flows/%s", baseURL, flowID),
		token,
		nil,
	)
	if err != nil {
		log.Fatalf("Failed to get updated flow: %v", err)
	}
	defer getUpdatedFlowResp.Body.Close()

	if getUpdatedFlowResp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(getUpdatedFlowResp.Body)
		log.Fatalf("Failed to get updated flow: %s", body)
	}

	updatedContentBytes, err := io.ReadAll(getUpdatedFlowResp.Body)
	if err != nil {
		log.Fatalf("Failed to read updated flow content: %v", err)
	}

	fmt.Printf("Retrieved updated flow content successfully (%d bytes)\n", len(updatedContentBytes))

	// Step 8: Delete flow
	fmt.Println("\nStep 8: Deleting flow...")
	deleteFlowResp, err := makeAuthenticatedRequest(
		http.MethodDelete,
		fmt.Sprintf("%s/api/v1/flows/%s", baseURL, flowID),
		token,
		nil,
	)
	if err != nil {
		log.Fatalf("Failed to delete flow: %v", err)
	}
	defer deleteFlowResp.Body.Close()

	if deleteFlowResp.StatusCode != http.StatusNoContent {
		body, _ := io.ReadAll(deleteFlowResp.Body)
		log.Fatalf("Failed to delete flow: %s", body)
	}

	fmt.Println("Flow deleted successfully")

	// Step 9: Verify flow is deleted
	fmt.Println("\nStep 9: Verifying flow is deleted...")
	verifyDeletedResp, err := makeAuthenticatedRequest(
		http.MethodGet,
		fmt.Sprintf("%s/api/v1/flows/%s", baseURL, flowID),
		token,
		nil,
	)
	if err != nil {
		log.Fatalf("Failed to verify deletion: %v", err)
	}
	defer verifyDeletedResp.Body.Close()

	if verifyDeletedResp.StatusCode != http.StatusNotFound {
		body, _ := io.ReadAll(verifyDeletedResp.Body)
		log.Fatalf("Flow was not deleted properly: %s", body)
	}

	fmt.Println("Flow deletion verified")

	fmt.Println("\nAll flow management tests passed successfully!")
	os.Exit(0)
}
