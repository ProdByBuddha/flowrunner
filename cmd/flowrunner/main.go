// Package main is the entry point for the flowrunner application.
package main

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"path/filepath"
	"strconv"
	"syscall"
	"time"

	"github.com/tcmartin/flowrunner/pkg/api"
	"github.com/tcmartin/flowrunner/pkg/config"
	"github.com/tcmartin/flowrunner/pkg/registry"
	"github.com/tcmartin/flowrunner/pkg/services"
	"github.com/tcmartin/flowrunner/pkg/storage"
)

var (
	// Command-line flags
	configPath = flag.String("config", "", "Path to config file")
	version    = flag.Bool("version", false, "Print version information")
)

// Version information
const (
	AppVersion = "0.1.0"
	AppName    = "flowrunner"
)

func main() {
	// Parse command-line flags
	flag.Parse()

	// Print version information if requested
	if *version {
		fmt.Printf("%s version %s\n", AppName, AppVersion)
		return
	}

	// Load configuration
	cfg, err := loadConfig()
	if err != nil {
		log.Fatalf("Failed to load configuration: %v", err)
	}

	// Initialize the application
	app, err := NewApp(cfg)
	if err != nil {
		log.Fatalf("Failed to initialize application: %v", err)
	}

	// Handle graceful shutdown
	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt, syscall.SIGTERM)

	// Start the application in a goroutine
	errCh := make(chan error)
	go func() {
		errCh <- app.Start()
	}()

	// Wait for interrupt signal or error
	select {
	case err := <-errCh:
		if err != nil {
			log.Fatalf("Application failed: %v", err)
		}
	case <-stop:
		log.Println("Shutting down gracefully...")
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		if err := app.Stop(ctx); err != nil {
			log.Fatalf("Error during shutdown: %v", err)
		}
	}
}

// loadConfig loads the configuration from the specified path or creates a default one
func loadConfig() (*config.Config, error) {
	var cfg *config.Config

	// If a config path is specified, load it
	if *configPath != "" {
		var err error
		cfg, err = config.LoadConfig(*configPath)
		if err != nil {
			return nil, fmt.Errorf("failed to load config from %s: %w", *configPath, err)
		}
	} else {
		// Otherwise, look for a config file in standard locations
		locations := []string{
			"./config.json",
			"./configs/config.json",
			filepath.Join(os.Getenv("HOME"), ".flowrunner", "config.json"),
			"/etc/flowrunner/config.json",
		}

		for _, path := range locations {
			if loadedCfg, err := config.LoadConfig(path); err == nil {
				cfg = loadedCfg
				break
			}
		}

		// If no config file is found, create a default one
		if cfg == nil {
			cfg = config.DefaultConfig()

			// Save the default config to the user's home directory
			defaultPath := filepath.Join(os.Getenv("HOME"), ".flowrunner", "config.json")
			if err := config.SaveConfig(cfg, defaultPath); err != nil {
				return nil, fmt.Errorf("failed to save default config: %w", err)
			}

			fmt.Printf("Created default configuration at %s\n", defaultPath)
		}
	}

	// Override with environment variables if present
	overrideConfigFromEnv(cfg)

	// Generate random JWT secret and encryption key if not set
	if cfg.Auth.JWTSecret == "" {
		secret, err := generateRandomKey(32)
		if err != nil {
			return nil, fmt.Errorf("failed to generate JWT secret: %w", err)
		}
		cfg.Auth.JWTSecret = secret
	}

	if cfg.Auth.EncryptionKey == "" {
		key, err := generateRandomKey(32)
		if err != nil {
			return nil, fmt.Errorf("failed to generate encryption key: %w", err)
		}
		cfg.Auth.EncryptionKey = key
	}

	return cfg, nil
}

