package storage

import (
	"github.com/tcmartin/flowrunner/pkg/runtime"
)

// ExecutionWrapper wraps an execution status with additional metadata
type ExecutionWrapper struct {
	runtime.ExecutionStatus
	AccountID string
}
