package api

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/gorilla/mux"
	"github.com/tcmartin/flowrunner/pkg/auth"
	"github.com/tcmartin/flowrunner/pkg/config"
	"github.com/tcmartin/flowrunner/pkg/middleware"
	"github.com/tcmartin/flowrunner/pkg/registry"
)

// Server represents the HTTP API server
type Server struct {
	config         *config.Config
	router         *mux.Router
	server         *http.Server
	flowRegistry   registry.FlowRegistry
	accountService auth.AccountService
}

// NewServer creates a new API server
func NewServer(cfg *config.Config, flowRegistry registry.FlowRegistry, accountService auth.AccountService) *Server {
	s := &Server{
		config:         cfg,
		router:         mux.NewRouter(),
		flowRegistry:   flowRegistry,
		accountService: accountService,
	}

	s.setupRoutes()
	return s
}

// Start starts the HTTP server
func (s *Server) Start() error {
	addr := fmt.Sprintf("%s:%d", s.config.Server.Host, s.config.Server.Port)
	s.server = &http.Server{
		Addr:         addr,
		Handler:      s.router,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	log.Printf("Starting HTTP server on %s", addr)

	var err error
	if s.config.Server.TLS.Enabled {
		err = s.server.ListenAndServeTLS(
			s.config.Server.TLS.CertFile,
			s.config.Server.TLS.KeyFile,
		)
	} else {
		err = s.server.ListenAndServe()
	}

	// If the server was shut down gracefully, this error is expected
	if err == http.ErrServerClosed {
		return nil
	}
	return err
}

// Stop stops the HTTP server gracefully
func (s *Server) Stop(ctx context.Context) error {
	return s.server.Shutdown(ctx)
}

// setupRoutes configures the API routes
func (s *Server) setupRoutes() {
	// Create authentication middleware
	authMiddleware := middleware.NewAuthMiddleware(s.accountService)

	// API router with version prefix
	api := s.router.PathPrefix("/api/v1").Subrouter()

	// Public routes (no authentication required)
	api.HandleFunc("/health", s.handleHealth).Methods(http.MethodGet, http.MethodOptions)
	api.HandleFunc("/login", s.handleLogin).Methods(http.MethodPost, http.MethodOptions)

	// Account routes
	accounts := api.PathPrefix("/accounts").Subrouter()
	accounts.HandleFunc("", s.handleCreateAccount).Methods(http.MethodPost, http.MethodOptions)

	// Authenticated routes
	authenticated := api.PathPrefix("").Subrouter()
	authenticated.Use(authMiddleware.Authenticate)

	// Flow routes
	flows := authenticated.PathPrefix("/flows").Subrouter()
	flows.HandleFunc("", s.handleListFlows).Methods(http.MethodGet, http.MethodOptions)
	flows.HandleFunc("", s.handleCreateFlow).Methods(http.MethodPost, http.MethodOptions)
	flows.HandleFunc("/{id}", s.handleGetFlow).Methods(http.MethodGet, http.MethodOptions)
	flows.HandleFunc("/{id}", s.handleUpdateFlow).Methods(http.MethodPut, http.MethodOptions)
	flows.HandleFunc("/{id}", s.handleDeleteFlow).Methods(http.MethodDelete, http.MethodOptions)
	flows.HandleFunc("/{id}/metadata", s.handleUpdateFlowMetadata).Methods(http.MethodPatch, http.MethodOptions)
	flows.HandleFunc("/search", s.handleSearchFlows).Methods(http.MethodPost, http.MethodOptions)

	// Account management routes (authenticated)
	accountsMgmt := authenticated.PathPrefix("/accounts").Subrouter()
	accountsMgmt.HandleFunc("/me", s.handleGetCurrentAccount).Methods(http.MethodGet, http.MethodOptions)
	accountsMgmt.HandleFunc("/refresh-token", s.handleRefreshToken).Methods(http.MethodPost, http.MethodOptions)

	// Debug middleware to log all requests
	s.router.Use(func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			log.Printf("Request: %s %s", r.Method, r.URL.Path)
			next.ServeHTTP(w, r)
		})
	})

	// CORS middleware for all routes
	s.router.Use(middleware.CORS)
}

