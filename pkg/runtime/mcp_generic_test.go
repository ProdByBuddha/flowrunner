package runtime

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// GenericMCPServer represents any MCP server implementation
type GenericMCPServer struct {
	server *httptest.Server
	tools  []GenericTool
}

// GenericTool represents a tool that any MCP server might provide
type GenericTool struct {
	Name        string                 `json:"name"`
	Description string                 `json:"description"`
	InputSchema map[string]interface{} `json:"inputSchema,omitempty"`
}

// GenericMCPResponse represents any MCP server response format
type GenericMCPResponse struct {
	Result interface{} `json:"result,omitempty"`
	Error  interface{} `json:"error,omitempty"`
}

// NewGenericMCPServer creates a server that mimics any MCP server behavior
func NewGenericMCPServer(tools []GenericTool) *GenericMCPServer {
	server := &GenericMCPServer{tools: tools}
	
	server.server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}

		var request map[string]interface{}
		if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
			http.Error(w, "Invalid JSON", http.StatusBadRequest)
			return
		}

		w.Header().Set("Content-Type", "application/json")

		method, ok := request["method"].(string)
		if !ok {
			response := GenericMCPResponse{
				Error: map[string]interface{}{
					"code":    -32600,
					"message": "Invalid Request",
				},
			}
			json.NewEncoder(w).Encode(response)
			return
		}

		switch method {
		case "tools/list":
			server.handleToolsList(w, request)
		case "tools/call":
			server.handleToolCall(w, request)
		default:
			response := GenericMCPResponse{
				Error: map[string]interface{}{
					"code":    -32601,
					"message": fmt.Sprintf("Method not found: %s", method),
				},
			}
			json.NewEncoder(w).Encode(response)
		}
	}))
	
	return server
}

func (g *GenericMCPServer) handleToolsList(w http.ResponseWriter, request map[string]interface{}) {
	response := GenericMCPResponse{
		Result: map[string]interface{}{
			"tools": g.tools,
		},
	}
	json.NewEncoder(w).Encode(response)
}

func (g *GenericMCPServer) handleToolCall(w http.ResponseWriter, request map[string]interface{}) {
	params, ok := request["params"].(map[string]interface{})
	if !ok {
		response := GenericMCPResponse{
			Error: map[string]interface{}{
				"code":    -32602,
				"message": "Invalid params",
			},
		}
		json.NewEncoder(w).Encode(response)
		return
	}

	toolName, ok := params["name"].(string)
	if !ok {
		response := GenericMCPResponse{
			Error: map[string]interface{}{
				"code":    -32602,
				"message": "Missing tool name",
			},
		}
		json.NewEncoder(w).Encode(response)
		return
	}

	// Find the tool
	var tool *GenericTool
	for _, t := range g.tools {
		if t.Name == toolName {
			tool = &t
			break
		}
	}

	if tool == nil {
		response := GenericMCPResponse{
			Error: map[string]interface{}{
				"code":    -32602,
				"message": fmt.Sprintf("Tool not found: %s", toolName),
			},
		}
		json.NewEncoder(w).Encode(response)
		return
	}

	// Generate generic response based on tool name
	result := g.generateToolResult(toolName, params)
	
	response := GenericMCPResponse{Result: result}
	json.NewEncoder(w).Encode(response)
}

