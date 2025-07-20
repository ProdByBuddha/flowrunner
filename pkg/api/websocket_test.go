package api

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gorilla/websocket"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/tcmartin/flowrunner/pkg/runtime"
)

// MockFlowRuntimeForWebSocket for focused WebSocket testing
type MockFlowRuntimeForWebSocket struct {
	mock.Mock
}

func (m *MockFlowRuntimeForWebSocket) Execute(accountID string, flowID string, input map[string]interface{}) (string, error) {
	args := m.Called(accountID, flowID, input)
	return args.String(0), args.Error(1)
}

func (m *MockFlowRuntimeForWebSocket) GetStatus(executionID string) (runtime.ExecutionStatus, error) {
	args := m.Called(executionID)
	return args.Get(0).(runtime.ExecutionStatus), args.Error(1)
}

func (m *MockFlowRuntimeForWebSocket) GetLogs(executionID string) ([]runtime.ExecutionLog, error) {
	args := m.Called(executionID)
	return args.Get(0).([]runtime.ExecutionLog), args.Error(1)
}

func (m *MockFlowRuntimeForWebSocket) SubscribeToLogs(executionID string) (<-chan runtime.ExecutionLog, error) {
	args := m.Called(executionID)
	return args.Get(0).(<-chan runtime.ExecutionLog), args.Error(1)
}

func (m *MockFlowRuntimeForWebSocket) Cancel(executionID string) error {
	args := m.Called(executionID)
	return args.Error(0)
}

func (m *MockFlowRuntimeForWebSocket) ListExecutions(accountID string) ([]runtime.ExecutionStatus, error) {
	args := m.Called(accountID)
	return args.Get(0).([]runtime.ExecutionStatus), args.Error(1)
}

func TestWebSocketManager_NewWebSocketManager(t *testing.T) {
	mockRuntime := &MockFlowRuntimeForWebSocket{}
	
	wsManager := NewWebSocketManager(mockRuntime)
	
	assert.NotNil(t, wsManager)
	assert.Equal(t, mockRuntime, wsManager.flowRuntime)
	assert.NotNil(t, wsManager.connections)
	assert.NotNil(t, wsManager.connectionMeta)
	assert.Equal(t, 0, wsManager.GetConnectedClients())
}

func TestWebSocketManager_HandleWebSocket(t *testing.T) {
	mockRuntime := &MockFlowRuntimeForWebSocket{}
	wsManager := NewWebSocketManager(mockRuntime)
	
	// Create test server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		wsManager.HandleWebSocket(w, r, "test-account")
	}))
	defer server.Close()
	
	// Convert http://127.0.0.1 to ws://127.0.0.1
	u := "ws" + strings.TrimPrefix(server.URL, "http") + "/"
	
	// Connect to the WebSocket
	ws, _, err := websocket.DefaultDialer.Dial(u, nil)
	assert.NoError(t, err)
	defer ws.Close()
	
	// Allow some time for connection to be registered
	time.Sleep(100 * time.Millisecond)
	
	// Check that connection is registered
	assert.Equal(t, 1, wsManager.GetConnectedClients())
}

func TestWebSocketManager_SubscribeToExecution(t *testing.T) {
	mockRuntime := &MockFlowRuntimeForWebSocket{}
	wsManager := NewWebSocketManager(mockRuntime)
	
	// Create a test execution status
	testStatus := runtime.ExecutionStatus{
		ID:        "test-execution",
		FlowID:    "test-flow",
		Status:    "running",
		StartTime: time.Now(),
		Progress:  50.0,
	}
	
	// Create a log channel for the mock
	logChan := make(chan runtime.ExecutionLog, 1)
	close(logChan) // Close immediately to avoid hanging
	
	// Mock the GetStatus and SubscribeToLogs calls
	mockRuntime.On("GetStatus", "test-execution").Return(testStatus, nil)
	mockRuntime.On("SubscribeToLogs", "test-execution").Return((<-chan runtime.ExecutionLog)(logChan), nil)
	
	// Create test server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		wsManager.HandleWebSocket(w, r, "test-account")
	}))
	defer server.Close()
	
	// Convert http://127.0.0.1 to ws://127.0.0.1
	u := "ws" + strings.TrimPrefix(server.URL, "http") + "/"
	
	// Connect to the WebSocket
	ws, _, err := websocket.DefaultDialer.Dial(u, nil)
	assert.NoError(t, err)
	defer ws.Close()
	
	// Subscribe to execution
	subscribeMsg := WebSocketMessage{
		Type:        "subscribe",
		ExecutionID: "test-execution",
	}
	
	err = ws.WriteJSON(subscribeMsg)
	assert.NoError(t, err)
	
	// Read the status update message
	var update ExecutionUpdate
	err = ws.ReadJSON(&update)
	assert.NoError(t, err)
	
	// Verify the update
	assert.Equal(t, "status", update.Type)
	assert.Equal(t, "test-execution", update.ExecutionID)
	assert.NotNil(t, update.Status)
	assert.Equal(t, "running", update.Status.Status)
	
	// Verify subscription count
	assert.Equal(t, 1, wsManager.GetExecutionSubscribers("test-execution"))
	
	// Wait a bit for the monitor goroutine to process
	time.Sleep(200 * time.Millisecond)
	
	mockRuntime.AssertExpectations(t)
}

