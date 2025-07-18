package storage

import (
	"errors"
	"strings"
	"time"
)

// UpdateFlowMetadata for MemoryFlowStore
func (s *MemoryFlowStore) UpdateFlowMetadata(accountID, flowID string, metadata FlowMetadata) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Check if the account and flow exist
	if _, ok := s.flows[accountID]; !ok {
		return ErrAccountNotFound
	}
	if _, ok := s.flows[accountID][flowID]; !ok {
		return ErrFlowNotFound
	}

	// Get the existing metadata
	existingMetadata := s.metadata[accountID][flowID]

	// Update the metadata fields
	existingMetadata.Tags = metadata.Tags
	existingMetadata.Category = metadata.Category
	existingMetadata.Status = metadata.Status
	existingMetadata.Custom = metadata.Custom

	// Update the last modified time
	existingMetadata.UpdatedAt = time.Now().Unix()

	// Save the updated metadata
	s.metadata[accountID][flowID] = existingMetadata

	return nil
}

// SearchFlows for MemoryFlowStore
func (s *MemoryFlowStore) SearchFlows(accountID string, filters map[string]interface{}) ([]FlowMetadata, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	// Check if the account exists
	if _, ok := s.metadata[accountID]; !ok {
		return []FlowMetadata{}, nil
	}

	// Get all metadata for the account
	var results []FlowMetadata

	for _, metadata := range s.metadata[accountID] {
		// Check if the metadata matches all filters
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
			return []FlowMetadata{}, nil
		}

		if end > len(results) {
			end = len(results)
		}

		results = results[start:end]
	}

	return results, nil
}

// matchesAllFilters checks if a flow metadata matches all the given filters
func matchesAllFilters(metadata FlowMetadata, filters map[string]interface{}) bool {
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

// UpdateFlowMetadata for DynamoDBFlowStore
func (s *DynamoDBFlowStore) UpdateFlowMetadata(accountID, flowID string, metadata FlowMetadata) error {
	// This would be implemented using DynamoDB's UpdateItem API
	// For now, we'll return a not implemented error
	return errors.New("not implemented")
}

// SearchFlows for DynamoDBFlowStore
func (s *DynamoDBFlowStore) SearchFlows(accountID string, filters map[string]interface{}) ([]FlowMetadata, error) {
	// This would be implemented using DynamoDB's Query API with filter expressions
	// For now, we'll return a not implemented error
	return nil, errors.New("not implemented")
}

// UpdateFlowMetadata for PostgreSQLFlowStore
func (s *PostgreSQLFlowStore) UpdateFlowMetadata(accountID, flowID string, metadata FlowMetadata) error {
	// This would be implemented using SQL UPDATE statements
	// For now, we'll return a not implemented error
	return errors.New("not implemented")
}

// SearchFlows for PostgreSQLFlowStore
func (s *PostgreSQLFlowStore) SearchFlows(accountID string, filters map[string]interface{}) ([]FlowMetadata, error) {
	// This would be implemented using SQL SELECT statements with WHERE clauses
	// For now, we'll return a not implemented error
	return nil, errors.New("not implemented")
}
