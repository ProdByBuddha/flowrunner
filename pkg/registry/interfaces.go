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
}

// FlowVersionInfo contains metadata about a specific flow version
type FlowVersionInfo struct {
	FlowID      string    `json:"flow_id"`
	Version     string    `json:"version"`
	Description string    `json:"description"`
	CreatedAt   time.Time `json:"created_at"`
	CreatedBy   string    `json:"created_by,omitempty"`
}

// FlowRegistryOptions contains options for creating a flow registry
type FlowRegistryOptions struct {
	// YAMLLoader is used to validate flow definitions
	YAMLLoader loader.YAMLLoader
}
