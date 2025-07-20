package runtime_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"

	
	"github.com/tcmartin/flowrunner/pkg/runtime"
	"github.com/tcmartin/flowlib"
)

// MockFlowRegistry is a mock implementation of the FlowRegistry interface
type MockFlowRegistry struct {
	mock.Mock
}

func (m *MockFlowRegistry) GetFlow(accountID, flowID string) (*runtime.Flow, error) {
	args := m.Called(accountID, flowID)
	return args.Get(0).(*runtime.Flow), args.Error(1)
}

// MockYAMLLoader is a mock implementation of the YAMLLoader interface
type MockYAMLLoader struct {
	mock.Mock
}

func (m *MockYAMLLoader) Parse(yamlContent string) (*flowlib.Flow, error) {
	args := m.Called(yamlContent)
	return args.Get(0).(*flowlib.Flow), args.Error(1)
}

func (m *MockYAMLLoader) Validate(yamlContent string) error {
	args := m.Called(yamlContent)
	return args.Error(0)
}

// MockNode is a mock implementation of flowlib.Node
type MockNode struct {
	mock.Mock
}

func (m *MockNode) SetParams(params map[string]interface{}) {
	m.Called(params)
}

func (m *MockNode) Params() map[string]interface{} {
	args := m.Called()
	return args.Get(0).(map[string]interface{})
}

func (m *MockNode) Next(action string, n flowlib.Node) {
	m.Called(action, n)
}

func (m *MockNode) Successors() map[flowlib.Action]flowlib.Node {
	args := m.Called()
	return args.Get(0).(map[flowlib.Action]flowlib.Node)
}

func (m *MockNode) Run(shared interface{}) (flowlib.Action, error) {
	args := m.Called(shared)
	return args.String(0), args.Error(1)
}



func TestFlowRuntime_Execute(t *testing.T) {
	// Create mocks
	mockRegistry := new(MockFlowRegistry)
	mockYAMLLoader := new(MockYAMLLoader)

	// Create a sample flow for testing
	flowDef := &runtime.Flow{
		ID:   "test-flow",
		YAML: "metadata:\n  name: test-flow\nnodes:\n  start:\n    type: base\n",
	}

	// Create a mock node
	mockNode := new(MockNode)
	mockNode.On("Run", mock.Anything).Return(flowlib.DefaultAction, nil)
	mockNode.On("Successors").Return(map[flowlib.Action]flowlib.Node{})

	// Create a real flowlib.Flow with the mock node as its start node
	flowlibFlow := flowlib.NewFlow(mockNode)

	// Set up mock expectations
	mockRegistry.On("GetFlow", "test-account", "test-flow").Return(flowDef, nil)
	mockYAMLLoader.On("Parse", flowDef.YAML).Return(flowlibFlow, nil)

	// Create the flow runtime with the mocks
	flowRuntime := runtime.NewFlowRuntime(mockRegistry, mockYAMLLoader)

	// Execute the flow
	executionID, err := flowRuntime.Execute("test-account", "test-flow", nil)

	// Assertions
	assert.NoError(t, err)
	assert.NotEmpty(t, executionID)

	// Verify mock expectations
	mockRegistry.AssertExpectations(t)
	mockYAMLLoader.AssertExpectations(t)
}
