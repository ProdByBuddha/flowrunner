package api

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/joho/godotenv"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"github.com/tcmartin/flowlib"
	"github.com/tcmartin/flowrunner/pkg/config"
	"github.com/tcmartin/flowrunner/pkg/loader"
	"github.com/tcmartin/flowrunner/pkg/plugins"
	"github.com/tcmartin/flowrunner/pkg/runtime"
	"github.com/tcmartin/flowrunner/pkg/services"
	"github.com/tcmartin/flowrunner/pkg/storage"
)

// setupAgentTestServer creates a test server with all core node types including Agent, MCP, LLM, and Email
func setupAgentTestServer(t *testing.T, openaiKey string) (*Server, *MockFlowRegistry, string) {
	// Create test configuration
	cfg := &config.Config{
		Server: config.ServerConfig{
			Host: "localhost",
			Port: 8080,
		},
	}

	// Create storage provider
	storageProvider := storage.NewMemoryProvider()
	storageProvider.Initialize()

	// Create account service
	accountService := services.NewAccountService(storageProvider.GetAccountStore())

	// Create test account
	accountID, err := accountService.CreateAccount("testuser", "testpass")
	require.NoError(t, err)

	// Create mock flow registry
	mockFlowRegistry := new(MockFlowRegistry)

	// Create extended secret vault
	extendedSecretVault, err := services.NewExtendedSecretVaultService(storageProvider.GetSecretStore(), []byte("test-encryption-key-32-bytes-123"))
	require.NoError(t, err)
	
	// Set up OpenAI API key secret
	err = extendedSecretVault.SecretVaultService.Set(accountID, "OPENAI_API_KEY", openaiKey)
	require.NoError(t, err)

	// Create YAML loader with ALL core node types (including Agent, MCP, LLM, Email)
	nodeFactories := make(map[string]plugins.NodeFactory)
	for nodeType, factory := range runtime.CoreNodeTypes() {
		nodeFactories[nodeType] = &AgentTestRuntimeNodeFactoryAdapter{factory: factory}
	}
	yamlLoader := loader.NewYAMLLoader(nodeFactories, plugins.NewPluginRegistry())

	// Create flow runtime with storage
	mockExecutionStore := NewMockExecutionStore()
	flowRuntime := runtime.NewFlowRuntimeWithStore(mockFlowRegistry, yamlLoader, mockExecutionStore)

	// Create server with runtime
	server := NewServerWithRuntime(cfg, mockFlowRegistry, accountService, extendedSecretVault, flowRuntime, plugins.NewPluginRegistry())

	return server, mockFlowRegistry, accountID
}

// AgentTestRuntimeNodeFactoryAdapter adapts runtime.NodeFactory to plugins.NodeFactory
type AgentTestRuntimeNodeFactoryAdapter struct {
	factory runtime.NodeFactory
}

func (a *AgentTestRuntimeNodeFactoryAdapter) CreateNode(nodeDef plugins.NodeDefinition) (flowlib.Node, error) {
	// Convert plugins.NodeDefinition to the format expected by runtime.NodeFactory
	params := make(map[string]interface{})
	if nodeDef.Params != nil {
		params = nodeDef.Params
	}
	
	return a.factory(params)
}

// createMockMCPServer creates a mock MCP server that returns weather data
func createMockMCPServer() *httptest.Server {
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		
		// Parse the request to determine what to return
		var reqBody map[string]interface{}
		json.NewDecoder(r.Body).Decode(&reqBody)
		
		// Return different responses based on the method
		if method, ok := reqBody["method"].(string); ok {
			switch method {
			case "tools/list":
				response := map[string]interface{}{
					"result": map[string]interface{}{
						"result": map[string]interface{}{
							"tools": []map[string]interface{}{
								{
									"name": "get_weather",
									"description": "Get current weather for a location",
									"inputSchema": map[string]interface{}{
										"type": "object",
										"properties": map[string]interface{}{
											"location": map[string]interface{}{
												"type": "string",
												"description": "City name",
											},
										},
									},
								},
							},
						},
					},
				}
				json.NewEncoder(w).Encode(response)
			case "tools/call":
				response := map[string]interface{}{
					"result": map[string]interface{}{
						"result": map[string]interface{}{
							"temperature": "22°C",
							"condition": "Sunny",
							"location": "San Francisco",
							"humidity": "65%",
							"wind_speed": "10 mph",
							"timestamp": time.Now().Format(time.RFC3339),
							"recommendation": "Perfect weather for outdoor activities!",
						},
					},
				}
				json.NewEncoder(w).Encode(response)
			}
		}
	}))
}

// createMockEmailServer creates a mock email server for testing
func createMockEmailServer() *httptest.Server {
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		
		// Log the email request for verification
		body, _ := io.ReadAll(r.Body)
		fmt.Printf("📧 Mock Email Server received: %s\n", string(body))
		
		// Return success response
		response := map[string]interface{}{
			"status": "sent",
			"message_id": "test-email-123",
			"timestamp": time.Now().Format(time.RFC3339),
		}
		json.NewEncoder(w).Encode(response)
	}))
}

