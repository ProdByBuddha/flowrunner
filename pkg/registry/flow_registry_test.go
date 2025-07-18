package registry

import (
	"errors"
	"testing"
	"time"

	"github.com/tcmartin/flowlib"
	"github.com/tcmartin/flowrunner/pkg/storage"
)

// MockFlowStore is a mock implementation of storage.FlowStore for testing
type MockFlowStore struct {
	flows    map[string]map[string][]byte
	metadata map[string]map[string]storage.FlowMetadata
}

func NewMockFlowStore() *MockFlowStore {
	return &MockFlowStore{
		flows:    make(map[string]map[string][]byte),
		metadata: make(map[string]map[string]storage.FlowMetadata),
	}
}

func (m *MockFlowStore) SaveFlow(accountID, flowID string, definition []byte) error {
	if _, ok := m.flows[accountID]; !ok {
		m.flows[accountID] = make(map[string][]byte)
		m.metadata[accountID] = make(map[string]storage.FlowMetadata)
	}
	m.flows[accountID][flowID] = definition

	// Extract metadata from the definition (simplified for testing)
	meta := storage.FlowMetadata{
		ID:        flowID,
		AccountID: accountID,
		Name:      "Test Flow",
		Version:   "1.0.0",
		CreatedAt: time.Now().Unix(),
		UpdatedAt: time.Now().Unix(),
	}
	m.metadata[accountID][flowID] = meta

	return nil
}

func (m *MockFlowStore) GetFlow(accountID, flowID string) ([]byte, error) {
	if _, ok := m.flows[accountID]; !ok {
		return nil, errors.New("account not found")
	}
	if def, ok := m.flows[accountID][flowID]; ok {
		return def, nil
	}
	return nil, errors.New("flow not found")
}

func (m *MockFlowStore) ListFlows(accountID string) ([]string, error) {
	if _, ok := m.flows[accountID]; !ok {
		return []string{}, nil
	}

	flowIDs := make([]string, 0, len(m.flows[accountID]))
	for id := range m.flows[accountID] {
		flowIDs = append(flowIDs, id)
	}
	return flowIDs, nil
}

func (m *MockFlowStore) DeleteFlow(accountID, flowID string) error {
	if _, ok := m.flows[accountID]; !ok {
		return errors.New("account not found")
	}
	if _, ok := m.flows[accountID][flowID]; !ok {
		return errors.New("flow not found")
	}

	delete(m.flows[accountID], flowID)
	delete(m.metadata[accountID], flowID)
	return nil
}

func (m *MockFlowStore) GetFlowMetadata(accountID, flowID string) (storage.FlowMetadata, error) {
	if _, ok := m.metadata[accountID]; !ok {
		return storage.FlowMetadata{}, errors.New("account not found")
	}
	if meta, ok := m.metadata[accountID][flowID]; ok {
		return meta, nil
	}
	return storage.FlowMetadata{}, errors.New("flow not found")
}

func (m *MockFlowStore) ListFlowsWithMetadata(accountID string) ([]storage.FlowMetadata, error) {
	if _, ok := m.metadata[accountID]; !ok {
		return []storage.FlowMetadata{}, nil
	}

	metadataList := make([]storage.FlowMetadata, 0, len(m.metadata[accountID]))
	for _, meta := range m.metadata[accountID] {
		metadataList = append(metadataList, meta)
	}
	return metadataList, nil
}

// MockYAMLLoader is a mock implementation of loader.YAMLLoader for testing
type MockYAMLLoader struct {
	validateFunc func(string) error
	parseFunc    func(string) (*flowlib.Flow, error)
}

func (m *MockYAMLLoader) Validate(yamlContent string) error {
	if m.validateFunc != nil {
		return m.validateFunc(yamlContent)
	}
	return nil
}

func (m *MockYAMLLoader) Parse(yamlContent string) (*flowlib.Flow, error) {
	if m.parseFunc != nil {
		return m.parseFunc(yamlContent)
	}
	return nil, nil
}

func TestFlowRegistryCreate(t *testing.T) {
	mockStore := NewMockFlowStore()
	mockLoader := &MockYAMLLoader{}

	registry := NewFlowRegistry(mockStore, FlowRegistryOptions{
		YAMLLoader: mockLoader,
	})

	// Test successful creation
	flowID, err := registry.Create("account1", "test-flow", `
metadata:
  name: Test Flow
  description: A test flow
  version: 1.0.0
nodes:
  start:
    type: test
    params:
      foo: bar
`)

	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}

	if flowID == "" {
		t.Error("Expected flow ID, got empty string")
	}

	// Test validation failure
	mockLoader.validateFunc = func(yamlContent string) error {
		return errors.New("validation error")
	}

	_, err = registry.Create("account1", "invalid-flow", "invalid yaml")
	if err == nil {
		t.Error("Expected validation error, got nil")
	}
}

