package api

import (
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/tcmartin/flowrunner/pkg/loader"
	"github.com/tcmartin/flowrunner/pkg/plugins"
)

// This test validates that the simplified ai.agent YAML parses successfully
// and constructs a graph with the expected start node.
// It does not invoke real LLMs.
func TestAIAgentYAMLAbstraction_ParseOnly(t *testing.T) {
	nodeFactories := map[string]plugins.NodeFactory{}
	pluginRegistry := plugins.NewPluginRegistry()
	// Register the ai.agent plugin
	require.NoError(t, pluginRegistry.Register("ai.agent", &plugins.AIAgentPlugin{}))

	yamlLoader := loader.NewYAMLLoader(nodeFactories, pluginRegistry)

	yaml := `
metadata:
  name: "AI Agent Simple Flow"
  version: "1.0.0"
  description: "Simple AI agent abstraction"

nodes:
  agent:
    type: "ai.agent"
    params:
      provider: "openai"
      api_key: "sk-TEST"
      model: "gpt-4.1-mini"
      prompt: "Answer: ${input.question}"
      max_steps: 4
      temperature: 0.2
      tools:
        - type: function
          function:
            name: search_web
            description: Search the web
            parameters:
              type: object
              properties:
                query:
                  type: string
              required: [query]
`

	flow, err := yamlLoader.Parse(yaml)
	require.NoError(t, err)
	require.NotNil(t, flow)

	// Ensure there is a start node
	start := flow.Start()
	require.NotNil(t, start)
}
