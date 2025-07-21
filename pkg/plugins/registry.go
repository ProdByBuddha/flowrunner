package plugins

import (
	"fmt"
	"sync"
)

// pluginRegistry implements the PluginRegistry interface.
type pluginRegistry struct {
	plugins map[string]NodePlugin
	mu      sync.RWMutex
}

// NewPluginRegistry creates a new PluginRegistry.
func NewPluginRegistry() PluginRegistry {
	return &pluginRegistry{
		plugins: make(map[string]NodePlugin),
	}
}

// Register adds a plugin to the registry.
func (r *pluginRegistry) Register(name string, plugin interface{}) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if _, exists := r.plugins[name]; exists {
		return fmt.Errorf("plugin with name '%s' already registered", name)
	}

	nodePlugin, ok := plugin.(NodePlugin)
	if !ok {
		return fmt.Errorf("plugin does not implement NodePlugin interface")
	}

	r.plugins[name] = nodePlugin
	return nil
}

// Get retrieves a plugin by name.
func (r *pluginRegistry) Get(name string) (interface{}, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	plugin, exists := r.plugins[name]
	if !exists {
		return nil, fmt.Errorf("plugin with name '%s' not found", name)
	}

	return plugin, nil
}

// List returns all registered plugin names.
func (r *pluginRegistry) List() []string {
	r.mu.RLock()
	defer r.mu.RUnlock()

	names := make([]string, 0, len(r.plugins))
	for name := range r.plugins {
		names = append(names, name)
	}

	return names
}

// Load loads plugins from a directory.
// This is a placeholder for now.
func (r *pluginRegistry) Load(directory string) error {
	// TODO: Implement dynamic plugin loading from shared libraries.
	return nil
}
