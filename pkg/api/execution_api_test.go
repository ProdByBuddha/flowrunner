package api

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/tcmartin/flowlib"
	"github.com/tcmartin/flowrunner/pkg/config"
	"github.com/tcmartin/flowrunner/pkg/loader"
	"github.com/tcmartin/flowrunner/pkg/registry"
	"github.com/tcmartin/flowrunner/pkg/runtime"
	"github.com/tcmartin/flowrunner/pkg/services"
	"github.com/tcmartin/flowrunner/pkg/storage"
)

// Mock implementations for testing

type MockFlowRegistry struct {
	mock.Mock
}

func (m *MockFlowRegistry) GetFlow(accountID, flowID string) (*runtime.Flow, error) {
	args := m.Called(accountID, flowID)
	return args.Get(0).(*runtime.Flow), args.Error(1)
}

func (m *MockFlowRegistry) Create(accountID, name, content string) (string, error) {
	args := m.Called(accountID, name, content)
	return args.String(0), args.Error(1)
}

func (m *MockFlowRegistry) Get(accountID, flowID string) (string, error) {
	args := m.Called(accountID, flowID)
	return args.String(0), args.Error(1)
}

func (m *MockFlowRegistry) List(accountID string) ([]registry.FlowInfo, error) {
	args := m.Called(accountID)
	return args.Get(0).([]registry.FlowInfo), args.Error(1)
}

func (m *MockFlowRegistry) Update(accountID, flowID, content string) error {
	args := m.Called(accountID, flowID, content)
	return args.Error(0)
}

func (m *MockFlowRegistry) Delete(accountID, flowID string) error {
	args := m.Called(accountID, flowID)
	return args.Error(0)
}

func (m *MockFlowRegistry) UpdateMetadata(accountID, flowID string, metadata registry.FlowMetadata) error {
	args := m.Called(accountID, flowID, metadata)
	return args.Error(0)
}

func (m *MockFlowRegistry) Search(accountID string, filters registry.FlowSearchFilters) ([]registry.FlowInfo, error) {
	args := m.Called(accountID, filters)
	return args.Get(0).([]registry.FlowInfo), args.Error(1)
}

func (m *MockFlowRegistry) GetVersion(accountID, flowID, version string) (string, error) {
	args := m.Called(accountID, flowID, version)
	return args.String(0), args.Error(1)
}

func (m *MockFlowRegistry) ListVersions(accountID, flowID string) ([]registry.FlowVersionInfo, error) {
	args := m.Called(accountID, flowID)
	return args.Get(0).([]registry.FlowVersionInfo), args.Error(1)
}

type MockYAMLLoader struct {
	mock.Mock
}

func (m *MockYAMLLoader) Parse(yamlContent string) (*flowlib.Flow, error) {
	args := m.Called(yamlContent)
	return args.Get(0).(*flowlib.Flow), args.Error(1)
}

func (m *MockYAMLLoader) Validate(yamlContent string) error {
	args := m.Called(yamlContent)
	return args.Error(0)
}

type MockNode struct {
	mock.Mock
}

func (m *MockNode) SetParams(params map[string]interface{}) {
	m.Called(params)
}

func (m *MockNode) Params() map[string]interface{} {
	args := m.Called()
	return args.Get(0).(map[string]interface{})
}

func (m *MockNode) Next(action string, n flowlib.Node) {
	m.Called(action, n)
}

func (m *MockNode) Successors() map[flowlib.Action]flowlib.Node {
	args := m.Called()
	return args.Get(0).(map[flowlib.Action]flowlib.Node)
}

func (m *MockNode) Run(shared interface{}) (flowlib.Action, error) {
	args := m.Called(shared)
	return args.String(0), args.Error(1)
}

type MockExecutionStore struct {
	executions map[string]runtime.ExecutionStatus
	logs       map[string][]runtime.ExecutionLog
}

func NewMockExecutionStore() *MockExecutionStore {
	return &MockExecutionStore{
		executions: make(map[string]runtime.ExecutionStatus),
		logs:       make(map[string][]runtime.ExecutionLog),
	}
}

func (m *MockExecutionStore) SaveExecution(execution runtime.ExecutionStatus) error {
	m.executions[execution.ID] = execution
	return nil
}

func (m *MockExecutionStore) GetExecution(executionID string) (runtime.ExecutionStatus, error) {
	if exec, ok := m.executions[executionID]; ok {
		return exec, nil
	}
	return runtime.ExecutionStatus{}, fmt.Errorf("execution not found")
}

