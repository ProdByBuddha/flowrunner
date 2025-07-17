package loader

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/tcmartin/flowlib"
)

// MockNodeFactory is a mock implementation of NodeFactory
type MockNodeFactory func(params map[string]interface{}) (flowlib.Node, error)

// MockExpressionEvaluator is a mock implementation of ExpressionEvaluator
type MockExpressionEvaluator struct {
	mock.Mock
}

func (m *MockExpressionEvaluator) Evaluate(expression string, context map[string]interface{}) (interface{}, error) {
	args := m.Called(expression, context)
	return args.Get(0), args.Error(1)
}

func (m *MockExpressionEvaluator) EvaluateInObject(obj map[string]interface{}, context map[string]interface{}) (map[string]interface{}, error) {
	args := m.Called(obj, context)
	return args.Get(0).(map[string]interface{}), args.Error(1)
}

// MockScriptEngine is a mock implementation of ScriptEngine
type MockScriptEngine struct {
	mock.Mock
}

func (m *MockScriptEngine) Execute(script string, context map[string]interface{}) (interface{}, error) {
	args := m.Called(script, context)
	return args.Get(0), args.Error(1)
}

func (m *MockScriptEngine) ExecuteWithTimeout(ctx context.Context, script string, context map[string]interface{}, timeout time.Duration) (interface{}, error) {
	args := m.Called(ctx, script, context, timeout)
	return args.Get(0), args.Error(1)
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

func (m *MockNode) Next(action string, n flowlib.Node) flowlib.Node {
	args := m.Called(action, n)
	return args.Get(0).(flowlib.Node)
}

func (m *MockNode) Successors() map[string]flowlib.Node {
	args := m.Called()
	return args.Get(0).(map[string]flowlib.Node)
}

func (m *MockNode) Run(shared interface{}) (string, error) {
	args := m.Called(shared)
	return args.String(0), args.Error(1)
}

func TestYAMLLoaderValidate(t *testing.T) {
	// Create mock dependencies
	nodeFactories := map[string]NodeFactory{
		"test": func(params map[string]interface{}) (flowlib.Node, error) {
			mockNode := new(MockNode)
			mockNode.On("SetParams", mock.Anything).Return()
			mockNode.On("Params").Return(map[string]interface{}{})
			mockNode.On("Next", mock.Anything, mock.Anything).Return(mockNode)
			mockNode.On("Successors").Return(map[string]flowlib.Node{})
			mockNode.On("Run", mock.Anything).Return("default", nil)
			return mockNode, nil
		},
	}
	evaluator := new(MockExpressionEvaluator)
	engine := new(MockScriptEngine)

	// Create YAML loader
	loader := NewYAMLLoader(nodeFactories, evaluator, engine)

	// Test cases
	tests := []struct {
		name    string
		yaml    string
		wantErr bool
	}{
		{
			name: "Valid YAML",
			yaml: `
metadata:
  name: "Test Flow"
  description: "A test flow"
  version: "1.0.0"
nodes:
  start:
    type: "test"
    params:
      key: "value"
    next:
      default: "end"
  end:
    type: "test"
    params:
      key: "value"
`,
			wantErr: false,
		},
		{
			name: "Invalid YAML - Missing metadata",
			yaml: `
nodes:
  start:
    type: "test"
    params:
      key: "value"
`,
			wantErr: true,
		},
		{
			name: "Invalid YAML - Missing nodes",
			yaml: `
metadata:
  name: "Test Flow"
`,
			wantErr: true,
		},
		{
			name: "Invalid YAML - Unknown node type",
			yaml: `
metadata:
  name: "Test Flow"
nodes:
  start:
    type: "unknown"
    params:
      key: "value"
`,
			wantErr: true,
		},
		{
			name: "Invalid YAML - Invalid reference",
			yaml: `
metadata:
  name: "Test Flow"
nodes:
  start:
    type: "test"
    params:
      key: "value"
    next:
      default: "nonexistent"
`,
			wantErr: true,
		},
	}

	// Run tests
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := loader.Validate(tt.yaml)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestYAMLLoaderParse(t *testing.T) {
	// Create mock dependencies
	mockNode := new(MockNode)
	mockNode.On("SetParams", mock.Anything).Return()
	mockNode.On("Params").Return(map[string]interface{}{})
	mockNode.On("Next", mock.Anything, mock.Anything).Return(mockNode)
	mockNode.On("Successors").Return(map[string]flowlib.Node{})
	mockNode.On("Run", mock.Anything).Return("default", nil)

	nodeFactories := map[string]NodeFactory{
		"test": func(params map[string]interface{}) (flowlib.Node, error) {
			return mockNode, nil
		},
	}
	evaluator := new(MockExpressionEvaluator)
	engine := new(MockScriptEngine)

	// Create YAML loader
	loader := NewYAMLLoader(nodeFactories, evaluator, engine)

	// Test valid YAML
	yaml := `
metadata:
  name: "Test Flow"
  description: "A test flow"
  version: "1.0.0"
nodes:
  start:
    type: "test"
    params:
      key: "value"
    next:
      default: "end"
  end:
    type: "test"
    params:
      key: "value"
`

	flow, err := loader.Parse(yaml)
	assert.NoError(t, err)
	assert.NotNil(t, flow)

	// Test invalid YAML
	invalidYaml := `
metadata:
  name: "Test Flow"
nodes:
  start:
    type: "unknown"
`

	flow, err = loader.Parse(invalidYaml)
	assert.Error(t, err)
	assert.Nil(t, flow)
}
