// Package registry provides functionality for managing flow definitions.
package registry

import (
	"time"

	"github.com/tcmartin/flowrunner/pkg/loader"
)

// FlowRegistry manages flow definitions
type FlowRegistry interface {
	// Create stores a new flow definition
	Create(accountID string, name string, yamlContent string) (string, error)

	// Get retrieves a flow definition by ID (latest version)
	Get(accountID string, id string) (string, error)

	// GetVersion retrieves a specific version of a flow definition
	GetVersion(accountID string, id string, version string) (string, error)

	// List returns all flows for an account
	List(accountID string) ([]FlowInfo, error)

	// ListVersions returns all versions of a flow
	ListVersions(accountID string, id string) ([]FlowVersionInfo, error)

	// Update modifies an existing flow definition and creates a new version
	Update(accountID string, id string, yamlContent string) error

	// Delete removes a flow definition and all its versions
	Delete(accountID string, id string) error
	
	// UpdateMetadata updates the metadata for a flow without changing the flow definition
	UpdateMetadata(accountID string, id string, metadata FlowMetadata) error
	
	// Search searches for flows based on metadata filters
	Search(accountID string, filters FlowSearchFilters) ([]FlowInfo, error)
}

// FlowInfo contains metadata about a flow
type FlowInfo struct {
	ID          string    `json:"id"`
	AccountID   string    `json:"account_id"`
	Name        string    `json:"name"`
	Description string    `json:"description"`
	Version     string    `json:"version"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
	Tags        []string  `json:"tags,omitempty"`
	Category    string    `json:"category,omitempty"`
	Status      string    `json:"status,omitempty"`
	Custom      map[string]interface{} `json:"custom,omitempty"`
}

// FlowVersionInfo contains metadata about a specific flow version
type FlowVersionInfo struct {
	FlowID      string    `json:"flow_id"`
	Version     string    `json:"version"`
	Description string    `json:"description"`
	CreatedAt   time.Time `json:"created_at"`
	CreatedBy   string    `json:"created_by,omitempty"`
}

// FlowMetadata contains additional metadata for a flow
type FlowMetadata struct {
	// Tags for categorizing and searching flows
	Tags []string `json:"tags,omitempty"`
	
	// Category for grouping flows
	Category string `json:"category,omitempty"`
	
	// Status of the flow (e.g., "draft", "published", "archived")
	Status string `json:"status,omitempty"`
	
	// Custom metadata fields
	Custom map[string]interface{} `json:"custom,omitempty"`
}

// FlowSearchFilters defines the filters for searching flows
type FlowSearchFilters struct {
	// Search by name (partial match)
	NameContains string `json:"name_contains,omitempty"`
	
	// Search by description (partial match)
	DescriptionContains string `json:"description_contains,omitempty"`
	
	// Filter by tags (exact match for any tag in the list)
	Tags []string `json:"tags,omitempty"`
	
	// Filter by category (exact match)
	Category string `json:"category,omitempty"`
	
	// Filter by status (exact match)
	Status string `json:"status,omitempty"`
	
	// Filter by creation date range
	CreatedAfter  *time.Time `json:"created_after,omitempty"`
	CreatedBefore *time.Time `json:"created_before,omitempty"`
	
	// Filter by update date range
	UpdatedAfter  *time.Time `json:"updated_after,omitempty"`
	UpdatedBefore *time.Time `json:"updated_before,omitempty"`
	
	// Pagination parameters
	Page     int `json:"page,omitempty"`      // 1-based page number
	PageSize int `json:"page_size,omitempty"` // Number of items per page
}

// FlowRegistryOptions contains options for creating a flow registry
type FlowRegistryOptions struct {
	// YAMLLoader is used to validate flow definitions
	YAMLLoader loader.YAMLLoader
}
