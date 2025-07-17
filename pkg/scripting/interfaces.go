// Package scripting provides JavaScript execution capabilities for flows.
package scripting

import (
	"context"
	"time"
)

// ScriptEngine executes JavaScript code
type ScriptEngine interface {
	// Execute runs a JavaScript snippet with the given context
	Execute(script string, context map[string]interface{}) (interface{}, error)

	// ExecuteWithTimeout runs a JavaScript snippet with a timeout
	ExecuteWithTimeout(ctx context.Context, script string, context map[string]interface{}, timeout time.Duration) (interface{}, error)

	// RegisterFunction makes a Go function available to JavaScript
	RegisterFunction(name string, function interface{}) error

	// RegisterObject makes a Go object available to JavaScript
	RegisterObject(name string, object interface{}) error
}

// ScriptContext provides access to flow context from JavaScript
type ScriptContext interface {
	// GetValue retrieves a value from the context
	GetValue(key string) (interface{}, error)

	// SetValue sets a value in the context
	SetValue(key string, value interface{}) error

	// GetSecret retrieves a secret from the vault
	GetSecret(key string) (string, error)

	// Log writes a log message
	Log(level string, message string, data map[string]interface{}) error
}

// ExpressionEvaluator evaluates expressions in YAML
type ExpressionEvaluator interface {
	// Evaluate processes an expression string with the given context
	Evaluate(expression string, context map[string]interface{}) (interface{}, error)

	// EvaluateInObject processes all expressions in an object
	EvaluateInObject(obj map[string]interface{}, context map[string]interface{}) (map[string]interface{}, error)
}