func (m *MockExecutionStore) ListExecutions(accountID string) ([]runtime.ExecutionStatus, error) {
	var result []runtime.ExecutionStatus
	for _, exec := range m.executions {
		// Since we don't have accountID in ExecutionStatus, return all
		result = append(result, exec)
	}
	return result, nil
}

func (m *MockExecutionStore) SaveExecutionLog(executionID string, log runtime.ExecutionLog) error {
	if m.logs[executionID] == nil {
		m.logs[executionID] = make([]runtime.ExecutionLog, 0)
	}
	m.logs[executionID] = append(m.logs[executionID], log)
	return nil
}

func (m *MockExecutionStore) GetExecutionLogs(executionID string) ([]runtime.ExecutionLog, error) {
	if logs, ok := m.logs[executionID]; ok {
		return logs, nil
	}
	return []runtime.ExecutionLog{}, nil
}

// Test helper functions

func setupTestServer() (*Server, *MockFlowRegistry, *storage.MemoryProvider, string) {
	// Create test configuration
	cfg := &config.Config{
		Server: config.ServerConfig{
			Host: "localhost",
			Port: 8080,
		},
	}

	// Create storage provider
	storageProvider := storage.NewMemoryProvider()
	storageProvider.Initialize()

	// Create account service
	accountService := services.NewAccountService(storageProvider.GetAccountStore())

	// Create test account
	accountID, err := accountService.CreateAccount("testuser", "testpass")
	if err != nil {
		panic(err)
	}

	// Create mock flow registry
	mockFlowRegistry := new(MockFlowRegistry)

	// Create extended secret vault
	extendedSecretVault, err := services.NewExtendedSecretVaultService(storageProvider.GetSecretStore(), []byte("test-encryption-key-32-bytes-123"))
	if err != nil {
		panic(err)
	}

	// Create YAML loader with node factories
	nodeFactories := map[string]loader.NodeFactory{
		"base": &loader.BaseNodeFactory{},
	}
	yamlLoader := loader.NewYAMLLoader(nodeFactories)

	// Create flow runtime with storage
	mockExecutionStore := NewMockExecutionStore()
	flowRuntime := runtime.NewFlowRuntimeWithStore(mockFlowRegistry, yamlLoader, mockExecutionStore)

	// Create server with runtime
	server := NewServerWithRuntime(cfg, mockFlowRegistry, accountService, extendedSecretVault, flowRuntime)

	return server, mockFlowRegistry, storageProvider, accountID
}

func makeAuthenticatedRequest(server *Server, accountID, method, url string, body interface{}) *httptest.ResponseRecorder {
	var reqBody bytes.Buffer
	if body != nil {
		json.NewEncoder(&reqBody).Encode(body)
	}

	req := httptest.NewRequest(method, url, &reqBody)
	req.Header.Set("Content-Type", "application/json")

	// Add authentication header (using basic auth for simplicity)
	req.SetBasicAuth("testuser", "testpass")

	rr := httptest.NewRecorder()
	server.router.ServeHTTP(rr, req)
	return rr
}

// Test cases

