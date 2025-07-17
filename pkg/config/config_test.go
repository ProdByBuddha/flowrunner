package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestDefaultConfig(t *testing.T) {
	cfg := DefaultConfig()

	// Check default values
	if cfg.Server.Host != "localhost" {
		t.Errorf("Expected default host to be 'localhost', got '%s'", cfg.Server.Host)
	}

	if cfg.Server.Port != 8080 {
		t.Errorf("Expected default port to be 8080, got %d", cfg.Server.Port)
	}

	if cfg.Storage.Type != "memory" {
		t.Errorf("Expected default storage type to be 'memory', got '%s'", cfg.Storage.Type)
	}
}

func TestSaveAndLoadConfig(t *testing.T) {
	// Create a temporary directory for the test
	tempDir, err := os.MkdirTemp("", "flowrunner-config-test")
	if err != nil {
		t.Fatalf("Failed to create temp directory: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Create a config file path
	configPath := filepath.Join(tempDir, "config.json")

	// Create a test config
	originalCfg := DefaultConfig()
	originalCfg.Server.Host = "testhost"
	originalCfg.Server.Port = 9090
	originalCfg.Storage.Type = "postgres"

	// Save the config
	if err := SaveConfig(originalCfg, configPath); err != nil {
		t.Fatalf("Failed to save config: %v", err)
	}

	// Load the config
	loadedCfg, err := LoadConfig(configPath)
	if err != nil {
		t.Fatalf("Failed to load config: %v", err)
	}

	// Check that the loaded config matches the original
	if loadedCfg.Server.Host != originalCfg.Server.Host {
		t.Errorf("Expected host to be '%s', got '%s'", originalCfg.Server.Host, loadedCfg.Server.Host)
	}

	if loadedCfg.Server.Port != originalCfg.Server.Port {
		t.Errorf("Expected port to be %d, got %d", originalCfg.Server.Port, loadedCfg.Server.Port)
	}

	if loadedCfg.Storage.Type != originalCfg.Storage.Type {
		t.Errorf("Expected storage type to be '%s', got '%s'", originalCfg.Storage.Type, loadedCfg.Storage.Type)
	}
}

func TestLoadConfigError(t *testing.T) {
	// Try to load a non-existent config file
	_, err := LoadConfig("non-existent-file.json")
	if err == nil {
		t.Error("Expected error when loading non-existent config file, got nil")
	}
}
