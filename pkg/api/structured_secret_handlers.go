package api

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/gorilla/mux"
	"github.com/tcmartin/flowrunner/pkg/auth"
	"github.com/tcmartin/flowrunner/pkg/middleware"
)

// StructuredSecretRequest represents a request to create or update a structured secret
type StructuredSecretRequest struct {
	Type        auth.SecretType           `json:"type"`
	Description string                    `json:"description,omitempty"`
	Tags        []string                  `json:"tags,omitempty"`
	ExpiresAt   *time.Time                `json:"expires_at,omitempty"`
	Data        map[string]interface{}    `json:"data"`
	Schema      *auth.SecretSchema        `json:"schema,omitempty"`
}

// OAuthSecretRequest represents an OAuth secret request
type OAuthSecretRequest struct {
	Description  string            `json:"description,omitempty"`
	Tags         []string          `json:"tags,omitempty"`
	ExpiresAt    *time.Time        `json:"expires_at,omitempty"`
	ClientID     string            `json:"client_id"`
	ClientSecret string            `json:"client_secret"`
	TokenURL     string            `json:"token_url,omitempty"`
	AuthURL      string            `json:"auth_url,omitempty"`
	RedirectURL  string            `json:"redirect_url,omitempty"`
	Scopes       []string          `json:"scopes,omitempty"`
	AccessToken  string            `json:"access_token,omitempty"`
	RefreshToken string            `json:"refresh_token,omitempty"`
	TokenExpires *time.Time        `json:"token_expires_at,omitempty"`
	Extra        map[string]string `json:"extra,omitempty"`
}

// APIKeySecretRequest represents an API key secret request
type APIKeySecretRequest struct {
	Description string            `json:"description,omitempty"`
	Tags        []string          `json:"tags,omitempty"`
	ExpiresAt   *time.Time        `json:"expires_at,omitempty"`
	Key         string            `json:"key"`
	HeaderName  string            `json:"header_name,omitempty"`
	Prefix      string            `json:"prefix,omitempty"`
	Headers     map[string]string `json:"headers,omitempty"`
	QueryParam  string            `json:"query_param,omitempty"`
	BaseURL     string            `json:"base_url,omitempty"`
	RateLimit   *auth.RateLimit   `json:"rate_limit,omitempty"`
}

// DatabaseSecretRequest represents a database secret request
type DatabaseSecretRequest struct {
	Description  string            `json:"description,omitempty"`
	Tags         []string          `json:"tags,omitempty"`
	ExpiresAt    *time.Time        `json:"expires_at,omitempty"`
	Type         string            `json:"type"`
	Host         string            `json:"host"`
	Port         int               `json:"port"`
	Database     string            `json:"database"`
	Username     string            `json:"username"`
	Password     string            `json:"password"`
	SSLMode      string            `json:"ssl_mode,omitempty"`
	SSLCert      string            `json:"ssl_cert,omitempty"`
	SSLKey       string            `json:"ssl_key,omitempty"`
	SSLRootCert  string            `json:"ssl_root_cert,omitempty"`
	Timezone     string            `json:"timezone,omitempty"`
	MaxConns     int               `json:"max_conns,omitempty"`
	MaxIdleConns int               `json:"max_idle_conns,omitempty"`
	ConnTimeout  string            `json:"conn_timeout,omitempty"`
	Options      map[string]string `json:"options,omitempty"`
}

// JWTSecretRequest represents a JWT secret request
type JWTSecretRequest struct {
	Description  string            `json:"description,omitempty"`
	Tags         []string          `json:"tags,omitempty"`
	ExpiresAt    *time.Time        `json:"expires_at,omitempty"`
	Token        string            `json:"token"`
	Algorithm    string            `json:"algorithm,omitempty"`
	Audience     string            `json:"audience,omitempty"`
	Issuer       string            `json:"issuer,omitempty"`
	Subject      string            `json:"subject,omitempty"`
	TokenExpires *time.Time        `json:"token_expires_at,omitempty"`
	IssuedAt     *time.Time        `json:"issued_at,omitempty"`
	Claims       map[string]string `json:"claims,omitempty"`
}

