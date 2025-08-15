package api

import (
	"os"
	"strings"
	"testing"
	"time"

	"github.com/joho/godotenv"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/tcmartin/flowlib"
	"github.com/tcmartin/flowrunner/pkg/loader"
	"github.com/tcmartin/flowrunner/pkg/plugins"
	"github.com/tcmartin/flowrunner/pkg/registry"
	"github.com/tcmartin/flowrunner/pkg/runtime"
	"github.com/tcmartin/flowrunner/pkg/services"
	"github.com/tcmartin/flowrunner/pkg/storage"
)

// TestSimpleLLMFlow tests a simple LLM flow with tool calling
func TestSimpleLLMFlow(t *testing.T) {
	// Load environment variables
	_ = godotenv.Load("../../.env")

	// Skip test if OpenAI API key is not available
	apiKey := os.Getenv("OPENAI_API_KEY")
	if apiKey == "" {
		t.Skip("Skipping simple LLM flow test: OPENAI_API_KEY environment variable not set")
	}

	// Create in-memory storage provider
	storageProvider := storage.NewMemoryProvider()
	require.NoError(t, storageProvider.Initialize())



	// Create secret vault
	encryptionKey, err := services.GenerateEncryptionKey()
	require.NoError(t, err)
	secretVault, err := services.NewExtendedSecretVaultService(storageProvider.GetSecretStore(), encryptionKey)
	require.NoError(t, err)

	// Create test account
	accountID := "test-account-simple"
	
	// Store OpenAI API key
	err = secretVault.Set(accountID, "OPENAI_API_KEY", apiKey)
	require.NoError(t, err)

	// Create plugin registry
	pluginRegistry := plugins.NewPluginRegistry()

	// Create YAML loader with core node types
	nodeFactories := make(map[string]plugins.NodeFactory)
	for nodeType, factory := range runtime.CoreNodeTypes() {
		nodeFactories[nodeType] = &SimpleLLMTestRuntimeNodeFactoryAdapter{factory: factory}
	}
	yamlLoader := loader.NewYAMLLoader(nodeFactories, pluginRegistry)

	// Create flow registry
	flowRegistry := registry.NewFlowRegistry(storageProvider.GetFlowStore(), registry.FlowRegistryOptions{
		YAMLLoader: yamlLoader,
	})

	// Create flow runtime with secret vault support
	registryAdapter := &SimpleLLMTestFlowRegistryAdapter{registry: flowRegistry}
	executionStore := storageProvider.GetExecutionStore()
	flowRuntime := runtime.NewFlowRuntimeWithStoreAndSecrets(registryAdapter, yamlLoader, executionStore, secretVault)

	// Load and register the simple flow
	flowContent, err := os.ReadFile("testlogs/simple_llm_tool_test.yaml")
	require.NoError(t, err, "Failed to read simple LLM test YAML")

	flowName := "simple-llm-tool-test"
	flowID, err := flowRegistry.Create(accountID, flowName, string(flowContent))
	require.NoError(t, err, "Failed to register flow")

	t.Logf("Registered flow: %s (ID: %s)", flowName, flowID)

	// Execute the flow
	t.Log("Executing simple LLM flow...")
	inputs := map[string]interface{}{}

	executionID, err := flowRuntime.Execute(accountID, flowID, inputs)
	require.NoError(t, err, "Failed to execute flow")

	t.Logf("Started execution: %s", executionID)

	// Poll for completion
	t.Log("Polling for execution completion...")
	maxAttempts := 30 // 30 seconds timeout
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

	// Log the results for inspection
	t.Logf("Execution results: %+v", executionResult.Results)

	// Verify logs contain evidence of tool calling
	logs, err := flowRuntime.GetLogs(executionID)
	if err == nil && len(logs) > 0 {
		foundToolCall := false
		for _, logEntry := range logs {
			messageStr := logEntry.Message
			if strings.Contains(strings.ToLower(messageStr), "tool") ||
				strings.Contains(strings.ToLower(messageStr), "http") ||
				strings.Contains(strings.ToLower(messageStr), "get_website") {
				foundToolCall = true
				t.Logf("Found tool-related log: %s", messageStr)
				break
			}
		}

		if foundToolCall {
			t.Log("✅ Found evidence of tool calling in logs")
		} else {
			t.Log("⚠️  No tool calling evidence found in logs")
		}
	}

	t.Log("✅ Simple LLM flow test completed")
}

// SimpleLLMTestRuntimeNodeFactoryAdapter adapts runtime.NodeFactory to plugins.NodeFactory
type SimpleLLMTestRuntimeNodeFactoryAdapter struct {
	factory runtime.NodeFactory
}

func (a *SimpleLLMTestRuntimeNodeFactoryAdapter) CreateNode(nodeDef plugins.NodeDefinition) (flowlib.Node, error) {
	return a.factory(nodeDef.Params)
}

// SimpleLLMTestFlowRegistryAdapter adapts registry.FlowRegistry to runtime.FlowRegistry
type SimpleLLMTestFlowRegistryAdapter struct {
	registry registry.FlowRegistry
}

func (a *SimpleLLMTestFlowRegistryAdapter) GetFlow(accountID, flowID string) (*runtime.Flow, error) {
	content, err := a.registry.Get(accountID, flowID)
	if err != nil {
		return nil, err
	}

	return &runtime.Flow{
		ID:   flowID,
		YAML: content,
	}, nil
}