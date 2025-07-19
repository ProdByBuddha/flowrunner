package main

import (
	"encoding/json"
	"log"
	"net/http"
	"time"

	"github.com/gorilla/mux"
	"github.com/tcmartin/flowrunner/pkg/auth"
	"github.com/tcmartin/flowrunner/pkg/middleware"
	"github.com/tcmartin/flowrunner/pkg/services"
	"github.com/tcmartin/flowrunner/pkg/storage"
)

func main() {
	// Create a memory account store
	accountStore := storage.NewMemoryAccountStore()

	// Create an account service
	accountService := services.NewAccountService(accountStore)

	// Create a router
	router := mux.NewRouter()

	// Create authentication middleware
	authMiddleware := middleware.NewAuthMiddleware(accountService)

	// API router with version prefix
	api := router.PathPrefix("/api/v1").Subrouter()

	// Public routes (no authentication required)
	api.HandleFunc("/health", handleHealth).Methods(http.MethodGet, http.MethodOptions)
	api.HandleFunc("/accounts", handleCreateAccount(accountService)).Methods(http.MethodPost, http.MethodOptions)

	// Authenticated routes
	authenticated := api.PathPrefix("").Subrouter()
	authenticated.Use(authMiddleware.Authenticate)

	// Account management routes (authenticated)
	authenticated.HandleFunc("/accounts/me", handleGetCurrentAccount(accountService)).Methods(http.MethodGet, http.MethodOptions)

	// Add CORS middleware
	router.Use(middleware.CORS)

	// Add debug middleware
	router.Use(func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			log.Printf("Request: %s %s", r.Method, r.URL.Path)
			next.ServeHTTP(w, r)
		})
	})

	// Start the server
	addr := "localhost:8090"
	log.Printf("Starting HTTP server on %s", addr)
	log.Fatal(http.ListenAndServe(addr, router))
}

// handleHealth handles the health check endpoint
func handleHealth(w http.ResponseWriter, r *http.Request) {
	log.Println("Health check requested")
	w.Header().Set("Content-Type", "application/json")
	response := map[string]string{
		"status": "ok",
		"time":   time.Now().Format(time.RFC3339),
	}

	responseBytes, err := json.Marshal(response)
	if err != nil {
		log.Printf("Error marshaling response: %v", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	log.Printf("Health response: %s", string(responseBytes))
	w.Write(responseBytes)
}

// handleCreateAccount handles account creation
func handleCreateAccount(accountService auth.AccountService) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req struct {
			Username string `json:"username"`
			Password string `json:"password"`
		}

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

// handleGetCurrentAccount handles retrieving the current account
func handleGetCurrentAccount(accountService auth.AccountService) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		accountID, ok := middleware.GetAccountID(r)
		if !ok {
			http.Error(w, "Authentication required", http.StatusUnauthorized)
			return
		}

		account, err := accountService.GetAccount(accountID)
		if err != nil {
			http.Error(w, "Failed to retrieve account", http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(account)
	}
}