// overrideConfigFromEnv overrides configuration values from environment variables
func overrideConfigFromEnv(cfg *config.Config) {
	// Server configuration
	if host := os.Getenv("FLOWRUNNER_SERVER_HOST"); host != "" {
		cfg.Server.Host = host
	}
	if port := os.Getenv("FLOWRUNNER_SERVER_PORT"); port != "" {
		if p, err := strconv.Atoi(port); err == nil {
			cfg.Server.Port = p
		}
	}

	// Storage configuration
	if storageType := os.Getenv("FLOWRUNNER_STORAGE_TYPE"); storageType != "" {
		cfg.Storage.Type = storageType
	}

	// DynamoDB configuration
	if region := os.Getenv("FLOWRUNNER_DYNAMODB_REGION"); region != "" {
		cfg.Storage.DynamoDB.Region = region
	}
	if endpoint := os.Getenv("FLOWRUNNER_DYNAMODB_ENDPOINT"); endpoint != "" {
		cfg.Storage.DynamoDB.Endpoint = endpoint
	}
	if tablePrefix := os.Getenv("FLOWRUNNER_DYNAMODB_TABLE_PREFIX"); tablePrefix != "" {
		cfg.Storage.DynamoDB.TablePrefix = tablePrefix
	}

	// PostgreSQL configuration
	if host := os.Getenv("FLOWRUNNER_POSTGRES_HOST"); host != "" {
		cfg.Storage.Postgres.Host = host
	}
	if port := os.Getenv("FLOWRUNNER_POSTGRES_PORT"); port != "" {
		if p, err := strconv.Atoi(port); err == nil {
			cfg.Storage.Postgres.Port = p
		}
	}
	if database := os.Getenv("FLOWRUNNER_POSTGRES_DATABASE"); database != "" {
		cfg.Storage.Postgres.Database = database
	}
	if user := os.Getenv("FLOWRUNNER_POSTGRES_USER"); user != "" {
		cfg.Storage.Postgres.User = user
	}
	if password := os.Getenv("FLOWRUNNER_POSTGRES_PASSWORD"); password != "" {
		cfg.Storage.Postgres.Password = password
	}
	if sslMode := os.Getenv("FLOWRUNNER_POSTGRES_SSL_MODE"); sslMode != "" {
		cfg.Storage.Postgres.SSLMode = sslMode
	}

	// Auth configuration
	if jwtSecret := os.Getenv("FLOWRUNNER_JWT_SECRET"); jwtSecret != "" {
		cfg.Auth.JWTSecret = jwtSecret
	}
	if tokenExpiration := os.Getenv("FLOWRUNNER_TOKEN_EXPIRATION"); tokenExpiration != "" {
		if exp, err := strconv.Atoi(tokenExpiration); err == nil {
			cfg.Auth.TokenExpiration = exp
		}
	}
	if encryptionKey := os.Getenv("FLOWRUNNER_ENCRYPTION_KEY"); encryptionKey != "" {
		cfg.Auth.EncryptionKey = encryptionKey
	}
}

// generateRandomKey generates a random key of the specified length
func generateRandomKey(length int) (string, error) {
	bytes := make([]byte, length)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	return hex.EncodeToString(bytes), nil
}

// App represents the flowrunner application
type App struct {
	config          *config.Config
	server          *api.Server
	storageProvider storage.StorageProvider
}

// NewApp creates a new application instance
func NewApp(cfg *config.Config) (*App, error) {
	// Initialize storage provider
	var storageProvider storage.StorageProvider
	var err error

	switch cfg.Storage.Type {
	case "memory":
		storageProvider = storage.NewMemoryProvider()
		log.Println("Using in-memory storage provider")
	case "dynamodb":
		log.Printf("Initializing DynamoDB storage provider with region: %s, endpoint: %s",
			cfg.Storage.DynamoDB.Region, cfg.Storage.DynamoDB.Endpoint)
		// Create a DynamoDB provider with the configuration
		// For now, we'll use the memory provider as a fallback
		storageProvider = storage.NewMemoryProvider()
		log.Println("Note: Using in-memory storage as fallback (DynamoDB implementation pending)")
	case "postgres":
		log.Printf("Initializing PostgreSQL storage provider with host: %s, port: %d, database: %s",
			cfg.Storage.Postgres.Host, cfg.Storage.Postgres.Port, cfg.Storage.Postgres.Database)
		// Create a PostgreSQL provider with the configuration
		// For now, we'll use the memory provider as a fallback
		storageProvider = storage.NewMemoryProvider()
		log.Println("Note: Using in-memory storage as fallback (PostgreSQL implementation pending)")
	default:
		return nil, fmt.Errorf("unsupported storage type: %s", cfg.Storage.Type)
	}

	if err != nil {
		return nil, fmt.Errorf("failed to initialize storage provider: %w", err)
	}

	// Initialize storage
	if err := storageProvider.Initialize(); err != nil {
		return nil, fmt.Errorf("failed to initialize storage: %w", err)
	}

	// We'll skip the YAML loader for now

	// Create flow registry
	flowRegistry := registry.NewFlowRegistry(storageProvider.GetFlowStore(), registry.FlowRegistryOptions{})

	// Create account service with JWT support
	accountService := services.NewAccountService(storageProvider.GetAccountStore())

	// Add JWT support if configured
	if cfg.Auth.JWTSecret != "" {
		accountService = accountService.WithJWTService(cfg.Auth.JWTSecret, cfg.Auth.TokenExpiration)
	}

	// Create API server
	server := api.NewServer(cfg, flowRegistry, accountService)

	return &App{
		config:          cfg,
		server:          server,
		storageProvider: storageProvider,
	}, nil
}

// Start starts the application
func (a *App) Start() error {
	fmt.Printf("Starting %s version %s\n", AppName, AppVersion)
	return a.server.Start()
}

// Stop stops the application gracefully
func (a *App) Stop(ctx context.Context) error {
	// Stop the server
	if err := a.server.Stop(ctx); err != nil {
		return err
	}

	// Close storage
	if err := a.storageProvider.Close(); err != nil {
		return fmt.Errorf("failed to close storage: %w", err)
	}

	return nil
}