func (g *GenericMCPServer) generateToolResult(toolName string, params map[string]interface{}) interface{} {
	// Generate results that any MCP server might return
	switch {
	case strings.Contains(toolName, "weather"):
		return map[string]interface{}{
			"temperature": "20°C",
			"humidity":    "65%",
			"condition":   "Partly cloudy",
			"timestamp":   time.Now().Format(time.RFC3339),
		}
	case strings.Contains(toolName, "search"):
		return map[string]interface{}{
			"query":   extractParam(params, "query", "default search"),
			"results": []map[string]interface{}{
				{"title": "Generic Result 1", "url": "https://example1.com"},
				{"title": "Generic Result 2", "url": "https://example2.com"},
			},
			"total": 2,
		}
	case strings.Contains(toolName, "time"):
		return map[string]interface{}{
			"current_time": time.Now().Format(time.RFC3339),
			"timezone":     extractParam(params, "timezone", "UTC"),
			"unix":         time.Now().Unix(),
		}
	case strings.Contains(toolName, "calculate"):
		return map[string]interface{}{
			"expression": extractParam(params, "expression", "1+1"),
			"result":     2,
			"type":       "number",
		}
	case strings.Contains(toolName, "translate"):
		return map[string]interface{}{
			"original":    extractParam(params, "text", "hello"),
			"translated":  "hola",
			"from":        extractParam(params, "from", "en"),
			"to":          extractParam(params, "to", "es"),
		}
	default:
		return map[string]interface{}{
			"tool":      toolName,
			"executed":  true,
			"timestamp": time.Now().Format(time.RFC3339),
			"params":    params,
		}
	}
}

func (g *GenericMCPServer) Close() {
	if g.server != nil {
		g.server.Close()
	}
}

func (g *GenericMCPServer) URL() string {
	return g.server.URL
}

// Helper functions
func extractParam(params map[string]interface{}, key, defaultValue string) string {
	if args, ok := params["arguments"].(map[string]interface{}); ok {
		if val, exists := args[key]; exists {
			if str, ok := val.(string); ok {
				return str
			}
		}
	}
	return defaultValue
}

// TestMCPNode_GenericWeatherServer tests MCP client with a weather-focused server
func TestMCPNode_GenericWeatherServer(t *testing.T) {
	// Create a weather-focused MCP server
	weatherTools := []GenericTool{
		{
			Name:        "get_current_weather",
			Description: "Get current weather for a location",
			InputSchema: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"location": map[string]interface{}{
						"type":        "string",
						"description": "City name or coordinates",
					},
				},
				"required": []string{"location"},
			},
		},
		{
			Name:        "get_weather_forecast",
			Description: "Get weather forecast for multiple days",
			InputSchema: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"location": map[string]interface{}{
						"type":        "string",
						"description": "City name or coordinates",
					},
					"days": map[string]interface{}{
						"type":        "number",
						"description": "Number of days to forecast",
					},
				},
				"required": []string{"location"},
			},
		},
	}

	server := NewGenericMCPServer(weatherTools)
	defer server.Close()

	// Test listing weather tools
	params := map[string]interface{}{
		"connectionType": "http",
		"operation":      "listTools",
		"url":            server.URL(),
	}

	node, err := NewMCPNodeWrapper(params)
	require.NoError(t, err)

	// Create shared context to capture results
	shared := make(map[string]interface{})
	
	action, err := node.Run(shared)
	require.NoError(t, err)
	require.Equal(t, "default", action)

	// Verify weather tools are listed
	resultData, exists := shared["result"]
	require.True(t, exists)
	
	// The result is nested: result.result.result.tools
	resultMap, ok := resultData.(map[string]interface{})
	require.True(t, ok, "Result should be a map")
	
	innerResult, exists := resultMap["result"]
	require.True(t, exists, "Inner result should exist")
	
	innerResultMap, ok := innerResult.(map[string]interface{})
	require.True(t, ok, "Inner result should be a map")
	
	finalResult, exists := innerResultMap["result"]
	require.True(t, exists, "Final result should exist")
	
	finalResultMap, ok := finalResult.(map[string]interface{})
	require.True(t, ok, "Final result should be a map")
	
	toolsInterface, exists := finalResultMap["tools"]
	require.True(t, exists, "Tools should exist in result")
	require.NotNil(t, toolsInterface, "Tools should not be nil")
	
	tools, ok := toolsInterface.([]interface{})
	require.True(t, ok, "Tools should be an array")
	assert.Len(t, tools, 2)

	// Test executing weather tool
	execParams := map[string]interface{}{
		"connectionType": "http",
		"operation":      "executeTool",
		"url":            server.URL(),
		"toolName":       "get_current_weather",
		"toolParameters": `{"location": "New York"}`,
	}

	execNode, err := NewMCPNodeWrapper(execParams)
	require.NoError(t, err)

	_ = map[string]interface{}{
		"params": execParams,
		"input":  map[string]interface{}{},
	}

	// Create shared context for execution
	execShared := make(map[string]interface{})
	
	execAction, err := execNode.Run(execShared)
	require.NoError(t, err)
	require.Equal(t, "default", execAction)

	// Verify weather data
	execResultData, exists := execShared["result"]
	require.True(t, exists)
	
	// The result is nested: result.result.result.temperature
	execResultMap, ok := execResultData.(map[string]interface{})
	require.True(t, ok, "Exec result should be a map")
	
	innerExecResult, exists := execResultMap["result"]
	require.True(t, exists, "Inner exec result should exist")
	
	innerExecResultMap, ok := innerExecResult.(map[string]interface{})
	require.True(t, ok, "Inner exec result should be a map")
	
	finalExecResult, exists := innerExecResultMap["result"]
	require.True(t, exists, "Final exec result should exist")
	
	weatherData, ok := finalExecResult.(map[string]interface{})
	require.True(t, ok, "Weather data should be a map")
	
	assert.Contains(t, weatherData, "temperature")
	assert.Contains(t, weatherData, "condition")
}

