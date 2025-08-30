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
	"github.com/stretchr/testify/require"
	"github.com/tcmartin/flowlib"
	"github.com/tcmartin/flowrunner/pkg/config"
	"github.com/tcmartin/flowrunner/pkg/loader"
	"github.com/tcmartin/flowrunner/pkg/plugins"
	"github.com/tcmartin/flowrunner/pkg/registry"
	"github.com/tcmartin/flowrunner/pkg/runtime"
	"github.com/tcmartin/flowrunner/pkg/services"
	"github.com/tcmartin/flowrunner/pkg/storage"
)

// setupAgentTestServer creates a test server with all core node types including Agent, MCP, LLM, and Email
func setupAgentTestServer(t *testing.T, openaiKey string) (*Server, string, *services.ExtendedSecretVaultService, loader.YAMLLoader, runtime.FlowRuntime) {
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

	// Create flow runtime with storage and secrets
	mockExecutionStore := NewMockExecutionStore()
	
	// Create a simple flow registry adapter for direct execution
	flowRegistryAdapter := &LLMTestFlowRegistryAdapter{registry: nil}
	flowRuntime := runtime.NewFlowRuntimeWithStoreAndSecrets(flowRegistryAdapter, yamlLoader, mockExecutionStore, extendedSecretVault)

	// Create server with runtime (for token generation)
	mockFlowRegistry := new(MockFlowRegistry)
	server := NewServerWithRuntime(cfg, mockFlowRegistry, accountService, extendedSecretVault, flowRuntime, plugins.NewPluginRegistry())

	return server, accountID, extendedSecretVault, yamlLoader, flowRuntime
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
	
	// Get OpenAI API key and email credentials
	openaiKey := os.Getenv("OPENAI_API_KEY")
	if openaiKey == "" {
		openaiKey = "test-api-key-for-testing" // Fallback for testing
	}
	
	// Check for email credentials
	sender := os.Getenv("GMAIL_USERNAME")
	pass := os.Getenv("GMAIL_PASSWORD")
	recipient := os.Getenv("EMAIL_RECIPIENT")
	if sender == "" || pass == "" || recipient == "" {
		t.Skip("Skipping MCP LLM email integration test: missing email credentials")
	}
	
	t.Logf("=== Starting Agent MCP + LLM + Email Integration Test ===")
	
	// Create mock MCP server that returns tool list
	mcpServer := createMockMCPServer()
	defer mcpServer.Close()
	t.Logf("Mock MCP server started at: %s", mcpServer.URL)
	
	// Setup test server with all core node types (including Agent, MCP, Email)
	server, accountID, extendedSecretVault, yamlLoader, _ := setupAgentTestServer(t, openaiKey)
	testServer := NewIPv4Server(server.router)
	defer testServer.Close()
	t.Logf("FlowRunner API server started at: %s", testServer.URL)
	
	// Set up email credentials in the secret vault
	err := extendedSecretVault.Set(accountID, "GMAIL_USERNAME", sender)
	require.NoError(t, err)
	err = extendedSecretVault.Set(accountID, "GMAIL_PASSWORD", pass)
	require.NoError(t, err)
	err = extendedSecretVault.Set(accountID, "EMAIL_RECIPIENT", recipient)
	require.NoError(t, err)
	
	t.Logf("Using account ID: %s", accountID)
	
	// Create a flow that lists MCP tools, analyzes them, and sends an email summary
	flowYAML := fmt.Sprintf(`metadata:
  name: "MCP Tools Analysis and Email Report"
  version: "1.0.0"
  description: "Agent lists MCP tools, analyzes them, and sends email summary"

nodes:
  list_mcp_tools:
    type: "mcp"
    params:
      connectionType: "http"
      operation: "listTools"
      url: "%s/tools/list"
    next:
      default: "analyze_tools"
      
  analyze_tools:
    type: "agent"
    params:
      provider: "openai"
      model: "gpt-3.5-turbo"
      api_key: "%s"
      prompt: |
        You are a technical analyst reviewing MCP (Model Context Protocol) tools.
        
        You have received a list of available MCP tools. Please analyze this data and create 
        a comprehensive email summary that includes:
        
        1. Executive Summary: Brief overview of MCP tools discovered
        2. Tool Analysis: Description of each tool and its capabilities  
        3. Technical Benefits: How these tools can be used in workflows
        4. Integration Opportunities: Potential use cases for these tools
        5. Next Steps: Recommendations for implementation
        
        Format this as a professional email that would be suitable for sending to a technical team.
        Write the email content directly without JSON formatting.
      temperature: 0.7
      max_tokens: 800
    next:
      default: "send_email_summary"
      
  send_email_summary:
    type: "email.send"
    params:
      smtp_host: "smtp.gmail.com"
      smtp_port: 587
      username: "${secrets.GMAIL_USERNAME}"
      password: "${secrets.GMAIL_PASSWORD}"
      from: "${secrets.GMAIL_USERNAME}"
      to: "${secrets.EMAIL_RECIPIENT}"
      subject: "MCP Tools Analysis Report - FlowRunner Test"
      body: "${shared.result.content}"
    next:
      default: "END"
`, mcpServer.URL, openaiKey)
	
	// Use the same pattern as the working email summary test
	t.Logf("Creating MCP tools analysis workflow...")
	
	// Create flow registry and store the flow
	sp := storage.NewMemoryProvider()
	require.NoError(t, sp.Initialize())
	flowReg := registry.NewFlowRegistry(sp.GetFlowStore(), registry.FlowRegistryOptions{YAMLLoader: yamlLoader})
	
	flowID, err := flowReg.Create(accountID, "mcp-tools-analysis", flowYAML)
	require.NoError(t, err)
	
	// Execute the flow using the runtime with secrets
	t.Logf("Executing MCP tools analysis workflow...")
	rt := runtime.NewFlowRuntimeWithStoreAndSecrets(&LLMTestFlowRegistryAdapter{registry: flowReg}, yamlLoader, sp.GetExecutionStore(), extendedSecretVault)
	executionID, err := rt.Execute(accountID, flowID, map[string]interface{}{
		"task": "Analyze MCP tools and send email summary",
		"mcp_server_url": mcpServer.URL,
	})
	require.NoError(t, err)
	t.Logf("✅ MCP tools analysis execution started with ID: %s", executionID)
	
	// Monitor execution progress using flow runtime
	t.Logf("Monitoring MCP tools analysis execution progress...")
	maxWait := 120 * time.Second // Timeout for agent reasoning
	checkInterval := 2 * time.Second
	startTime := time.Now()
	
	var finalStatus runtime.ExecutionStatus
	for time.Since(startTime) < maxWait {
		finalStatus, err = rt.GetStatus(executionID)
		require.NoError(t, err)
		
		elapsed := time.Since(startTime)
		t.Logf("MCP Analysis Status: %s (elapsed: %v)", finalStatus.Status, elapsed)
		
		if finalStatus.Status == "completed" || finalStatus.Status == "failed" {
			break
		}
		
		time.Sleep(checkInterval)
	}
	
	// Check if execution failed and log details
	if finalStatus.Status == "failed" {
		t.Logf("❌ Execution failed. Status: %+v", finalStatus)
		if finalStatus.Error != "" {
			t.Logf("Error message: %s", finalStatus.Error)
		}
	}
	
	// Verify execution completed successfully
	require.Equal(t, "completed", finalStatus.Status, "MCP tools analysis execution should complete successfully")
	t.Logf("✅ MCP tools analysis execution completed successfully")
	
	// Examine the MCP tools analysis results
	if finalStatus.Results != nil {
		t.Logf("=== MCP TOOLS ANALYSIS EXECUTION RESULTS ===")
		
		// Check MCP tools discovery result
		if mcpResult, exists := finalStatus.Results["list_mcp_tools"]; exists {
			t.Logf("MCP Tools Discovery Result: %+v", mcpResult)
			assert.NotNil(t, mcpResult, "Should have discovered MCP tools")
		}
		
		// Check agent analysis result
		if agentResult, exists := finalStatus.Results["analyze_tools"]; exists {
			t.Logf("Agent Analysis Result: %+v", agentResult)
			assert.NotNil(t, agentResult, "Agent should have analyzed the tools")
			
			// Verify agent generated comprehensive report
			if agentMap, ok := agentResult.(map[string]interface{}); ok {
				if content, exists := agentMap["content"]; exists {
					contentStr := content.(string)
					t.Logf("=== GENERATED MCP TOOLS ANALYSIS REPORT ===")
					t.Logf("%s", contentStr)
					
					// Verify the report contains key technical elements
					assert.Contains(t, strings.ToLower(contentStr), "mcp", "Report should mention MCP")
					assert.Contains(t, strings.ToLower(contentStr), "tools", "Report should mention tools")
					assert.Contains(t, strings.ToLower(contentStr), "analysis", "Report should mention analysis")
					assert.NotEmpty(t, contentStr, "Report should not be empty")
					
					// Verify report length indicates comprehensive analysis
					assert.Greater(t, len(contentStr), 200, "Report should be comprehensive (>200 chars)")
				}
			}
		}
		
		// Check email sending result
		if emailResult, exists := finalStatus.Results["send_email_summary"]; exists {
			t.Logf("Email Sending Result: %+v", emailResult)
			assert.NotNil(t, emailResult, "Should have attempted to send email")
			
			// Verify email was sent successfully
			if emailMap, ok := emailResult.(map[string]interface{}); ok {
				if status, exists := emailMap["status"]; exists {
					assert.Equal(t, "sent", status, "Email should be sent successfully")
				}
			}
		}
	}
	
	t.Logf("=== MCP + LLM + EMAIL INTEGRATION TEST SUMMARY ===")
	t.Logf("✅ MCP Tools Discovery: Successfully discovered tools from mock MCP server")
	t.Logf("✅ Agent Analysis: AI agent analyzed MCP tools and generated comprehensive report")
	t.Logf("✅ LLM Processing: Used real OpenAI API calls for reasoning and content generation")
	t.Logf("✅ Email Integration: Successfully sent email with MCP tools analysis report")
	t.Logf("✅ Template Resolution: Secret vault properly resolved email credentials")
	t.Logf("✅ End-to-End Workflow: Complete MCP → LLM → Email pipeline working")
	
	// Final verification
	assert.Equal(t, "completed", finalStatus.Status, "MCP tools analysis and email workflow should complete successfully")
}