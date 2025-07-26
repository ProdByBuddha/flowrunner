// Package scripting provides JavaScript execution capabilities for flows.
package scripting

import (
	"strings"

	"github.com/tcmartin/flowrunner/pkg/auth"
)

// SecretsProxy provides dynamic access to secrets for JavaScript evaluation
type SecretsProxy struct {
	vault     auth.SecretVault
	accountID string
	cache     map[string]any
}

// Get retrieves a secret value by key
func (sp *SecretsProxy) Get(key string) (any, error) {
	// Check cache first
	if sp.cache == nil {
		sp.cache = make(map[string]any)
	}

	if value, exists := sp.cache[key]; exists {
		return value, nil
	}

	// Fetch from vault
	value, err := sp.vault.Get(sp.accountID, key)
	if err != nil {
		return nil, err
	}

	// Cache the value
	sp.cache[key] = value
	return value, nil
}

// GetField retrieves a specific field from a structured secret
func (sp *SecretsProxy) GetField(key, field string) (any, error) {
	// For structured secrets, we might need to implement field-level access
	// For now, just return the whole secret
	return sp.Get(key)
}

// Has checks if a secret exists
func (sp *SecretsProxy) Has(key string) bool {
	_, err := sp.Get(key)
	return err == nil
}

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

	// Create enhanced context with secrets and node results
	enhancedContext := e.createEnhancedContext(context)

	// Use the parent evaluator with our enhanced context
	return e.JSExpressionEvaluator.Evaluate(expression, enhancedContext)
}

// createEnhancedContext creates a context with secrets and node results
func (e *SecretAwareExpressionEvaluator) createEnhancedContext(context map[string]any) map[string]any {
	// Create a copy of the context to avoid modifying the original
	enhancedContext := make(map[string]any)
	for k, v := range context {
		enhancedContext[k] = v
	}

	// Add secrets object to the context
	if e.secretVault != nil && context["accountID"] != nil {
		if accountID, ok := context["accountID"].(string); ok && accountID != "" {
			// Create a dynamic secrets object that resolves secrets on-demand
			secretsObj := &SecretsProxy{
				vault:     e.secretVault,
				accountID: accountID,
			}
			enhancedContext["secrets"] = secretsObj
		}
	}

	// Add node results to context for easy access
	if flowContext, ok := context["_flow_context"].(map[string]any); ok {
		// Add node results as 'results' object
		if nodeResults, ok := flowContext["node_results"].(map[string]any); ok {
			enhancedContext["results"] = nodeResults
		}

		// Add shared context data as 'shared' object
		if sharedData, ok := flowContext["shared_data"].(map[string]any); ok {
			enhancedContext["shared"] = sharedData
		}
	}

	return enhancedContext
}

// EvaluateInObject processes all expressions in an object with secret access
func (e *SecretAwareExpressionEvaluator) EvaluateInObject(obj map[string]any, context map[string]any) (map[string]any, error) {
	// Create enhanced context with secrets and node results
	enhancedContext := e.createEnhancedContext(context)

	// Use the parent evaluator with our enhanced context
	return e.JSExpressionEvaluator.EvaluateInObject(obj, enhancedContext)
}