// TestMCPNode_GenericSearchServer tests MCP client with a search-focused server
func TestMCPNode_GenericSearchServer(t *testing.T) {
	// Create a search-focused MCP server
	searchTools := []GenericTool{
		{
			Name:        "web_search",
			Description: "Search the web for information",
			InputSchema: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"query": map[string]interface{}{
						"type":        "string",
						"description": "Search query",
					},
					"limit": map[string]interface{}{
						"type":        "number",
						"description": "Maximum number of results",
					},
				},
				"required": []string{"query"},
			},
		},
		{
			Name:        "image_search",
			Description: "Search for images",
			InputSchema: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"query": map[string]interface{}{
						"type":        "string",
						"description": "Image search query",
					},
				},
				"required": []string{"query"},
			},
		},
	}

	server := NewGenericMCPServer(searchTools)
	defer server.Close()

	// Test web search tool
	params := map[string]interface{}{
		"connectionType": "http",
		"operation":      "executeTool",
		"url":            server.URL(),
		"toolName":       "web_search",
		"toolParameters": `{"query": "FlowRunner MCP integration"}`,
	}

	node, err := NewMCPNodeWrapper(params)
	require.NoError(t, err)

	// Create shared context to capture results
	shared := make(map[string]interface{})
	
	action, err := node.Run(shared)
	require.NoError(t, err)
	require.Equal(t, "default", action)

	// Verify search results
	resultData, exists := shared["result"]
	require.True(t, exists)
	
	// The result is nested: result.result.result.results
	resultMap, ok := resultData.(map[string]interface{})
	require.True(t, ok, "Result should be a map")
	
	innerResult, exists := resultMap["result"]
	require.True(t, exists, "Inner result should exist")
	
	innerResultMap, ok := innerResult.(map[string]interface{})
	require.True(t, ok, "Inner result should be a map")
	
	finalResult, exists := innerResultMap["result"]
	require.True(t, exists, "Final result should exist")
	
	searchData, ok := finalResult.(map[string]interface{})
	require.True(t, ok, "Search data should be a map")
	
	assert.Contains(t, searchData, "results")
	assert.Contains(t, searchData, "total")
	
	resultsInterface, exists := searchData["results"]
	require.True(t, exists, "Results should exist")
	require.NotNil(t, resultsInterface, "Results should not be nil")
	
	results, ok := resultsInterface.([]interface{})
	require.True(t, ok, "Results should be an array")
	assert.Len(t, results, 2)
}

