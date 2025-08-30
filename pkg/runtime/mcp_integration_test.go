package runtime

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestMCPIntegration_NodeChaining tests chaining MCP nodes together
func TestMCPIntegration_NodeChaining(t *testing.T) {
	// Create mock MCP server
	mockServer := NewMockMCPServer()
	defer mockServer.Close()

	// Test 1: List tools
	listParams := map[string]interface{}{
		"connectionType": "http",
		"operation":      "listTools",
		"url":            mockServer.URL(),
	}

	listNode, err := NewMCPNodeWrapper(listParams)
	require.NoError(t, err)

	listInput := map[string]interface{}{
		"params": listParams,
		"input":  map[string]interface{}{},
	}

	listAction, err := listNode.Run(listInput)
	require.NoError(t, err)
	require.Equal(t, "default", listAction)

	// Get result from shared context
	listResultData := listInput["result"]
	
	var listResponse MCPResponse
	if resultStr, ok := listResultData.(string); ok {
		err = json.Unmarshal([]byte(resultStr), &listResponse)
		require.NoError(t, err)
	} else {
		resultBytes, _ := json.Marshal(listResultData)
		err = json.Unmarshal(resultBytes, &listResponse)
		require.NoError(t, err)
	}

	// Verify tools are available - handle nested structure
	require.NotNil(t, listResponse.Result)
	
	// The result is nested: result.result.tools
	resultMap, ok := listResponse.Result.(map[string]interface{})
	require.True(t, ok, "Result should be a map")
	
	innerResult, exists := resultMap["result"]
	require.True(t, exists, "Inner result should exist")
	
	resultObj, ok := innerResult.(map[string]interface{})
	require.True(t, ok, "Inner result should be a map")
	
	toolsInterface, exists := resultObj["tools"]
	require.True(t, exists, "Tools should exist")
	require.NotNil(t, toolsInterface, "Tools should not be nil")
	
	tools, ok := toolsInterface.([]interface{})
	require.True(t, ok, "Tools should be an array")
	require.Greater(t, len(tools), 0)

	// Test 2: Execute a tool from the list
	execParams := map[string]interface{}{
		"connectionType": "http",
		"operation":      "executeTool",
		"url":            mockServer.URL(),
		"toolName":       "get_weather",
		"toolParameters": `{"location": "Integration Test City"}`,
	}

	execNode, err := NewMCPNodeWrapper(execParams)
	require.NoError(t, err)

	execInput := map[string]interface{}{
		"params": execParams,
		"input":  map[string]interface{}{},
	}

	execAction, err := execNode.Run(execInput)
	require.NoError(t, err)
	require.Equal(t, "default", execAction)

	// Get result from shared context
	execResultData := execInput["result"]
	
	var execResponse MCPResponse
	if resultStr, ok := execResultData.(string); ok {
		err = json.Unmarshal([]byte(resultStr), &execResponse)
		require.NoError(t, err)
	} else {
		resultBytes, _ := json.Marshal(execResultData)
		err = json.Unmarshal(resultBytes, &execResponse)
		require.NoError(t, err)
	}

	// Verify execution result - handle nested structure
	require.NotNil(t, execResponse.Result)
	
	// The result is nested: result.result.temperature
	execResultMap, ok := execResponse.Result.(map[string]interface{})
	require.True(t, ok, "Result should be a map")
	
	execInnerResult, exists := execResultMap["result"]
	require.True(t, exists, "Inner result should exist")
	
	weatherData, ok := execInnerResult.(map[string]interface{})
	require.True(t, ok, "Weather data should be a map")
	
	assert.Contains(t, weatherData, "temperature")
	assert.Equal(t, "22°C", weatherData["temperature"])
}

