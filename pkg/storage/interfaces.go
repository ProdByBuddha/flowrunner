// Package storage provides interfaces for persistent storage.
package storage

import (
	"github.com/tcmartin/flowrunner/pkg/auth"
	"github.com/tcmartin/flowrunner/pkg/runtime"
)

// StorageProvider defines the interface for persistence backends
type StorageProvider interface {
	// Initialize sets up the storage backend
	Initialize() error

	// Close cleans up resources
	Close() error

	// GetFlowStore returns a store for flow definitions
	GetFlowStore() FlowStore

	// GetSecretStore returns a store for secrets
	GetSecretStore() SecretStore

	// GetExecutionStore returns a store for execution data
	GetExecutionStore() ExecutionStore

	// GetAccountStore returns a store for account data
	GetAccountStore() AccountStore
}

// FlowStore manages flow definition persistence
type FlowStore interface {
	// SaveFlow persists a flow definition
	SaveFlow(accountID, flowID string, definition []byte) error

	// SaveFlowVersion persists a new version of a flow definition
	SaveFlowVersion(accountID, flowID string, definition []byte, version string) error

	// GetFlow retrieves a flow definition (latest version)
	GetFlow(accountID, flowID string) ([]byte, error)

	// GetFlowVersion retrieves a specific version of a flow definition
	GetFlowVersion(accountID, flowID, version string) ([]byte, error)

	// ListFlowVersions returns all versions of a flow
	ListFlowVersions(accountID, flowID string) ([]string, error)

	// ListFlows returns all flow IDs for an account
	ListFlows(accountID string) ([]string, error)

	// DeleteFlow removes a flow definition and all its versions
	DeleteFlow(accountID, flowID string) error

	// GetFlowMetadata retrieves metadata for a flow
	GetFlowMetadata(accountID, flowID string) (FlowMetadata, error)

	// ListFlowsWithMetadata returns all flows with metadata for an account
	ListFlowsWithMetadata(accountID string) ([]FlowMetadata, error)
	
	// UpdateFlowMetadata updates the metadata for a flow
	UpdateFlowMetadata(accountID, flowID string, metadata FlowMetadata) error
	
	// SearchFlows searches for flows based on metadata filters
	SearchFlows(accountID string, filters map[string]interface{}) ([]FlowMetadata, error)
}

// FlowMetadata contains information about a stored flow
type FlowMetadata struct {
	// ID of the flow
	ID string `json:"id"`

	// AccountID is the ID of the account that owns the flow
	AccountID string `json:"account_id"`

	// Name of the flow
	Name string `json:"name"`

	// Description of the flow
	Description string `json:"description"`

	// Version of the flow (latest version)
	Version string `json:"version"`

	// CreatedAt is when the flow was created
	CreatedAt int64 `json:"created_at"`

	// UpdatedAt is when the flow was last updated
	UpdatedAt int64 `json:"updated_at"`
	
	// Tags for categorizing and searching flows
	Tags []string `json:"tags,omitempty"`
	
	// Category for grouping flows
	Category string `json:"category,omitempty"`
	
	// Status of the flow (e.g., "draft", "published", "archived")
	Status string `json:"status,omitempty"`
	
	// Custom metadata fields
	Custom map[string]interface{} `json:"custom,omitempty"`
}

// FlowVersion contains information about a specific version of a flow
type FlowVersion struct {
	// FlowID is the ID of the flow
	FlowID string `json:"flow_id"`

	// Version is the version identifier
	Version string `json:"version"`

	// Description is the version description
	Description string `json:"description"`

	// CreatedAt is when the version was created
	CreatedAt int64 `json:"created_at"`

	// CreatedBy is the user who created the version
	CreatedBy string `json:"created_by,omitempty"`

	// Definition is the flow definition for this version
	Definition []byte `json:"definition"`
}

// SecretStore manages secret persistence
type SecretStore interface {
	// SaveSecret persists a secret
	SaveSecret(secret auth.Secret) error

	// GetSecret retrieves a secret
	GetSecret(accountID, key string) (auth.Secret, error)

	// ListSecrets returns all secrets for an account
	ListSecrets(accountID string) ([]auth.Secret, error)

	// DeleteSecret removes a secret
	DeleteSecret(accountID, key string) error
}

// ExecutionStore manages execution data persistence
type ExecutionStore interface {
	// SaveExecution persists execution data
	SaveExecution(execution runtime.ExecutionStatus) error

	// GetExecution retrieves execution data
	GetExecution(executionID string) (runtime.ExecutionStatus, error)

	// ListExecutions returns all executions for an account
	ListExecutions(accountID string) ([]runtime.ExecutionStatus, error)

	// SaveExecutionLog persists an execution log entry
	SaveExecutionLog(executionID string, log runtime.ExecutionLog) error

	// GetExecutionLogs retrieves logs for an execution
	GetExecutionLogs(executionID string) ([]runtime.ExecutionLog, error)
}

// AccountStore manages account persistence
type AccountStore interface {
	// SaveAccount persists an account
	SaveAccount(account auth.Account) error

	// GetAccount retrieves an account
	GetAccount(accountID string) (auth.Account, error)

	// GetAccountByUsername retrieves an account by username
	GetAccountByUsername(username string) (auth.Account, error)

	// GetAccountByToken retrieves an account by API token
	GetAccountByToken(token string) (auth.Account, error)

	// ListAccounts returns all accounts
	ListAccounts() ([]auth.Account, error)

	// DeleteAccount removes an account
	DeleteAccount(accountID string) error
}
