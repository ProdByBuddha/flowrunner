package runtime

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
)

// TestAgentNodeCreation tests that agent node can be created and delegates to LLM node
func TestAgentNodeCreation(t *testing.T) {
	// Test agent node creation with minimal parameters
	node, err := NewAgentNodeWrapper(map[string]interface{}{
		"provider": "openai",
		"api_key":  "test_key",
		"model":    "gpt-4",
		"prompt":   "Hello, world!",
	})

	assert.NoError(t, err)
	assert.NotNil(t, node)

	// Verify it's a NodeWrapper
	wrapper, ok := node.(*NodeWrapper)
	assert.True(t, ok)
	assert.NotNil(t, wrapper)
}

// TestAgentNodeParameterHandling tests parameter handling and backwards compatibility
func TestAgentNodeParameterHandling(t *testing.T) {
	// Test with various parameter combinations
	testCases := []map[string]interface{}{
		{
			"provider": "openai",
			"api_key":  "test_key",
			"model":    "gpt-4",
			"prompt":   "Test prompt",
		},
		{
			"provider":    "anthropic",
			"api_key":     "test_key",
			"model":       "claude-3",
			"temperature": 0.5,
		},
		{
			"provider":   "openai",
			"api_key":    "test_key",
			"model":      "gpt-4",
			"max_tokens": 100,
			"tools": []interface{}{
				map[string]interface{}{
					"type": "function",
					"function": map[string]interface{}{
						"name":        "test_tool",
						"description": "A test tool",
						"parameters":  map[string]interface{}{},
					},
				},
			},
		},
	}

	for i, params := range testCases {
		t.Run(fmt.Sprintf("TestCase_%d", i), func(t *testing.T) {
			node, err := NewAgentNodeWrapper(params)
			assert.NoError(t, err)
			assert.NotNil(t, node)
		})
	}
}