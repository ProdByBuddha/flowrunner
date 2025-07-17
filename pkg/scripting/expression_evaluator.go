// Package scripting provides JavaScript execution capabilities for flows.
package scripting

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
)

// SimpleExpressionEvaluator is a basic implementation of the ExpressionEvaluator interface
type SimpleExpressionEvaluator struct{}

// NewSimpleExpressionEvaluator creates a new SimpleExpressionEvaluator
func NewSimpleExpressionEvaluator() *SimpleExpressionEvaluator {
	return &SimpleExpressionEvaluator{}
}

// Evaluate processes an expression string with the given context
func (e *SimpleExpressionEvaluator) Evaluate(expression string, context map[string]interface{}) (interface{}, error) {
	// Check if this is an expression
	if !strings.HasPrefix(expression, "${") || !strings.HasSuffix(expression, "}") {
		return expression, nil
	}

	// Extract the expression content
	expr := expression[2 : len(expression)-1]

	// Handle variable references
	if strings.Contains(expr, ".") {
		parts := strings.Split(expr, ".")
		current := context

		for i, part := range parts {
			if i == len(parts)-1 {
				// Last part, return the value
				if val, ok := current[part]; ok {
					return val, nil
				}
				return nil, fmt.Errorf("variable not found: %s", expr)
			}

			// Navigate to the next level
			if val, ok := current[part]; ok {
				if nextMap, ok := val.(map[string]interface{}); ok {
					current = nextMap
				} else {
					return nil, fmt.Errorf("cannot navigate path: %s is not a map", part)
				}
			} else {
				return nil, fmt.Errorf("variable not found: %s", expr)
			}
		}
	}

	// Simple variable reference
	if val, ok := context[expr]; ok {
		return val, nil
	}

	// Handle simple math expressions
	if result, err := e.evaluateMathExpression(expr); err == nil {
		return result, nil
	}

	return nil, fmt.Errorf("unknown expression: %s", expr)
}

// EvaluateInObject processes all expressions in an object
func (e *SimpleExpressionEvaluator) EvaluateInObject(obj map[string]interface{}, context map[string]interface{}) (map[string]interface{}, error) {
	result := make(map[string]interface{})

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
		var evaluatedValue interface{}
		var err error

		switch v := value.(type) {
		case string:
			evaluatedValue, err = e.Evaluate(v, context)
			if err != nil {
				return nil, fmt.Errorf("failed to evaluate expression '%s': %w", v, err)
			}
		case map[string]interface{}:
			evaluatedValue, err = e.EvaluateInObject(v, context)
			if err != nil {
				return nil, err
			}
		case []interface{}:
			evaluatedArray := make([]interface{}, len(v))
			for i, item := range v {
				if strItem, ok := item.(string); ok {
					evaluatedArray[i], err = e.Evaluate(strItem, context)
					if err != nil {
						return nil, fmt.Errorf("failed to evaluate expression '%s': %w", strItem, err)
					}
				} else if mapItem, ok := item.(map[string]interface{}); ok {
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

// evaluateMathExpression evaluates simple math expressions
func (e *SimpleExpressionEvaluator) evaluateMathExpression(expr string) (float64, error) {
	// This is a very simple implementation that only handles basic operations
	// In a real implementation, you would use a proper expression parser

	// Check for Math.random()
	if expr == "Math.random()" {
		return 0.5, nil // Placeholder for testing
	}

	// Simple addition
	if match := regexp.MustCompile(`^(\d+(?:\.\d+)?)\s*\+\s*(\d+(?:\.\d+)?)$`).FindStringSubmatch(expr); match != nil {
		a, _ := strconv.ParseFloat(match[1], 64)
		b, _ := strconv.ParseFloat(match[2], 64)
		return a + b, nil
	}

	// Simple subtraction
	if match := regexp.MustCompile(`^(\d+(?:\.\d+)?)\s*-\s*(\d+(?:\.\d+)?)$`).FindStringSubmatch(expr); match != nil {
		a, _ := strconv.ParseFloat(match[1], 64)
		b, _ := strconv.ParseFloat(match[2], 64)
		return a - b, nil
	}

	// Simple multiplication
	if match := regexp.MustCompile(`^(\d+(?:\.\d+)?)\s*\*\s*(\d+(?:\.\d+)?)$`).FindStringSubmatch(expr); match != nil {
		a, _ := strconv.ParseFloat(match[1], 64)
		b, _ := strconv.ParseFloat(match[2], 64)
		return a * b, nil
	}

	// Simple division
	if match := regexp.MustCompile(`^(\d+(?:\.\d+)?)\s*\/\s*(\d+(?:\.\d+)?)$`).FindStringSubmatch(expr); match != nil {
		a, _ := strconv.ParseFloat(match[1], 64)
		b, _ := strconv.ParseFloat(match[2], 64)
		if b == 0 {
			return 0, fmt.Errorf("division by zero")
		}
		return a / b, nil
	}

	return 0, fmt.Errorf("unsupported expression: %s", expr)
}