func TestFlowRegistryGet(t *testing.T) {
	mockStore := NewMockFlowStore()
	mockLoader := &MockYAMLLoader{}

	registry := NewFlowRegistry(mockStore, FlowRegistryOptions{
		YAMLLoader: mockLoader,
	})

	// Create a flow first
	yamlContent := `
metadata:
  name: Test Flow
  description: A test flow
  version: 1.0.0
nodes:
  start:
    type: test
    params:
      foo: bar
`
	flowID, _ := registry.Create("account1", "test-flow", yamlContent)

	// Test successful retrieval
	retrievedYAML, err := registry.Get("account1", flowID)
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}

	if retrievedYAML != yamlContent {
		t.Errorf("Expected %s, got %s", yamlContent, retrievedYAML)
	}

	// Test flow not found
	_, err = registry.Get("account1", "non-existent")
	if err == nil {
		t.Error("Expected error for non-existent flow, got nil")
	}

	// Test unauthorized access
	_, err = registry.Get("account2", flowID)
	if err == nil {
		t.Error("Expected error for unauthorized access, got nil")
	}
}

func TestFlowRegistryList(t *testing.T) {
	mockStore := NewMockFlowStore()
	mockLoader := &MockYAMLLoader{}

	registry := NewFlowRegistry(mockStore, FlowRegistryOptions{
		YAMLLoader: mockLoader,
	})

	// Create a few flows
	registry.Create("account1", "flow1", `
metadata:
  name: Flow 1
  description: First test flow
  version: 1.0.0
nodes:
  start:
    type: test
`)

	registry.Create("account1", "flow2", `
metadata:
  name: Flow 2
  description: Second test flow
  version: 1.0.0
nodes:
  start:
    type: test
`)

	registry.Create("account2", "flow3", `
metadata:
  name: Flow 3
  description: Third test flow
  version: 1.0.0
nodes:
  start:
    type: test
`)

	// Test listing flows for account1
	flows, err := registry.List("account1")
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}

	if len(flows) != 2 {
		t.Errorf("Expected 2 flows, got %d", len(flows))
	}

	// Test listing flows for account2
	flows, err = registry.List("account2")
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}

	if len(flows) != 1 {
		t.Errorf("Expected 1 flow, got %d", len(flows))
	}

	// Test listing flows for non-existent account
	flows, err = registry.List("account3")
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}

	if len(flows) != 0 {
		t.Errorf("Expected 0 flows, got %d", len(flows))
	}
}

func TestFlowRegistryUpdate(t *testing.T) {
	mockStore := NewMockFlowStore()
	mockLoader := &MockYAMLLoader{}

	registry := NewFlowRegistry(mockStore, FlowRegistryOptions{
		YAMLLoader: mockLoader,
	})

	// Create a flow first
	yamlContent := `
metadata:
  name: Test Flow
  description: A test flow
  version: 1.0.0
nodes:
  start:
    type: test
    params:
      foo: bar
`
	flowID, _ := registry.Create("account1", "test-flow", yamlContent)

	// Test successful update
	updatedYAML := `
metadata:
  name: Updated Flow
  description: An updated test flow
  version: 1.1.0
nodes:
  start:
    type: test
    params:
      foo: baz
`
	err := registry.Update("account1", flowID, updatedYAML)
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}

	// Verify the update
	retrievedYAML, _ := registry.Get("account1", flowID)
	if retrievedYAML != updatedYAML {
		t.Errorf("Expected %s, got %s", updatedYAML, retrievedYAML)
	}

	// Test validation failure during update
	mockLoader.validateFunc = func(yamlContent string) error {
		return errors.New("validation error")
	}

	err = registry.Update("account1", flowID, "invalid yaml")
	if err == nil {
		t.Error("Expected validation error, got nil")
	}

	// Test flow not found
	err = registry.Update("account1", "non-existent", updatedYAML)
	if err == nil {
		t.Error("Expected error for non-existent flow, got nil")
	}

	// Test unauthorized access
	err = registry.Update("account2", flowID, updatedYAML)
	if err == nil {
		t.Error("Expected error for unauthorized access, got nil")
	}
}

func TestFlowRegistryDelete(t *testing.T) {
	mockStore := NewMockFlowStore()
	mockLoader := &MockYAMLLoader{}

	registry := NewFlowRegistry(mockStore, FlowRegistryOptions{
		YAMLLoader: mockLoader,
	})

	// Create a flow first
	yamlContent := `
metadata:
  name: Test Flow
  description: A test flow
  version: 1.0.0
nodes:
  start:
    type: test
    params:
      foo: bar
`
	flowID, _ := registry.Create("account1", "test-flow", yamlContent)

	// Test successful deletion
	err := registry.Delete("account1", flowID)
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}

	// Verify the deletion
	_, err = registry.Get("account1", flowID)
	if err == nil {
		t.Error("Expected error after deletion, got nil")
	}

	// Test flow not found
	err = registry.Delete("account1", "non-existent")
	if err == nil {
		t.Error("Expected error for non-existent flow, got nil")
	}

	// Test unauthorized access
	flowID, _ = registry.Create("account1", "another-flow", yamlContent)
	err = registry.Delete("account2", flowID)
	if err == nil {
		t.Error("Expected error for unauthorized access, got nil")
	}
}
