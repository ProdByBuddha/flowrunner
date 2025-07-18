package registry

import (
	"errors"
	"testing"
	"time"

	"github.com/tcmartin/flowlib"
	"github.com/tcmartin/flowrunner/pkg/storage"
	"gopkg.in/yaml.v3"
)

// MockFlowStore is a mock implementation of storage.FlowStore for testing
type MockFlowStore struct {
	flows    map[string]map[string][]byte
	metadata map[string]map[string]storage.FlowMetadata
	versions map[string]map[string]map[string][]byte // accountID -> flowID -> version -> definition
}

func NewMockFlowStore() *MockFlowStore {
	return &MockFlowStore{
		flows:    make(map[string]map[string][]byte),
		metadata: make(map[string]map[string]storage.FlowMetadata),
		versions: make(map[string]map[string]map[string][]byte),
	}
}

func (m *MockFlowStore) SaveFlow(accountID, flowID string, definition []byte) error {
	if _, ok := m.flows[accountID]; !ok {
		m.flows[accountID] = make(map[string][]byte)
		m.metadata[accountID] = make(map[string]storage.FlowMetadata)
		m.versions[accountID] = make(map[string]map[string][]byte)
	}
	m.flows[accountID][flowID] = definition

	// Parse the YAML to extract version info (simplified for testing)
	var flowDef struct {
		Metadata struct {
			Name        string `yaml:"name"`
			Description string `yaml:"description"`
			Version     string `yaml:"version"`
		} `yaml:"metadata"`
	}
	
	// Default version if parsing fails
	version := "1.0.0"
	name := "Test Flow"
	description := ""
	
	// Try to parse version from YAML
	if err := yaml.Unmarshal(definition, &flowDef); err == nil {
		if flowDef.Metadata.Version != "" {
			version = flowDef.Metadata.Version
		}
		if flowDef.Metadata.Name != "" {
			name = flowDef.Metadata.Name
		}
		description = flowDef.Metadata.Description
	}
	
	now := time.Now().Unix()

	// Create metadata with parsed information
	meta := storage.FlowMetadata{
		ID:          flowID,
		AccountID:   accountID,
		Name:        name,
		Description: description,
		Version:     version,
		CreatedAt:   now,
		UpdatedAt:   now,
	}
	m.metadata[accountID][flowID] = meta

	// Store the version
	if _, ok := m.versions[accountID][flowID]; !ok {
		m.versions[accountID][flowID] = make(map[string][]byte)
	}
	m.versions[accountID][flowID][version] = definition

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

	// Delete all versions
	if _, ok := m.versions[accountID]; ok {
		delete(m.versions[accountID], flowID)
	}

	return nil
}

func (m *MockFlowStore) SaveFlowVersion(accountID, flowID string, definition []byte, version string) error {
	if _, ok := m.flows[accountID]; !ok {
		return errors.New("account not found")
	}
	if _, ok := m.flows[accountID][flowID]; !ok {
		return errors.New("flow not found")
	}

	// Update the main flow definition to latest version
	m.flows[accountID][flowID] = definition

	// Parse the YAML to extract metadata (simplified for testing)
	var flowDef struct {
		Metadata struct {
			Name        string `yaml:"name"`
			Description string `yaml:"description"`
		} `yaml:"metadata"`
	}
	
	// Get existing metadata and update it
	meta := m.metadata[accountID][flowID]
	
	// Try to parse metadata from YAML
	if err := yaml.Unmarshal(definition, &flowDef); err == nil {
		if flowDef.Metadata.Name != "" {
			meta.Name = flowDef.Metadata.Name
		}
		meta.Description = flowDef.Metadata.Description
	}
	
	meta.Version = version
	meta.UpdatedAt = time.Now().Unix()
	m.metadata[accountID][flowID] = meta

	// Store the version
	if _, ok := m.versions[accountID][flowID]; !ok {
		m.versions[accountID][flowID] = make(map[string][]byte)
	}
	m.versions[accountID][flowID][version] = definition

	return nil
}

func (m *MockFlowStore) GetFlowVersion(accountID, flowID, version string) ([]byte, error) {
	if _, ok := m.versions[accountID]; !ok {
		return nil, errors.New("account not found")
	}
	if _, ok := m.versions[accountID][flowID]; !ok {
		return nil, errors.New("flow not found")
	}
	if def, ok := m.versions[accountID][flowID][version]; ok {
		return def, nil
	}
	return nil, errors.New("flow version not found")
}

func (m *MockFlowStore) ListFlowVersions(accountID, flowID string) ([]string, error) {
	if _, ok := m.versions[accountID]; !ok {
		return nil, errors.New("account not found")
	}
	if _, ok := m.versions[accountID][flowID]; !ok {
		return nil, errors.New("flow not found")
	}

	versions := make([]string, 0, len(m.versions[accountID][flowID]))
	for version := range m.versions[accountID][flowID] {
		versions = append(versions, version)
	}
	return versions, nil
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