// StructuredSecretResponse represents a structured secret in API responses
type StructuredSecretResponse struct {
	Key       string                 `json:"key"`
	Type      auth.SecretType        `json:"type"`
	Metadata  auth.SecretMetadata    `json:"metadata"`
	Data      map[string]interface{} `json:"data,omitempty"`
	Schema    *auth.SecretSchema     `json:"schema,omitempty"`
	CreatedAt time.Time              `json:"created_at"`
	UpdatedAt time.Time              `json:"updated_at"`
}

// SecretSearchRequest represents a search request
type SecretSearchRequest struct {
	Type           auth.SecretType `json:"type,omitempty"`
	Tags           []string        `json:"tags,omitempty"`
	Description    string          `json:"description,omitempty"`
	ExpiringWithin string          `json:"expiring_within,omitempty"` // Duration string like "24h", "7d"
	Limit          int             `json:"limit,omitempty"`
	Offset         int             `json:"offset,omitempty"`
}

// handleCreateStructuredSecret handles POST /api/v1/accounts/{accountId}/secrets/{key}/structured
func (s *Server) handleCreateStructuredSecret(w http.ResponseWriter, r *http.Request) {
	accountID, key, ok := s.extractAccountAndKey(w, r)
	if !ok {
		return
	}

	var req StructuredSecretRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// Validate required fields
	if req.Type == "" {
		req.Type = auth.SecretTypeCustom
	}

	// Convert data to JSON
	dataJSON, err := auth.ToJSON(req.Data)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to marshal secret data: %v", err), http.StatusBadRequest)
		return
	}

	// Create metadata
	metadata := auth.SecretMetadata{
		Type:        req.Type,
		Description: req.Description,
		Tags:        req.Tags,
		ExpiresAt:   req.ExpiresAt,
		Version:     1,
	}

	// Create structured secret
	structured := auth.StructuredSecret{
		AccountID: accountID,
		Key:       key,
		Metadata:  metadata,
		Value:     dataJSON,
		Schema:    req.Schema,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	// Store the secret
	if err := s.secretVault.SetStructured(accountID, structured); err != nil {
		http.Error(w, fmt.Sprintf("Failed to store secret: %v", err), http.StatusInternalServerError)
		return
	}

	// Return success response (without sensitive data)
	response := StructuredSecretResponse{
		Key:       key,
		Type:      metadata.Type,
		Metadata:  metadata,
		Schema:    req.Schema,
		CreatedAt: structured.CreatedAt,
		UpdatedAt: structured.UpdatedAt,
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(response)
}

// handleCreateOAuthSecret handles POST /api/v1/accounts/{accountId}/secrets/{key}/oauth
func (s *Server) handleCreateOAuthSecret(w http.ResponseWriter, r *http.Request) {
	accountID, key, ok := s.extractAccountAndKey(w, r)
	if !ok {
		return
	}

	var req OAuthSecretRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// Validate required fields
	if req.ClientID == "" || req.ClientSecret == "" {
		http.Error(w, "client_id and client_secret are required", http.StatusBadRequest)
		return
	}

	// Create OAuth secret
	oauth := auth.OAuthSecret{
		ClientID:     req.ClientID,
		ClientSecret: req.ClientSecret,
		TokenURL:     req.TokenURL,
		AuthURL:      req.AuthURL,
		RedirectURL:  req.RedirectURL,
		Scopes:       req.Scopes,
		AccessToken:  req.AccessToken,
		RefreshToken: req.RefreshToken,
		ExpiresAt:    req.TokenExpires,
		Extra:        req.Extra,
	}

	// Create metadata
	metadata := auth.SecretMetadata{
		Type:        auth.SecretTypeOAuth,
		Description: req.Description,
		Tags:        req.Tags,
		ExpiresAt:   req.ExpiresAt,
		Version:     1,
	}

	// Store the secret
	if err := s.secretVault.SetOAuth(accountID, key, oauth, metadata); err != nil {
		http.Error(w, fmt.Sprintf("Failed to store OAuth secret: %v", err), http.StatusInternalServerError)
		return
	}

	// Return success response
	response := StructuredSecretResponse{
		Key:       key,
		Type:      auth.SecretTypeOAuth,
		Metadata:  metadata,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(response)
}

// handleCreateAPIKeySecret handles POST /api/v1/accounts/{accountId}/secrets/{key}/apikey
func (s *Server) handleCreateAPIKeySecret(w http.ResponseWriter, r *http.Request) {
	accountID, key, ok := s.extractAccountAndKey(w, r)
	if !ok {
		return
	}

	var req APIKeySecretRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// Validate required fields
	if req.Key == "" {
		http.Error(w, "key is required", http.StatusBadRequest)
		return
	}

	// Create API key secret
	apiKey := auth.APIKeySecret{
		Key:        req.Key,
		HeaderName: req.HeaderName,
		Prefix:     req.Prefix,
		Headers:    req.Headers,
		QueryParam: req.QueryParam,
		BaseURL:    req.BaseURL,
		RateLimit:  req.RateLimit,
	}

	// Create metadata
	metadata := auth.SecretMetadata{
		Type:        auth.SecretTypeAPIKey,
		Description: req.Description,
		Tags:        req.Tags,
		ExpiresAt:   req.ExpiresAt,
		Version:     1,
	}

	// Store the secret
	if err := s.secretVault.SetAPIKey(accountID, key, apiKey, metadata); err != nil {
		http.Error(w, fmt.Sprintf("Failed to store API key secret: %v", err), http.StatusInternalServerError)
		return
	}

	// Return success response
	response := StructuredSecretResponse{
		Key:       key,
		Type:      auth.SecretTypeAPIKey,
		Metadata:  metadata,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(response)
}

// handleCreateDatabaseSecret handles POST /api/v1/accounts/{accountId}/secrets/{key}/database
func (s *Server) handleCreateDatabaseSecret(w http.ResponseWriter, r *http.Request) {
	accountID, key, ok := s.extractAccountAndKey(w, r)
	if !ok {
		return
	}

	var req DatabaseSecretRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// Validate required fields
	if req.Type == "" || req.Host == "" || req.Username == "" || req.Password == "" {
		http.Error(w, "type, host, username, and password are required", http.StatusBadRequest)
		return
	}

	// Create database secret
	db := auth.DatabaseSecret{
		Type:         req.Type,
		Host:         req.Host,
		Port:         req.Port,
		Database:     req.Database,
		Username:     req.Username,
		Password:     req.Password,
		SSLMode:      req.SSLMode,
		SSLCert:      req.SSLCert,
		SSLKey:       req.SSLKey,
		SSLRootCert:  req.SSLRootCert,
		Timezone:     req.Timezone,
		MaxConns:     req.MaxConns,
		MaxIdleConns: req.MaxIdleConns,
		ConnTimeout:  req.ConnTimeout,
		Options:      req.Options,
	}

	// Create metadata
	metadata := auth.SecretMetadata{
		Type:        auth.SecretTypeDatabase,
		Description: req.Description,
		Tags:        req.Tags,
		ExpiresAt:   req.ExpiresAt,
		Version:     1,
	}

	// Store the secret
	if err := s.secretVault.SetDatabase(accountID, key, db, metadata); err != nil {
		http.Error(w, fmt.Sprintf("Failed to store database secret: %v", err), http.StatusInternalServerError)
		return
	}

	// Return success response
	response := StructuredSecretResponse{
		Key:       key,
		Type:      auth.SecretTypeDatabase,
		Metadata:  metadata,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(response)
}

// handleGetStructuredSecret handles GET /api/v1/accounts/{accountId}/secrets/{key}/structured
func (s *Server) handleGetStructuredSecret(w http.ResponseWriter, r *http.Request) {
	accountID, key, ok := s.extractAccountAndKey(w, r)
	if !ok {
		return
	}

	// Get field parameter for specific field extraction
	field := r.URL.Query().Get("field")

	if field != "" {
		// Get specific field
		value, err := s.secretVault.GetField(accountID, key, field)
		if err != nil {
			if strings.Contains(err.Error(), "not found") {
				http.Error(w, "Secret or field not found", http.StatusNotFound)
				return
			}
			http.Error(w, fmt.Sprintf("Failed to retrieve field: %v", err), http.StatusInternalServerError)
			return
		}

		response := map[string]interface{}{
			"key":   key,
			"field": field,
			"value": value,
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
		return
	}

	// Get full structured secret
	secret, err := s.secretVault.GetStructured(accountID, key)
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			http.Error(w, "Secret not found", http.StatusNotFound)
			return
		}
		http.Error(w, fmt.Sprintf("Failed to retrieve secret: %v", err), http.StatusInternalServerError)
		return
	}

	// Parse data
	var data map[string]interface{}
	if err := json.Unmarshal([]byte(secret.Value), &data); err != nil {
		// Fallback to simple value
		data = map[string]interface{}{"value": secret.Value}
	}

	response := StructuredSecretResponse{
		Key:       secret.Key,
		Type:      secret.Metadata.Type,
		Metadata:  secret.Metadata,
		Data:      data,
		Schema:    secret.Schema,
		CreatedAt: secret.CreatedAt,
		UpdatedAt: secret.UpdatedAt,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// handleSearchSecrets handles POST /api/v1/accounts/{accountId}/secrets/search
func (s *Server) handleSearchSecrets(w http.ResponseWriter, r *http.Request) {
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

	var req SecretSearchRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// Convert to auth.SecretQuery
	query := auth.SecretQuery{
		Type:        req.Type,
		Tags:        req.Tags,
		Description: req.Description,
		Limit:       req.Limit,
		Offset:      req.Offset,
	}

	// Parse expiring within duration
	if req.ExpiringWithin != "" {
		duration, err := time.ParseDuration(req.ExpiringWithin)
		if err != nil {
			http.Error(w, fmt.Sprintf("Invalid duration format: %v", err), http.StatusBadRequest)
			return
		}
		query.ExpiringWithin = &duration
	}

	// Search secrets
	secrets, err := s.secretVault.Search(accountID, query)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to search secrets: %v", err), http.StatusInternalServerError)
		return
	}

	// Convert to response format
	responses := make([]StructuredSecretResponse, len(secrets))
	for i, secret := range secrets {
		responses[i] = StructuredSecretResponse{
			Key:       secret.Key,
			Type:      secret.Metadata.Type,
			Metadata:  secret.Metadata,
			Schema:    secret.Schema,
			CreatedAt: secret.CreatedAt,
			UpdatedAt: secret.UpdatedAt,
			// Data is intentionally omitted for security
		}
	}

	response := map[string]interface{}{
		"secrets": responses,
		"total":   len(responses),
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// handleGetSecretsByType handles GET /api/v1/accounts/{accountId}/secrets/type/{type}
func (s *Server) handleGetSecretsByType(w http.ResponseWriter, r *http.Request) {
	// Get account ID and type from URL
	vars := mux.Vars(r)
	accountID := vars["accountId"]
	secretType := auth.SecretType(vars["type"])

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

	// Get secrets by type
	secrets, err := s.secretVault.ListByType(accountID, secretType)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to list secrets: %v", err), http.StatusInternalServerError)
		return
	}

	// Convert to response format
	responses := make([]StructuredSecretResponse, len(secrets))
	for i, secret := range secrets {
		responses[i] = StructuredSecretResponse{
			Key:       secret.Key,
			Type:      secret.Metadata.Type,
			Metadata:  secret.Metadata,
			Schema:    secret.Schema,
			CreatedAt: secret.CreatedAt,
			UpdatedAt: secret.UpdatedAt,
		}
	}

	response := map[string]interface{}{
		"secrets": responses,
		"total":   len(responses),
		"type":    secretType,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// handleListStructuredSecrets handles GET /api/v1/accounts/{accountId}/structured-secrets
func (s *Server) handleListStructuredSecrets(w http.ResponseWriter, r *http.Request) {
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

	// Parse query parameters
	secretType := r.URL.Query().Get("type")
	tags := r.URL.Query()["tag"] // Multiple tags

	// Create search query
	query := auth.SecretQuery{}
	if secretType != "" {
		query.Type = auth.SecretType(secretType)
	}
	if len(tags) > 0 {
		query.Tags = tags
	}

	// Search or list all
	var secrets []auth.StructuredSecret
	var err error

	if secretType != "" || len(tags) > 0 {
		secrets, err = s.secretVault.Search(accountID, query)
	} else {
		// List all structured secrets (we'll simulate this with a custom method)
		secrets, err = s.secretVault.Search(accountID, auth.SecretQuery{})
	}

	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to list secrets: %v", err), http.StatusInternalServerError)
		return
	}

	// Clear sensitive values
	for i := range secrets {
		secrets[i].Value = ""
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"secrets": secrets,
		"total":   len(secrets),
	})
}

// handleSearchStructuredSecrets handles POST /api/v1/accounts/{accountId}/structured-secrets/search
func (s *Server) handleSearchStructuredSecrets(w http.ResponseWriter, r *http.Request) {
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

	// Parse search query
	var query auth.SecretQuery
	if err := json.NewDecoder(r.Body).Decode(&query); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// Search secrets
	secrets, err := s.secretVault.Search(accountID, query)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to search secrets: %v", err), http.StatusInternalServerError)
		return
	}

	// Clear sensitive values
	for i := range secrets {
		secrets[i].Value = ""
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"secrets": secrets,
		"total":   len(secrets),
	})
}

// handleUpdateStructuredSecret handles PUT /api/v1/accounts/{accountId}/structured-secrets/{key}
func (s *Server) handleUpdateStructuredSecret(w http.ResponseWriter, r *http.Request) {
	accountID, key, ok := s.extractAccountAndKey(w, r)
	if !ok {
		return
	}

	// Parse request body
	var req StructuredSecretRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// Get existing secret to preserve creation time
	existingSecret, err := s.secretVault.GetStructured(accountID, key)
	if err != nil {
		http.Error(w, "Secret not found", http.StatusNotFound)
		return
	}

	// Create updated secret
	secret := auth.StructuredSecret{
		AccountID: accountID,
		Key:       key,
		CreatedAt: existingSecret.CreatedAt, // Preserve creation time
		UpdatedAt: time.Now(),
		Metadata:  req.toMetadata(),
		Schema:    req.Schema,
	}

	// Store the secret
	if err := s.secretVault.SetStructured(accountID, secret); err != nil {
		http.Error(w, fmt.Sprintf("Failed to update secret: %v", err), http.StatusInternalServerError)
		return
	}

	// Return the created secret (without value)
	secret.Value = ""
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(secret)
}

// handleDeleteStructuredSecret handles DELETE /api/v1/accounts/{accountId}/structured-secrets/{key}
func (s *Server) handleDeleteStructuredSecret(w http.ResponseWriter, r *http.Request) {
	accountID, key, ok := s.extractAccountAndKey(w, r)
	if !ok {
		return
	}

	// Delete the secret
	if err := s.secretVault.Delete(accountID, key); err != nil {
		http.Error(w, fmt.Sprintf("Failed to delete secret: %v", err), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// handleGetStructuredSecretField handles GET /api/v1/accounts/{accountId}/structured-secrets/{key}/field/{field}
func (s *Server) handleGetStructuredSecretField(w http.ResponseWriter, r *http.Request) {
	accountID, key, ok := s.extractAccountAndKey(w, r)
	if !ok {
		return
	}

	// Get field from URL
	vars := mux.Vars(r)
	field := vars["field"]
	if field == "" {
		http.Error(w, "Field name cannot be empty", http.StatusBadRequest)
		return
	}

	// Get the field value
	value, err := s.secretVault.GetField(accountID, key, field)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to get field: %v", err), http.StatusInternalServerError)
		return
	}

	// Mark secret as used
	s.secretVault.MarkUsed(accountID, key)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"field": field,
		"value": value,
	})
}

// handleUpdateStructuredSecretMetadata handles PATCH /api/v1/accounts/{accountId}/structured-secrets/{key}/metadata
func (s *Server) handleUpdateStructuredSecretMetadata(w http.ResponseWriter, r *http.Request) {
	accountID, key, ok := s.extractAccountAndKey(w, r)
	if !ok {
		return
	}

	// Parse metadata update request
	var metadata auth.SecretMetadata
	if err := json.NewDecoder(r.Body).Decode(&metadata); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// Update metadata
	if err := s.secretVault.UpdateMetadata(accountID, key, metadata); err != nil {
		http.Error(w, fmt.Sprintf("Failed to update metadata: %v", err), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// handleGetOAuthSecret handles GET /api/v1/accounts/{accountId}/oauth-secrets/{key}
func (s *Server) handleGetOAuthSecret(w http.ResponseWriter, r *http.Request) {
	accountID, key, ok := s.extractAccountAndKey(w, r)
	if !ok {
		return
	}

	// Get the structured secret
	secret, err := s.secretVault.GetStructured(accountID, key)
	if err != nil {
		http.Error(w, "Secret not found", http.StatusNotFound)
		return
	}

	// Ensure it's an OAuth secret
	if secret.Metadata.Type != auth.SecretTypeOAuth {
		http.Error(w, "Secret is not an OAuth secret", http.StatusBadRequest)
		return
	}

	// Mark as used
	s.secretVault.MarkUsed(accountID, key)

	// Parse OAuth data
	var oauth auth.OAuthSecret
	if err := json.Unmarshal([]byte(secret.Value), &oauth); err != nil {
		http.Error(w, "Failed to parse OAuth secret", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"key":      secret.Key,
		"metadata": secret.Metadata,
		"oauth":    oauth,
	})
}

// handleGetAPIKeySecret handles GET /api/v1/accounts/{accountId}/api-key-secrets/{key}
func (s *Server) handleGetAPIKeySecret(w http.ResponseWriter, r *http.Request) {
	accountID, key, ok := s.extractAccountAndKey(w, r)
	if !ok {
		return
	}

	// Get the structured secret
	secret, err := s.secretVault.GetStructured(accountID, key)
	if err != nil {
		http.Error(w, "Secret not found", http.StatusNotFound)
		return
	}

	// Ensure it's an API key secret
	if secret.Metadata.Type != auth.SecretTypeAPIKey {
		http.Error(w, "Secret is not an API key secret", http.StatusBadRequest)
		return
	}

	// Mark as used
	s.secretVault.MarkUsed(accountID, key)

	// Parse API key data
	var apiKey auth.APIKeySecret
	if err := json.Unmarshal([]byte(secret.Value), &apiKey); err != nil {
		http.Error(w, "Failed to parse API key secret", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"key":      secret.Key,
		"metadata": secret.Metadata,
		"api_key":  apiKey,
	})
}

// handleGetDatabaseSecret handles GET /api/v1/accounts/{accountId}/database-secrets/{key}
func (s *Server) handleGetDatabaseSecret(w http.ResponseWriter, r *http.Request) {
	accountID, key, ok := s.extractAccountAndKey(w, r)
	if !ok {
		return
	}

	// Get the structured secret
	secret, err := s.secretVault.GetStructured(accountID, key)
	if err != nil {
		http.Error(w, "Secret not found", http.StatusNotFound)
		return
	}

	// Ensure it's a database secret
	if secret.Metadata.Type != auth.SecretTypeDatabase {
		http.Error(w, "Secret is not a database secret", http.StatusBadRequest)
		return
	}

	// Mark as used
	s.secretVault.MarkUsed(accountID, key)

	// Parse database data
	var database auth.DatabaseSecret
	if err := json.Unmarshal([]byte(secret.Value), &database); err != nil {
		http.Error(w, "Failed to parse database secret", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"key":      secret.Key,
		"metadata": secret.Metadata,
		"database": database,
	})
}

// handleCreateJWTSecret handles POST /api/v1/accounts/{accountId}/jwt-secrets/{key}
func (s *Server) handleCreateJWTSecret(w http.ResponseWriter, r *http.Request) {
	accountID, key, ok := s.extractAccountAndKey(w, r)
	if !ok {
		return
	}

	// Parse JWT secret request
	var req JWTSecretRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// Create JWT secret
	var audience []string
	if req.Audience != "" {
		audience = []string{req.Audience}
	}

	// Convert claims from map[string]string to map[string]interface{}
	claims := make(map[string]interface{})
	for k, v := range req.Claims {
		claims[k] = v
	}

	jwtSecret := auth.JWTSecret{
		Token:     req.Token,
		Algorithm: req.Algorithm,
		Audience:  audience,
		Issuer:    req.Issuer,
		Subject:   req.Subject,
		ExpiresAt: req.TokenExpires,
		IssuedAt:  req.IssuedAt,
		Claims:    claims,
	}

	// Set the JWT secret
	metadata := auth.SecretMetadata{
		Type:        auth.SecretTypeJWT,
		Description: req.Description,
		Tags:        req.Tags,
		ExpiresAt:   req.ExpiresAt,
	}

	if err := s.secretVault.SetJWT(accountID, key, jwtSecret, metadata); err != nil {
		http.Error(w, fmt.Sprintf("Failed to create JWT secret: %v", err), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"key":      key,
		"metadata": metadata,
		"message":  "JWT secret created successfully",
	})
}

// handleGetJWTSecret handles GET /api/v1/accounts/{accountId}/jwt-secrets/{key}
func (s *Server) handleGetJWTSecret(w http.ResponseWriter, r *http.Request) {
	accountID, key, ok := s.extractAccountAndKey(w, r)
	if !ok {
		return
	}

	// Get the structured secret
	secret, err := s.secretVault.GetStructured(accountID, key)
	if err != nil {
		http.Error(w, "Secret not found", http.StatusNotFound)
		return
	}

	// Ensure it's a JWT secret
	if secret.Metadata.Type != auth.SecretTypeJWT {
		http.Error(w, "Secret is not a JWT secret", http.StatusBadRequest)
		return
	}

	// Mark as used
	s.secretVault.MarkUsed(accountID, key)

	// Parse JWT data
	var jwt auth.JWTSecret
	if err := json.Unmarshal([]byte(secret.Value), &jwt); err != nil {
		http.Error(w, "Failed to parse JWT secret", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"key":      secret.Key,
		"metadata": secret.Metadata,
		"jwt":      jwt,
	})
}

// Helper method to convert StructuredSecretRequest to SecretMetadata
func (req StructuredSecretRequest) toMetadata() auth.SecretMetadata {
	return auth.SecretMetadata{
		Type:        req.Type,
		Description: req.Description,
		Tags:        req.Tags,
		ExpiresAt:   req.ExpiresAt,
	}
}

// Helper method to extract account ID and key from request
func (s *Server) extractAccountAndKey(w http.ResponseWriter, r *http.Request) (accountID, key string, ok bool) {
	// Get account ID and key from URL
	vars := mux.Vars(r)
	accountID = vars["accountId"]
	key = vars["key"]

	// Get authenticated account ID from context
	authAccountID, authenticated := middleware.GetAccountID(r)
	if !authenticated {
		http.Error(w, "Authentication required", http.StatusUnauthorized)
		return "", "", false
	}

	// Ensure user can only access their own secrets
	if accountID != authAccountID {
		http.Error(w, "Access denied", http.StatusForbidden)
		return "", "", false
	}

	// Validate input
	if key == "" {
		http.Error(w, "Secret key cannot be empty", http.StatusBadRequest)
		return "", "", false
	}

	return accountID, key, true
}
