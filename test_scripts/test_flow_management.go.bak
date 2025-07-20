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

	"github.com/tcmartin/flowrunner/pkg/auth"
	"github.com/tcmartin/flowrunner/pkg/middleware"
	"github.com/tcmartin/flowrunner/pkg/registry"
	"github.com/tcmartin/flowrunner/pkg/services"
	"github.com/tcmartin/flowrunner/pkg/storage"
)

// Test server configuration
const (
	serverPort = 8096
	serverHost = "localhost"
	jwtSecret  = "test-jwt-secret-key-for-flow-management-testing"
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

type UpdateFlowMetadataRequest struct {
	Tags     []string               `json:"tags,omitempty"`
	Category string                 `json:"category,omitempty"`
	Status   string                 `json:"status,omitempty"`
	Custom   map[string]interface{} `json:"custom,omitempty"`
}

type SearchFlowsRequest struct {
	NameContains        string   `json:"name_contains,omitempty"`
	DescriptionContains string   `json:"description_contains,omitempty"`
	Tags                []string `json:"tags,omitempty"`
	Category            string   `json:"category,omitempty"`
	Status              string   `json:"status,omitempty"`
	CreatedAfter        *int64   `json:"created_after,omitempty"`
	CreatedBefore       *int64   `json:"created_before,omitempty"`
	UpdatedAfter        *int64   `json:"updated_after,omitempty"`
	UpdatedBefore       *int64   `json:"updated_before,omitempty"`
	Page                int      `json:"page,omitempty"`
	PageSize            int      `json:"page_size,omitempty"`
}

// Test server
type TestServer struct {
	server         *http.Server
	accountService *services.AccountService
	flowRegistry   registry.FlowRegistry
	mux            *http.ServeMux
}

// NewTestServer creates a new test server
func NewTestServer() *TestServer {
	// Create memory stores
	accountStore := storage.NewMemoryAccountStore()
	flowStore := storage.NewMemoryFlowStore()

	// Create services
	accountService := services.NewAccountService(accountStore)
	accountService = accountService.WithJWTService(jwtSecret, 24)
	flowRegistry := registry.NewFlowRegistry(flowStore, registry.FlowRegistryOptions{})

	// Create router
	mux := http.NewServeMux()

	// Create server
	server := &http.Server{
		Addr:    fmt.Sprintf("%s:%d", serverHost, serverPort),
		Handler: mux,
	}

	ts := &TestServer{
		server:         server,
		accountService: accountService,
		flowRegistry:   flowRegistry,
		mux:            mux,
	}

	// Set up routes
	ts.setupRoutes()

	return ts
}

// setupRoutes configures the API routes
func (s *TestServer) setupRoutes() {
	// Create authentication middleware
	authMiddleware := middleware.NewAuthMiddleware(s.accountService)

	// Public routes
	s.mux.HandleFunc("/api/v1/accounts", s.handleCreateAccount)
	s.mux.HandleFunc("/api/v1/login", s.handleLogin)

	// Protected routes
	s.mux.HandleFunc("/api/v1/flows", func(w http.ResponseWriter, r *http.Request) {
		handler := authMiddleware.Authenticate(http.HandlerFunc(s.handleFlows))
		handler.ServeHTTP(w, r)
	})

	s.mux.HandleFunc("/api/v1/flows/", func(w http.ResponseWriter, r *http.Request) {
		handler := authMiddleware.Authenticate(http.HandlerFunc(s.handleFlowByID))
		handler.ServeHTTP(w, r)
	})

	s.mux.HandleFunc("/api/v1/flows/search", func(w http.ResponseWriter, r *http.Request) {
		handler := authMiddleware.Authenticate(http.HandlerFunc(s.handleSearchFlows))
		handler.ServeHTTP(w, r)
	})
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

	accountID, err := s.accountService.CreateAccount(req.Username, req.Password)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	account, err := s.accountService.GetAccount(accountID)
	if err != nil {
		http.Error(w, "Failed to retrieve account", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(account)
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

	// Authenticate the user
	accountID, err := s.accountService.Authenticate(req.Username, req.Password)
	if err != nil {
		http.Error(w, "Authentication failed", http.StatusUnauthorized)
		return
	}

	// Get the account
	account, err := s.accountService.GetAccount(accountID)
	if err != nil {
		http.Error(w, "Failed to retrieve account", http.StatusInternalServerError)
		return
	}

	// Generate a JWT token
	token, err := s.accountService.GenerateJWT(accountID)
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

// handleFlows handles flow listing and creation
func (s *TestServer) handleFlows(w http.ResponseWriter, r *http.Request) {
	// Get account ID from context
	accountID, ok := r.Context().Value(middleware.AccountIDKey).(string)
	if !ok {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	switch r.Method {
	case http.MethodGet:
		// List flows
		flows, err := s.flowRegistry.List(accountID)
		if err != nil {
			http.Error(w, "Failed to list flows", http.StatusInternalServerError)
			return
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

		flowID, err := s.flowRegistry.Create(accountID, req.Name, req.Content)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(CreateFlowResponse{ID: flowID})

	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

// handleFlowByID handles flow retrieval, update, and deletion
func (s *TestServer) handleFlowByID(w http.ResponseWriter, r *http.Request) {
	// Get account ID from context
	accountID, ok := r.Context().Value(middleware.AccountIDKey).(string)
	if !ok {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	// Extract flow ID from URL
	path := r.URL.Path
	if len(path) <= len("/api/v1/flows/") {
		http.Error(w, "Invalid flow ID", http.StatusBadRequest)
		return
	}

	flowID := path[len("/api/v1/flows/"):]

	// Check if this is a metadata update
	if len(flowID) > len("metadata") && flowID[len(flowID)-len("metadata"):] == "metadata" {
		// Handle metadata update
		actualFlowID := flowID[:len(flowID)-len("/metadata")]
		if r.Method == http.MethodPatch {
			var req UpdateFlowMetadataRequest
			if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
				http.Error(w, "Invalid request body", http.StatusBadRequest)
				return
			}

			metadata := registry.FlowMetadata{
				Tags:     req.Tags,
				Category: req.Category,
				Status:   req.Status,
				Custom:   req.Custom,
			}

			err := s.flowRegistry.UpdateMetadata(accountID, actualFlowID, metadata)
			if err != nil {
				http.Error(w, err.Error(), http.StatusBadRequest)
				return
			}

			w.WriteHeader(http.StatusNoContent)
			return
		}

		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	switch r.Method {
	case http.MethodGet:
		// Get flow
		content, err := s.flowRegistry.Get(accountID, flowID)
		if err != nil {
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

		err := s.flowRegistry.Update(accountID, flowID, req.Content)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		w.WriteHeader(http.StatusNoContent)

	case http.MethodDelete:
		// Delete flow
		err := s.flowRegistry.Delete(accountID, flowID)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		w.WriteHeader(http.StatusNoContent)

	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

// handleSearchFlows handles flow searching
func (s *TestServer) handleSearchFlows(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Get account ID from context
	accountID, ok := r.Context().Value(middleware.AccountIDKey).(string)
	if !ok {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	var req SearchFlowsRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	filters := registry.FlowSearchFilters{
		NameContains:        req.NameContains,
		DescriptionContains: req.DescriptionContains,
		Tags:                req.Tags,
		Category:            req.Category,
		Status:              req.Status,
		Page:                req.Page,
		PageSize:            req.PageSize,
	}

	if req.CreatedAfter != nil {
		t := time.Unix(*req.CreatedAfter, 0)
		filters.CreatedAfter = &t
	}

	if req.CreatedBefore != nil {
		t := time.Unix(*req.CreatedBefore, 0)
		filters.CreatedBefore = &t
	}

	if req.UpdatedAfter != nil {
		t := time.Unix(*req.UpdatedAfter, 0)
		filters.UpdatedAfter = &t
	}

	if req.UpdatedBefore != nil {
		t := time.Unix(*req.UpdatedBefore, 0)
		filters.UpdatedBefore = &t
	}

	flows, err := s.flowRegistry.Search(accountID, filters)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(flows)
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

	var account auth.Account
	if err := json.NewDecoder(createResp.Body).Decode(&account); err != nil {
		log.Fatalf("Failed to parse create account response: %v", err)
	}

	fmt.Printf("Account created: %s (%s)\n", account.Username, account.ID)

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

	var flows []registry.FlowInfo
	if err := json.NewDecoder(listFlowsResp.Body).Decode(&flows); err != nil {
		log.Fatalf("Failed to parse list flows response: %v", err)
	}

	fmt.Printf("Found %d flows:\n", len(flows))
	for _, flow := range flows {
		fmt.Printf("- %s (%s)\n", flow.Name, flow.ID)
	}

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
	flowContent = string(flowContentBytes)

	fmt.Printf("Retrieved flow content:\n%s\n", flowContent)

	// Step 6: Update flow metadata
	fmt.Println("\nStep 6: Updating flow metadata...")
	updateMetadataReq := UpdateFlowMetadataRequest{
		Tags:     []string{"test", "api", "example"},
		Category: "testing",
		Status:   "draft",
		Custom: map[string]interface{}{
			"owner":      "flowuser",
			"department": "engineering",
			"priority":   1,
		},
	}

	updateMetadataResp, err := makeAuthenticatedRequest(
		http.MethodPatch,
		fmt.Sprintf("%s/api/v1/flows/%s/metadata", baseURL, flowID),
		token,
		updateMetadataReq,
	)
	if err != nil {
		log.Fatalf("Failed to update flow metadata: %v", err)
	}
	defer updateMetadataResp.Body.Close()

	if updateMetadataResp.StatusCode != http.StatusNoContent {
		body, _ := io.ReadAll(updateMetadataResp.Body)
		log.Fatalf("Failed to update flow metadata: %s", body)
	}

	fmt.Println("Flow metadata updated successfully")

	// Step 7: Search flows
	fmt.Println("\nStep 7: Searching flows...")
	searchReq := SearchFlowsRequest{
		Tags:     []string{"test"},
		Category: "testing",
		Status:   "draft",
	}

	searchResp, err := makeAuthenticatedRequest(
		http.MethodPost,
		fmt.Sprintf("%s/api/v1/flows/search", baseURL),
		token,
		searchReq,
	)
	if err != nil {
		log.Fatalf("Failed to search flows: %v", err)
	}
	defer searchResp.Body.Close()

	if searchResp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(searchResp.Body)
		log.Fatalf("Failed to search flows: %s", body)
	}

	var searchResults []registry.FlowInfo
	if err := json.NewDecoder(searchResp.Body).Decode(&searchResults); err != nil {
		log.Fatalf("Failed to parse search results: %v", err)
	}

	fmt.Printf("Search found %d flows:\n", len(searchResults))
	for _, flow := range searchResults {
		fmt.Printf("- %s (%s)\n", flow.Name, flow.ID)
	}

	// Step 8: Update flow content
	fmt.Println("\nStep 8: Updating flow content...")
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

	// Step 9: Get updated flow
	fmt.Println("\nStep 9: Getting updated flow...")
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
	updatedContent := string(updatedContentBytes)

	fmt.Printf("Retrieved updated flow content:\n%s\n", updatedContent)

	// Step 10: Delete flow
	fmt.Println("\nStep 10: Deleting flow...")
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

	// Step 11: Verify flow is deleted
	fmt.Println("\nStep 11: Verifying flow is deleted...")
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
