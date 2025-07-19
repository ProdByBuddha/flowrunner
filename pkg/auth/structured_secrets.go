package auth

import (
	"encoding/json"
	"time"
)

// SecretType represents the type of secret for better organization and validation
type SecretType string

const (
	SecretTypeGeneral    SecretType = "general"    // Simple key-value secret
	SecretTypeOAuth      SecretType = "oauth"      // OAuth credentials
	SecretTypeAPIKey     SecretType = "api_key"    // API key with optional headers
	SecretTypeDatabase   SecretType = "database"   // Database connection info
	SecretTypeJWT        SecretType = "jwt"        // JWT token with metadata
	SecretTypeCustom     SecretType = "custom"     // Arbitrary structured data
)

// SecretMetadata contains metadata about a secret
type SecretMetadata struct {
	// Type categorizes the secret for better organization
	Type SecretType `json:"type"`
	
	// Description provides human-readable information about the secret
	Description string `json:"description,omitempty"`
	
	// Tags for categorization and searching
	Tags []string `json:"tags,omitempty"`
	
	// ExpiresAt indicates when the secret expires (optional)
	ExpiresAt *time.Time `json:"expires_at,omitempty"`
	
	// LastUsed tracks when the secret was last accessed
	LastUsed *time.Time `json:"last_used,omitempty"`
	
	// Version for secret versioning
	Version int `json:"version"`
}

