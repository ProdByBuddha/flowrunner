package scripting

// ExpressionEvaluator evaluates expressions in YAML
type ExpressionEvaluator interface {
	// Evaluate processes an expression string with the given context
	Evaluate(expression string, context map[string]any) (any, error)

	// EvaluateInObject processes all expressions in an object
	EvaluateInObject(obj map[string]any, context map[string]any) (map[string]any, error)
}

// ScriptEngine executes JavaScript code
type ScriptEngine interface {
	// Execute runs a JavaScript snippet with the given context
	Execute(script string, context map[string]any) (any, error)

	// ExecuteWithTimeout runs a JavaScript snippet with a timeout
	ExecuteWithTimeout(ctx any, script string, context map[string]any, timeout any) (any, error)
}
