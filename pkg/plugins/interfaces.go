// Package plugins provides functionality for loading and managing custom node plugins.
package plugins

import (
	"github.com/tcmartin/flowlib"
)

// PluginRegistry manages custom node plugins
type PluginRegistry interface {
	// Register adds a plugin to the registry
	Register(name string, plugin interface{}) error

	// Get retrieves a plugin by name
	Get(name string) (interface{}, error)

	// List returns all registered plugin names
	List() []string

	// Load loads plugins from a directory
	Load(directory string) error
}

// NodePlugin is the interface that all node plugins must implement
type NodePlugin interface {
	// Name returns the name of the plugin
	Name() string

	// Description returns a description of the plugin
	Description() string

	// Version returns the version of the plugin
	Version() string

	// CreateNode creates a new instance of the node
	CreateNode(params map[string]interface{}) (flowlib.Node, error)
}

// PluginMetadata contains information about a plugin
type PluginMetadata struct {
	// Name of the plugin
	Name string `json:"name"`

	// Description of the plugin
	Description string `json:"description"`

	// Version of the plugin
	Version string `json:"version"`

	// Author of the plugin
	Author string `json:"author"`

	// Parameters that the plugin accepts
	Parameters []PluginParameter `json:"parameters"`
}

// PluginParameter describes a parameter for a plugin
type PluginParameter struct {
	// Name of the parameter
	Name string `json:"name"`

	// Type of the parameter
	Type string `json:"type"`

	// Description of the parameter
	Description string `json:"description"`

	// Required indicates whether the parameter is required
	Required bool `json:"required"`

	// DefaultValue is the default value for the parameter
	DefaultValue interface{} `json:"default_value,omitempty"`
}
