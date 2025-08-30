package runtime

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/tcmartin/flowlib"
)

// Helper function to run a node and extract the result from shared context
func runNodeAndGetResult(t *testing.T, node flowlib.Node, input map[string]interface{}) (interface{}, error) {
	shared := make(map[string]interface{})
	action, err := node.Run(shared)
	if err != nil {
		return nil, err
	}
	require.Equal(t, "default", action)
	
	result, exists := shared["result"]
	require.True(t, exists)
	return result, nil
}

// MockMCPServer represents a generic MCP server for testing
type MockMCPServer struct {
	server *httptest.Server
	tools  []MCPTool
}

// MCPTool represents a tool in the MCP protocol
type MCPTool struct {
	Name        string                 `json:"name"`
	Description string                 `json:"description"`
	InputSchema map[string]interface{} `json:"inputSchema,omitempty"`
}

// MCPResponse represents a standard MCP response
type MCPResponse struct {
	Result interface{} `json:"result,omitempty"`
	Error  *MCPError   `json:"error,omitempty"`
}

// MCPError represents an MCP error
type MCPError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

// NewMockMCPServer creates a new mock MCP server with predefined tools
func NewMockMCPServer() *MockMCPServer {
	mock := &MockMCPServer{
		tools: []MCPTool{
			{
				Name:        "get_weather",
				Description: "Get weather information for a location",
				InputSchema: map[string]interface{}{
					"type": "object",
					"properties": map[string]interface{}{
						"location": map[string]interface{}{
							"type":        "string",
							"description": "The location to get weather for",
						},
					},
					"required": []string{"location"},
				},
			},
			{
				Name:        "search_web",
				Description: "Search the web for information",
				InputSchema: map[string]interface{}{
					"type": "object",
					"properties": map[string]interface{}{
						"query": map[string]interface{}{
							"type":        "string",
							"description": "The search query",
						},
					},
					"required": []string{"query"},
				},
			},
			{
				Name:        "get_time",
				Description: "Get current time",
				InputSchema: map[string]interface{}{
					"type": "object",
					"properties": map[string]interface{}{
						"timezone": map[string]interface{}{
							"type":        "string",
							"description": "Timezone (optional)",
						},
					},
				},
			},
		},
	}

	mock.server = httptest.NewServer(http.HandlerFunc(mock.handleRequest))
	return mock
}

// Close shuts down the mock server
func (m *MockMCPServer) Close() {
	if m.server != nil {
		m.server.Close()
	}
}

// URL returns the server URL
func (m *MockMCPServer) URL() string {
	return m.server.URL
}

// handleRequest handles MCP requests
func (m *MockMCPServer) handleRequest(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var request map[string]interface{}
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}

	method, ok := request["method"].(string)
	if !ok {
		http.Error(w, "Missing method", http.StatusBadRequest)
		return
	}

	w.Header().Set("Content-Type", "application/json")

	switch method {
	case "tools/list":
		m.handleToolsList(w, request)
	case "tools/call":
		m.handleToolsCall(w, request)
	default:
		response := MCPResponse{
			Error: &MCPError{
				Code:    -32601,
				Message: fmt.Sprintf("Method not found: %s", method),
			},
		}
		json.NewEncoder(w).Encode(response)
	}
}

// handleToolsList handles tools/list requests
func (m *MockMCPServer) handleToolsList(w http.ResponseWriter, request map[string]interface{}) {
	response := MCPResponse{
		Result: map[string]interface{}{
			"tools": m.tools,
		},
	}
	json.NewEncoder(w).Encode(response)
}

