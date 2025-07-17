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

	// GetFlow retrieves a flow definition
	GetFlow(accountID, flowID string) ([]byte, error)

	// ListFlows returns all flow IDs for an account
	ListFlows(accountID string) ([]string, error)

	// DeleteFlow removes a flow definition
	DeleteFlow(accountID, flowID string) error

	// GetFlowMetadata retrieves metadata for a flow
	GetFlowMetadata(accountID, flowID string) (FlowMetadata, error)

	// ListFlowsWithMetadata returns all flows with metadata for an account
	ListFlowsWithMetadata(accountID string) ([]FlowMetadata, error)
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

	// Version of the flow
	Version string `json:"version"`

	// CreatedAt is when the flow was created
	CreatedAt int64 `json:"created_at"`

	// UpdatedAt is when the flow was last updated
	UpdatedAt int64 `json:"updated_at"`
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
