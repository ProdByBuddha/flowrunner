package registry

import (
	"fmt"
	"time"

	"github.com/tcmartin/flowrunner/pkg/storage"
)

// UpdateMetadata updates the metadata for a flow without changing the flow definition
func (r *FlowRegistryService) UpdateMetadata(accountID string, id string, metadata FlowMetadata) error {
	// Check if the flow exists and belongs to the account
	existingMetadata, err := r.flowStore.GetFlowMetadata(accountID, id)
	if err != nil {
		return fmt.Errorf("failed to get flow metadata: %w", err)
	}

	// Update the metadata fields while preserving the original fields
	if metadata.Tags != nil {
		existingMetadata.Tags = metadata.Tags
	}
	
	if metadata.Category != "" {
		existingMetadata.Category = metadata.Category
	}
	
	if metadata.Status != "" {
		existingMetadata.Status = metadata.Status
	}
	
	if metadata.Custom != nil {
		if existingMetadata.Custom == nil {
			existingMetadata.Custom = make(map[string]interface{})
		}
		
		// Merge custom fields
		for k, v := range metadata.Custom {
			existingMetadata.Custom[k] = v
		}
	}
	
	// Update the metadata in the store
	if err := r.flowStore.UpdateFlowMetadata(accountID, id, existingMetadata); err != nil {
		return fmt.Errorf("failed to update flow metadata: %w", err)
	}
	
	return nil
}

// Search searches for flows based on metadata filters
func (r *FlowRegistryService) Search(accountID string, filters FlowSearchFilters) ([]FlowInfo, error) {
	// Convert the filters to a map for the storage layer
	filterMap := make(map[string]interface{})
	
	if filters.NameContains != "" {
		filterMap["name_contains"] = filters.NameContains
	}
	
	if filters.DescriptionContains != "" {
		filterMap["description_contains"] = filters.DescriptionContains
	}
	
	if len(filters.Tags) > 0 {
		filterMap["tags"] = filters.Tags
	}
	
	if filters.Category != "" {
		filterMap["category"] = filters.Category
	}
	
	if filters.Status != "" {
		filterMap["status"] = filters.Status
	}
	
	if filters.CreatedAfter != nil {
		filterMap["created_after"] = filters.CreatedAfter.Unix()
	}
	
	if filters.CreatedBefore != nil {
		filterMap["created_before"] = filters.CreatedBefore.Unix()
	}
	
	if filters.UpdatedAfter != nil {
		filterMap["updated_after"] = filters.UpdatedAfter.Unix()
	}
	
	if filters.UpdatedBefore != nil {
		filterMap["updated_before"] = filters.UpdatedBefore.Unix()
	}
	
	if filters.Page > 0 {
		filterMap["page"] = filters.Page
	}
	
	if filters.PageSize > 0 {
		filterMap["page_size"] = filters.PageSize
	}
	
	// Call the storage layer to perform the search
	results, err := r.flowStore.SearchFlows(accountID, filterMap)
	if err != nil {
		return nil, fmt.Errorf("failed to search flows: %w", err)
	}
	
	// Convert storage.FlowMetadata to registry.FlowInfo
	flowInfos := make([]FlowInfo, len(results))
	for i, metadata := range results {
		flowInfos[i] = FlowInfo{
			ID:          metadata.ID,
			AccountID:   metadata.AccountID,
			Name:        metadata.Name,
			Description: metadata.Description,
			Version:     metadata.Version,
			CreatedAt:   time.Unix(metadata.CreatedAt, 0),
			UpdatedAt:   time.Unix(metadata.UpdatedAt, 0),
			Tags:        metadata.Tags,
			Category:    metadata.Category,
			Status:      metadata.Status,
			Custom:      metadata.Custom,
		}
	}
	
	return flowInfos, nil
}
