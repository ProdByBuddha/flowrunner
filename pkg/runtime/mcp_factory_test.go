package runtime

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/tcmartin/flowlib"
)

// TestMCPNodeFactory_Registration tests that MCP nodes are properly registered
func TestMCPNodeFactory_Registration(t *testing.T) {
	coreTypes := CoreNodeTypes()
	
	// Test that MCP node is registered
	mcpFactory, exists := coreTypes["mcp"]
	assert.True(t, exists)
	assert.NotNil(t, mcpFactory)
	
	// Test node creation through factory
	params := map[string]interface{}{
		"connectionType": "http",
		"operation":      "listTools",
		"url":            "http://example.com",
	}
	
	node, err := mcpFactory(params)
	require.NoError(t, err)
	assert.NotNil(t, node)
	
	// Verify it's a flowlib.Node
	_, ok := node.(flowlib.Node)
	assert.True(t, ok)
}

// TestMCPNodeFactory_InvalidParams tests error handling for invalid parameters
func TestMCPNodeFactory_InvalidParams(t *testing.T) {
	coreTypes := CoreNodeTypes()
	mcpFactory := coreTypes["mcp"]
	
	tests := []struct {
		name        string
		params      map[string]interface{}
		expectError bool
	}{
		{
			name: "valid http params",
			params: map[string]interface{}{
				"connectionType": "http",
				"operation":      "listTools",
				"url":            "http://example.com",
			},
			expectError: false,
		},
		{
			name: "valid cmd params",
			params: map[string]interface{}{
				"connectionType": "cmd",
				"operation":      "listTools",
				"command":        "echo",
			},
			expectError: false,
		},
		{
			name: "valid raw body params",
			params: map[string]interface{}{
				"connectionType": "http",
				"url":            "http://example.com",
				"rawBody":        `{"method":"tools/list","params":{}}`,
			},
			expectError: false,
		},
		{
			name: "missing connection type",
			params: map[string]interface{}{
				"operation": "listTools",
				"url":       "http://example.com",
			},
			expectError: false, // Should use default "cmd"
		},
		{
			name: "empty params",
			params: map[string]interface{}{},
			expectError: true, // Should fail validation during execution
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			node, err := mcpFactory(tt.params)
			
			if tt.expectError {
				// For some cases, error might occur during execution rather than creation
				if err == nil {
					// Try to run the node to see if it fails
					_, runErr := node.Run(map[string]interface{}{
						"params": tt.params,
						"input":  map[string]interface{}{},
					})
					assert.Error(t, runErr, "Expected error during node execution")
				} else {
					assert.Error(t, err, "Expected error during node creation")
				}
			} else {
				assert.NoError(t, err, "Should not error with valid params")
				assert.NotNil(t, node, "Node should be created")
			}
		})
	}
}

// TestMCPNodeFactory_UnknownNodeType tests handling of unknown node types
func TestMCPNodeFactory_UnknownNodeType(t *testing.T) {
	coreTypes := CoreNodeTypes()
	
	unknownFactory, exists := coreTypes["unknown_type"]
	assert.False(t, exists)
	assert.Nil(t, unknownFactory)
}

// TestMCPNodeFactory_AllSupportedTypes tests that all expected types are supported
func TestMCPNodeFactory_AllSupportedTypes(t *testing.T) {
	coreTypes := CoreNodeTypes()
	
	// Get all supported type names
	var supportedTypes []string
	for nodeType := range coreTypes {
		supportedTypes = append(supportedTypes, nodeType)
	}
	
	// Verify MCP is in the list of supported types
	assert.Contains(t, supportedTypes, "mcp")
	
	// Verify other core types are still supported
	expectedTypes := []string{
		"http.request",
		"transform", 
		"condition",
		"webhook",
		"store",
		"llm",
		"mcp", // Our new type
	}
	
	for _, expectedType := range expectedTypes {
		assert.Contains(t, supportedTypes, expectedType, 
			"Expected node type %s to be supported", expectedType)
	}
}