// TestMCPNode_GenericUtilityServer tests MCP client with utility tools
func TestMCPNode_GenericUtilityServer(t *testing.T) {
	// Create a utility-focused MCP server
	utilityTools := []GenericTool{
		{
			Name:        "get_current_time",
			Description: "Get current time in specified timezone",
			InputSchema: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"timezone": map[string]interface{}{
						"type":        "string",
						"description": "Timezone (e.g., UTC, America/New_York)",
					},
				},
			},
		},
		{
			Name:        "calculate_expression",
			Description: "Calculate mathematical expressions",
			InputSchema: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"expression": map[string]interface{}{
						"type":        "string",
						"description": "Mathematical expression to calculate",
					},
				},
				"required": []string{"expression"},
			},
		},
		{
			Name:        "translate_text",
			Description: "Translate text between languages",
			InputSchema: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"text": map[string]interface{}{
						"type":        "string",
						"description": "Text to translate",
					},
					"from": map[string]interface{}{
						"type":        "string",
						"description": "Source language code",
					},
					"to": map[string]interface{}{
						"type":        "string",
						"description": "Target language code",
					},
				},
				"required": []string{"text", "to"},
			},
		},
	}

	server := NewGenericMCPServer(utilityTools)
	defer server.Close()

	// Test each utility tool
	testCases := []struct {
		name       string
		toolName   string
		parameters string
		checkField string
	}{
		{
			name:       "time tool",
			toolName:   "get_current_time",
			parameters: `{"timezone": "UTC"}`,
			checkField: "current_time",
		},
		{
			name:       "calculator tool",
			toolName:   "calculate_expression",
			parameters: `{"expression": "2+2"}`,
			checkField: "result",
		},
		{
			name:       "translation tool",
			toolName:   "translate_text",
			parameters: `{"text": "hello", "from": "en", "to": "es"}`,
			checkField: "translated",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			params := map[string]interface{}{
				"connectionType": "http",
				"operation":      "executeTool",
				"url":            server.URL(),
				"toolName":       tc.toolName,
				"toolParameters": tc.parameters,
			}

			node, err := NewMCPNodeWrapper(params)
			require.NoError(t, err)

			// Create shared context to capture results
			shared := make(map[string]interface{})
			
			action, err := node.Run(shared)
			require.NoError(t, err)
			require.Equal(t, "default", action)

			// Verify tool result
			resultData, exists := shared["result"]
			require.True(t, exists)
			
			// The result is nested: result.result.result.field
			resultMap, ok := resultData.(map[string]interface{})
			require.True(t, ok, "Result should be a map")
			
			innerResult, exists := resultMap["result"]
			require.True(t, exists, "Inner result should exist")
			
			innerResultMap, ok := innerResult.(map[string]interface{})
			require.True(t, ok, "Inner result should be a map")
			
			finalResult, exists := innerResultMap["result"]
			require.True(t, exists, "Final result should exist")
			
			toolData, ok := finalResult.(map[string]interface{})
			require.True(t, ok, "Tool data should be a map")
			
			assert.Contains(t, toolData, tc.checkField)
		})
	}
}