// handleToolsCall handles tools/call requests
func (m *MockMCPServer) handleToolsCall(w http.ResponseWriter, request map[string]interface{}) {
	params, ok := request["params"].(map[string]interface{})
	if !ok {
		response := MCPResponse{
			Error: &MCPError{
				Code:    -32602,
				Message: "Invalid params",
			},
		}
		json.NewEncoder(w).Encode(response)
		return
	}

	toolName, ok := params["name"].(string)
	if !ok {
		response := MCPResponse{
			Error: &MCPError{
				Code:    -32602,
				Message: "Missing tool name",
			},
		}
		json.NewEncoder(w).Encode(response)
		return
	}

	// Find the tool
	var tool *MCPTool
	for _, t := range m.tools {
		if t.Name == toolName {
			tool = &t
			break
		}
	}

	if tool == nil {
		response := MCPResponse{
			Error: &MCPError{
				Code:    -32602,
				Message: fmt.Sprintf("Tool not found: %s", toolName),
			},
		}
		json.NewEncoder(w).Encode(response)
		return
	}

	// Mock tool execution results
	var result interface{}
	switch toolName {
	case "get_weather":
		result = map[string]interface{}{
			"temperature": "22°C",
			"condition":   "Sunny",
			"location":    "Default Location",
		}
	case "search_web":
		result = map[string]interface{}{
			"results": []map[string]interface{}{
				{"title": "Test Result", "url": "https://example.com"},
			},
		}
	case "get_time":
		result = map[string]interface{}{
			"time":     time.Now().Format(time.RFC3339),
			"timezone": "UTC",
		}
	default:
		result = map[string]interface{}{
			"message": fmt.Sprintf("Executed tool: %s", toolName),
		}
	}

	response := MCPResponse{
		Result: result,
	}
	json.NewEncoder(w).Encode(response)
}

func TestMCPNode_HTTP_ListTools(t *testing.T) {
	// Create mock MCP server
	mockServer := NewMockMCPServer()
	defer mockServer.Close()

	// Create MCP node for listing tools
	params := map[string]interface{}{
		"connectionType": "http",
		"operation":      "listTools",
		"url":            mockServer.URL(),
	}

	node, err := NewMCPNodeWrapper(params)
	require.NoError(t, err)
	require.NotNil(t, node)

	// Execute the node
	input := map[string]interface{}{
		"params": params,
		"input":  map[string]interface{}{},
	}

	result, err := runNodeAndGetResult(t, node, input)
	require.NoError(t, err)
	require.NotNil(t, result)

	// Verify the result contains tools
	resultData := result

	// Parse the nested result
	var mcpResponse MCPResponse
	if resultStr, ok := resultData.(string); ok {
		err = json.Unmarshal([]byte(resultStr), &mcpResponse)
		require.NoError(t, err)
	} else {
		// Result might already be parsed
		resultBytes, err := json.Marshal(resultData)
		require.NoError(t, err)
		err = json.Unmarshal(resultBytes, &mcpResponse)
		require.NoError(t, err)
	}

	// Verify tools are present - handle nested structure
	require.NotNil(t, mcpResponse.Result)
	resultObj, ok := mcpResponse.Result.(map[string]interface{})
	require.True(t, ok)

	// The result is nested: result.result.tools
	innerResult, exists := resultObj["result"]
	require.True(t, exists)
	
	innerResultObj, ok := innerResult.(map[string]interface{})
	require.True(t, ok)

	tools, exists := innerResultObj["tools"]
	require.True(t, exists)

	toolsArray, ok := tools.([]interface{})
	require.True(t, ok)
	assert.Len(t, toolsArray, 3) // Should have 3 mock tools
}

func TestMCPNode_HTTP_ExecuteTool(t *testing.T) {
	// Create mock MCP server
	mockServer := NewMockMCPServer()
	defer mockServer.Close()

	// Create MCP node for executing a tool
	params := map[string]interface{}{
		"connectionType": "http",
		"operation":      "executeTool",
		"url":            mockServer.URL(),
		"toolName":       "get_weather",
		"toolParameters": `{"location": "San Francisco"}`,
	}

	node, err := NewMCPNodeWrapper(params)
	require.NoError(t, err)
	require.NotNil(t, node)

	// Execute the node
	input := map[string]interface{}{
		"params": params,
		"input":  map[string]interface{}{},
	}

	result, err := runNodeAndGetResult(t, node, input)
	require.NoError(t, err)
	require.NotNil(t, result)

	// Verify the result contains weather data
	resultData := result

	// Parse the result
	var mcpResponse MCPResponse
	if resultStr, ok := resultData.(string); ok {
		err = json.Unmarshal([]byte(resultStr), &mcpResponse)
		require.NoError(t, err)
	} else {
		resultBytes, err := json.Marshal(resultData)
		require.NoError(t, err)
		err = json.Unmarshal(resultBytes, &mcpResponse)
		require.NoError(t, err)
	}

	// Verify weather result - handle nested structure
	require.NotNil(t, mcpResponse.Result)
	resultObj, ok := mcpResponse.Result.(map[string]interface{})
	require.True(t, ok)

	// The result is nested: result.result.temperature
	innerResult, exists := resultObj["result"]
	require.True(t, exists)
	
	weatherData, ok := innerResult.(map[string]interface{})
	require.True(t, ok)

	temperature, exists := weatherData["temperature"]
	require.True(t, exists)
	assert.Equal(t, "22°C", temperature)
}

