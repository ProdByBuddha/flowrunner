// Package api provides the core API interfaces for flowrunner.
package api

import (
	"time"
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
	// ID of the flow
	ID string `json:"id"`

	// Name of the flow
	Name string `json:"name"`

	// Description of the flow
	Description string `json:"description"`

	// Version of the flow
	Version string `json:"version"`

	// CreatedAt is when the flow was created
	CreatedAt time.Time `json:"created_at"`

	// UpdatedAt is when the flow was last updated
	UpdatedAt time.Time `json:"updated_at"`
}
