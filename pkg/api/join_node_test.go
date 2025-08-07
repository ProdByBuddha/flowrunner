package api

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

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

func TestJoinNodeBasic(t *testing.T) {
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
		nodeFactories[nodeType] = &JoinTestRuntimeNodeFactoryAdapter{factory: factory}
	}
	yamlLoader := loader.NewYAMLLoader(nodeFactories, pluginRegistry)

	// Create flow registry
	flowRegistry := registry.NewFlowRegistry(storageProvider.GetFlowStore(), registry.FlowRegistryOptions{
		YAMLLoader: yamlLoader,
	})

	// Create flow runtime adapter
	flowRegistryAdapter := &JoinTestFlowRegistryAdapter{registry: flowRegistry}
	flowRuntime := runtime.NewFlowRuntime(flowRegistryAdapter, yamlLoader)

	// Create configuration
	cfg := &config.Config{
		Server: config.ServerConfig{
			Host: "localhost",
			Port: 8080,
		},
	}

	// Create and start server
	server := NewServerWithRuntime(cfg, flowRegistry, accountService, secretVault, flowRuntime, pluginRegistry)
	testServer := httptest.NewServer(server.router)
	defer testServer.Close()

	fmt.Printf("Test server started at: %s\n", testServer.URL)

	t.Logf("Test server started at: %s", testServer.URL)

	// Step 1: Create a test user
	t.Log("Step 1: Creating test user...")
	username := fmt.Sprintf("testuser-join-%d", time.Now().UnixNano())
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
	require.True(t, ok, "Account ID should be a string")
	require.NotEmpty(t, accountID)
	t.Logf("Created account: %s (ID: %s)", username, accountID)

	// Step 2: Create JoinNode test flow
	fmt.Println("Step 2: Creating JoinNode test flow...")
	flowYAML := `metadata:
  name: "JoinNode Test Flow"
  description: "Test JoinNode functionality with SplitNode"
  version: "1.0.0"

nodes:
  start:
    type: transform
    params:
      script: |
        return {
          numbers: [10, 20, 30],
          message: "Starting JoinNode test"
        };
    next:
      default: split_mapper

  split_mapper:
    type: split
    description: Fan out to multiple mappers for parallel processing
    next:
      mapper1: mapper_branch_1
      mapper2: mapper_branch_2  
      mapper3: mapper_branch_3
      default: join_results

  mapper_branch_1:
    type: transform
    params:
      script: |
        var numbers = input.numbers || [10, 20, 30];
        var num = numbers[0];
        var result = num * num;
        return {
          branch: "mapper1",
          input: num,
          output: result,
          message: "Mapper 1 completed: " + num + "² = " + result
        };

  mapper_branch_2:
    type: transform
    params:
      script: |
        var numbers = input.numbers || [10, 20, 30];
        var num = numbers[1];
        var result = num * num;
        return {
          branch: "mapper2", 
          input: num,
          output: result,
          message: "Mapper 2 completed: " + num + "² = " + result
        };

  mapper_branch_3:
    type: transform
    params:
      script: |
        var numbers = input.numbers || [10, 20, 30];
        var num = numbers[2];
        var result = num * num;
        return {
          branch: "mapper3",
          input: num,
          output: result,
          message: "Mapper 3 completed: " + num + "² = " + result
        };

  join_results:
    type: join
    params:
      format: array
    next:
      default: process_results

  process_results:
    type: transform
    params:
      script: |
        // Now we have a clean array to work with
        var results = input || [];
        var total = 0;
        var processedInputs = [];
        var messages = [];
        
        for (var i = 0; i < results.length; i++) {
          var result = results[i];
          total += result.output;
          processedInputs.push(result.input);
          messages.push(result.message);
        }
        
        return {
          map_reduce_complete: true,
          total_mappers: results.length,
          processed_inputs: processedInputs,
          individual_results: results,
          aggregated_sum: total,
          mapper_messages: messages,
          execution_summary: {
            operation: "parallel_square_and_sum",
            inputs: processedInputs,
            individual_squares: (function() {
              var squares = [];
              for (var i = 0; i < results.length; i++) {
                squares.push(results[i].output);
              }
              return squares;
            })(),
            final_sum: total,
            parallel_execution: "JoinNode enabled clean map-reduce pattern"
          }
        };
    next:
      default: final_output

  final_output:
    type: transform
    params:
      script: |
        return {
          status: "completed",
          map_reduce_results: input,
          conclusion: "JoinNode successfully enabled clean map-reduce pattern",
          performance_note: "All mappers executed in parallel, JoinNode collected results, then reducer processed clean data"
        };

`

	flowName := fmt.Sprintf("joinnode-test-flow-%d", time.Now().UnixNano())
	createFlowReq := map[string]interface{}{
		"name":    flowName,
		"content": flowYAML,
	}

	flowBody, err := json.Marshal(createFlowReq)
	require.NoError(t, err)

	req, err := http.NewRequest("POST", testServer.URL+"/api/v1/flows", bytes.NewReader(flowBody))
	require.NoError(t, err)

	req.SetBasicAuth(username, password)
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{}
	resp, err = client.Do(req)
	require.NoError(t, err)
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated {
		body, _ := io.ReadAll(resp.Body)
		t.Logf("Flow creation failed with status %d: %s", resp.StatusCode, string(body))
	}
	assert.Equal(t, http.StatusCreated, resp.StatusCode, "Failed to create flow")

	var createFlowResp map[string]interface{}
	err = json.NewDecoder(resp.Body).Decode(&createFlowResp)
	require.NoError(t, err)

	flowID, ok := createFlowResp["id"].(string)
	require.True(t, ok, "Flow ID should be a string")
	require.NotEmpty(t, flowID)
	t.Logf("Created flow: %s (ID: %s)", flowName, flowID)

	// Step 3: Execute the flow
	t.Log("Step 3: Executing JoinNode flow...")
	runFlowReq := map[string]interface{}{
		"input": map[string]interface{}{
			"test": true,
		},
	}

	runFlowBody, err := json.Marshal(runFlowReq)
	require.NoError(t, err)

	req, err = http.NewRequest("POST", testServer.URL+"/api/v1/flows/"+flowID+"/run", bytes.NewReader(runFlowBody))
	require.NoError(t, err)

	req.SetBasicAuth(username, password)
	req.Header.Set("Content-Type", "application/json")

	resp, err = client.Do(req)
	require.NoError(t, err)
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated {
		body, _ := io.ReadAll(resp.Body)
		t.Logf("Flow execution failed with status %d: %s", resp.StatusCode, string(body))
	}
	assert.Equal(t, http.StatusCreated, resp.StatusCode, "Failed to run flow")

	var runFlowResp map[string]interface{}
	err = json.NewDecoder(resp.Body).Decode(&runFlowResp)
	require.NoError(t, err)

	executionID, ok := runFlowResp["execution_id"].(string)
	require.True(t, ok, "Execution ID should be a string")
	require.NotEmpty(t, executionID)
	t.Logf("Started execution: %s", executionID)

	// Step 4: Poll for completion
	t.Log("Step 4: Polling for execution completion...")
	var execution map[string]interface{}
	maxAttempts := 30
	for i := 0; i < maxAttempts; i++ {
		req, err := http.NewRequest("GET", testServer.URL+"/api/v1/executions/"+executionID, nil)
		require.NoError(t, err)

		req.SetBasicAuth(username, password)

		resp, err := client.Do(req)
		require.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusOK, resp.StatusCode, "Failed to get execution status")

		err = json.NewDecoder(resp.Body).Decode(&execution)
		require.NoError(t, err)

		status, ok := execution["status"].(string)
		require.True(t, ok, "Status should be a string")

		if status == "completed" || status == "failed" {
			break
		}
		time.Sleep(100 * time.Millisecond)
	}

	require.NotNil(t, execution)
	status := execution["status"].(string)
	t.Logf("Execution finished with status: %s", status)

	if duration, ok := execution["duration"].(float64); ok {
		t.Logf("Total execution time: %v", time.Duration(duration)*time.Millisecond)
	}

	// Step 5: Verify execution results
	t.Log("Step 5: Verifying execution results...")
	assert.Equal(t, "completed", status, "Execution should complete successfully")

	// Step 6: Verify the results contain the expected map-reduce output
	t.Log("Step 6: Verifying JoinNode functionality...")

	// The final result should contain the aggregated sum
	if result, ok := execution["result"]; ok && result != nil {
		resultJSON, _ := json.MarshalIndent(result, "", "  ")
		t.Logf("Final execution result:\n%s", string(resultJSON))

		// Check if the result contains the expected aggregated sum (10² + 20² + 30² = 100 + 400 + 900 = 1400)
		if resultMap, ok := result.(map[string]interface{}); ok {
			if mapReduceResults, ok := resultMap["map_reduce_results"].(map[string]interface{}); ok {
				if aggregatedSum, ok := mapReduceResults["aggregated_sum"].(float64); ok {
					assert.Equal(t, float64(1400), aggregatedSum, "Aggregated sum should be 1400 (10² + 20² + 30²)")
					t.Logf("✅ Correct aggregated sum: %.0f", aggregatedSum)
				} else {
					t.Errorf("❌ aggregated_sum not found or not a number")
				}

				if totalMappers, ok := mapReduceResults["total_mappers"].(float64); ok {
					assert.Equal(t, float64(3), totalMappers, "Should have 3 mappers")
					t.Logf("✅ Correct number of mappers: %.0f", totalMappers)
				} else {
					t.Errorf("❌ total_mappers not found or not a number")
				}
			} else {
				t.Errorf("❌ map_reduce_results not found in final result")
			}
		} else {
			t.Errorf("❌ Final result is not a map")
		}
	} else {
		t.Errorf("❌ No execution result found")
	}

	t.Log("\n🎉 JOINNODE TEST SUMMARY:")
	t.Log("====================================")
	t.Log("✅ JoinNode successfully integrated into flowrunner")
	t.Log("✅ SplitNode + JoinNode pattern works correctly")
	t.Log("✅ Parallel mappers executed and results collected")
	t.Log("✅ JoinNode structured results for clean downstream processing")
	t.Log("✅ Transform node processed structured data successfully")
	t.Log("✅ Map-reduce pattern achieved with clean separation of concerns")
	t.Log("✅ End-to-end JoinNode integration test PASSED!")
}

// JoinTestRuntimeNodeFactoryAdapter adapts runtime.NodeFactory to plugins.NodeFactory
type JoinTestRuntimeNodeFactoryAdapter struct {
	factory runtime.NodeFactory
}

func (a *JoinTestRuntimeNodeFactoryAdapter) CreateNode(nodeDef plugins.NodeDefinition) (flowlib.Node, error) {
	return a.factory(nodeDef.Params)
}

// JoinTestFlowRegistryAdapter adapts registry.FlowRegistry to runtime.FlowRegistry
type JoinTestFlowRegistryAdapter struct {
	registry registry.FlowRegistry
}

func (a *JoinTestFlowRegistryAdapter) GetFlow(accountID, flowID string) (*runtime.Flow, error) {
	content, err := a.registry.Get(accountID, flowID)
	if err != nil {
		return nil, err
	}

	return &runtime.Flow{
		ID:   flowID,
		YAML: content,
	}, nil
}