// handleHealth handles the health check endpoint
func (s *Server) handleHealth(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{
		"status": "ok",
		"time":   time.Now().Format(time.RFC3339),
	})
}

// handleCreateAccount handles account creation
func (s *Server) handleCreateAccount(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Username string `json:"username"`
		Password string `json:"password"`
	}

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

// handleGetCurrentAccount handles retrieving the current account
func (s *Server) handleGetCurrentAccount(w http.ResponseWriter, r *http.Request) {
	accountID, ok := middleware.GetAccountID(r)
	if !ok {
		http.Error(w, "Authentication required", http.StatusUnauthorized)
		return
	}

	account, err := s.accountService.GetAccount(accountID)
	if err != nil {
		http.Error(w, "Failed to retrieve account", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(account)
}

// handleListFlows handles listing flows
func (s *Server) handleListFlows(w http.ResponseWriter, r *http.Request) {
	accountID, ok := middleware.GetAccountID(r)
	if !ok {
		http.Error(w, "Authentication required", http.StatusUnauthorized)
		return
	}

	flows, err := s.flowRegistry.List(accountID)
	if err != nil {
		http.Error(w, "Failed to list flows", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(flows)
}

// handleCreateFlow handles flow creation
func (s *Server) handleCreateFlow(w http.ResponseWriter, r *http.Request) {
	accountID, ok := middleware.GetAccountID(r)
	if !ok {
		http.Error(w, "Authentication required", http.StatusUnauthorized)
		return
	}

	var req struct {
		Name    string `json:"name"`
		Content string `json:"content"`
	}

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
	json.NewEncoder(w).Encode(map[string]string{
		"id": flowID,
	})
}

// handleGetFlow handles retrieving a flow
func (s *Server) handleGetFlow(w http.ResponseWriter, r *http.Request) {
	accountID, ok := middleware.GetAccountID(r)
	if !ok {
		http.Error(w, "Authentication required", http.StatusUnauthorized)
		return
	}

	vars := mux.Vars(r)
	flowID := vars["id"]

	content, err := s.flowRegistry.Get(accountID, flowID)
	if err != nil {
		http.Error(w, "Flow not found", http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/yaml")
	w.Write([]byte(content))
}

// handleUpdateFlow handles updating a flow
func (s *Server) handleUpdateFlow(w http.ResponseWriter, r *http.Request) {
	accountID, ok := middleware.GetAccountID(r)
	if !ok {
		http.Error(w, "Authentication required", http.StatusUnauthorized)
		return
	}

	vars := mux.Vars(r)
	flowID := vars["id"]

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
}

// handleDeleteFlow handles deleting a flow
func (s *Server) handleDeleteFlow(w http.ResponseWriter, r *http.Request) {
	accountID, ok := middleware.GetAccountID(r)
	if !ok {
		http.Error(w, "Authentication required", http.StatusUnauthorized)
		return
	}

	vars := mux.Vars(r)
	flowID := vars["id"]

	err := s.flowRegistry.Delete(accountID, flowID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// handleUpdateFlowMetadata handles updating flow metadata
func (s *Server) handleUpdateFlowMetadata(w http.ResponseWriter, r *http.Request) {
	accountID, ok := middleware.GetAccountID(r)
	if !ok {
		http.Error(w, "Authentication required", http.StatusUnauthorized)
		return
	}

	vars := mux.Vars(r)
	flowID := vars["id"]

	var metadata registry.FlowMetadata
	if err := json.NewDecoder(r.Body).Decode(&metadata); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	err := s.flowRegistry.UpdateMetadata(accountID, flowID, metadata)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// handleSearchFlows handles searching for flows
func (s *Server) handleSearchFlows(w http.ResponseWriter, r *http.Request) {
	accountID, ok := middleware.GetAccountID(r)
	if !ok {
		http.Error(w, "Authentication required", http.StatusUnauthorized)
		return
	}

	var filters registry.FlowSearchFilters
	if err := json.NewDecoder(r.Body).Decode(&filters); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	flows, err := s.flowRegistry.Search(accountID, filters)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(flows)
}
