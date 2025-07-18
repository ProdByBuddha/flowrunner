package registry

import (
	"errors"
	"strings"
	"time"

	"github.com/tcmartin/flowrunner/pkg/storage"
)

// UpdateFlowMetadata implements the storage.FlowStore interface method for updating flow metadata
func (m *MockFlowStore) UpdateFlowMetadata(accountID, flowID string, metadata storage.FlowMetadata) error {
	if _, ok := m.metadata[accountID]; !ok {
		return errors.New("account not found")
	}
	if _, ok := m.metadata[accountID][flowID]; !ok {
		return errors.New("flow not found")
	}

	// Update the metadata while preserving the ID, account ID, creation time, etc.
	existingMetadata := m.metadata[accountID][flowID]
	existingMetadata.Tags = metadata.Tags
	existingMetadata.Category = metadata.Category
	existingMetadata.Status = metadata.Status
	existingMetadata.Custom = metadata.Custom
	existingMetadata.UpdatedAt = time.Now().Unix()
	
	m.metadata[accountID][flowID] = existingMetadata
	
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
