// Package runtime provides functionality for executing flows.
package runtime

import (
	"time"
)

// FlowRuntime executes flows
type FlowRuntime interface {
	// Execute runs a flow with the given input
	Execute(accountID string, flowID string, input map[string]interface{}) (string, error)

	// GetStatus retrieves the status of a flow execution
	GetStatus(executionID string) (ExecutionStatus, error)

	// GetLogs retrieves logs for a flow execution
	GetLogs(executionID string) ([]ExecutionLog, error)

	// SubscribeToLogs creates a channel that receives real-time logs for an execution
	SubscribeToLogs(executionID string) (<-chan ExecutionLog, error)

	// Cancel stops a running flow execution
	Cancel(executionID string) error
}

// ExecutionStatus represents the current state of a flow execution
type ExecutionStatus struct {
	// ID of the execution
	ID string `json:"id"`

	// FlowID is the ID of the flow being executed
	FlowID string `json:"flow_id"`

	// Status of the execution
	Status string `json:"status"` // "running", "completed", "failed", "canceled"

	// StartTime is when the execution started
	StartTime time.Time `json:"start_time"`

	// EndTime is when the execution completed
	EndTime time.Time `json:"end_time,omitempty"`

	// Error message if the execution failed
	Error string `json:"error,omitempty"`

	// Results of the execution
	Results map[string]interface{} `json:"results,omitempty"`

	// Progress of the execution (0-100%)
	Progress float64 `json:"progress"`

	// CurrentNode is the ID of the currently executing node
	CurrentNode string `json:"current_node,omitempty"`
}

// ExecutionLog represents a log entry for an execution
type ExecutionLog struct {
	// Timestamp of the log entry
	Timestamp time.Time `json:"timestamp"`

	// NodeID is the ID of the node that generated the log
	NodeID string `json:"node_id,omitempty"`

	// Level of the log entry
	Level string `json:"level"` // "info", "warning", "error", "debug"

	// Message is the log message
	Message string `json:"message"`

	// Data is additional context for the log entry
	Data map[string]interface{} `json:"data,omitempty"`
}
