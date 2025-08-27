package runtime

import (
    "encoding/json"
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

    // Add input context for backward compatibility. Provide a view of shared data
    // and also surface commonly-used fields from the last node result for
    // templates that reference `${input.*}` directly.
    inputView := make(map[string]any, len(fc.sharedData))
    for k, v := range fc.sharedData {
        inputView[k] = v
    }
    if res, ok := fc.sharedData["result"].(map[string]any); ok {
        if tp, ok := res["tool_params"]; ok {
            inputView["tool_params"] = tp
        } else if tp := tryExtractToolParams(res); tp != nil {
            inputView["tool_params"] = tp
        }
    }
    if _, exists := inputView["tool_params"]; !exists {
        if res2, ok := fc.sharedData["result_result"].(map[string]any); ok {
            if tp := tryExtractToolParams(res2); tp != nil {
                inputView["tool_params"] = tp
            }
        }
    }
    if tp, ok := fc.sharedData["tool_params"]; ok {
        inputView["tool_params"] = tp
    }
    context["input"] = inputView

	// Add shared context for template expressions that use shared.variable
	context["shared"] = fc.sharedData

	return context
}

// GetEvaluationContext returns the current evaluation context for external use
func (fc *FlowContext) GetEvaluationContext() map[string]any {
    return fc.createEvaluationContext()
}

// tryExtractToolParams attempts to derive tool parameters from a structure
// that may contain llm_result.tool_calls[*].function.arguments (JSON string).
func tryExtractToolParams(v map[string]any) map[string]any {
    // Check direct llm_result first
    if lr, ok := v["llm_result"].(map[string]any); ok {
        if tp := parseToolCallsForParams(lr); tp != nil {
            return tp
        }
    }
    // Some shapes nest under result
    if inner, ok := v["result"].(map[string]any); ok {
        if lr, ok := inner["llm_result"].(map[string]any); ok {
            if tp := parseToolCallsForParams(lr); tp != nil {
                return tp
            }
        }
        if tp := parseToolCallsForParams(inner); tp != nil {
            return tp
        }
    }
    // Try at top-level as well
    if tp := parseToolCallsForParams(v); tp != nil {
        return tp
    }
    return nil
}

func parseToolCallsForParams(container map[string]any) map[string]any {
    raw, ok := container["tool_calls"].([]interface{})
    if !ok || len(raw) == 0 {
        return nil
    }
    first, ok := raw[0].(map[string]any)
    if !ok {
        return nil
    }
    fn, ok := first["function"].(map[string]any)
    if !ok {
        return nil
    }
    if argStr, ok := fn["arguments"].(string); ok && argStr != "" {
        var m map[string]any
        if err := json.Unmarshal([]byte(argStr), &m); err == nil {
            return m
        }
    }
    return nil
}
