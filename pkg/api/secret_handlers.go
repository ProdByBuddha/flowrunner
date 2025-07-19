package api

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/gorilla/mux"
	"github.com/tcmartin/flowrunner/pkg/middleware"
)

// SecretRequest represents a request to create or update a secret
type SecretRequest struct {
	Value string `json:"value"`
}

// SecretResponse represents a secret in API responses (without the value)
type SecretResponse struct {
	Key       string    `json:"key"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// SecretListResponse represents a list of secrets
type SecretListResponse struct {
	Secrets []SecretResponse `json:"secrets"`
	Total   int              `json:"total"`
}

// handleListSecrets handles GET /api/v1/accounts/{accountId}/secrets
func (s *Server) handleListSecrets(w http.ResponseWriter, r *http.Request) {
	// Get account ID from URL
	vars := mux.Vars(r)
	accountID := vars["accountId"]

	// Get authenticated account ID from context
	authAccountID, ok := middleware.GetAccountID(r)
	if !ok {
		http.Error(w, "Authentication required", http.StatusUnauthorized)
		return
	}

	// Ensure user can only access their own secrets
	if accountID != authAccountID {
		http.Error(w, "Access denied", http.StatusForbidden)
		return
	}

	// Get secrets using the basic list method and create metadata manually
	keys, err := s.secretVault.List(accountID)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to list secrets: %v", err), http.StatusInternalServerError)
		return
	}

	// For basic secrets, we'll just return the keys with minimal metadata
	secretResponses := make([]SecretResponse, len(keys))
	for i, key := range keys {
		// Note: For basic secrets we don't have creation/update times easily available
		// This is a limitation of the basic SecretVault interface
		secretResponses[i] = SecretResponse{
			Key:       key,
			CreatedAt: time.Time{}, // Will be zero time for basic secrets
			UpdatedAt: time.Time{}, // Will be zero time for basic secrets
		}
	}

	response := SecretListResponse{
		Secrets: secretResponses,
		Total:   len(secretResponses),
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// handleCreateSecret handles POST /api/v1/accounts/{accountId}/secrets/{key}
func (s *Server) handleCreateSecret(w http.ResponseWriter, r *http.Request) {
	// Get account ID and key from URL
	vars := mux.Vars(r)
	accountID := vars["accountId"]
	key := vars["key"]

	// Get authenticated account ID from context
	authAccountID, ok := middleware.GetAccountID(r)
	if !ok {
		http.Error(w, "Authentication required", http.StatusUnauthorized)
		return
	}

	// Ensure user can only access their own secrets
	if accountID != authAccountID {
		http.Error(w, "Access denied", http.StatusForbidden)
		return
	}

	// Parse request body
	var req SecretRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// Validate input
	if key == "" {
		http.Error(w, "Secret key cannot be empty", http.StatusBadRequest)
		return
	}

	// Store the secret
	if err := s.secretVault.Set(accountID, key, req.Value); err != nil {
		http.Error(w, fmt.Sprintf("Failed to store secret: %v", err), http.StatusInternalServerError)
		return
	}

	// Return success response
	response := SecretResponse{
		Key:       key,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(response)
}

// handleGetSecret handles GET /api/v1/accounts/{accountId}/secrets/{key}
func (s *Server) handleGetSecret(w http.ResponseWriter, r *http.Request) {
	// Get account ID and key from URL
	vars := mux.Vars(r)
	accountID := vars["accountId"]
	key := vars["key"]

	// Get authenticated account ID from context
	authAccountID, ok := middleware.GetAccountID(r)
	if !ok {
		http.Error(w, "Authentication required", http.StatusUnauthorized)
		return
	}

	// Ensure user can only access their own secrets
	if accountID != authAccountID {
		http.Error(w, "Access denied", http.StatusForbidden)
		return
	}

	// Get the secret value
	value, err := s.secretVault.Get(accountID, key)
	if err != nil {
		if err.Error() == "secret not found" {
			http.Error(w, "Secret not found", http.StatusNotFound)
			return
		}
		http.Error(w, fmt.Sprintf("Failed to retrieve secret: %v", err), http.StatusInternalServerError)
		return
	}

	// Return the secret value
	response := map[string]interface{}{
		"key":   key,
		"value": value,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// handleUpdateSecret handles PUT /api/v1/accounts/{accountId}/secrets/{key}
func (s *Server) handleUpdateSecret(w http.ResponseWriter, r *http.Request) {
	// Get account ID and key from URL
	vars := mux.Vars(r)
	accountID := vars["accountId"]
	key := vars["key"]

	// Get authenticated account ID from context
	authAccountID, ok := middleware.GetAccountID(r)
	if !ok {
		http.Error(w, "Authentication required", http.StatusUnauthorized)
		return
	}

	// Ensure user can only access their own secrets
	if accountID != authAccountID {
		http.Error(w, "Access denied", http.StatusForbidden)
		return
	}

	// Parse request body
	var req SecretRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// Check if secret exists
	_, err := s.secretVault.Get(accountID, key)
	if err != nil {
		if err.Error() == "secret not found" {
			http.Error(w, "Secret not found", http.StatusNotFound)
			return
		}
		http.Error(w, fmt.Sprintf("Failed to check secret existence: %v", err), http.StatusInternalServerError)
		return
	}

	// Update the secret
	if err := s.secretVault.Set(accountID, key, req.Value); err != nil {
		http.Error(w, fmt.Sprintf("Failed to update secret: %v", err), http.StatusInternalServerError)
		return
	}

	// Return success response
	response := SecretResponse{
		Key:       key,
		CreatedAt: time.Now(), // This will be updated by the service to preserve creation time
		UpdatedAt: time.Now(),
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// handleDeleteSecret handles DELETE /api/v1/accounts/{accountId}/secrets/{key}
func (s *Server) handleDeleteSecret(w http.ResponseWriter, r *http.Request) {
	// Get account ID and key from URL
	vars := mux.Vars(r)
	accountID := vars["accountId"]
	key := vars["key"]

	// Get authenticated account ID from context
	authAccountID, ok := middleware.GetAccountID(r)
	if !ok {
		http.Error(w, "Authentication required", http.StatusUnauthorized)
		return
	}

	// Ensure user can only access their own secrets
	if accountID != authAccountID {
		http.Error(w, "Access denied", http.StatusForbidden)
		return
	}

	// Delete the secret
	if err := s.secretVault.Delete(accountID, key); err != nil {
		if err.Error() == "secret not found" {
			http.Error(w, "Secret not found", http.StatusNotFound)
			return
		}
		http.Error(w, fmt.Sprintf("Failed to delete secret: %v", err), http.StatusInternalServerError)
		return
	}

	// Return success response
	w.WriteHeader(http.StatusNoContent)
}

// handleSecretKeys handles GET /api/v1/accounts/{accountId}/secrets/keys (convenience endpoint)
func (s *Server) handleSecretKeys(w http.ResponseWriter, r *http.Request) {
	// Get account ID from URL
	vars := mux.Vars(r)
	accountID := vars["accountId"]

	// Get authenticated account ID from context
	authAccountID, ok := middleware.GetAccountID(r)
	if !ok {
		http.Error(w, "Authentication required", http.StatusUnauthorized)
		return
	}

	// Ensure user can only access their own secrets
	if accountID != authAccountID {
		http.Error(w, "Access denied", http.StatusForbidden)
		return
	}

	// Get secret keys
	keys, err := s.secretVault.List(accountID)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to list secret keys: %v", err), http.StatusInternalServerError)
		return
	}

	// Return the keys
	response := map[string]interface{}{
		"keys":  keys,
		"total": len(keys),
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}
