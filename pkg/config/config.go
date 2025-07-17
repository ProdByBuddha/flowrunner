// Package config provides configuration handling for flowrunner.
package config

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

// Config represents the application configuration
type Config struct {
	// Server configuration
	Server ServerConfig `json:"server"`

	// Storage configuration
	Storage StorageConfig `json:"storage"`

	// Auth configuration
	Auth AuthConfig `json:"auth"`

	// Plugins configuration
	Plugins PluginsConfig `json:"plugins"`

	// Logging configuration
	Logging LoggingConfig `json:"logging"`
}

// ServerConfig contains HTTP server settings
type ServerConfig struct {
	// Host to bind to
	Host string `json:"host"`

	// Port to listen on
	Port int `json:"port"`

	// TLS configuration
	TLS TLSConfig `json:"tls"`
}

// TLSConfig contains TLS settings
type TLSConfig struct {
	// Enabled indicates whether TLS is enabled
	Enabled bool `json:"enabled"`

	// CertFile is the path to the certificate file
	CertFile string `json:"cert_file"`

	// KeyFile is the path to the key file
	KeyFile string `json:"key_file"`
}

// StorageConfig contains storage settings
type StorageConfig struct {
	// Type of storage to use
	Type string `json:"type"` // "memory", "dynamodb", "postgres"

	// DynamoDB configuration
	DynamoDB DynamoDBConfig `json:"dynamodb"`

	// PostgreSQL configuration
	Postgres PostgresConfig `json:"postgres"`
}

// DynamoDBConfig contains DynamoDB settings
type DynamoDBConfig struct {
	// Region is the AWS region
	Region string `json:"region"`

	// Endpoint is the DynamoDB endpoint (for local development)
	Endpoint string `json:"endpoint"`

	// TablePrefix is the prefix for all tables
	TablePrefix string `json:"table_prefix"`
}

// PostgresConfig contains PostgreSQL settings
type PostgresConfig struct {
	// Host is the database host
	Host string `json:"host"`

	// Port is the database port
	Port int `json:"port"`

	// Database is the database name
	Database string `json:"database"`

	// User is the database user
	User string `json:"user"`

	// Password is the database password
	Password string `json:"password"`

	// SSLMode is the SSL mode
	SSLMode string `json:"ssl_mode"`
}

// AuthConfig contains authentication settings
type AuthConfig struct {
	// JWTSecret is the secret for signing JWT tokens
	JWTSecret string `json:"jwt_secret"`

	// TokenExpiration is the token expiration time in hours
	TokenExpiration int `json:"token_expiration"`

	// EncryptionKey is the key for encrypting secrets
	EncryptionKey string `json:"encryption_key"`
}

// PluginsConfig contains plugin settings
type PluginsConfig struct {
	// Directory is the path to the plugins directory
	Directory string `json:"directory"`

	// AutoLoad indicates whether to load plugins on startup
	AutoLoad bool `json:"auto_load"`
}

// LoggingConfig contains logging settings
type LoggingConfig struct {
	// Level is the logging level
	Level string `json:"level"` // "debug", "info", "warn", "error"

	// Format is the log format
	Format string `json:"format"` // "json", "text"

	// Output is the log output
	Output string `json:"output"` // "stdout", "file"

	// FilePath is the path to the log file
	FilePath string `json:"file_path"`
}

// LoadConfig loads the configuration from a file
func LoadConfig(path string) (*Config, error) {
	// Read the file
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	// Parse the JSON
	var config Config
	if err := json.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("failed to parse config file: %w", err)
	}

	return &config, nil
}

// DefaultConfig returns the default configuration
func DefaultConfig() *Config {
	return &Config{
		Server: ServerConfig{
			Host: "localhost",
			Port: 8080,
			TLS: TLSConfig{
				Enabled: false,
			},
		},
		Storage: StorageConfig{
			Type: "memory",
			DynamoDB: DynamoDBConfig{
				Region:      "us-west-2",
				TablePrefix: "flowrunner_",
			},
			Postgres: PostgresConfig{
				Host:     "localhost",
				Port:     5432,
				Database: "flowrunner",
				User:     "flowrunner",
				SSLMode:  "disable",
			},
		},
		Auth: AuthConfig{
			TokenExpiration: 24,
		},
		Plugins: PluginsConfig{
			Directory: "./plugins",
			AutoLoad:  true,
		},
		Logging: LoggingConfig{
			Level:  "info",
			Format: "json",
			Output: "stdout",
		},
	}
}

// SaveConfig saves the configuration to a file
func SaveConfig(config *Config, path string) error {
	// Create the directory if it doesn't exist
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create config directory: %w", err)
	}

	// Marshal the JSON
	data, err := json.MarshalIndent(config, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal config: %w", err)
	}

	// Write the file
	if err := os.WriteFile(path, data, 0644); err != nil {
		return fmt.Errorf("failed to write config file: %w", err)
	}

	return nil
}
