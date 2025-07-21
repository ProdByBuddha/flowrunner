package plugins

import (
	"fmt"
	"github.com/tcmartin/flowlib"
)

// ExampleNodePlugin is a simple custom node plugin that demonstrates how to implement NodePlugin.
type ExampleNodePlugin struct{}

// Name returns the name of the plugin.
func (p *ExampleNodePlugin) Name() string {
	return "example_node"
}

// Description returns a description of the plugin.
func (p *ExampleNodePlugin) Description() string {
	return "A simple example custom node for demonstration purposes."
}

// Version returns the version of the plugin.
func (p *ExampleNodePlugin) Version() string {
	return "1.0.0"
}

// CreateNode creates a new instance of the example_node.
// It demonstrates how to use parameters from the YAML definition and return a flowlib.Node.
func (p *ExampleNodePlugin) CreateNode(params map[string]interface{}) (flowlib.Node, error) {
	message, ok := params["message"].(string)
	if !ok || message == "" {
		return nil, fmt.Errorf("example_node: 'message' parameter is required and must be a string")
	}

	// Create a basic flowlib.NodeWithRetry
	node := flowlib.NewNode(0, 0) // No retries or wait for this example

	// Set the execution function for the node
	node.SetExecFn(func(input interface{}) (interface{}, error) {
		fmt.Printf("ExampleNode: %s (Input: %v)\n", message, input)
		return map[string]interface{}{"status": "executed", "message": message, "input_received": input}, nil
	})

	return node, nil
}