// StructuredSecret represents a complex secret with metadata and structured data
type StructuredSecret struct {
	// Basic secret fields
	AccountID string    `json:"-"`
	Key       string    `json:"key"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
	
	// Metadata about the secret
	Metadata SecretMetadata `json:"metadata"`
	
	// Encrypted value (JSON structure encrypted as string)
	Value string `json:"-"`
	
	// Schema defines the expected structure (optional, for validation)
	Schema *SecretSchema `json:"schema,omitempty"`
}

// SecretSchema defines the expected structure of a secret's data
type SecretSchema struct {
	// Fields defines the expected fields and their types
	Fields map[string]FieldDefinition `json:"fields"`
	
	// Required lists the required field names
	Required []string `json:"required,omitempty"`
}

// FieldDefinition defines a field in a secret schema
type FieldDefinition struct {
	// Type of the field (string, number, boolean, object, array)
	Type string `json:"type"`
	
	// Description of the field
	Description string `json:"description,omitempty"`
	
	// Sensitive indicates if this field should be masked in logs/UI
	Sensitive bool `json:"sensitive,omitempty"`
	
	// Pattern for string validation (regex)
	Pattern string `json:"pattern,omitempty"`
}

// OAuthSecret represents OAuth credentials
type OAuthSecret struct {
	ClientID     string            `json:"client_id"`
	ClientSecret string            `json:"client_secret"`
	TokenURL     string            `json:"token_url,omitempty"`
	AuthURL      string            `json:"auth_url,omitempty"`
	RedirectURL  string            `json:"redirect_url,omitempty"`
	Scopes       []string          `json:"scopes,omitempty"`
	AccessToken  string            `json:"access_token,omitempty"`
	RefreshToken string            `json:"refresh_token,omitempty"`
	ExpiresAt    *time.Time        `json:"expires_at,omitempty"`
	Extra        map[string]string `json:"extra,omitempty"`
}

// APIKeySecret represents an API key with optional headers and configuration
type APIKeySecret struct {
	Key         string            `json:"key"`
	HeaderName  string            `json:"header_name,omitempty"`  // e.g., "Authorization", "X-API-Key"
	Prefix      string            `json:"prefix,omitempty"`       // e.g., "Bearer ", "ApiKey "
	Headers     map[string]string `json:"headers,omitempty"`      // Additional headers
	QueryParam  string            `json:"query_param,omitempty"`  // If passed as query parameter
	BaseURL     string            `json:"base_url,omitempty"`     // Associated API base URL
	RateLimit   *RateLimit        `json:"rate_limit,omitempty"`   // Rate limiting info
}

// RateLimit represents rate limiting information for an API
type RateLimit struct {
	RequestsPerSecond int    `json:"requests_per_second,omitempty"`
	RequestsPerHour   int    `json:"requests_per_hour,omitempty"`
	RequestsPerDay    int    `json:"requests_per_day,omitempty"`
	BurstLimit        int    `json:"burst_limit,omitempty"`
	ResetTime         string `json:"reset_time,omitempty"` // e.g., "daily", "hourly"
}

// DatabaseSecret represents database connection information
type DatabaseSecret struct {
	Type         string            `json:"type"`          // postgres, mysql, mongodb, etc.
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

// JWTSecret represents JWT token information
type JWTSecret struct {
	Token      string            `json:"token"`
	Algorithm  string            `json:"algorithm,omitempty"`  // HS256, RS256, etc.
	Issuer     string            `json:"issuer,omitempty"`
	Subject    string            `json:"subject,omitempty"`
	Audience   []string          `json:"audience,omitempty"`
	ExpiresAt  *time.Time        `json:"expires_at,omitempty"`
	IssuedAt   *time.Time        `json:"issued_at,omitempty"`
	NotBefore  *time.Time        `json:"not_before,omitempty"`
	KeyID      string            `json:"key_id,omitempty"`
	Claims     map[string]interface{} `json:"claims,omitempty"`
}

// SecretReference represents a reference to a secret for use in YAML flows
type SecretReference struct {
	// AccountID is implied from the execution context
	SecretKey string `json:"secret_key" yaml:"secret_key"`
	
	// Field specifies which field to extract from structured secrets
	Field string `json:"field,omitempty" yaml:"field,omitempty"`
	
	// Default value if secret or field is not found
	Default string `json:"default,omitempty" yaml:"default,omitempty"`
	
	// Required indicates if this secret reference must resolve
	Required bool `json:"required,omitempty" yaml:"required,omitempty"`
}

// ExtendedSecretVault extends the basic SecretVault interface for structured secrets
type ExtendedSecretVault interface {
	SecretVault
	
	// SetStructured stores a structured secret
	SetStructured(accountID string, secret StructuredSecret) error
	
	// GetStructured retrieves a structured secret
	GetStructured(accountID string, key string) (StructuredSecret, error)
	
	// GetField retrieves a specific field from a structured secret
	GetField(accountID string, key string, field string) (interface{}, error)
	
	// ListByType returns secrets filtered by type
	ListByType(accountID string, secretType SecretType) ([]StructuredSecret, error)
	
	// ListByTags returns secrets filtered by tags
	ListByTags(accountID string, tags []string) ([]StructuredSecret, error)
	
	// Search searches secrets by metadata
	Search(accountID string, query SecretQuery) ([]StructuredSecret, error)
	
	// UpdateMetadata updates only the metadata of a secret
	UpdateMetadata(accountID string, key string, metadata SecretMetadata) error
	
	// SetOAuth stores OAuth credentials
	SetOAuth(accountID string, key string, oauth OAuthSecret, metadata SecretMetadata) error
	
	// SetAPIKey stores API key credentials
	SetAPIKey(accountID string, key string, apiKey APIKeySecret, metadata SecretMetadata) error
	
	// SetDatabase stores database credentials
	SetDatabase(accountID string, key string, db DatabaseSecret, metadata SecretMetadata) error
	
	// SetJWT stores JWT token
	SetJWT(accountID string, key string, jwt JWTSecret, metadata SecretMetadata) error
	
	// SetCustom stores arbitrary structured data
	SetCustom(accountID string, key string, data map[string]interface{}, metadata SecretMetadata) error
	
	// MarkUsed updates the last used timestamp
	MarkUsed(accountID string, key string) error
	
	// GetExpiring returns secrets that expire within the given duration
	GetExpiring(accountID string, within time.Duration) ([]StructuredSecret, error)
}

// SecretQuery represents search criteria for secrets
type SecretQuery struct {
	// Type filters by secret type
	Type SecretType `json:"type,omitempty"`
	
	// Tags filters by tags (AND operation)
	Tags []string `json:"tags,omitempty"`
	
	// Description searches in description
	Description string `json:"description,omitempty"`
	
	// ExpiringWithin finds secrets expiring within duration
	ExpiringWithin *time.Duration `json:"expiring_within,omitempty"`
	
	// LastUsedBefore finds secrets not used since given time
	LastUsedBefore *time.Time `json:"last_used_before,omitempty"`
	
	// Limit limits the number of results
	Limit int `json:"limit,omitempty"`
	
	// Offset for pagination
	Offset int `json:"offset,omitempty"`
}

// GetOAuthSecretSchema returns the schema for OAuth secrets
func GetOAuthSecretSchema() SecretSchema {
	return SecretSchema{
		Fields: map[string]FieldDefinition{
			"client_id":     {Type: "string", Description: "OAuth Client ID", Sensitive: false},
			"client_secret": {Type: "string", Description: "OAuth Client Secret", Sensitive: true},
			"token_url":     {Type: "string", Description: "Token endpoint URL", Sensitive: false},
			"auth_url":      {Type: "string", Description: "Authorization endpoint URL", Sensitive: false},
			"redirect_url":  {Type: "string", Description: "Redirect URL", Sensitive: false},
			"scopes":        {Type: "array", Description: "OAuth scopes", Sensitive: false},
			"access_token":  {Type: "string", Description: "Access token", Sensitive: true},
			"refresh_token": {Type: "string", Description: "Refresh token", Sensitive: true},
		},
		Required: []string{"client_id", "client_secret"},
	}
}

// GetAPIKeySecretSchema returns the schema for API key secrets
func GetAPIKeySecretSchema() SecretSchema {
	return SecretSchema{
		Fields: map[string]FieldDefinition{
			"key":         {Type: "string", Description: "API Key", Sensitive: true},
			"header_name": {Type: "string", Description: "Header name for the API key", Sensitive: false},
			"prefix":      {Type: "string", Description: "Prefix for the API key", Sensitive: false},
			"headers":     {Type: "object", Description: "Additional headers", Sensitive: false},
			"query_param": {Type: "string", Description: "Query parameter name", Sensitive: false},
			"base_url":    {Type: "string", Description: "API base URL", Sensitive: false},
		},
		Required: []string{"key"},
	}
}

// GetDatabaseSecretSchema returns the schema for database secrets
func GetDatabaseSecretSchema() SecretSchema {
	return SecretSchema{
		Fields: map[string]FieldDefinition{
			"type":     {Type: "string", Description: "Database type", Sensitive: false},
			"host":     {Type: "string", Description: "Database host", Sensitive: false},
			"port":     {Type: "number", Description: "Database port", Sensitive: false},
			"database": {Type: "string", Description: "Database name", Sensitive: false},
			"username": {Type: "string", Description: "Database username", Sensitive: false},
			"password": {Type: "string", Description: "Database password", Sensitive: true},
		},
		Required: []string{"type", "host", "username", "password"},
	}
}

// ToJSON converts any secret data structure to JSON
func ToJSON(data interface{}) (string, error) {
	bytes, err := json.Marshal(data)
	if err != nil {
		return "", err
	}
	return string(bytes), nil
}

// FromJSON converts JSON string to a data structure
func FromJSON(jsonStr string, target interface{}) error {
	return json.Unmarshal([]byte(jsonStr), target)
}
