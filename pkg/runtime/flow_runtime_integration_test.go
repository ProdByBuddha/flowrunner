package runtime_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"

	"github.com/tcmartin/flowrunner/pkg/loader"
	"github.com/tcmartin/flowrunner/pkg/runtime"
)

// MockFlowRegistry for integration tests
type IntegrationMockFlowRegistry struct {
	mock.Mock
}

func (m *IntegrationMockFlowRegistry) GetFlow(accountID, flowID string) (*runtime.Flow, error) {
	args := m.Called(accountID, flowID)
	return args.Get(0).(*runtime.Flow), args.Error(1)
}

func TestFlowRuntime_Integration_SimpleFlow(t *testing.T) {
	// Setup
	mockRegistry := new(IntegrationMockFlowRegistry)
	nodeFactories := map[string]loader.NodeFactory{
		"base": &loader.BaseNodeFactory{},
	}
	yamlLoader := loader.NewYAMLLoader(nodeFactories)
	flowRuntime := runtime.NewFlowRuntime(mockRegistry, yamlLoader)

	// Define a simple flow in YAML
	yamlContent := `
metadata:
  name: simple-flow
nodes:
  start:
    type: base
    next:
      default: end
  end:
    type: base
`
	flowDef := &runtime.Flow{
		ID:   "simple-flow",
		YAML: yamlContent,
	}
	mockRegistry.On("GetFlow", "test-account", "simple-flow").Return(flowDef, nil)

	// Execute the flow
	executionID, err := flowRuntime.Execute("test-account", "simple-flow", nil)

	// Assertions
	assert.NoError(t, err)
	assert.NotEmpty(t, executionID)
	mockRegistry.AssertExpectations(t)
}

func TestFlowRuntime_Integration_FlowWithRetry(t *testing.T) {
	// Setup
	mockRegistry := new(IntegrationMockFlowRegistry)
	nodeFactories := map[string]loader.NodeFactory{
		"base": &loader.BaseNodeFactory{},
	}
	yamlLoader := loader.NewYAMLLoader(nodeFactories)
	flowRuntime := runtime.NewFlowRuntime(mockRegistry, yamlLoader)

	// Define a flow with retry logic
	yamlContent := `
metadata:
  name: retry-flow
nodes:
  start:
    type: base
    retry:
      max_retries: 2
      wait: 10ms
    next:
      default: end
  end:
    type: base
`
	flowDef := &runtime.Flow{
		ID:   "retry-flow",
		YAML: yamlContent,
	}
	mockRegistry.On("GetFlow", "test-account", "retry-flow").Return(flowDef, nil)

	// Execute the flow
	executionID, err := flowRuntime.Execute("test-account", "retry-flow", nil)

	// Assertions
	assert.NoError(t, err)
	assert.NotEmpty(t, executionID)
	mockRegistry.AssertExpectations(t)
}

func TestFlowRuntime_Integration_FlowWithBatch(t *testing.T) {
	// Setup
	mockRegistry := new(IntegrationMockFlowRegistry)
	nodeFactories := map[string]loader.NodeFactory{
		"base":  &loader.BaseNodeFactory{},
		"batch": &loader.BatchNodeFactory{},
	}
	yamlLoader := loader.NewYAMLLoader(nodeFactories)
	flowRuntime := runtime.NewFlowRuntime(mockRegistry, yamlLoader)

	// Define a flow with batch logic
	yamlContent := `
metadata:
  name: batch-flow
nodes:
  start:
    type: base
    next:
      default: process_items
  process_items:
    type: batch
    next:
      default: end
  end:
    type: base
`
	flowDef := &runtime.Flow{
		ID:   "batch-flow",
		YAML: yamlContent,
	}
	mockRegistry.On("GetFlow", "test-account", "batch-flow").Return(flowDef, nil)

	// Execute the flow
	executionID, err := flowRuntime.Execute("test-account", "batch-flow", nil)

	// Assertions
	assert.NoError(t, err)
	assert.NotEmpty(t, executionID)
	mockRegistry.AssertExpectations(t)
}

func TestFlowRuntime_Integration_FlowWithWorkerPool(t *testing.T) {
	// Setup
	mockRegistry := new(IntegrationMockFlowRegistry)
	nodeFactories := map[string]loader.NodeFactory{
		"base":        &loader.BaseNodeFactory{},
		"worker_pool": &loader.WorkerPoolBatchNodeFactory{},
	}
	yamlLoader := loader.NewYAMLLoader(nodeFactories)
	flowRuntime := runtime.NewFlowRuntime(mockRegistry, yamlLoader)

	// Define a flow with worker pool logic
	yamlContent := `
metadata:
  name: worker-pool-flow
nodes:
  start:
    type: base
    next:
      default: process_items
  process_items:
    type: worker_pool
    batch:
      max_parallel: 2
    next:
      default: end
  end:
    type: base
`
	flowDef := &runtime.Flow{
		ID:   "worker-pool-flow",
		YAML: yamlContent,
	}
	mockRegistry.On("GetFlow", "test-account", "worker-pool-flow").Return(flowDef, nil)

	// Execute the flow
	executionID, err := flowRuntime.Execute("test-account", "worker-pool-flow", nil)

	// Assertions
	assert.NoError(t, err)
	assert.NotEmpty(t, executionID)
	mockRegistry.AssertExpectations(t)
}