// TestMCPNodeFactory_ParameterValidation tests parameter validation
func TestMCPNodeFactory_ParameterValidation(t *testing.T) {
	coreTypes := CoreNodeTypes()
	mcpFactory := coreTypes["mcp"]
	
	// Test various parameter combinations
	testCases := []struct {
		name   string
		params map[string]interface{}
		valid  bool
	}{
		{
			name: "http with all required params",
			params: map[string]interface{}{
				"connectionType": "http",
				"operation":      "listTools",
				"url":            "https://api.example.com/mcp",
			},
			valid: true,
		},
		{
			name: "http with headers",
			params: map[string]interface{}{
				"connectionType": "http",
				"operation":      "executeTool",
				"url":            "https://api.example.com/mcp",
				"toolName":       "test_tool",
				"toolParameters": `{"param": "value"}`,
				"headers": map[string]interface{}{
					"Authorization": "Bearer token",
					"Content-Type":  "application/json",
				},
			},
			valid: true,
		},
		{
			name: "cmd with command and args",
			params: map[string]interface{}{
				"connectionType": "cmd",
				"operation":      "listTools",
				"command":        "npx",
				"args":           []string{"mcp-server", "--config", "config.json"},
			},
			valid: true,
		},
		{
			name: "sse with url and post endpoint",
			params: map[string]interface{}{
				"connectionType":        "sse",
				"operation":             "listTools",
				"url":                   "https://api.example.com/sse",
				"messagesPostEndpoint":  "https://api.example.com/messages",
			},
			valid: true,
		},
		{
			name: "raw body override",
			params: map[string]interface{}{
				"connectionType": "http",
				"url":            "https://api.example.com/mcp",
				"rawBody":        `{"method": "tools/list", "params": {}}`,
			},
			valid: true,
		},
		{
			name: "template support",
			params: map[string]interface{}{
				"connectionType": "http",
				"url":            "https://api.example.com/mcp",
				"rawBodyTemplate": `{"method": "tools/call", "params": {"name": "{{.input.toolName}}"}}`,
				"headersTemplate": map[string]interface{}{
					"Authorization": "Bearer {{.input.token}}",
				},
			},
			valid: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			node, err := mcpFactory(tc.params)
			
			if tc.valid {
				assert.NoError(t, err, "Should create node successfully")
				assert.NotNil(t, node, "Node should not be nil")
				
				// Verify the node implements flowlib.Node
				_, ok := node.(flowlib.Node)
				assert.True(t, ok, "Node should implement flowlib.Node interface")
			} else {
				// Note: Some validation might happen during execution rather than creation
				if err == nil {
					assert.NotNil(t, node, "If no creation error, node should exist")
				}
			}
		})
	}
}

// TestMCPNodeFactory_Concurrency tests concurrent node creation
func TestMCPNodeFactory_Concurrency(t *testing.T) {
	coreTypes := CoreNodeTypes()
	mcpFactory := coreTypes["mcp"]
	
	// Test concurrent node creation
	const numGoroutines = 10
	const nodesPerGoroutine = 5
	
	results := make(chan error, numGoroutines*nodesPerGoroutine)
	
	for i := 0; i < numGoroutines; i++ {
		go func(routineID int) {
			for j := 0; j < nodesPerGoroutine; j++ {
				params := map[string]interface{}{
					"connectionType": "http",
					"operation":      "listTools",
					"url":            "https://api.example.com/mcp",
				}
				
				node, err := mcpFactory(params)
				if err != nil {
					results <- err
					return
				}
				
				if node == nil {
					results <- assert.AnError
					return
				}
				
				results <- nil
			}
		}(i)
	}
	
	// Collect results
	for i := 0; i < numGoroutines*nodesPerGoroutine; i++ {
		err := <-results
		assert.NoError(t, err, "Concurrent node creation should not fail")
	}
}

// BenchmarkMCPNodeFactory_CreateNode benchmarks node creation performance
func BenchmarkMCPNodeFactory_CreateNode(b *testing.B) {
	coreTypes := CoreNodeTypes()
	mcpFactory := coreTypes["mcp"]
	params := map[string]interface{}{
		"connectionType": "http",
		"operation":      "listTools",
		"url":            "https://api.example.com/mcp",
	}
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		node, err := mcpFactory(params)
		if err != nil {
			b.Fatal(err)
		}
		if node == nil {
			b.Fatal("Node should not be nil")
		}
	}
}

// BenchmarkMCPNodeFactory_GetSupportedTypes benchmarks getting supported types
func BenchmarkMCPNodeFactory_GetSupportedTypes(b *testing.B) {
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		coreTypes := CoreNodeTypes()
		if len(coreTypes) == 0 {
			b.Fatal("Should return supported types")
		}
	}
}