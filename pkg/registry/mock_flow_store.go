package registry

import (
	"errors"
	"strings"
	"time"

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

	// Parse the YAML to extract metadata
	var flowDef struct {
		Metadata struct {
			Name        string   `yaml:"name"`
			Description string   `yaml:"description"`
			Version     string   `yaml:"version"`
			Tags        []string `yaml:"tags"`
			Category    string   `yaml:"category"`
			Status      string   `yaml:"status"`
		} `yaml:"metadata"`
	}

	// Default values if parsing fails
	version := "1.0.0"
	name := "Test Flow"
	description := ""
	var tags []string
	category := ""
	status := "draft"

	// Try to parse metadata from YAML
	if err := yaml.Unmarshal(definition, &flowDef); err == nil {
		if flowDef.Metadata.Version != "" {
			version = flowDef.Metadata.Version
		}
		if flowDef.Metadata.Name != "" {
			name = flowDef.Metadata.Name
		}
		if flowDef.Metadata.Description != "" {
			description = flowDef.Metadata.Description
		}
		if len(flowDef.Metadata.Tags) > 0 {
			tags = flowDef.Metadata.Tags
		}
		if flowDef.Metadata.Category != "" {
			category = flowDef.Metadata.Category
		}
		if flowDef.Metadata.Status != "" {
			status = flowDef.Metadata.Status
		}
	}

	now := time.Now().Unix()

	// Create metadata with parsed information
	meta := storage.FlowMetadata{
		ID:          flowID,
		AccountID:   accountID,
		Name:        name,
		Description: description,
		Version:     version,
		Tags:        tags,
		Category:    category,
		Status:      status,
		CreatedAt:   now,
		UpdatedAt:   now,
	}
	m.metadata[accountID][flowID] = meta

	// Also save in versions
	if _, ok := m.versions[accountID][flowID]; !ok {
		m.versions[accountID][flowID] = make(map[string][]byte)
	}
	m.versions[accountID][flowID][version] = definition

	return nil
}

func (m *MockFlowStore) GetFlow(accountID, flowID string) ([]byte, error) {
	if _, ok := m.flows[accountID]; !ok {
		return nil, storage.ErrFlowNotFound
	}
	if definition, ok := m.flows[accountID][flowID]; ok {
		return definition, nil
	}
	return nil, storage.ErrFlowNotFound
}

func (m *MockFlowStore) ListFlows(accountID string) ([]string, error) {
	var flowIDs []string
	if flows, ok := m.flows[accountID]; ok {
		for flowID := range flows {
			flowIDs = append(flowIDs, flowID)
		}
	}
	return flowIDs, nil
}

func (m *MockFlowStore) DeleteFlow(accountID, flowID string) error {
	if _, ok := m.flows[accountID]; !ok {
		return storage.ErrFlowNotFound
	}
	if _, ok := m.flows[accountID][flowID]; !ok {
		return storage.ErrFlowNotFound
	}
	delete(m.flows[accountID], flowID)
	delete(m.metadata[accountID], flowID)
	delete(m.versions[accountID], flowID)
	return nil
}

func (m *MockFlowStore) GetFlowMetadata(accountID, flowID string) (storage.FlowMetadata, error) {
	if _, ok := m.metadata[accountID]; !ok {
		return storage.FlowMetadata{}, storage.ErrFlowNotFound
	}
	if metadata, ok := m.metadata[accountID][flowID]; ok {
		return metadata, nil
	}
	return storage.FlowMetadata{}, storage.ErrFlowNotFound
}

func (m *MockFlowStore) ListFlowsWithMetadata(accountID string) ([]storage.FlowMetadata, error) {
	var metadataList []storage.FlowMetadata
	if metadata, ok := m.metadata[accountID]; ok {
		for _, meta := range metadata {
			metadataList = append(metadataList, meta)
		}
	}
	return metadataList, nil
}

func (m *MockFlowStore) SaveFlowVersion(accountID, flowID string, definition []byte, version string) error {
	if _, ok := m.flows[accountID]; !ok {
		return storage.ErrFlowNotFound
	}
	if _, ok := m.flows[accountID][flowID]; !ok {
		return storage.ErrFlowNotFound
	}

	// Update the main flow definition to latest version
	m.flows[accountID][flowID] = definition

	// Parse the YAML to extract metadata
	var flowDef struct {
		Metadata struct {
			Name        string   `yaml:"name"`
			Description string   `yaml:"description"`
			Tags        []string `yaml:"tags"`
			Category    string   `yaml:"category"`
			Status      string   `yaml:"status"`
		} `yaml:"metadata"`
	}

	// Get existing metadata and update it
	meta := m.metadata[accountID][flowID]

	// Try to parse metadata from YAML
	if err := yaml.Unmarshal(definition, &flowDef); err == nil {
		if flowDef.Metadata.Name != "" {
			meta.Name = flowDef.Metadata.Name
		}
		if flowDef.Metadata.Description != "" {
			meta.Description = flowDef.Metadata.Description
		}
		// Only update tags if they are specified in the YAML
		if len(flowDef.Metadata.Tags) > 0 {
			meta.Tags = flowDef.Metadata.Tags
		}
		// Only update category if it's specified in the YAML
		if flowDef.Metadata.Category != "" {
			meta.Category = flowDef.Metadata.Category
		}
		// Only update status if it's specified in the YAML
		if flowDef.Metadata.Status != "" {
			meta.Status = flowDef.Metadata.Status
		}
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
		return nil, storage.ErrFlowNotFound
	}
	if _, ok := m.versions[accountID][flowID]; !ok {
		return nil, storage.ErrFlowNotFound
	}
	if definition, ok := m.versions[accountID][flowID][version]; ok {
		return definition, nil
	}
	return nil, storage.ErrFlowNotFound
}

