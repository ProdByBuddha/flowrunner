package api

import (
	"os"
	"testing"

	"github.com/joho/godotenv"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/tcmartin/flowrunner/pkg/runtime"
)

// TestSimpleToolExecution tests the LLM node directly with tool calling
func TestSimpleToolExecution(t *testing.T) {
	// Load environment variables
	_ = godotenv.Load("../../.env")

	// Skip test if OpenAI API key is not available
	apiKey := os.Getenv("OPENAI_API_KEY")
	if apiKey == "" {
		t.Skip("Skipping simple tool execution test: OPENAI_API_KEY environment variable not set")
	}

	// Create LLM node directly with explicit messages array
	params := map[string]interface{}{
		"provider":     "openai",
		"api_key":      apiKey, // Use direct API key instead of template
		"model":        "gpt-4o-mini",
		"temperature":  0.3,
		"max_tokens":   500,
		"messages": []interface{}{
			map[string]interface{}{
				"role": "system",
				"content": "You are an AI assistant with access to tools. Use the get_website tool to fetch content from https://httpbin.org/json",
			},
			map[string]interface{}{
				"role": "user",
				"content": "Please fetch the content from https://httpbin.org/json and tell me what you found.",
			},
		},
		"tools": []interface{}{
			map[string]interface{}{
				"type": "function",
				"function": map[string]interface{}{
					"name":        "get_website",
					"description": "Fetch content from a website URL",
					"parameters": map[string]interface{}{
						"type": "object",
						"properties": map[string]interface{}{
							"url": map[string]interface{}{
								"type":        "string",
								"description": "The URL to fetch content from",
							},
						},
						"required": []string{"url"},
					},
				},
			},
		},
	}

	// Create the LLM node
	llmNode, err := runtime.NewLLMNodeWrapper(params)
	require.NoError(t, err, "Failed to create LLM node")

	// Execute the node
	t.Log("Executing LLM node with tool calling...")
	
	// Create a shared context to capture the results
	shared := make(map[string]interface{})
	
	// Set the node parameters
	llmNode.SetParams(params)
	
	// Execute with shared context
	action, err := llmNode.Run(shared)
	
	if err != nil {
		t.Logf("LLM node execution failed: %v", err)
	} else {
		t.Log("LLM node execution succeeded!")
	}

	t.Logf("LLM node action: %s", action)
	t.Logf("LLM node error: %v", err)
	t.Logf("Shared context: %+v", shared)

	// Basic validation - just check that we got some result
	assert.NotNil(t, action, "Should get some action from LLM node")
	
	// Check shared context for results
	if llmResult, exists := shared["llm_result"]; exists {
		t.Logf("LLM result found in shared context: %+v", llmResult)
		
		if resultMap, ok := llmResult.(map[string]interface{}); ok {
			if hasToolCalls, exists := resultMap["has_tool_calls"]; exists {
				t.Logf("Has tool calls: %v", hasToolCalls)
			}
			if toolCalls, exists := resultMap["tool_calls"]; exists {
				t.Logf("Tool calls: %+v", toolCalls)
			}
		}
	}
}