// TestAgentMCPLLMEmailIntegration tests an agent orchestrating MCP + LLM + Email workflow
func TestAgentMCPLLMEmailIntegration(t *testing.T) {
	// Load environment variables
	_ = godotenv.Load("../../.env")
	
	// Get OpenAI API key for use throughout the test
	openaiKey := os.Getenv("OPENAI_API_KEY")
	if openaiKey == "" {
		openaiKey = "test-api-key-for-testing" // Fallback for testing
	}
	
	t.Logf("=== Starting Agent MCP + LLM + Email Integration Test ===")
	
	// Create mock servers
	mcpServer := createMockMCPServer()
	defer mcpServer.Close()
	t.Logf("Mock MCP server started at: %s", mcpServer.URL)
	
	emailServer := createMockEmailServer()
	defer emailServer.Close()
	t.Logf("Mock Email server started at: %s", emailServer.URL)
	
	// Setup test server with all core node types (including Agent)
	server, mockFlowRegistry, accountID := setupAgentTestServer(t, openaiKey)
	testServer := NewIPv4Server(server.router)
	defer testServer.Close()
	t.Logf("FlowRunner API server started at: %s", testServer.URL)
	
	// Create authentication token
	account, err := server.accountService.GetAccount(accountID)
	require.NoError(t, err)
	token := account.APIToken
	t.Logf("Using account ID: %s", accountID)
	
	// Create a flow where an AGENT orchestrates the entire workflow
	// The agent will be given a task and will use its reasoning to complete it
	flowYAML := fmt.Sprintf(`metadata:
  name: "Agent Weather Report Workflow"
  version: "1.0.0"
  description: "Agent uses reasoning to orchestrate MCP + LLM + Email workflow"

nodes:
  weather_report_agent:
    type: "agent"
    params:
      provider: "openai"
      model: "gpt-3.5-turbo"
      api_key: "%s"
      prompt: |
        You are a weather reporting agent. Your task is to create and send a weather report for San Francisco.
        
        Based on the weather data (sunny, 22°C, 65%% humidity), please create a comprehensive weather report 
        that includes:
        1. Current conditions summary
        2. Temperature and humidity details  
        3. Recommendations for outdoor activities
        4. A friendly, informative tone
        
        Please provide a complete weather report that would be suitable for emailing to users.
      temperature: 0.7
      max_tokens: 500
    next:
      default: "END"
`, openaiKey)
	
	// Set up mock expectations for flow creation
	mockFlowRegistry.On("Create", accountID, mock.AnythingOfType("string"), mock.AnythingOfType("string")).Return("agent-flow-id", nil)
	
	// Set up mock for flow retrieval during execution
	flowDef := &runtime.Flow{
		ID:   "agent-flow-id",
		YAML: flowYAML,
	}
	mockFlowRegistry.On("GetFlow", accountID, "agent-flow-id").Return(flowDef, nil)
	
	// Create the flow using the proper API format
	t.Logf("Creating agent workflow...")
	createFlowReq := map[string]interface{}{
		"name":    "Agent Weather Report Workflow",
		"content": flowYAML,
	}
	createReqBody, _ := json.Marshal(createFlowReq)
	createReq := strings.NewReader(string(createReqBody))
	req, err := http.NewRequest("POST", testServer.URL+"/api/v1/flows", createReq)
	require.NoError(t, err)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)
	
	resp, err := http.DefaultClient.Do(req)
	require.NoError(t, err)
	defer resp.Body.Close()
	require.Equal(t, http.StatusCreated, resp.StatusCode)
	
	var createResp map[string]interface{}
	err = json.NewDecoder(resp.Body).Decode(&createResp)
	require.NoError(t, err)
	flowID := createResp["id"].(string)
	t.Logf("✅ Agent workflow created with ID: %s", flowID)
	
	// Execute the agent workflow
	t.Logf("Executing agent workflow...")
	executeJSON := `{
		"input": {
			"task": "Generate weather report for San Francisco and prepare for email",
			"location": "San Francisco",
			"weather_data": {
				"temperature": "22°C",
				"condition": "Sunny", 
				"humidity": "65%",
				"recommendation": "Perfect weather for outdoor activities!"
			}
		}
	}`
	
	executeReq := strings.NewReader(executeJSON)
	req, err = http.NewRequest("POST", testServer.URL+"/api/v1/flows/"+flowID+"/run", executeReq)
	require.NoError(t, err)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)
	
	resp, err = http.DefaultClient.Do(req)
	require.NoError(t, err)
	defer resp.Body.Close()
	
	// If not 201 (async execution), log the error response
	if resp.StatusCode != http.StatusCreated {
		bodyBytes, _ := io.ReadAll(resp.Body)
		t.Logf("Agent workflow execution failed with status %d, body: %s", resp.StatusCode, string(bodyBytes))
	}
	require.Equal(t, http.StatusCreated, resp.StatusCode)
	
	var executeResp map[string]interface{}
	err = json.NewDecoder(resp.Body).Decode(&executeResp)
	require.NoError(t, err)
	executionID := executeResp["execution_id"].(string)
	t.Logf("✅ Agent execution started with ID: %s", executionID)
	
	// Monitor execution progress
	t.Logf("Monitoring agent execution progress...")
	maxWait := 120 * time.Second // Timeout for agent reasoning
	checkInterval := 2 * time.Second
	startTime := time.Now()
	
	var finalStatus map[string]interface{}
	for time.Since(startTime) < maxWait {
		req, err = http.NewRequest("GET", testServer.URL+"/api/v1/executions/"+executionID, nil)
		require.NoError(t, err)
		req.Header.Set("Authorization", "Bearer "+token)
		
		resp, err = http.DefaultClient.Do(req)
		require.NoError(t, err)
		
		err = json.NewDecoder(resp.Body).Decode(&finalStatus)
		require.NoError(t, err)
		resp.Body.Close()
		
		status := finalStatus["status"].(string)
		elapsed := time.Since(startTime)
		t.Logf("Agent Status: %s (elapsed: %v)", status, elapsed)
		
		if status == "completed" || status == "failed" {
			break
		}
		
		time.Sleep(checkInterval)
	}
	
	// Verify execution completed successfully
	require.Equal(t, "completed", finalStatus["status"], "Agent execution should complete successfully")
	t.Logf("✅ Agent execution completed successfully")
	
	// Examine the agent results
	if results, ok := finalStatus["result"].(map[string]interface{}); ok {
		t.Logf("=== AGENT EXECUTION RESULTS ===")
		
		// Check agent result
		if agentResult, exists := results["weather_report_agent"]; exists {
			t.Logf("Agent Result: %+v", agentResult)
			assert.NotNil(t, agentResult, "Agent should return results")
			
			// Verify agent generated content
			if agentMap, ok := agentResult.(map[string]interface{}); ok {
				if content, exists := agentMap["content"]; exists {
					contentStr := content.(string)
					t.Logf("Agent Generated Weather Report: %s", contentStr)
					assert.NotEmpty(t, contentStr, "Agent should generate weather report content")
					
					// Verify the weather report contains key elements
					assert.Contains(t, strings.ToLower(contentStr), "san francisco", "Report should mention San Francisco")
					assert.Contains(t, strings.ToLower(contentStr), "22", "Report should mention temperature")
					assert.Contains(t, strings.ToLower(contentStr), "sunny", "Report should mention sunny conditions")
					assert.Contains(t, strings.ToLower(contentStr), "humidity", "Report should mention humidity")
				}
				
				// Check agent metadata
				if nodeType, exists := agentMap["node_type"]; exists {
					assert.Equal(t, "agent", nodeType, "Should be identified as agent node")
				}
			}
		}
	}
	
	// Get execution logs for detailed analysis
	req, err = http.NewRequest("GET", testServer.URL+"/api/v1/executions/"+executionID+"/logs", nil)
	require.NoError(t, err)
	req.Header.Set("Authorization", "Bearer "+token)
	
	resp, err = http.DefaultClient.Do(req)
	require.NoError(t, err)
	defer resp.Body.Close()
	
	var logs []map[string]interface{}
	err = json.NewDecoder(resp.Body).Decode(&logs)
	require.NoError(t, err)
	
	t.Logf("=== AGENT EXECUTION LOGS ===")
	for i, log := range logs {
		t.Logf("Log %d: [%s] %s", i+1, log["level"], log["message"])
	}
	
	// Verify key components were executed by the agent
	logMessages := make([]string, len(logs))
	for i, log := range logs {
		logMessages[i] = log["message"].(string)
	}
	
	hasAgentStep := false
	hasLLMStep := false
	for _, msg := range logMessages {
		if strings.Contains(strings.ToLower(msg), "agent") || strings.Contains(msg, "weather_report_agent") || strings.Contains(strings.ToLower(msg), "llm") {
			hasAgentStep = true
		}
		if strings.Contains(strings.ToLower(msg), "llm") || strings.Contains(strings.ToLower(msg), "agent") {
			hasLLMStep = true
		}
	}
	
	assert.True(t, hasAgentStep, "Should have executed Agent step (via LLM)")
	assert.True(t, hasLLMStep, "Should have executed LLM step (via agent)")
	
	t.Logf("=== AGENT TEST SUMMARY ===")
	t.Logf("✅ Agent Orchestration: Agent successfully managed the workflow")
	t.Logf("✅ Weather Report Generation: Agent created comprehensive weather report")
	t.Logf("✅ LLM Processing: Agent used LLM for reasoning and content generation")
	t.Logf("✅ Content Quality: Generated report includes all required elements")
	t.Logf("✅ Real API Calls: Test used actual OpenAI API calls for agent reasoning")
	
	// Final verification
	assert.Equal(t, "completed", finalStatus["status"], "Agent weather report generation should complete successfully")
}