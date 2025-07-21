package api

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/tcmartin/flowlib"
	"github.com/tcmartin/flowrunner/pkg/config"
	"github.com/tcmartin/flowrunner/pkg/plugins"
	"github.com/tcmartin/flowrunner/pkg/registry"
	"github.com/tcmartin/flowrunner/pkg/services"
	"github.com/tcmartin/flowrunner/pkg/storage"
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
	return flowlib.NewNode(0, 0), nil
}

func TestPluginHandlers(t *testing.T) {
	// Setup test server with a plugin registry
	cfg := &config.Config{
		Server: config.ServerConfig{
			Host: "localhost",
			Port: 8080,
		},
	}

	storageProvider := storage.NewMemoryProvider()
	require.NoError(t, storageProvider.Initialize())

	accountService := services.NewAccountService(storageProvider.GetAccountStore())
	encKey, err := services.GenerateEncryptionKey()
	require.NoError(t, err)
	secretVault, err := services.NewExtendedSecretVaultService(storageProvider.GetSecretStore(), encKey)
	require.NoError(t, err)

	flowRegistry := registry.NewFlowRegistry(storageProvider.GetFlowStore(), registry.FlowRegistryOptions{})

	pluginRegistry := plugins.NewPluginRegistry()

	// Register a dummy plugin for testing
	dummyPlugin := &MockNodePlugin{
		name:        "dummy_plugin",
		description: "A dummy plugin for testing",
		version:     "1.0.0",
	}
	err = pluginRegistry.Register(dummyPlugin.Name(), dummyPlugin)
	require.NoError(t, err)

	server := NewServer(cfg, flowRegistry, accountService, secretVault, pluginRegistry)
	testServer := httptest.NewServer(server.router)
	defer testServer.Close()

	// Create a test account and get auth header
	_, authHeader := createTestAccountAndAuth(t, server)

	t.Run("list_plugins", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, testServer.URL+"/api/v1/plugins", nil)
		req.Header.Set("Authorization", authHeader)

		w := httptest.NewRecorder()
		server.router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)

		var response []plugins.PluginMetadata
		err := json.Unmarshal(w.Body.Bytes(), &response)
		require.NoError(t, err)

		assert.Len(t, response, 1)
		assert.Equal(t, dummyPlugin.Name(), response[0].Name)
		assert.Equal(t, dummyPlugin.Description(), response[0].Description)
		assert.Equal(t, dummyPlugin.Version(), response[0].Version)
	})

	t.Run("get_plugin_by_name", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, testServer.URL+"/api/v1/plugins/dummy_plugin", nil)
		req.Header.Set("Authorization", authHeader)

		w := httptest.NewRecorder()
		server.router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)

		var response plugins.PluginMetadata
		err := json.Unmarshal(w.Body.Bytes(), &response)
		require.NoError(t, err)

		assert.Equal(t, dummyPlugin.Name(), response.Name)
		assert.Equal(t, dummyPlugin.Description(), response.Description)
		assert.Equal(t, dummyPlugin.Version(), response.Version)
	})

	t.Run("get_non_existent_plugin", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, testServer.URL+"/api/v1/plugins/non_existent", nil)
		req.Header.Set("Authorization", authHeader)

		w := httptest.NewRecorder()
		server.router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusNotFound, w.Code)
	})

	t.Run("list_plugins_unauthenticated", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, testServer.URL+"/api/v1/plugins", nil)

		w := httptest.NewRecorder()
		server.router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusUnauthorized, w.Code)
	})
}

