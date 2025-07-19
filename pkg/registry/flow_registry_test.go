package registry

import (
	"errors"
	"testing"

	"github.com/tcmartin/flowlib"
	"gopkg.in/yaml.v3"
)

// MockYAMLLoader is a mock implementation of loader.YAMLLoader for testing
type MockYAMLLoader struct {
	validateFunc func(string) error
	parseFunc    func(string) (*flowlib.Flow, error)
}

func (m *MockYAMLLoader) Validate(yamlContent string) error {
	if m.validateFunc != nil {
		return m.validateFunc(yamlContent)
	}
	// Basic validation - check if it's valid YAML
	var data interface{}
	err := yaml.Unmarshal([]byte(yamlContent), &data)
	return err
}

func (m *MockYAMLLoader) Parse(yamlContent string) (*flowlib.Flow, error) {
	if m.parseFunc != nil {
		return m.parseFunc(yamlContent)
	}

	// Create a simple flow for testing
	flow := &flowlib.Flow{}
	return flow, nil
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

// These methods are already implemented above
func TestFlowRegistryVersioning(t *testing.T) {
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

	// Update the flow to create a new version
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
		t.Errorf("Expected no error on update, got %v", err)
	}

	// Test getting a specific version
	v1YAML, err := registry.GetVersion("account1", flowID, "1.0.0")
	if err != nil {
		t.Errorf("Expected no error getting version 1.0.0, got %v", err)
	}
	if v1YAML != yamlContent {
		t.Errorf("Expected original content for version 1.0.0, got different content")
	}

	v2YAML, err := registry.GetVersion("account1", flowID, "1.1.0")
	if err != nil {
		t.Errorf("Expected no error getting version 1.1.0, got %v", err)
	}
	if v2YAML != updatedYAML {
		t.Errorf("Expected updated content for version 1.1.0, got different content")
	}

	// Test listing versions
	versions, err := registry.ListVersions("account1", flowID)
	if err != nil {
		t.Errorf("Expected no error listing versions, got %v", err)
	}
	if len(versions) != 2 {
		t.Errorf("Expected 2 versions, got %d", len(versions))
	}

	// Test getting a non-existent version
	_, err = registry.GetVersion("account1", flowID, "2.0.0")
	if err == nil {
		t.Error("Expected error for non-existent version, got nil")
	}

	// Test getting a version for a non-existent flow
	_, err = registry.GetVersion("account1", "non-existent", "1.0.0")
	if err == nil {
		t.Error("Expected error for non-existent flow, got nil")
	}

	// Test unauthorized access to a version
	_, err = registry.GetVersion("account2", flowID, "1.0.0")
	if err == nil {
		t.Error("Expected error for unauthorized access, got nil")
	}
}
