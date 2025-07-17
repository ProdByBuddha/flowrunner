// Package scripting provides JavaScript execution capabilities for flows.
package scripting

import (
	"fmt"
	"regexp"
	"strings"
)

// SimpleExpressionEvaluator is a basic implementation of the ExpressionEvaluator interface
type SimpleExpressionEvaluator struct {
	// Regular expression for matching expressions
	exprRegex *regexp.Regexp
}

// NewSimpleExpressionEvaluator creates a new expression evaluator
func NewSimpleExpressionEvaluator() *SimpleExpressionEvaluator {
	return &SimpleExpressionEvaluator{
		exprRegex: regexp.MustCompile(`\${([^}]+)}`),
	}
}

// Evaluate processes an expression string with the given context
func (e *SimpleExpressionEvaluator) Evaluate(expression string, context map[string]interface{}) (interface{}, error) {
	// Check if the expression is a simple reference
	if strings.HasPrefix(expression, "${") && strings.HasSuffix(expression, "}") {
		// Extract the reference path
		path := expression[2 : len(expression)-1]
		return e.resolvePathReference(path, context)
	}

	// Replace all expressions in the string
	result := e.exprRegex.ReplaceAllStringFunc(expression, func(match string) string {
		// Extract the reference path
		path := match[2 : len(match)-1]
		value, err := e.resolvePathReference(path, context)
		if err != nil {
			// Return the original expression if there's an error
			return match
		}
		return fmt.Sprintf("%v", value)
	})

	return result, nil
}

// EvaluateInObject processes all expressions in an object
func (e *SimpleExpressionEvaluator) EvaluateInObject(obj map[string]interface{}, context map[string]interface{}) (map[string]interface{}, error) {
	result := make(map[string]interface{})

	for key, value := range obj {
		// Evaluate the key if it contains expressions
		evaluatedKey := key
		if strings.Contains(key, "${") {
			keyResult, err := e.Evaluate(key, context)
			if err != nil {
				return nil, fmt.Errorf("failed to evaluate key '%s': %w", key, err)
			}
			evaluatedKey = fmt.Sprintf("%v", keyResult)
		}

		// Evaluate the value based on its type
		var evaluatedValue interface{}
		var err error

		switch v := value.(type) {
		case string:
			if strings.Contains(v, "${") {
				evaluatedValue, err = e.Evaluate(v, context)
				if err != nil {
					return nil, fmt.Errorf("failed to evaluate value for key '%s': %w", key, err)
				}
			} else {
				evaluatedValue = v
			}
		case map[string]interface{}:
			evaluatedValue, err = e.EvaluateInObject(v, context)
			if err != nil {
				return nil, fmt.Errorf("failed to evaluate object for key '%s': %w", key, err)
			}
		case []interface{}:
			evaluatedValue, err = e.evaluateArray(v, context)
			if err != nil {
				return nil, fmt.Errorf("failed to evaluate array for key '%s': %w", key, err)
			}
		default:
			evaluatedValue = v
		}

		result[evaluatedKey] = evaluatedValue
	}

	return result, nil
}

// evaluateArray processes all expressions in an array
func (e *SimpleExpressionEvaluator) evaluateArray(arr []interface{}, context map[string]interface{}) ([]interface{}, error) {
	result := make([]interface{}, len(arr))

	for i, value := range arr {
		var evaluatedValue interface{}
		var err error

		switch v := value.(type) {
		case string:
			if strings.Contains(v, "${") {
				evaluatedValue, err = e.Evaluate(v, context)
				if err != nil {
					return nil, fmt.Errorf("failed to evaluate array item %d: %w", i, err)
				}
			} else {
				evaluatedValue = v
			}
		case map[string]interface{}:
			evaluatedValue, err = e.EvaluateInObject(v, context)
			if err != nil {
				return nil, fmt.Errorf("failed to evaluate object in array at index %d: %w", i, err)
			}
		case []interface{}:
			evaluatedValue, err = e.evaluateArray(v, context)
			if err != nil {
				return nil, fmt.Errorf("failed to evaluate array at index %d: %w", i, err)
			}
		default:
			evaluatedValue = v
		}

		result[i] = evaluatedValue
	}

	return result, nil
}

// resolvePathReference resolves a path reference in the context
func (e *SimpleExpressionEvaluator) resolvePathReference(path string, context map[string]interface{}) (interface{}, error) {
	// Handle special references
	if strings.HasPrefix(path, "secrets.") {
		// In a real implementation, this would access the secret vault
		return fmt.Sprintf("[SECRET:%s]", path[8:]), nil
	}

	// Split the path into parts
	parts := strings.Split(path, ".")
	var current interface{} = context

	// Navigate through the path
	for i, part := range parts {
		// Check if the current value is a map
		currentMap, ok := current.(map[string]interface{})
		if !ok {
			return nil, fmt.Errorf("cannot access '%s' in '%s': not an object", part, strings.Join(parts[:i], "."))
		}

		// Get the next value
		next, ok := currentMap[part]
		if !ok {
			return nil, fmt.Errorf("reference not found: %s", path)
		}

		current = next
	}

	return current, nil
}
