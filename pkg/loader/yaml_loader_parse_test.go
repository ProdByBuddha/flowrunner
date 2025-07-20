package loader_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	

	"github.com/tcmartin/flowrunner/pkg/loader"
)

func TestYAMLLoader_Parse_SimpleFlow(t *testing.T) {
	// Create a map of node factories
	nodeFactories := map[string]loader.NodeFactory{
		"base": &loader.BaseNodeFactory{},
	}

	// Create a new YAML loader
	yamlLoader := loader.NewYAMLLoader(nodeFactories)

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
}