func TestWebSocketManager_UnsubscribeFromExecution(t *testing.T) {
	mockRuntime := &MockFlowRuntimeForWebSocket{}
	wsManager := NewWebSocketManager(mockRuntime)
	
	// Create a test execution status
	testStatus := runtime.ExecutionStatus{
		ID:        "test-execution",
		FlowID:    "test-flow",
		Status:    "running",
		StartTime: time.Now(),
		Progress:  50.0,
	}
	
	// Create a log channel for the mock
	logChan := make(chan runtime.ExecutionLog, 1)
	close(logChan) // Close immediately to avoid hanging
	
	// Mock the GetStatus and SubscribeToLogs calls
	mockRuntime.On("GetStatus", "test-execution").Return(testStatus, nil)
	mockRuntime.On("SubscribeToLogs", "test-execution").Return((<-chan runtime.ExecutionLog)(logChan), nil)
	
	// Create test server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		wsManager.HandleWebSocket(w, r, "test-account")
	}))
	defer server.Close()
	
	// Convert http://127.0.0.1 to ws://127.0.0.1
	u := "ws" + strings.TrimPrefix(server.URL, "http") + "/"
	
	// Connect to the WebSocket
	ws, _, err := websocket.DefaultDialer.Dial(u, nil)
	assert.NoError(t, err)
	defer ws.Close()
	
	// Subscribe to execution
	subscribeMsg := WebSocketMessage{
		Type:        "subscribe",
		ExecutionID: "test-execution",
	}
	
	err = ws.WriteJSON(subscribeMsg)
	assert.NoError(t, err)
	
	// Read the status update message
	var update ExecutionUpdate
	err = ws.ReadJSON(&update)
	assert.NoError(t, err)
	
	// Verify subscription
	assert.Equal(t, 1, wsManager.GetExecutionSubscribers("test-execution"))
	
	// Unsubscribe from execution
	unsubscribeMsg := WebSocketMessage{
		Type:        "unsubscribe",
		ExecutionID: "test-execution",
	}
	
	err = ws.WriteJSON(unsubscribeMsg)
	assert.NoError(t, err)
	
	// Allow some time for unsubscription
	time.Sleep(100 * time.Millisecond)
	
	// Verify unsubscription
	assert.Equal(t, 0, wsManager.GetExecutionSubscribers("test-execution"))
	
	mockRuntime.AssertExpectations(t)
}

func TestWebSocketManager_PingPong(t *testing.T) {
	mockRuntime := &MockFlowRuntimeForWebSocket{}
	wsManager := NewWebSocketManager(mockRuntime)
	
	// Create test server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		wsManager.HandleWebSocket(w, r, "test-account")
	}))
	defer server.Close()
	
	// Convert http://127.0.0.1 to ws://127.0.0.1
	u := "ws" + strings.TrimPrefix(server.URL, "http") + "/"
	
	// Connect to the WebSocket
	ws, _, err := websocket.DefaultDialer.Dial(u, nil)
	assert.NoError(t, err)
	defer ws.Close()
	
	// Send ping message
	pingMsg := WebSocketMessage{
		Type: "ping",
	}
	
	err = ws.WriteJSON(pingMsg)
	assert.NoError(t, err)
	
	// Read the pong response
	var update ExecutionUpdate
	err = ws.ReadJSON(&update)
	assert.NoError(t, err)
	
	// Verify the pong response
	assert.Equal(t, "pong", update.Type)
}

func TestWebSocketManager_ExecutionNotFound(t *testing.T) {
	mockRuntime := &MockFlowRuntimeForWebSocket{}
	wsManager := NewWebSocketManager(mockRuntime)
	
	// Mock the GetStatus call to return error
	mockRuntime.On("GetStatus", "nonexistent-execution").Return(runtime.ExecutionStatus{}, assert.AnError)
	
	// Create test server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		wsManager.HandleWebSocket(w, r, "test-account")
	}))
	defer server.Close()
	
	// Convert http://127.0.0.1 to ws://127.0.0.1
	u := "ws" + strings.TrimPrefix(server.URL, "http") + "/"
	
	// Connect to the WebSocket
	ws, _, err := websocket.DefaultDialer.Dial(u, nil)
	assert.NoError(t, err)
	defer ws.Close()
	
	// Subscribe to nonexistent execution
	subscribeMsg := WebSocketMessage{
		Type:        "subscribe",
		ExecutionID: "nonexistent-execution",
	}
	
	err = ws.WriteJSON(subscribeMsg)
	assert.NoError(t, err)
	
	// Read the error message
	var update ExecutionUpdate
	err = ws.ReadJSON(&update)
	assert.NoError(t, err)
	
	// Verify the error response
	assert.Equal(t, "error", update.Type)
	assert.Equal(t, "nonexistent-execution", update.ExecutionID)
	assert.Contains(t, update.Message, "not found")
	
	mockRuntime.AssertExpectations(t)
}
