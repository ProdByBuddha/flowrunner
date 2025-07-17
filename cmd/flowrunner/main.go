// Package main is the entry point for the flowrunner application.
package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"

	"github.com/tcmartin/flowrunner/pkg/config"
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

	// Start the application
	if err := app.Start(); err != nil {
		log.Fatalf("Application failed: %v", err)
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

	// Save the default config to the user's home directory
	defaultPath := filepath.Join(os.Getenv("HOME"), ".flowrunner", "config.json")
	if err := config.SaveConfig(cfg, defaultPath); err != nil {
		return nil, fmt.Errorf("failed to save default config: %w", err)
	}

	fmt.Printf("Created default configuration at %s\n", defaultPath)
	return cfg, nil
}

// App represents the flowrunner application
type App struct {
	config *config.Config
	// Add other components here
}

// NewApp creates a new application instance
func NewApp(cfg *config.Config) (*App, error) {
	return &App{
		config: cfg,
	}, nil
}

// Start starts the application
func (a *App) Start() error {
	fmt.Printf("Starting %s version %s\n", AppName, AppVersion)
	fmt.Printf("Listening on %s:%d\n", a.config.Server.Host, a.config.Server.Port)

	// TODO: Initialize components and start the server

	return nil
}