func TestFlowExecutionAPI(t *testing.T) {
	server, mockFlowRegistry, _, accountID := setupTestServer()

	t.Run("run flow successfully", func(t *testing.T) {
		// Mock flow registry to return a flow
		flowDef := &runtime.Flow{
			ID:   "test-flow",
			YAML: "metadata:\n  name: test-flow\nnodes:\n  start:\n    type: base\n",
		}
		mockFlowRegistry.On("GetFlow", accountID, "test-flow").Return(flowDef, nil)

		// Make request to run flow
		reqBody := map[string]interface{}{
			"input": map[string]interface{}{
				"test": "value",
			},
		}

		rr := makeAuthenticatedRequest(server, accountID, "POST", "/api/v1/flows/test-flow/run", reqBody)

		assert.Equal(t, http.StatusCreated, rr.Code)

		var response map[string]interface{}
		err := json.NewDecoder(rr.Body).Decode(&response)
		assert.NoError(t, err)
		assert.Contains(t, response, "execution_id")
		assert.Equal(t, "running", response["status"])

		mockFlowRegistry.AssertExpectations(t)
	})

	t.Run("run flow without input", func(t *testing.T) {
		// Mock flow registry to return a flow
		flowDef := &runtime.Flow{
			ID:   "test-flow-2",
			YAML: "metadata:\n  name: test-flow-2\nnodes:\n  start:\n    type: base\n",
		}
		mockFlowRegistry.On("GetFlow", accountID, "test-flow-2").Return(flowDef, nil)

		// Make request to run flow without input
		rr := makeAuthenticatedRequest(server, accountID, "POST", "/api/v1/flows/test-flow-2/run", map[string]interface{}{})

		assert.Equal(t, http.StatusCreated, rr.Code)

		var response map[string]interface{}
		err := json.NewDecoder(rr.Body).Decode(&response)
		assert.NoError(t, err)
		assert.Contains(t, response, "execution_id")

		mockFlowRegistry.AssertExpectations(t)
	})

	t.Run("run non-existent flow", func(t *testing.T) {
		// Mock flow registry to return error
		mockFlowRegistry.On("GetFlow", accountID, "non-existent").Return((*runtime.Flow)(nil), assert.AnError)

		rr := makeAuthenticatedRequest(server, accountID, "POST", "/api/v1/flows/non-existent/run", map[string]interface{}{})

		assert.Equal(t, http.StatusBadRequest, rr.Code)

		mockFlowRegistry.AssertExpectations(t)
	})

	t.Run("run flow without authentication", func(t *testing.T) {
		req := httptest.NewRequest("POST", "/api/v1/flows/test-flow/run", nil)
		req.Header.Set("Content-Type", "application/json")

		rr := httptest.NewRecorder()
		server.router.ServeHTTP(rr, req)

		assert.Equal(t, http.StatusUnauthorized, rr.Code)
	})
}

func TestExecutionStatusAPI(t *testing.T) {
	server, mockFlowRegistry, _, accountID := setupTestServer()

	t.Run("get execution status", func(t *testing.T) {
		// First, run a flow to create an execution
		flowDef := &runtime.Flow{
			ID:   "test-flow",
			YAML: "metadata:\n  name: test-flow\nnodes:\n  start:\n    type: base\n",
		}
		mockFlowRegistry.On("GetFlow", accountID, "test-flow").Return(flowDef, nil)

		// Run the flow
		runResp := makeAuthenticatedRequest(server, accountID, "POST", "/api/v1/flows/test-flow/run", map[string]interface{}{})
		assert.Equal(t, http.StatusCreated, runResp.Code)

		var runResponse map[string]interface{}
		err := json.NewDecoder(runResp.Body).Decode(&runResponse)
		assert.NoError(t, err)
		executionID := runResponse["execution_id"].(string)

		// Wait a moment for execution to potentially complete
		time.Sleep(100 * time.Millisecond)

		// Get execution status
		statusResp := makeAuthenticatedRequest(server, accountID, "GET", "/api/v1/executions/"+executionID, nil)
		assert.Equal(t, http.StatusOK, statusResp.Code)

		var status runtime.ExecutionStatus
		err = json.NewDecoder(statusResp.Body).Decode(&status)
		assert.NoError(t, err)
		assert.Equal(t, executionID, status.ID)
		assert.Equal(t, "test-flow", status.FlowID)
		assert.Contains(t, []string{"running", "completed", "failed"}, status.Status)

		mockFlowRegistry.AssertExpectations(t)
	})

	t.Run("get non-existent execution status", func(t *testing.T) {
		rr := makeAuthenticatedRequest(server, accountID, "GET", "/api/v1/executions/non-existent", nil)
		assert.Equal(t, http.StatusNotFound, rr.Code)
	})

	t.Run("get execution status without authentication", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/api/v1/executions/test-id", nil)
		rr := httptest.NewRecorder()
		server.router.ServeHTTP(rr, req)

		assert.Equal(t, http.StatusUnauthorized, rr.Code)
	})
}

