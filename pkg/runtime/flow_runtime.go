package runtime

import (
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/tcmartin/flowrunner/pkg/loader"
	"github.com/tcmartin/flowrunner/pkg/models"
	"github.com/tcmartin/flowrunner/pkg/storage"
)

// flowRuntime is the implementation of the FlowRuntime interface
type flowRuntime struct {
	registry       FlowRegistry
	yamlLoader     loader.YAMLLoader
	executionStore ExecutionStore
}

// NewFlowRuntime creates a new FlowRuntime
func NewFlowRuntime(registry FlowRegistry, yamlLoader loader.YAMLLoader, executionStore ExecutionStore) FlowRuntime {
	return &flowRuntime{
		registry:       registry,
		yamlLoader:     yamlLoader,
		executionStore: executionStore,
	}
}

func (r *flowRuntime) Execute(accountID string, flowID string, input map[string]interface{}) (string, error) {
	flowDef, err := r.registry.GetFlow(accountID, flowID)
	if err != nil {
		return "", fmt.Errorf("failed to get flow: %w", err)
	}

	flow, err := r.yamlLoader.Parse(flowDef.YAML)
	if err != nil {
		return "", fmt.Errorf("failed to parse flow YAML: %w", err)
	}

	executionID := uuid.New().String()
	status := ExecutionStatus{
		ID:        executionID,
		FlowID:    flowID,
		Status:    "running",
		StartTime: time.Now(),
		Progress:  0,
	}
	if err := r.executionStore.SaveExecution(status); err != nil {
		return "", fmt.Errorf("failed to save execution status: %w", err)
	}

	// Initialize a shared context for the flow execution
	sharedContext := make(map[string]interface{})
	for k, v := range input {
		sharedContext[k] = v
	}

	// For now, just run the flow synchronously. We'll add async execution and status tracking later.
	_, err = flow.Run(sharedContext)
	if err != nil {
		status.Status = "failed"
		status.Error = err.Error()
		status.EndTime = time.Now()
		r.executionStore.SaveExecution(status) // Best effort save
		return "", fmt.Errorf("flow execution failed: %w", err)
	}

	status.Status = "completed"
	status.EndTime = time.Now()
	status.Progress = 100
	status.Results = sharedContext
	if err := r.executionStore.SaveExecution(status); err != nil {
		return "", fmt.Errorf("failed to save final execution status: %w", err)
	}

	return executionID, nil
}

func (r *flowRuntime) GetStatus(executionID string) (ExecutionStatus, error) {
	status, err := r.executionStore.GetExecution(executionID)
	if err != nil {
		return ExecutionStatus{}, fmt.Errorf("failed to get execution status: %w", err)
	}
	return status, nil
}

func (r *flowRuntime) GetLogs(executionID string) ([]ExecutionLog, error) {
	logs, err := r.executionStore.GetExecutionLogs(executionID)
	if err != nil {
		return nil, fmt.Errorf("failed to get execution logs: %w", err)
	}
	return logs, nil
}

func (r *flowRuntime) SubscribeToLogs(executionID string) (<-chan ExecutionLog, error) {
	// This will be implemented later with WebSockets or similar
	return nil, fmt.Errorf("not implemented")
}

func (r *flowRuntime) Cancel(executionID string) error {
	// This will be implemented later
	return fmt.Errorf("not implemented")
}
