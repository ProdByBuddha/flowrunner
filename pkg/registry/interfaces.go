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

	// Get retrieves a flow definition by ID
	Get(accountID string, id string) (string, error)

	// List returns all flows for an account
	List(accountID string) ([]FlowInfo, error)

	// Update modifies an existing flow definition
	Update(accountID string, id string, yamlContent string) error

	// Delete removes a flow definition
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

// FlowRegistryOptions contains options for creating a flow registry
type FlowRegistryOptions struct {
	// YAMLLoader is used to validate flow definitions
	YAMLLoader loader.YAMLLoader
}
