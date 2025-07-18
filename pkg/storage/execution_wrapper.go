package storage

import (
	"github.com/tcmartin/flowrunner/pkg/runtime"
)

// ExecutionWrapper wraps a runtime.ExecutionStatus with additional storage-specific fields
type ExecutionWrapper struct {
	// The actual execution status
	runtime.ExecutionStatus

	// AccountID is the ID of the account that owns the execution
	// This is not part of the runtime.ExecutionStatus struct but is needed for storage
	AccountID string
}

// NewExecutionWrapper creates a new execution wrapper
func NewExecutionWrapper(execution runtime.ExecutionStatus, accountID string) ExecutionWrapper {
	return ExecutionWrapper{
		ExecutionStatus: execution,
		AccountID:       accountID,
	}
}

// ToExecutionStatus converts an ExecutionWrapper to a runtime.ExecutionStatus
func (w ExecutionWrapper) ToExecutionStatus() runtime.ExecutionStatus {
	return w.ExecutionStatus
}

// ExecutionStore interface extension for account-aware operations
type AccountAwareExecutionStore interface {
	// SaveExecutionWithAccount persists execution data with account ID
	SaveExecutionWithAccount(execution runtime.ExecutionStatus, accountID string) error
}

// Helper function to get account ID from execution store if it supports it
func GetAccountID(store ExecutionStore, executionID string) (string, bool) {
	if _, ok := store.(AccountAwareExecutionStore); ok {
		// If the store implements the extended interface, use it
		return "", false // Not implemented yet
	}
	return "", false
}