func TestMCPNode_HTTP_RawBody(t *testing.T) {
	// Create mock MCP server
	mockServer := NewMockMCPServer()
	defer mockServer.Close()

	// Create MCP node with raw body
	params := map[string]interface{}{
		"connectionType": "http",
		"url":            mockServer.URL(),
		"rawBody":        `{"method":"tools/list","params":{}}`,
	}

	node, err := NewMCPNodeWrapper(params)
	require.NoError(t, err)
	require.NotNil(t, node)

	// Execute the node
	input := map[string]interface{}{
		"params": params,
		"input":  map[string]interface{}{},
	}

	action, err := node.Run(input)
	require.NoError(t, err)
	require.Equal(t, "default", action)

	// Verify the result is in shared context
	_, exists := input["result"]
	require.True(t, exists)
}

func TestMCPNode_HTTP_RawBodyTemplate(t *testing.T) {
	// Create mock MCP server
	mockServer := NewMockMCPServer()
	defer mockServer.Close()

	// Create MCP node with raw body template
	params := map[string]interface{}{
		"connectionType": "http",
		"url":            mockServer.URL(),
		"rawBodyTemplate": `{
			"method": "tools/call",
			"params": {
				"name": "{{.input.toolName}}",
				"arguments": {"location": "{{.input.location}}"}
			}
		}`,
	}

	node, err := NewMCPNodeWrapper(params)
	require.NoError(t, err)
	require.NotNil(t, node)

	// Execute the node with template variables
	input := map[string]interface{}{
		"params": params,
		"input": map[string]interface{}{
			"toolName": "get_weather",
			"location": "Tokyo",
		},
	}

	action, err := node.Run(input)
	require.NoError(t, err)
	require.Equal(t, "default", action)

	// Verify the result is in shared context
	_, exists := input["result"]
	require.True(t, exists)
}

func TestMCPNode_HTTP_HeadersTemplate(t *testing.T) {
	// Create mock MCP server that checks headers
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Check for templated header
		authHeader := r.Header.Get("Authorization")
		if !strings.HasPrefix(authHeader, "Bearer test-token-") {
			http.Error(w, "Invalid auth header", http.StatusUnauthorized)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		response := MCPResponse{
			Result: map[string]interface{}{
				"message": "authenticated",
			},
		}
		json.NewEncoder(w).Encode(response)
	}))
	defer server.Close()

	// Create MCP node with headers template
	params := map[string]interface{}{
		"connectionType": "http",
		"url":            server.URL,
		"rawBody":        `{"method":"tools/list","params":{}}`,
		"headersTemplate": map[string]interface{}{
			"Authorization": "Bearer test-token-{{.input.userId}}",
		},
	}

	node, err := NewMCPNodeWrapper(params)
	require.NoError(t, err)
	require.NotNil(t, node)

	// Execute the node with template variables
	input := map[string]interface{}{
		"params": params,
		"input": map[string]interface{}{
			"userId": "12345",
		},
	}

	action, err := node.Run(input)
	require.NoError(t, err)
	require.Equal(t, "default", action)

	// Verify the result is in shared context
	_, exists := input["result"]
	require.True(t, exists)
}

func TestMCPNode_HTTP_ErrorHandling(t *testing.T) {
	// Create mock server that returns errors
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		response := MCPResponse{
			Error: &MCPError{
				Code:    -32602,
				Message: "Invalid request",
			},
		}
		json.NewEncoder(w).Encode(response)
	}))
	defer server.Close()

	// Create MCP node
	params := map[string]interface{}{
		"connectionType": "http",
		"url":            server.URL,
		"rawBody":        `{"method":"invalid/method","params":{}}`,
	}

	node, err := NewMCPNodeWrapper(params)
	require.NoError(t, err)
	require.NotNil(t, node)

	// Execute the node
	input := map[string]interface{}{
		"params": params,
		"input":  map[string]interface{}{},
	}

	_, err = node.Run(input)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "400")
}

