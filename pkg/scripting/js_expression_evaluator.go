// Package scripting provides JavaScript execution capabilities for flows.
package scripting

import (
	"fmt"
	"strings"

	"github.com/robertkrimen/otto"
)

// JSExpressionEvaluator is an implementation of the ExpressionEvaluator interface using a JavaScript engine
type JSExpressionEvaluator struct {
	vm *otto.Otto
}

// NewJSExpressionEvaluator creates a new JSExpressionEvaluator
func NewJSExpressionEvaluator() *JSExpressionEvaluator {
	vm := otto.New()
	return &JSExpressionEvaluator{
		vm: vm,
	}
}

// Evaluate processes an expression string with the given context
func (e *JSExpressionEvaluator) Evaluate(expression string, context map[string]any) (any, error) {
	// Check if this is an expression
	if !strings.HasPrefix(expression, "${") || !strings.HasSuffix(expression, "}") {
		return expression, nil
	}

	// Extract the expression content
	expr := expression[2 : len(expression)-1]

	// Set up the context in the JavaScript VM
	for key, value := range context {
		e.vm.Set(key, value)
	}

	// Evaluate the expression
	result, err := e.vm.Run(expr)
	if err != nil {
		return nil, fmt.Errorf("failed to evaluate expression '%s': %w", expr, err)
	}

	// Convert the result to a Go value
	goValue, err := result.Export()
	if err != nil {
		return nil, fmt.Errorf("failed to convert result to Go value: %w", err)
	}

	return goValue, nil
}

// EvaluateInObject processes all expressions in an object
func (e *JSExpressionEvaluator) EvaluateInObject(obj map[string]any, context map[string]any) (map[string]any, error) {
	result := make(map[string]any)

	for key, value := range obj {
		// Evaluate the key if it's an expression
		evaluatedKey := key
		if strings.HasPrefix(key, "${") && strings.HasSuffix(key, "}") {
			keyResult, err := e.Evaluate(key, context)
			if err != nil {
				return nil, fmt.Errorf("failed to evaluate key expression '%s': %w", key, err)
			}
			evaluatedKey = fmt.Sprintf("%v", keyResult)
		}

		// Evaluate the value based on its type
		var evaluatedValue any
		var err error

		switch v := value.(type) {
		case string:
			evaluatedValue, err = e.Evaluate(v, context)
			if err != nil {
				return nil, fmt.Errorf("failed to evaluate expression '%s': %w", v, err)
			}
		case map[string]any:
			evaluatedValue, err = e.EvaluateInObject(v, context)
			if err != nil {
				return nil, err
			}
		case []any:
			evaluatedArray := make([]any, len(v))
			for i, item := range v {
				if strItem, ok := item.(string); ok {
					evaluatedArray[i], err = e.Evaluate(strItem, context)
					if err != nil {
						return nil, fmt.Errorf("failed to evaluate expression '%s': %w", strItem, err)
					}
				} else if mapItem, ok := item.(map[string]any); ok {
					evaluatedArray[i], err = e.EvaluateInObject(mapItem, context)
					if err != nil {
						return nil, err
					}
				} else {
					evaluatedArray[i] = item
				}
			}
			evaluatedValue = evaluatedArray
		default:
			evaluatedValue = value
		}

		result[evaluatedKey] = evaluatedValue
	}

	return result, nil
}
