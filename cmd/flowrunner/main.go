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
	// If a config path is specified, load it
	if *configPath != "" {
		return config.LoadConfig(*configPath)
	}

	// Otherwise, look for a config file in standard locations
	locations := []string{
		"./config.json",
		"./configs/config.json",
		filepath.Join(os.Getenv("HOME"), ".flowrunner", "config.json"),
		"/etc/flowrunner/config.json",
	}

	for _, path := range locations {
		if cfg, err := config.LoadConfig(path); err == nil {
			return cfg, nil
		}
	}

	// If no config file is found, create a default one
	cfg := config.DefaultConfig()

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

	// Save the default config to the user's home directory
	defaultPath := filepath.Join(os.Getenv("HOME"), ".flowrunner", "config.json")
	if err := config.SaveConfig(cfg, defaultPath); err != nil {
		return nil, fmt.Errorf("failed to save default config: %w", err)
	}

	fmt.Printf("Created default configuration at %s\n", defaultPath)
	return cfg, nil
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
	case "dynamodb":
		// For now, we'll use the memory provider for DynamoDB
		storageProvider = storage.NewMemoryProvider()
	case "postgres":
		// For now, we'll use the memory provider for PostgreSQL
		storageProvider = storage.NewMemoryProvider()
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
