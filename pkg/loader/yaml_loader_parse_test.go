package loader_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/tcmartin/flowlib"

	"github.com/tcmartin/flowrunner/pkg/loader"
	"github.com/tcmartin/flowrunner/pkg/plugins"
)

func TestYAMLLoader_Parse_SimpleFlow(t *testing.T) {
	// Create a map of node factories
	nodeFactories := map[string]plugins.NodeFactory{
		"base": &loader.BaseNodeFactory{},
	}

	// Create a new YAML loader
	yamlLoader := loader.NewYAMLLoader(nodeFactories, plugins.NewPluginRegistry())

	// Define a simple flow in YAML
	yamlContent := `
metadata:
  name: simple-flow
nodes:
  start:
    type: base
    next:
      default: end
  end:
    type: base
`

	// Parse the YAML
	flow, err := yamlLoader.Parse(yamlContent)

	// Assert that there was no error
	assert.NoError(t, err)
	assert.NotNil(t, flow)

	// Verify the flow structure (basic check)
	startNode := flow.Start()
	assert.NotNil(t, startNode)

	// Check if the start node has a successor
	successors := startNode.Successors()
	assert.Contains(t, successors, flowlib.DefaultAction)
	assert.NotNil(t, successors[flowlib.DefaultAction])
}

func TestYAMLLoader_Parse_FlowWithRetryAndBatch(t *testing.T) {
	// Create a map of node factories including all new types
	nodeFactories := map[string]plugins.NodeFactory{
		"base":           &loader.BaseNodeFactory{},
		"batch":          &loader.BatchNodeFactory{},
		"async_batch":    &loader.AsyncBatchNodeFactory{},
		"parallel_batch": &loader.AsyncParallelBatchNodeFactory{},
		"worker_pool":    &loader.WorkerPoolBatchNodeFactory{},
	}

	// Create a new YAML loader
	yamlLoader := loader.NewYAMLLoader(nodeFactories, plugins.NewPluginRegistry())

	// Define a complex flow in YAML
	yamlContent := `
metadata:
  name: complex-flow
nodes:
  start:
    type: base
    retry:
      max_retries: 3
      wait: 100ms
    next:
      default: process_data
  process_data:
    type: worker_pool
    batch:
      strategy: worker_pool
      max_parallel: 5
    next:
      success: end
      failure: error_node
  error_node:
    type: base
  end:
    type: base
`

	// Parse the YAML
	flow, err := yamlLoader.Parse(yamlContent)

	// Assert that there was no error
	assert.NoError(t, err)
	assert.NotNil(t, flow)

	// Verify the flow structure and node properties
	startNode := flow.Start()
	assert.NotNil(t, startNode)

	// Check start node properties
	baseNode, ok := startNode.(*flowlib.NodeWithRetry)
	assert.True(t, ok)
	assert.Equal(t, 3, baseNode.MaxRetries)
	assert.Equal(t, 100*time.Millisecond, baseNode.Wait)

	// Check process_data node properties
	processDataNode := startNode.Successors()[flowlib.DefaultAction]
	assert.NotNil(t, processDataNode)
	workerPoolNode, ok := processDataNode.(*flowlib.WorkerPoolBatchNode)
	assert.True(t, ok)
	assert.Equal(t, 5, workerPoolNode.MaxParallel)

	// Check branching
	assert.Contains(t, processDataNode.Successors(), "success")
	assert.Contains(t, processDataNode.Successors(), "failure")
}

func TestYAMLLoader_Parse_NoStartNode(t *testing.T) {
	// Create a map of node factories
	nodeFactories := map[string]plugins.NodeFactory{
		"base": &loader.BaseNodeFactory{},
	}

	// Create a new YAML loader
	yamlLoader := loader.NewYAMLLoader(nodeFactories, plugins.NewPluginRegistry())

	// Define a flow with no clear start node (circular reference)
	yamlContent := `
metadata:
  name: circular-flow
nodes:
  nodeA:
    type: base
    next:
      default: nodeB
  nodeB:
    type: base
    next:
      default: nodeA
`

	// Parse the YAML
	flow, err := yamlLoader.Parse(yamlContent)

	// Assert that an error occurred and flow is nil
	assert.Error(t, err)
	assert.Nil(t, flow)
	assert.Contains(t, err.Error(), "no start node found")
}

func TestYAMLLoader_Parse_MultipleStartNodes(t *testing.T) {
	// Create a map of node factories
	nodeFactories := map[string]plugins.NodeFactory{
		"base": &loader.BaseNodeFactory{},
	}

	// Create a new YAML loader
	yamlLoader := loader.NewYAMLLoader(nodeFactories, plugins.NewPluginRegistry())

	// Define a flow with multiple potential start nodes
	yamlContent := `
metadata:
  name: multiple-start-nodes
nodes:
  nodeA:
    type: base
  nodeB:
    type: base
`

	// Parse the YAML
	flow, err := yamlLoader.Parse(yamlContent)

	// Assert that an error occurred and flow is nil
	assert.Error(t, err)
	assert.Nil(t, flow)
	assert.Contains(t, err.Error(), "multiple start nodes found")
}