func TestMCPNode_CMD_Success(t *testing.T) {
	// Skip if not in a suitable environment for CMD tests
	if os.Getenv("SKIP_CMD_TESTS") == "true" {
		t.Skip("CMD tests skipped")
	}

	// Create MCP node with echo command (should work on most systems)
	params := map[string]interface{}{
		"connectionType": "cmd",
		"operation":      "listTools",
		"command":        "echo",
		"args":           []string{`{"result":{"tools":[{"name":"echo_tool","description":"Echo tool"}]}}`},
	}

	node, err := NewMCPNodeWrapper(params)
	require.NoError(t, err)
	require.NotNil(t, node)

	// Execute the node
	input := map[string]interface{}{
		"params": params,
		"input":  map[string]interface{}{},
	}

	action, err := node.Run(input)
	require.NoError(t, err)
	require.Equal(t, "default", action)

	// Verify the result is in shared context
	_, exists := input["result"]
	require.True(t, exists)
}

func TestMCPNode_ValidationErrors(t *testing.T) {
	tests := []struct {
		name        string
		params      map[string]interface{}
		expectedErr string
	}{
		{
			name: "missing connection type",
			params: map[string]interface{}{
				"operation": "listTools",
			},
			expectedErr: "'command' parameter is required for 'cmd'",
		},
		{
			name: "http missing url",
			params: map[string]interface{}{
				"connectionType": "http",
				"operation":      "listTools",
			},
			expectedErr: "'url' parameter is required for 'http'",
		},
		{
			name: "cmd missing command",
			params: map[string]interface{}{
				"connectionType": "cmd",
				"operation":      "listTools",
			},
			expectedErr: "'command' parameter is required for 'cmd'",
		},
		{
			name: "missing operation without rawBody",
			params: map[string]interface{}{
				"connectionType": "http",
				"url":            "http://example.com",
			},
			expectedErr: "'operation' parameter is required unless 'rawBody'",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			node, err := NewMCPNodeWrapper(tt.params)
			if err != nil {
				// Error during node creation
				assert.Contains(t, err.Error(), tt.expectedErr)
				return
			}

			// Error during execution
			input := map[string]interface{}{
				"params": tt.params,
				"input":  map[string]interface{}{},
			}

			_, err = node.Run(input)
			require.Error(t, err)
			assert.Contains(t, err.Error(), tt.expectedErr)
		})
	}
}

func TestMCPNode_Timeout(t *testing.T) {
	// Create a slow server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(2 * time.Second) // Longer than our timeout
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"result": "too slow"}`))
	}))
	defer server.Close()

	// Create MCP node with short timeout
	params := map[string]interface{}{
		"connectionType": "http",
		"url":            server.URL,
		"rawBody":        `{"method":"tools/list","params":{}}`,
		"timeout":        "500ms",
	}

	node, err := NewMCPNodeWrapper(params)
	require.NoError(t, err)
	require.NotNil(t, node)

	// Execute the node
	input := map[string]interface{}{
		"params": params,
		"input":  map[string]interface{}{},
	}

	_, err = node.Run(input)
	require.Error(t, err)
	// Should timeout
	assert.Contains(t, strings.ToLower(err.Error()), "timeout")
}

// Benchmark tests
func BenchmarkMCPNode_HTTP_ListTools(b *testing.B) {
	mockServer := NewMockMCPServer()
	defer mockServer.Close()

	params := map[string]interface{}{
		"connectionType": "http",
		"operation":      "listTools",
		"url":            mockServer.URL(),
	}

	node, err := NewMCPNodeWrapper(params)
	require.NoError(b, err)

	input := map[string]interface{}{
		"params": params,
		"input":  map[string]interface{}{},
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := node.Run(input)
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkMCPNode_HTTP_ExecuteTool(b *testing.B) {
	mockServer := NewMockMCPServer()
	defer mockServer.Close()

	params := map[string]interface{}{
		"connectionType": "http",
		"operation":      "executeTool",
		"url":            mockServer.URL(),
		"toolName":       "get_weather",
		"toolParameters": `{"location": "Test"}`,
	}

	node, err := NewMCPNodeWrapper(params)
	require.NoError(b, err)

	input := map[string]interface{}{
		"params": params,
		"input":  map[string]interface{}{},
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := node.Run(input)
		if err != nil {
			b.Fatal(err)
		}
	}
}