package api

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/joho/godotenv"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/tcmartin/flowrunner/pkg/config"
	"github.com/tcmartin/flowlib"
	"github.com/tcmartin/flowrunner/pkg/loader"
	"github.com/tcmartin/flowrunner/pkg/plugins"
	"github.com/tcmartin/flowrunner/pkg/registry"
	"github.com/tcmartin/flowrunner/pkg/runtime"
	"github.com/tcmartin/flowrunner/pkg/services"
	"github.com/tcmartin/flowrunner/pkg/storage"
)

// TestComprehensiveToolExecution tests both LLM and Agent nodes with serial and parallel tool execution
func TestComprehensiveToolExecution(t *testing.T) {
	// Load environment variables
	_ = godotenv.Load("../../.env")

	// Skip test if OpenAI API key is not available
	apiKey := os.Getenv("OPENAI_API_KEY")
	if apiKey == "" {
		t.Skip("Skipping comprehensive tool execution test: OPENAI_API_KEY environment variable not set")
	}

	// Create in-memory storage provider
	storageProvider := storage.NewMemoryProvider()
	require.NoError(t, storageProvider.Initialize())

	// Create account service
	accountService := services.NewAccountService(storageProvider.GetAccountStore())

	// Create secret vault
	encryptionKey, err := services.GenerateEncryptionKey()
	require.NoError(t, err)
	secretVault, err := services.NewExtendedSecretVaultService(storageProvider.GetSecretStore(), encryptionKey)
	require.NoError(t, err)

	// Create plugin registry
	pluginRegistry := plugins.NewPluginRegistry()

	// Create YAML loader with core node types
	nodeFactories := make(map[string]plugins.NodeFactory)
	for nodeType, factory := range runtime.CoreNodeTypes() {
		nodeFactories[nodeType] = &ComprehensiveTestRuntimeNodeFactoryAdapter{factory: factory}
	}
	yamlLoader := loader.NewYAMLLoader(nodeFactories, pluginRegistry)

	// Create flow registry
	flowRegistry := registry.NewFlowRegistry(storageProvider.GetFlowStore(), registry.FlowRegistryOptions{
		YAMLLoader: yamlLoader,
	})

	// Create flow runtime adapter with secret vault support
	registryAdapter := &ComprehensiveTestFlowRegistryAdapter{registry: flowRegistry}
	executionStore := storageProvider.GetExecutionStore()
	flowRuntime := runtime.NewFlowRuntimeWithStoreAndSecrets(registryAdapter, yamlLoader, executionStore, secretVault)

	// Create configuration
	cfg := &config.Config{
		Server: config.ServerConfig{
			Host: "localhost",
			Port: 8080,
		},
	}

	// Using real httpbin.org endpoints for reliable testing

	// Create and start server
	server := NewServerWithRuntime(cfg, flowRegistry, accountService, secretVault, flowRuntime, pluginRegistry)
	testServer := httptest.NewServer(server.router)
	defer testServer.Close()

	t.Logf("Test server started at: %s", testServer.URL)
	t.Log("Using real httpbin.org endpoints for tool calls")

	// Create test user
	t.Log("Creating test user...")
	username := fmt.Sprintf("testuser-comprehensive-%d", time.Now().UnixNano())
	password := "testpassword123"

	accountReq := map[string]interface{}{
		"username": username,
		"password": password,
	}

	accountBody, err := json.Marshal(accountReq)
	require.NoError(t, err)

	resp, err := http.Post(
		testServer.URL+"/api/v1/accounts",
		"application/json",
		bytes.NewReader(accountBody),
	)
	require.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, http.StatusCreated, resp.StatusCode, "Failed to create account")

	var accountResp map[string]interface{}
	err = json.NewDecoder(resp.Body).Decode(&accountResp)
	require.NoError(t, err)

	accountID, ok := accountResp["id"].(string)
	require.True(t, ok, "Account ID should be returned")
	t.Logf("Created account: %s (ID: %s)", username, accountID)

	// Store OpenAI API key in secret vault directly (no HTTP API needed for test)
	t.Log("Storing OpenAI API key as secret...")
	err = secretVault.Set(accountID, "OPENAI_API_KEY", apiKey)
	require.NoError(t, err, "Failed to store OpenAI API key")
	
	// Verify the secret was stored correctly
	retrievedKey, err := secretVault.Get(accountID, "OPENAI_API_KEY")
	require.NoError(t, err, "Failed to retrieve stored API key")
	assert.Equal(t, apiKey, retrievedKey, "Retrieved API key should match stored key")
	t.Logf("Successfully stored and verified OpenAI API key")

	// Test scenarios
	testScenarios := []struct {
		name     string
		yamlFile string
		nodeType string
		toolMode string
	}{
		{
			name:     "LLM Node Serial Tool Execution",
			yamlFile: "testlogs/llm_serial_tool_execution.yaml",
			nodeType: "llm",
			toolMode: "serial",
		},
		{
			name:     "LLM Node Parallel Tool Execution",
			yamlFile: "testlogs/llm_parallel_tool_execution.yaml",
			nodeType: "llm",
			toolMode: "parallel",
		},
		{
			name:     "Agent Node Serial Tool Execution",
			yamlFile: "testlogs/agent_serial_tool_execution.yaml",
			nodeType: "agent",
			toolMode: "serial",
		},
		{
			name:     "Agent Node Parallel Tool Execution",
			yamlFile: "testlogs/agent_parallel_tool_execution.yaml",
			nodeType: "agent",
			toolMode: "parallel",
		},
	}

	for _, scenario := range testScenarios {
		t.Run(scenario.name, func(t *testing.T) {
			t.Logf("Testing scenario: %s", scenario.name)

			// Load and register flow
			flowContent, err := os.ReadFile(scenario.yamlFile)
			require.NoError(t, err, "Failed to read YAML file: %s", scenario.yamlFile)

			flowName := fmt.Sprintf("test-flow-%s-%d", strings.ToLower(strings.ReplaceAll(scenario.name, " ", "-")), time.Now().UnixNano())

			// Register flow directly with flow registry (no HTTP API needed for test)
			t.Logf("Registering flow: %s", flowName)
			flowID, err := flowRegistry.Create(accountID, flowName, string(flowContent))
			require.NoError(t, err, "Failed to register flow")
			
			t.Logf("Registered flow: %s (ID: %s)", flowName, flowID)



			// Execute flow directly with flow runtime
			t.Logf("Executing flow: %s", flowName)
			inputs := map[string]interface{}{}
			
			executionID, err := flowRuntime.Execute(accountID, flowID, inputs)
			require.NoError(t, err, "Failed to execute flow")
			
			t.Logf("Started execution: %s", executionID)
			
			// Poll for completion
			t.Log("Polling for execution completion...")
			maxAttempts := 60 // 60 seconds timeout
			var executionResult runtime.ExecutionStatus
			
			for attempt := 0; attempt < maxAttempts; attempt++ {
				executionResult, err = flowRuntime.GetStatus(executionID)
				require.NoError(t, err, "Failed to get execution status")
				
				t.Logf("Execution status (attempt %d): %s", attempt+1, executionResult.Status)
				
				if executionResult.Status == "completed" || executionResult.Status == "failed" {
					break
				}
				
				time.Sleep(1 * time.Second)
			}
			
			// Verify execution completed successfully
			assert.Equal(t, "completed", executionResult.Status, "Execution should complete successfully")

			// Verify results exist
			assert.NotEmpty(t, executionResult.Results, "Results should not be empty")

			// Scenario-specific validations
			switch scenario.nodeType {
			case "llm":
				// Verify LLM response exists
				if llmOutput, exists := executionResult.Results["final_response"]; exists {
					llmStr, ok := llmOutput.(string)
					require.True(t, ok, "LLM output should be string")
					assert.NotEmpty(t, llmStr, "LLM response should not be empty")
					t.Logf("LLM Response: %s", llmStr)

					// For tool calling scenarios, verify tool usage is mentioned
					if scenario.toolMode == "serial" {
						// Should mention content from httpbin.org/json response
						lowerResponse := strings.ToLower(llmStr)
						assert.True(t, strings.Contains(lowerResponse, "json") || strings.Contains(lowerResponse, "data") || strings.Contains(lowerResponse, "response"),
							"LLM should mention content from tool call response")
					} else if scenario.toolMode == "parallel" {
						// Should mention content from multiple httpbin endpoints
						lowerResponse := strings.ToLower(llmStr)
						assert.True(t, strings.Contains(lowerResponse, "json") || strings.Contains(lowerResponse, "uuid") || strings.Contains(lowerResponse, "data"),
							"LLM should mention content from parallel tool calls")
					}
				}

			case "agent":
				// Verify agent response exists
				if agentOutput, exists := executionResult.Results["final_response"]; exists {
					agentStr, ok := agentOutput.(string)
					require.True(t, ok, "Agent output should be string")
					assert.NotEmpty(t, agentStr, "Agent response should not be empty")
					t.Logf("Agent Response: %s", agentStr)

					// For tool calling scenarios, verify tool usage is mentioned
					if scenario.toolMode == "serial" {
						// Should mention content from httpbin.org/json response
						lowerResponse := strings.ToLower(agentStr)
						assert.True(t, strings.Contains(lowerResponse, "json") || strings.Contains(lowerResponse, "data") || strings.Contains(lowerResponse, "response"),
							"Agent should mention content from tool call response")
					} else if scenario.toolMode == "parallel" {
						// Should mention content from multiple httpbin endpoints
						lowerResponse := strings.ToLower(agentStr)
						assert.True(t, strings.Contains(lowerResponse, "json") || strings.Contains(lowerResponse, "uuid") || strings.Contains(lowerResponse, "data"),
							"Agent should mention content from parallel tool calls")
					}
				}
			}

			// Verify execution logs contain tool calls
			logs, err := flowRuntime.GetLogs(executionID)
			if err == nil && len(logs) > 0 {
				foundToolCall := false
				for _, logEntry := range logs {
					messageStr := logEntry.Message
					if strings.Contains(strings.ToLower(messageStr), "tool") ||
						strings.Contains(strings.ToLower(messageStr), "http") ||
						strings.Contains(strings.ToLower(messageStr), "request") {
						foundToolCall = true
						t.Logf("Found tool-related log: %s", messageStr)
						break
					}
				}

				assert.True(t, foundToolCall, "Should find evidence of tool calling in logs")
			}

			t.Logf("✅ Scenario %s completed successfully", scenario.name)
		})
	}
}

// ComprehensiveTestRuntimeNodeFactoryAdapter adapts runtime.NodeFactory to plugins.NodeFactory
type ComprehensiveTestRuntimeNodeFactoryAdapter struct {
	factory runtime.NodeFactory
}

func (a *ComprehensiveTestRuntimeNodeFactoryAdapter) CreateNode(nodeDef plugins.NodeDefinition) (flowlib.Node, error) {
	return a.factory(nodeDef.Params)
}

// ComprehensiveTestFlowRegistryAdapter adapts registry.FlowRegistry to runtime.FlowRegistry
type ComprehensiveTestFlowRegistryAdapter struct {
	registry registry.FlowRegistry
}

func (a *ComprehensiveTestFlowRegistryAdapter) GetFlow(accountID, flowID string) (*runtime.Flow, error) {
	content, err := a.registry.Get(accountID, flowID)
	if err != nil {
		return nil, err
	}

	return &runtime.Flow{
		ID:   flowID,
		YAML: content,
	}, nil
}