// TestMCPIntegration_MultipleTools tests executing multiple tools in sequence
func TestMCPIntegration_MultipleTools(t *testing.T) {
	// Create mock MCP server
	mockServer := NewMockMCPServer()
	defer mockServer.Close()

	// Test executing multiple different tools
	tools := []string{"get_weather", "search_web", "get_time"}
	results := make([]interface{}, len(tools))

	for i, toolName := range tools {
		params := map[string]interface{}{
			"connectionType": "http",
			"operation":      "executeTool",
			"url":            mockServer.URL(),
			"toolName":       toolName,
			"toolParameters": `{"test": "parameter"}`,
		}

		node, err := NewMCPNodeWrapper(params)
		require.NoError(t, err)

		input := map[string]interface{}{
			"params": params,
			"input":  map[string]interface{}{},
		}

		action, err := node.Run(input)
		require.NoError(t, err)
		require.Equal(t, "default", action)

		// Store the actual result from shared context
		results[i] = input["result"]
	}

	// Verify all tools executed successfully
	for i, result := range results {
		resultMap := result.(map[string]interface{})
		resultData := resultMap["result"]
		
		var response MCPResponse
		if resultStr, ok := resultData.(string); ok {
			err := json.Unmarshal([]byte(resultStr), &response)
			require.NoError(t, err)
		} else {
			resultBytes, _ := json.Marshal(resultData)
			err := json.Unmarshal(resultBytes, &response)
			require.NoError(t, err)
		}

		require.NotNil(t, response.Result, "Tool %s should return result", tools[i])
		
		// Verify tool-specific results
		toolResult := response.Result.(map[string]interface{})
		switch tools[i] {
		case "get_weather":
			assert.Contains(t, toolResult, "temperature")
		case "search_web":
			assert.Contains(t, toolResult, "results")
		case "get_time":
			assert.Contains(t, toolResult, "time")
		}
	}
}

// TestMCPIntegration_ErrorRecovery tests error handling and recovery
func TestMCPIntegration_ErrorRecovery(t *testing.T) {
	// Create mock MCP server
	mockServer := NewMockMCPServer()
	defer mockServer.Close()

	// Test 1: Try to execute non-existent tool (should fail)
	failParams := map[string]interface{}{
		"connectionType": "http",
		"operation":      "executeTool",
		"url":            mockServer.URL(),
		"toolName":       "nonexistent_tool",
		"toolParameters": `{}`,
	}

	failNode, err := NewMCPNodeWrapper(failParams)
	require.NoError(t, err)

	failInput := map[string]interface{}{
		"params": failParams,
		"input":  map[string]interface{}{},
	}

	action, err := failNode.Run(failInput)
	require.NoError(t, err) // Node execution succeeds
	require.Equal(t, "default", action)
	
	// But the result should contain an MCP error
	resultData := failInput["result"]
	require.NotNil(t, resultData)
	
	resultMap := resultData.(map[string]interface{})
	resultInner := resultMap["result"]
	
	var response MCPResponse
	if resultStr, ok := resultInner.(string); ok {
		err := json.Unmarshal([]byte(resultStr), &response)
		require.NoError(t, err)
	} else {
		resultBytes, _ := json.Marshal(resultInner)
		err := json.Unmarshal(resultBytes, &response)
		require.NoError(t, err)
	}
	
	// Should have an error in the MCP response
	require.NotNil(t, response.Error)
	assert.Contains(t, response.Error.Message, "Tool not found")

	// Test 2: After failure, try a valid tool (should succeed)
	successParams := map[string]interface{}{
		"connectionType": "http",
		"operation":      "executeTool",
		"url":            mockServer.URL(),
		"toolName":       "get_weather",
		"toolParameters": `{"location": "Recovery Test"}`,
	}

	successNode, err := NewMCPNodeWrapper(successParams)
	require.NoError(t, err)

	successInput := map[string]interface{}{
		"params": successParams,
		"input":  map[string]interface{}{},
	}

	action, err = successNode.Run(successInput)
	require.NoError(t, err) // Should succeed
	require.Equal(t, "default", action)

	// Get result from shared context
	resultData = successInput["result"]
	
	if resultStr, ok := resultData.(string); ok {
		err = json.Unmarshal([]byte(resultStr), &response)
		require.NoError(t, err)
	} else {
		resultBytes, _ := json.Marshal(resultData)
		err = json.Unmarshal(resultBytes, &response)
		require.NoError(t, err)
	}

	require.NotNil(t, response.Result)
	resultObj := response.Result.(map[string]interface{})
	
	// Handle nested structure: result.result.temperature
	innerResult := resultObj["result"]
	weatherData := innerResult.(map[string]interface{})
	assert.Contains(t, weatherData, "temperature")
}