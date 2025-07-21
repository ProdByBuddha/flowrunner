package api

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gorilla/websocket"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/tcmartin/flowrunner/pkg/config"
	"github.com/tcmartin/flowrunner/pkg/registry"
	"github.com/tcmartin/flowrunner/pkg/runtime"
	"github.com/tcmartin/flowrunner/pkg/services"
	"github.com/tcmartin/flowrunner/pkg/storage"
)

func TestWebSocketIntegration_EndToEnd(t *testing.T) {
	// Create storage provider
	memoryProvider, err := storage.NewProvider(storage.ProviderConfig{
		Type: storage.MemoryProviderType,
	})
	require.NoError(t, err)
	
	// Create account service
	accountService := services.NewAccountService(memoryProvider.AccountStore())
	
	// Create secret vault
	encKey, err := services.GenerateEncryptionKey()
	require.NoError(t, err)
	secretVault := services.NewSecretVaultService(memoryProvider.SecretStore(), encKey)
	
	// Create flow registry
	flowRegistry := registry.NewFlowRegistry(memoryProvider.FlowStore(), registry.FlowRegistryOptions{})
	
	// Create test account
	accountID, err := accountService.CreateAccount("testuser", "testpass")
	require.NoError(t, err)
	
	// Create a simple test flow
	testFlowYAML := `
metadata:
  name: "Test Flow"
  version: "1.0.0"
  description: "A simple test flow for WebSocket integration"

nodes:
  start:
    type: "wait"
    parameters:
      duration: "100ms"
    
edges:
  - from: "start"
    to: "end"
`
	
	// Register the flow
	flowID, err := flowRegistry.Create(accountID, "test-flow", testFlowYAML)
	require.NoError(t, err)
	
	// Create flow runtime with execution store
	executionStore := memoryProvider.ExecutionStore()
	flowRuntime := runtime.NewFlowRuntimeWithStore(flowRegistry, nil, executionStore)
	
	// Create server with runtime
	cfg := &config.Config{
		Server: config.ServerConfig{
			Host: "localhost",
			Port: 8080,
		},
	}
	
	server := NewServerWithRuntime(cfg, flowRegistry, accountService, secretVault, flowRuntime)
	
	// Create test server
	testServer := httptest.NewServer(server.router)
	defer testServer.Close()
	
	// Create WebSocket connection with basic auth
	wsURL := "ws" + strings.TrimPrefix(testServer.URL, "http") + "/api/v1/ws"
	header := make(http.Header)
	header.Set("Authorization", "Basic dGVzdHVzZXI6dGVzdHBhc3M=") // testuser:testpass in base64
	
	ws, resp, err := websocket.DefaultDialer.Dial(wsURL, header)
	require.NoError(t, err, "Response status: %d", resp.StatusCode)
	defer ws.Close()
	
	// Start a flow execution via HTTP API
	execURL := testServer.URL + "/api/v1/flows/" + flowID + "/run"
	req, err := http.NewRequest("POST", execURL, strings.NewReader(`{"input":{"test":"value"}}`))
	require.NoError(t, err)
	
	req.SetBasicAuth("testuser", "testpass")
	req.Header.Set("Content-Type", "application/json")
	
	client := &http.Client{}
	execResp, err := client.Do(req)
	require.NoError(t, err)
	defer execResp.Body.Close()
	
	// Parse execution response
	var execResult map[string]interface{}
	err = json.NewDecoder(execResp.Body).Decode(&execResult)
	require.NoError(t, err)
	
	executionID := execResult["execution_id"].(string)
	require.NotEmpty(t, executionID)
	
	// Subscribe to execution updates via WebSocket
	subscribeMsg := WebSocketMessage{
		Type:        "subscribe",
		ExecutionID: executionID,
	}
	
	err = ws.WriteJSON(subscribeMsg)
	require.NoError(t, err)
	
	// Collect WebSocket updates
	updates := []ExecutionUpdate{}
	
	// Set read deadline to avoid hanging
	ws.SetReadDeadline(time.Now().Add(10 * time.Second))
	
	// Read updates until we get a completion event or timeout
	for {
		var update ExecutionUpdate
		err := ws.ReadJSON(&update)
		if err != nil {
			break // Timeout or connection closed
		}
		
		updates = append(updates, update)
		
		// Stop when we get a completion event
		if update.Type == "complete" || update.Type == "status" && 
		   update.Status != nil && (update.Status.Status == "completed" || update.Status.Status == "failed") {
			break
		}
	}
	
	// Verify we received updates
	assert.NotEmpty(t, updates, "Should have received WebSocket updates")
	
	// Should have received at least a status update
	hasStatus := false
	for _, update := range updates {
		if update.Type == "status" {
			hasStatus = true
			assert.Equal(t, executionID, update.ExecutionID)
			assert.NotNil(t, update.Status)
		}
	}
	assert.True(t, hasStatus, "Should have received at least one status update")
	
	// Verify final execution status via HTTP API
	statusURL := testServer.URL + "/api/v1/executions/" + executionID
	statusReq, err := http.NewRequest("GET", statusURL, nil)
	require.NoError(t, err)
	statusReq.SetBasicAuth("testuser", "testpass")
	
	statusResp, err := client.Do(statusReq)
	require.NoError(t, err)
	defer statusResp.Body.Close()
	
	var finalStatus runtime.ExecutionStatus
	err = json.NewDecoder(statusResp.Body).Decode(&finalStatus)
	require.NoError(t, err)
	
	// The execution should be completed (the wait node should finish quickly)
	assert.Contains(t, []string{"completed", "running"}, finalStatus.Status)
	assert.Equal(t, executionID, finalStatus.ID)
	assert.Equal(t, flowID, finalStatus.FlowID)
}

