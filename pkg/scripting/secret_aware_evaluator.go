// Package scripting provides JavaScript execution capabilities for flows.
package scripting

import (
	"strings"

	"github.com/tcmartin/flowrunner/pkg/auth"
)

// SecretAwareExpressionEvaluator extends JSExpressionEvaluator with secret access
type SecretAwareExpressionEvaluator struct {
	*JSExpressionEvaluator
	secretVault auth.SecretVault
}

// SetSecretVault sets the secret vault to use for secret access
func (e *SecretAwareExpressionEvaluator) SetSecretVault(vault auth.SecretVault) {
	e.secretVault = vault
}

// NewSecretAwareExpressionEvaluator creates a new SecretAwareExpressionEvaluator
func NewSecretAwareExpressionEvaluator(secretVault auth.SecretVault) *SecretAwareExpressionEvaluator {
	return &SecretAwareExpressionEvaluator{
		JSExpressionEvaluator: NewJSExpressionEvaluator(),
		secretVault:           secretVault,
	}
}

// Evaluate processes an expression string with the given context and access to secrets
func (e *SecretAwareExpressionEvaluator) Evaluate(expression string, context map[string]any) (any, error) {
	// Check if this is an expression that references secrets but no account ID is provided
	if strings.HasPrefix(expression, "${") && strings.HasSuffix(expression, "}") {
		expr := expression[2 : len(expression)-1]
		if strings.Contains(expr, "secrets") && context["accountID"] == nil {
			return nil, nil
		}
	}

	// Create a copy of the context to avoid modifying the original
	contextWithSecrets := make(map[string]any)
	for k, v := range context {
		contextWithSecrets[k] = v
	}

	// Add a secrets object to the context
	if e.secretVault != nil && context["accountID"] != nil {
		accountID, ok := context["accountID"].(string)
		if ok && accountID != "" {
			// Create a secrets object
			secretsObj := make(map[string]any)

			// Pre-load known secrets for this test
			// In a real implementation, we might want to lazy-load secrets as needed
			// but for simplicity, we'll pre-load the ones we know we'll need
			if keys, err := e.secretVault.List(accountID); err == nil {
				for _, key := range keys {
					if value, err := e.secretVault.Get(accountID, key); err == nil {
						secretsObj[key] = value
					}
				}
			}

			// Add the secrets object to the context
			contextWithSecrets["secrets"] = secretsObj
		}
	}

	// Use the parent evaluator with our enhanced context
	return e.JSExpressionEvaluator.Evaluate(expression, contextWithSecrets)
}

// EvaluateInObject processes all expressions in an object with secret access
func (e *SecretAwareExpressionEvaluator) EvaluateInObject(obj map[string]any, context map[string]any) (map[string]any, error) {
	// Create a copy of the context to avoid modifying the original
	contextWithSecrets := make(map[string]any)
	for k, v := range context {
		contextWithSecrets[k] = v
	}

	// Add a secrets object to the context
	if e.secretVault != nil && context["accountID"] != nil {
		accountID, ok := context["accountID"].(string)
		if ok && accountID != "" {
			// Create a secrets object
			secretsObj := make(map[string]any)

			// Pre-load known secrets for this test
			// In a real implementation, we might want to lazy-load secrets as needed
			// but for simplicity, we'll pre-load the ones we know we'll need
			if keys, err := e.secretVault.List(accountID); err == nil {
				for _, key := range keys {
					if value, err := e.secretVault.Get(accountID, key); err == nil {
						secretsObj[key] = value
					}
				}
			}

			// Add the secrets object to the context
			contextWithSecrets["secrets"] = secretsObj
		}
	}

	// Use the parent evaluator with our enhanced context
	return e.JSExpressionEvaluator.EvaluateInObject(obj, contextWithSecrets)
}
