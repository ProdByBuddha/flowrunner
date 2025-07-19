package api

import (
	"encoding/json"
	"net/http"

	"github.com/tcmartin/flowrunner/pkg/middleware"
	"github.com/tcmartin/flowrunner/pkg/services"
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

// handleLogin handles user login and returns a JWT token
func (s *Server) handleLogin(w http.ResponseWriter, r *http.Request) {
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
	accountService, ok := s.accountService.(*services.AccountService)
	if !ok {
		http.Error(w, "JWT authentication not supported", http.StatusInternalServerError)
		return
	}

	token, err := accountService.GenerateJWT(accountID)
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

// handleRefreshToken handles token refresh
func (s *Server) handleRefreshToken(w http.ResponseWriter, r *http.Request) {
	// Get the account ID from the context
	accountID, ok := middleware.GetAccountID(r)
	if !ok {
		http.Error(w, "Authentication required", http.StatusUnauthorized)
		return
	}

	// Get the account
	account, err := s.accountService.GetAccount(accountID)
	if err != nil {
		http.Error(w, "Failed to retrieve account", http.StatusInternalServerError)
		return
	}

	// Generate a new JWT token
	accountService, ok := s.accountService.(*services.AccountService)
	if !ok {
		http.Error(w, "JWT authentication not supported", http.StatusInternalServerError)
		return
	}

	token, err := accountService.GenerateJWT(accountID)
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