// TestMCPNode_GenericServerCompatibility tests compatibility with various MCP server formats
func TestMCPNode_GenericServerCompatibility(t *testing.T) {
	// Test different response formats that MCP servers might use
	testCases := []struct {
		name         string
		tools        []GenericTool
		expectedTool string
	}{
		{
			name: "minimal tools",
			tools: []GenericTool{
				{Name: "simple_tool", Description: "A simple tool"},
			},
			expectedTool: "simple_tool",
		},
		{
			name: "complex tools with schemas",
			tools: []GenericTool{
				{
					Name:        "complex_tool",
					Description: "A complex tool with detailed schema",
					InputSchema: map[string]interface{}{
						"type": "object",
						"properties": map[string]interface{}{
							"param1": map[string]interface{}{
								"type":        "string",
								"description": "First parameter",
								"enum":        []string{"option1", "option2"},
							},
							"param2": map[string]interface{}{
								"type":        "number",
								"description": "Second parameter",
								"minimum":     0,
								"maximum":     100,
							},
						},
						"required": []string{"param1"},
					},
				},
			},
			expectedTool: "complex_tool",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			server := NewGenericMCPServer(tc.tools)
			defer server.Close()

			// Test tool listing
			params := map[string]interface{}{
				"connectionType": "http",
				"operation":      "listTools",
				"url":            server.URL(),
			}

			node, err := NewMCPNodeWrapper(params)
			require.NoError(t, err)

			// Create shared context to capture results
			shared := make(map[string]interface{})
			
			action, err := node.Run(shared)
			require.NoError(t, err)
			require.Equal(t, "default", action)

			// Verify tools are listed correctly
			resultData, exists := shared["result"]
			require.True(t, exists)
			
			// The result is nested: result.result.result.tools
			resultMap, ok := resultData.(map[string]interface{})
			require.True(t, ok, "Result should be a map")
			
			innerResult, exists := resultMap["result"]
			require.True(t, exists, "Inner result should exist")
			
			innerResultMap, ok := innerResult.(map[string]interface{})
			require.True(t, ok, "Inner result should be a map")
			
			finalResult, exists := innerResultMap["result"]
			require.True(t, exists, "Final result should exist")
			
			resultObj, ok := finalResult.(map[string]interface{})
			require.True(t, ok, "Final result should be a map")
			
			toolsInterface, exists := resultObj["tools"]
			require.True(t, exists, "Tools should exist")
			require.NotNil(t, toolsInterface, "Tools should not be nil")
			
			tools, ok := toolsInterface.([]interface{})
			require.True(t, ok, "Tools should be an array")
			assert.Len(t, tools, len(tc.tools))

			// Verify expected tool is present
			found := false
			for _, tool := range tools {
				toolMap := tool.(map[string]interface{})
				if toolMap["name"] == tc.expectedTool {
					found = true
					break
				}
			}
			assert.True(t, found, "Expected tool %s not found", tc.expectedTool)
		})
	}
}

// TestMCPNode_GenericErrorHandling tests error handling with various MCP servers
func TestMCPNode_GenericErrorHandling(t *testing.T) {
	// Create server that returns errors for certain tools
	errorTools := []GenericTool{
		{Name: "working_tool", Description: "This tool works"},
		{Name: "broken_tool", Description: "This tool is broken"},
	}

	server := NewGenericMCPServer(errorTools)
	defer server.Close()

	// Test calling non-existent tool
	params := map[string]interface{}{
		"connectionType": "http",
		"operation":      "executeTool",
		"url":            server.URL(),
		"toolName":       "nonexistent_tool",
		"toolParameters": `{}`,
	}

	node, err := NewMCPNodeWrapper(params)
	require.NoError(t, err)

	// Create shared context
	shared := make(map[string]interface{})
	
	action, err := node.Run(shared)
	require.NoError(t, err) // HTTP request succeeds
	require.Equal(t, "default", action)
	
	// But the result should contain an error
	resultData, exists := shared["result"]
	require.True(t, exists)
	
	resultMap, ok := resultData.(map[string]interface{})
	require.True(t, ok)
	
	innerResult, exists := resultMap["result"]
	require.True(t, exists)
	
	innerResultMap, ok := innerResult.(map[string]interface{})
	require.True(t, ok)
	
	errorData, exists := innerResultMap["error"]
	require.True(t, exists, "Response should contain an error")
	
	errorMap, ok := errorData.(map[string]interface{})
	require.True(t, ok)
	
	assert.Contains(t, errorMap, "message")
	assert.Contains(t, errorMap["message"], "Tool not found")
}

// BenchmarkMCPNode_GenericServer benchmarks MCP operations with generic servers
func BenchmarkMCPNode_GenericServer(b *testing.B) {
	tools := []GenericTool{
		{Name: "benchmark_tool", Description: "Tool for benchmarking"},
	}

	server := NewGenericMCPServer(tools)
	defer server.Close()

	params := map[string]interface{}{
		"connectionType": "http",
		"operation":      "executeTool",
		"url":            server.URL(),
		"toolName":       "benchmark_tool",
		"toolParameters": `{}`,
	}

	node, err := NewMCPNodeWrapper(params)
	require.NoError(b, err)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		shared := make(map[string]interface{})
		_, err := node.Run(shared)
		if err != nil {
			b.Fatal(err)
		}
	}
}