package runtime

import (
	"github.com/tcmartin/flowrunner/pkg/auth"
	"github.com/tcmartin/flowrunner/pkg/scripting"
)

// FlowContext manages the execution context for a flow, including expression evaluation
type FlowContext struct {
	executionID string
	flowID      string
	accountID   string
	nodeResults map[string]any
	sharedData  map[string]any
	evaluator   scripting.SecretAwareEvaluator
}

// NewFlowContext creates a new flow context
func NewFlowContext(executionID, flowID, accountID string, secretVault auth.SecretVault) *FlowContext {
	evaluator := scripting.NewSecretAwareExpressionEvaluator(secretVault)

	return &FlowContext{
		executionID: executionID,
		flowID:      flowID,
		accountID:   accountID,
		nodeResults: make(map[string]any),
		sharedData:  make(map[string]any),
		evaluator:   evaluator,
	}
}

// SetNodeResult stores the result of a node execution
func (fc *FlowContext) SetNodeResult(nodeName string, result any) {
	fc.nodeResults[nodeName] = result
}

// GetNodeResult retrieves the result of a node execution
func (fc *FlowContext) GetNodeResult(nodeName string) (any, bool) {
	result, exists := fc.nodeResults[nodeName]
	return result, exists
}

// SetSharedData stores data in the shared context
func (fc *FlowContext) SetSharedData(key string, value any) {
	fc.sharedData[key] = value
}

// GetSharedData retrieves data from the shared context
func (fc *FlowContext) GetSharedData(key string) (any, bool) {
	value, exists := fc.sharedData[key]
	return value, exists
}

// EvaluateExpression evaluates an expression with full flow context
func (fc *FlowContext) EvaluateExpression(expression string) (any, error) {
	context := fc.createEvaluationContext()
	return fc.evaluator.Evaluate(expression, context)
}

// EvaluateInObject evaluates all expressions in an object with full flow context
func (fc *FlowContext) EvaluateInObject(obj map[string]any) (map[string]any, error) {
	context := fc.createEvaluationContext()
	return fc.evaluator.EvaluateInObject(obj, context)
}

// ProcessNodeParams processes node parameters to resolve expressions
func (fc *FlowContext) ProcessNodeParams(params map[string]any) (map[string]any, error) {
	context := fc.createEvaluationContext()
	return fc.evaluator.EvaluateInObject(params, context)
}

// createEvaluationContext creates the full context for expression evaluation
func (fc *FlowContext) createEvaluationContext() map[string]any {
	context := map[string]any{
		"accountID": fc.accountID,
		"_flow_context": map[string]any{
			"node_results": fc.nodeResults,
			"shared_data":  fc.sharedData,
		},
	}

	// Add execution metadata
	context["execution"] = map[string]any{
		"id":      fc.executionID,
		"flow_id": fc.flowID,
	}

	// Add input context for backward compatibility
	context["input"] = fc.sharedData

	// Add shared context for template expressions that use shared.variable
	context["shared"] = fc.sharedData

	return context
}

// GetEvaluationContext returns the current evaluation context for external use
func (fc *FlowContext) GetEvaluationContext() map[string]any {
	return fc.createEvaluationContext()
}