func (m *MockFlowStore) ListFlowVersions(accountID, flowID string) ([]string, error) {
	if _, ok := m.versions[accountID]; !ok {
		return nil, storage.ErrFlowNotFound
	}
	if versions, ok := m.versions[accountID][flowID]; ok {
		var versionList []string
		for version := range versions {
			versionList = append(versionList, version)
		}
		return versionList, nil
	}
	return nil, storage.ErrFlowNotFound
}

func (m *MockFlowStore) DeleteFlowVersion(accountID, flowID, version string) error {
	if _, ok := m.versions[accountID]; !ok {
		return storage.ErrFlowNotFound
	}
	if _, ok := m.versions[accountID][flowID]; !ok {
		return storage.ErrFlowNotFound
	}
	if _, ok := m.versions[accountID][flowID][version]; !ok {
		return storage.ErrFlowNotFound
	}
	delete(m.versions[accountID][flowID], version)
	return nil
}

// UpdateFlowMetadata implements the storage.FlowStore interface method for updating flow metadata
func (m *MockFlowStore) UpdateFlowMetadata(accountID, flowID string, metadata storage.FlowMetadata) error {
	if _, ok := m.metadata[accountID]; !ok {
		return errors.New("flow not found")
	}
	if _, ok := m.metadata[accountID][flowID]; !ok {
		return errors.New("flow not found")
	}

	// Update the metadata while preserving certain fields
	existing := m.metadata[accountID][flowID]
	metadata.ID = existing.ID
	metadata.AccountID = existing.AccountID
	metadata.CreatedAt = existing.CreatedAt
	metadata.UpdatedAt = time.Now().Unix()
	
	// Ensure we preserve the name, description, and version from existing if not provided
	if metadata.Name == "" {
		metadata.Name = existing.Name
	}
	if metadata.Description == "" {
		metadata.Description = existing.Description
	}
	if metadata.Version == "" {
		metadata.Version = existing.Version
	}

	m.metadata[accountID][flowID] = metadata
	return nil
}

// SearchFlows implements the storage.FlowStore interface method for searching flows
func (m *MockFlowStore) SearchFlows(accountID string, filters map[string]interface{}) ([]storage.FlowMetadata, error) {
	if _, ok := m.metadata[accountID]; !ok {
		return []storage.FlowMetadata{}, nil
	}

	var results []storage.FlowMetadata
	
	for _, metadata := range m.metadata[accountID] {
		if matchesAllFilters(metadata, filters) {
			results = append(results, metadata)
		}
	}
	
	// Handle pagination if specified
	if page, ok := filters["page"].(int); ok && page > 0 {
		pageSize := 10 // Default page size
		if size, ok := filters["page_size"].(int); ok && size > 0 {
			pageSize = size
		}
		
		start := (page - 1) * pageSize
		end := start + pageSize
		
		if start >= len(results) {
			return []storage.FlowMetadata{}, nil
		}
		
		if end > len(results) {
			end = len(results)
		}
		
		results = results[start:end]
	}
	
	return results, nil
}

// matchesAllFilters checks if a flow metadata matches the given filters
func matchesAllFilters(metadata storage.FlowMetadata, filters map[string]interface{}) bool {
	// Check name_contains filter
	if nameContains, ok := filters["name_contains"].(string); ok && nameContains != "" {
		if !strings.Contains(strings.ToLower(metadata.Name), strings.ToLower(nameContains)) {
			return false
		}
	}
	
	// Check description_contains filter
	if descContains, ok := filters["description_contains"].(string); ok && descContains != "" {
		if !strings.Contains(strings.ToLower(metadata.Description), strings.ToLower(descContains)) {
			return false
		}
	}
	
	// Check tags filter
	if tags, ok := filters["tags"].([]string); ok && len(tags) > 0 {
		found := false
		for _, tag := range tags {
			for _, metaTag := range metadata.Tags {
				if tag == metaTag {
					found = true
					break
				}
			}
			if found {
				break
			}
		}
		if !found {
			return false
		}
	}
	
	// Check category filter
	if category, ok := filters["category"].(string); ok && category != "" {
		if metadata.Category != category {
			return false
		}
	}
	
	// Check status filter
	if status, ok := filters["status"].(string); ok && status != "" {
		if metadata.Status != status {
			return false
		}
	}
	
	// Check created_after filter
	if createdAfter, ok := filters["created_after"].(int64); ok {
		if metadata.CreatedAt < createdAfter {
			return false
		}
	}
	
	// Check created_before filter
	if createdBefore, ok := filters["created_before"].(int64); ok {
		if metadata.CreatedAt > createdBefore {
			return false
		}
	}
	
	// Check updated_after filter
	if updatedAfter, ok := filters["updated_after"].(int64); ok {
		if metadata.UpdatedAt < updatedAfter {
			return false
		}
	}
	
	// Check updated_before filter
	if updatedBefore, ok := filters["updated_before"].(int64); ok {
		if metadata.UpdatedAt > updatedBefore {
			return false
		}
	}
	
	return true
}
