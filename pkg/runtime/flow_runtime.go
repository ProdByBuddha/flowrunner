package runtime

import (
	"fmt"

	"github.com/google/uuid"
	"github.com/tcmartin/flowrunner/pkg/loader"
)

// flowRuntime is the implementation of the FlowRuntime interface
type flowRuntime struct {
	registry   FlowRegistry
	yamlLoader loader.YAMLLoader
}

// NewFlowRuntime creates a new FlowRuntime
func NewFlowRuntime(registry FlowRegistry, yamlLoader loader.YAMLLoader) FlowRuntime {
	return &flowRuntime{
		registry:   registry,
		yamlLoader: yamlLoader,
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

	// For now, just run the flow synchronously. We'll add async execution and status tracking later.
	_, err = flow.Run(input)
	if err != nil {
		return "", fmt.Errorf("flow execution failed: %w", err)
	}

	return executionID, nil
}

func (r *flowRuntime) GetStatus(executionID string) (ExecutionStatus, error) {
	panic("not implemented")
}

func (r *flowRuntime) GetLogs(executionID string) ([]ExecutionLog, error) {
	panic("not implemented")
}

func (r *flowRuntime) SubscribeToLogs(executionID string) (<-chan ExecutionLog, error) {
	panic("not implemented")
}

func (r *flowRuntime) Cancel(executionID string) error {
	panic("not implemented")
}
