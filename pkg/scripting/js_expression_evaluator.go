// Package scripting provides JavaScript execution capabilities for flows.
package scripting

import (
    "fmt"
    "strings"

    "github.com/dop251/goja"
)

// JSExpressionEvaluator is an implementation of the ExpressionEvaluator interface using a JavaScript engine
type JSExpressionEvaluator struct {
    vm *goja.Runtime
}

// NewJSExpressionEvaluator creates a new JSExpressionEvaluator
func NewJSExpressionEvaluator() *JSExpressionEvaluator {
    vm := goja.New()
    return &JSExpressionEvaluator{vm: vm}
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
		e.setContextValue(key, value)
	}

	// Evaluate the expression
    result, err := e.vm.RunString(expr)
	if err != nil {
		return nil, fmt.Errorf("failed to evaluate expression '%s': %w", expr, err)
	}

	// Convert the result to a Go value
    goValue := result.Export()

	// Ensure consistent type conversion for numeric values
	switch v := goValue.(type) {
	case int:
		return float64(v), nil
	case int64:
		return float64(v), nil
	default:
		return goValue, nil
	}
}

// setContextValue sets a value in the JavaScript VM with special handling for certain types
func (e *JSExpressionEvaluator) setContextValue(key string, value any) {
	// Handle SecretsProxy specially to make secrets accessible in JavaScript
	if key == "secrets" {
		if secretsProxy, ok := value.(*SecretsProxy); ok {
			// Create a JavaScript-accessible secrets object
			secretsObj := make(map[string]any)

			// Get all available secret keys and create lazy getters
			if keys, err := secretsProxy.vault.List(secretsProxy.accountID); err == nil {
				for _, secretKey := range keys {
					secretKey := secretKey // capture for closure
					// Create a getter function for each secret
					secretsObj[secretKey] = func() any {
						if val, err := secretsProxy.Get(secretKey); err == nil {
							return val
						}
						return nil
					}()

					// Also try to get the value directly
					if val, err := secretsProxy.Get(secretKey); err == nil {
						secretsObj[secretKey] = val
					}
				}
			}
            e.vm.Set(key, secretsObj)
			return
		}
	}

	// Default handling for other values
    e.vm.Set(key, value)
}

// EvaluateInObject processes all expressions in an object
func (e *JSExpressionEvaluator) EvaluateInObject(obj map[string]any, context map[string]any) (map[string]any, error) {
	result := make(map[string]any)

	// Set up the context in the JavaScript VM with special handling
	for key, value := range context {
		e.setContextValue(key, value)
	}

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
					// Convert numeric types to float64 for consistency
					switch numVal := item.(type) {
					case int:
						evaluatedArray[i] = float64(numVal)
					case int64:
						evaluatedArray[i] = float64(numVal)
					default:
						evaluatedArray[i] = item
					}
				}
			}
			evaluatedValue = evaluatedArray
		case int:
			evaluatedValue = float64(v)
		case int64:
			evaluatedValue = float64(v)
		default:
			evaluatedValue = value
		}

		result[evaluatedKey] = evaluatedValue
	}

	return result, nil
}