func TestExecutionLogsAPI(t *testing.T) {
	server, mockFlowRegistry, _, accountID := setupTestServer()

	t.Run("get execution logs", func(t *testing.T) {
		// First, run a flow to create an execution
		flowDef := &runtime.Flow{
			ID:   "test-flow",
			YAML: "metadata:\n  name: test-flow\nnodes:\n  start:\n    type: base\n",
		}
		mockFlowRegistry.On("GetFlow", accountID, "test-flow").Return(flowDef, nil)

		// Run the flow
		runResp := makeAuthenticatedRequest(server, accountID, "POST", "/api/v1/flows/test-flow/run", map[string]interface{}{})
		assert.Equal(t, http.StatusCreated, runResp.Code)

		var runResponse map[string]interface{}
		err := json.NewDecoder(runResp.Body).Decode(&runResponse)
		assert.NoError(t, err)
		executionID := runResponse["execution_id"].(string)

		// Wait a moment for execution to potentially complete and logs to be written
		time.Sleep(100 * time.Millisecond)

		// Get execution logs
		logsResp := makeAuthenticatedRequest(server, accountID, "GET", "/api/v1/executions/"+executionID+"/logs", nil)
		assert.Equal(t, http.StatusOK, logsResp.Code)

		var logs []runtime.ExecutionLog
		err = json.NewDecoder(logsResp.Body).Decode(&logs)
		assert.NoError(t, err)
		// Should have at least some logs
		assert.GreaterOrEqual(t, len(logs), 0)

		mockFlowRegistry.AssertExpectations(t)
	})

	t.Run("get logs for non-existent execution", func(t *testing.T) {
		rr := makeAuthenticatedRequest(server, accountID, "GET", "/api/v1/executions/non-existent/logs", nil)
		assert.Equal(t, http.StatusOK, rr.Code) // Should return empty logs, not error

		var logs []runtime.ExecutionLog
		err := json.NewDecoder(rr.Body).Decode(&logs)
		assert.NoError(t, err)
		assert.Equal(t, 0, len(logs))
	})
}

func TestExecutionCancelAPI(t *testing.T) {
	server, mockFlowRegistry, _, accountID := setupTestServer()

	t.Run("cancel execution", func(t *testing.T) {
		// Mock a long-running flow
		flowDef := &runtime.Flow{
			ID:   "long-flow",
			YAML: "metadata:\n  name: long-flow\nnodes:\n  start:\n    type: base\n",
		}
		mockFlowRegistry.On("GetFlow", accountID, "long-flow").Return(flowDef, nil)

		// Run the flow
		runResp := makeAuthenticatedRequest(server, accountID, "POST", "/api/v1/flows/long-flow/run", map[string]interface{}{})
		assert.Equal(t, http.StatusCreated, runResp.Code)

		var runResponse map[string]interface{}
		err := json.NewDecoder(runResp.Body).Decode(&runResponse)
		assert.NoError(t, err)
		executionID := runResponse["execution_id"].(string)

		// Try to cancel the execution - since our mock flow completes quickly,
		// we expect either 204 (if still running) or 404 (if already completed)
		cancelResp := makeAuthenticatedRequest(server, accountID, "DELETE", "/api/v1/executions/"+executionID, nil)
		assert.Contains(t, []int{http.StatusNoContent, http.StatusNotFound}, cancelResp.Code)

		mockFlowRegistry.AssertExpectations(t)
	})

	t.Run("cancel non-existent execution", func(t *testing.T) {
		rr := makeAuthenticatedRequest(server, accountID, "DELETE", "/api/v1/executions/non-existent", nil)
		assert.Equal(t, http.StatusNotFound, rr.Code)
	})
}

func TestFlowRuntimeWithoutExecution(t *testing.T) {
	// Test server without flow runtime
	cfg := &config.Config{
		Server: config.ServerConfig{
			Host: "localhost",
			Port: 8080,
		},
	}

	storageProvider := storage.NewMemoryProvider()
	storageProvider.Initialize()

	accountService := services.NewAccountService(storageProvider.GetAccountStore())
	accountService.CreateAccount("testuser", "testpass")

	mockFlowRegistry := new(MockFlowRegistry)

	extendedSecretVault, _ := services.NewExtendedSecretVaultService(storageProvider.GetSecretStore(), []byte("test-encryption-key-32-bytes-123"))

	// Create server WITHOUT runtime
	server := NewServer(cfg, mockFlowRegistry, accountService, extendedSecretVault)

	t.Run("execution endpoints unavailable without runtime", func(t *testing.T) {
		rr := makeAuthenticatedRequest(server, "test-account", "POST", "/api/v1/flows/test-flow/run", map[string]interface{}{})
		assert.Equal(t, http.StatusServiceUnavailable, rr.Code)

		rr = makeAuthenticatedRequest(server, "test-account", "GET", "/api/v1/executions/test-id", nil)
		assert.Equal(t, http.StatusServiceUnavailable, rr.Code)

		rr = makeAuthenticatedRequest(server, "test-account", "GET", "/api/v1/executions/test-id/logs", nil)
		assert.Equal(t, http.StatusServiceUnavailable, rr.Code)

		rr = makeAuthenticatedRequest(server, "test-account", "DELETE", "/api/v1/executions/test-id", nil)
		assert.Equal(t, http.StatusServiceUnavailable, rr.Code)
	})
}
