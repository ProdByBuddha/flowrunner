// Package runtime provides functionality for executing flows.
package runtime

import (
	"github.com/tcmartin/flowrunner/pkg/models"
)

// FlowRuntime executes flows
type FlowRuntime interface {
	// Execute runs a flow with the given input
	Execute(accountID string, flowID string, input map[string]interface{}) (string, error)

	// GetStatus retrieves the status of a flow execution
	GetStatus(executionID string) (models.ExecutionStatus, error)

	// GetLogs retrieves logs for a flow execution
	GetLogs(executionID string) ([]models.ExecutionLog, error)

	// SubscribeToLogs creates a channel that receives real-time logs for an execution
	SubscribeToLogs(executionID string) (<-chan models.ExecutionLog, error)

	// Cancel stops a running flow execution
	Cancel(executionID string) error
}

// FlowRegistry is an interface for retrieving flow definitions
type FlowRegistry interface {
	GetFlow(accountID, flowID string) (*Flow, error)
}

type Flow represents a flow definition
type Flow struct {
	ID string
	YAML string
}

// ExecutionStore defines the interface for storing and retrieving execution data
type ExecutionStore interface {
	SaveExecution(status models.ExecutionStatus) error
	GetExecution(executionID string) (models.ExecutionStatus, error)
	ListExecutions(accountID string) ([]models.ExecutionStatus, error)
	SaveExecutionLog(executionID string, log models.ExecutionLog) error
	GetExecutionLogs(executionID string) ([]models.ExecutionLog, error)
}