func TestWebSocketAuthentication(t *testing.T) {
	// Create minimal services for auth testing
	memoryProvider := storage.NewProvider("memory", "", nil)
	accountService, err := services.NewAccountService(memoryProvider)
	require.NoError(t, err)
	
	secretVault, err := services.NewSecretVaultService(memoryProvider, "0123456789abcdef0123456789abcdef")
	require.NoError(t, err)
	
	flowRegistry := registry.NewFlowRegistry(memoryProvider, registry.FlowRegistryOptions{})
	
	// Create test account
	_, err = accountService.CreateAccount("testuser", "testpass")
	require.NoError(t, err)
	
	cfg := &config.Config{
		Server: config.ServerConfig{
			Host: "localhost",
			Port: 8080,
		},
	}
	
	server := NewServerWithRuntime(cfg, flowRegistry, accountService, secretVault, nil)
	testServer := httptest.NewServer(server.router)
	defer testServer.Close()
	
	wsURL := "ws" + strings.TrimPrefix(testServer.URL, "http") + "/api/v1/ws"
	
	t.Run("WebSocket connection with valid auth", func(t *testing.T) {
		header := make(http.Header)
		header.Set("Authorization", "Basic dGVzdHVzZXI6dGVzdHBhc3M=") // testuser:testpass
		
		ws, resp, err := websocket.DefaultDialer.Dial(wsURL, header)
		assert.NoError(t, err)
		if ws != nil {
			ws.Close()
		}
		assert.Equal(t, http.StatusSwitchingProtocols, resp.StatusCode)
	})
	
	t.Run("WebSocket connection with invalid auth", func(t *testing.T) {
		header := make(http.Header)
		header.Set("Authorization", "Basic aW52YWxpZDppbnZhbGlk") // invalid:invalid
		
		ws, resp, err := websocket.DefaultDialer.Dial(wsURL, header)
		assert.Error(t, err)
		if ws != nil {
			ws.Close()
		}
		assert.Equal(t, http.StatusUnauthorized, resp.StatusCode)
	})
	
	t.Run("WebSocket connection without auth", func(t *testing.T) {
		ws, resp, err := websocket.DefaultDialer.Dial(wsURL, nil)
		assert.Error(t, err)
		if ws != nil {
			ws.Close()
		}
		assert.Equal(t, http.StatusUnauthorized, resp.StatusCode)
	})
}

func TestWebSocketConcurrentConnections(t *testing.T) {
	// Create minimal services
	memoryProvider := storage.NewProvider("memory", "", nil)
	accountService, err := services.NewAccountService(memoryProvider)
	require.NoError(t, err)
	
	secretVault, err := services.NewSecretVaultService(memoryProvider, "0123456789abcdef0123456789abcdef")
	require.NoError(t, err)
	
	flowRegistry := registry.NewFlowRegistry(memoryProvider, registry.FlowRegistryOptions{})
	flowRuntime := runtime.NewFlowRuntime(flowRegistry, nil)
	
	// Create test account
	_, err = accountService.CreateAccount("testuser", "testpass")
	require.NoError(t, err)
	
	cfg := &config.Config{
		Server: config.ServerConfig{
			Host: "localhost",
			Port: 8080,
		},
	}
	
	server := NewServerWithRuntime(cfg, flowRegistry, accountService, secretVault, flowRuntime)
	testServer := httptest.NewServer(server.router)
	defer testServer.Close()
	
	wsURL := "ws" + strings.TrimPrefix(testServer.URL, "http") + "/api/v1/ws"
	header := make(http.Header)
	header.Set("Authorization", "Basic dGVzdHVzZXI6dGVzdHBhc3M=")
	
	// Create multiple concurrent WebSocket connections
	numConnections := 5
	connections := make([]*websocket.Conn, numConnections)
	
	for i := 0; i < numConnections; i++ {
		ws, resp, err := websocket.DefaultDialer.Dial(wsURL, header)
		require.NoError(t, err, "Failed to create connection %d", i)
		require.Equal(t, http.StatusSwitchingProtocols, resp.StatusCode)
		connections[i] = ws
	}
	
	// Verify all connections are tracked
	assert.Equal(t, numConnections, server.wsManager.GetConnectedClients())
	
	// Send ping messages on all connections
	for i, ws := range connections {
		pingMsg := WebSocketMessage{Type: "ping"}
		err := ws.WriteJSON(pingMsg)
		assert.NoError(t, err, "Failed to send ping on connection %d", i)
	}
	
	// Read pong responses
	for i, ws := range connections {
		var update ExecutionUpdate
		err := ws.ReadJSON(&update)
		assert.NoError(t, err, "Failed to read pong on connection %d", i)
		assert.Equal(t, "pong", update.Type)
	}
	
	// Close all connections
	for _, ws := range connections {
		ws.Close()
	}
	
	// Give some time for cleanup
	time.Sleep(100 * time.Millisecond)
	
	// Verify connections are cleaned up
	assert.Equal(t, 0, server.wsManager.GetConnectedClients())
}
