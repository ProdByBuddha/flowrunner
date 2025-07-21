package api

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/gorilla/mux"
	"github.com/tcmartin/flowrunner/pkg/middleware"
)

// AccountRequest represents a request to create or update an account
type AccountRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

// AccountResponse represents an account in API responses (without sensitive data)
type AccountResponse struct {
	ID        string `json:"id"`
	Username  string `json:"username"`
	CreatedAt string `json:"created_at"`
	UpdatedAt string `json:"updated_at"`
}

// AccountListResponse represents a list of accounts
type AccountListResponse struct {
	Accounts []AccountResponse `json:"accounts"`
	Total    int               `json:"total"`
}

// handleListAccounts handles GET /api/v1/accounts
// This is an admin-only endpoint
func (s *Server) handleListAccounts(w http.ResponseWriter, r *http.Request) {
	// Get authenticated account ID from context
	_, ok := middleware.GetAccountID(r)
	if !ok {
		http.Error(w, "Authentication required", http.StatusUnauthorized)
		return
	}

	// TODO: Add admin check here
	// For now, we'll allow any authenticated user to list accounts
	// In a production system, this should be restricted to admin users

	// Get all accounts
	accounts, err := s.accountService.ListAccounts()
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to list accounts: %v", err), http.StatusInternalServerError)
		return
	}

	// Convert to response format (without sensitive data)
	accountResponses := make([]AccountResponse, len(accounts))
	for i, account := range accounts {
		accountResponses[i] = AccountResponse{
			ID:        account.ID,
			Username:  account.Username,
			CreatedAt: account.CreatedAt.Format(http.TimeFormat),
			UpdatedAt: account.UpdatedAt.Format(http.TimeFormat),
		}
	}

	response := AccountListResponse{
		Accounts: accountResponses,
		Total:    len(accountResponses),
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// handleGetAccount handles GET /api/v1/accounts/{id}
func (s *Server) handleGetAccount(w http.ResponseWriter, r *http.Request) {
	// Get account ID from URL
	vars := mux.Vars(r)
	accountID := vars["id"]

	// Get authenticated account ID from context
	authAccountID, ok := middleware.GetAccountID(r)
	if !ok {
		http.Error(w, "Authentication required", http.StatusUnauthorized)
		return
	}

	// Ensure user can only access their own account unless they are an admin
	// TODO: Add admin check here
	if accountID != authAccountID {
		http.Error(w, "Access denied", http.StatusForbidden)
		return
	}

	// Get the account
	account, err := s.accountService.GetAccount(accountID)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to get account: %v", err), http.StatusInternalServerError)
		return
	}

	// Convert to response format (without sensitive data)
	response := AccountResponse{
		ID:        account.ID,
		Username:  account.Username,
		CreatedAt: account.CreatedAt.Format(http.TimeFormat),
		UpdatedAt: account.UpdatedAt.Format(http.TimeFormat),
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// handleDeleteAccount handles DELETE /api/v1/accounts/{id}
func (s *Server) handleDeleteAccount(w http.ResponseWriter, r *http.Request) {
	// Get account ID from URL
	vars := mux.Vars(r)
	accountID := vars["id"]

	// Get authenticated account ID from context
	authAccountID, ok := middleware.GetAccountID(r)
	if !ok {
		http.Error(w, "Authentication required", http.StatusUnauthorized)
		return
	}

	// Ensure user can only delete their own account unless they are an admin
	// TODO: Add admin check here
	if accountID != authAccountID {
		http.Error(w, "Access denied", http.StatusForbidden)
		return
	}

	// Delete the account
	if err := s.accountService.DeleteAccount(accountID); err != nil {
		http.Error(w, fmt.Sprintf("Failed to delete account: %v", err), http.StatusInternalServerError)
		return
	}

	// Return success response
	w.WriteHeader(http.StatusNoContent)
}

// handleUpdateAccount handles PUT /api/v1/accounts/{id}
func (s *Server) handleUpdateAccount(w http.ResponseWriter, r *http.Request) {
	// Get account ID from URL
	vars := mux.Vars(r)
	accountID := vars["id"]

	// Get authenticated account ID from context
	authAccountID, ok := middleware.GetAccountID(r)
	if !ok {
		http.Error(w, "Authentication required", http.StatusUnauthorized)
		return
	}

	// Ensure user can only update their own account unless they are an admin
	// TODO: Add admin check here
	if accountID != authAccountID {
		http.Error(w, "Access denied", http.StatusForbidden)
		return
	}

	// Parse request body
	var req AccountRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// For now, we only support updating the password
	// This would require extending the AccountService interface to support this operation
	// For now, we'll return a not implemented error
	http.Error(w, "Account update not implemented", http.StatusNotImplemented)
}
