package plugins

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/tcmartin/flowlib"
)

// MockNodePlugin is a mock implementation of the NodePlugin interface for testing.
type MockNodePlugin struct {
	name        string
	description string
	version     string
}

func (m *MockNodePlugin) Name() string {
	return m.name
}

func (m *MockNodePlugin) Description() string {
	return m.description
}

func (m *MockNodePlugin) Version() string {
	return m.version
}

func (m *MockNodePlugin) CreateNode(params map[string]interface{}) (flowlib.Node, error) {
	// For testing purposes, we can return a simple node or nil.
	return flowlib.NewNode(0, 0), nil
}

func TestPluginRegistry(t *testing.T) {
	registry := NewPluginRegistry()

	t.Run("Register and Get", func(t *testing.T) {
		plugin := &MockNodePlugin{name: "test_plugin"}
		err := registry.Register("test_plugin", plugin)
		assert.NoError(t, err)

		retrieved, err := registry.Get("test_plugin")
		assert.NoError(t, err)
		assert.Equal(t, plugin, retrieved)
	})

	t.Run("Register duplicate", func(t *testing.T) {
		plugin := &MockNodePlugin{name: "test_plugin_2"}
		err := registry.Register("test_plugin_2", plugin)
		assert.NoError(t, err)

		err = registry.Register("test_plugin_2", plugin)
		assert.Error(t, err)
	})

	t.Run("Get non-existent", func(t *testing.T) {
		_, err := registry.Get("non_existent_plugin")
		assert.Error(t, err)
	})

	t.Run("List plugins", func(t *testing.T) {
		registry := NewPluginRegistry() // New registry for clean test
		assert.Empty(t, registry.List())

		registry.Register("plugin1", &MockNodePlugin{name: "plugin1"})
		registry.Register("plugin2", &MockNodePlugin{name: "plugin2"})

		list := registry.List()
		assert.Len(t, list, 2)
		assert.Contains(t, list, "plugin1")
		assert.Contains(t, list, "plugin2")
	})
}
