package runtime

import (
	"fmt"
	"time"

	"github.com/tcmartin/flowlib"
)

// AgentNodeWrapper is a wrapper for AI agent nodes that delegates to LLM node
type AgentNodeWrapper struct {
	*NodeWrapper
	llmNode flowlib.Node
}

// Agent node now delegates to LLM node for consistency with the new architecture

// NewAgentNodeWrapper creates a new AI agent node wrapper that delegates to LLM node
func NewAgentNodeWrapper(params map[string]interface{}) (flowlib.Node, error) {
	// Create the base node
	baseNode := flowlib.NewNode(3, 30*time.Second)

	// Create an LLM node with agent-specific defaults
	llmParams := make(map[string]interface{})
	
	// Copy all params to LLM node
	for k, v := range params {
		llmParams[k] = v
	}
	
	// Set agent-specific defaults if not provided
	if _, ok := llmParams["temperature"]; !ok {
		llmParams["temperature"] = 0.7
	}
	
	// Add agent system message if not provided
	if _, ok := llmParams["messages"]; !ok {
		llmParams["messages"] = []map[string]interface{}{
			{
				"role":    "system",
				"content": "You are a helpful AI assistant with reasoning capabilities. You can use tools to help you answer questions. When you need to use a tool, call it and wait for the result before continuing.",
			},
		}
	}
	
	// Create the underlying LLM node
	llmNode, err := NewLLMNodeWrapper(llmParams)
	if err != nil {
		return nil, fmt.Errorf("failed to create underlying LLM node: %w", err)
	}

	// Create the wrapper
	wrapper := &NodeWrapper{
		node: baseNode,
		exec: func(input interface{}) (interface{}, error) {
			// Handle both old format (direct params) and new format (combined input)
			var nodeParams map[string]interface{}
			var flowInput map[string]interface{}
			
			if combinedInput, ok := input.(map[string]interface{}); ok {
				if paramsField, hasParams := combinedInput["params"]; hasParams {
					// New format: combined input with params and input
					if paramsMap, ok := paramsField.(map[string]interface{}); ok {
						nodeParams = paramsMap
					} else {
						return nil, fmt.Errorf("expected params to be map[string]interface{}")
					}
					
					// Extract flow input
					if inputField, hasInput := combinedInput["input"]; hasInput {
						if inputMap, ok := inputField.(map[string]interface{}); ok {
							flowInput = inputMap
						}
					}
				} else {
					// Old format: direct params (backwards compatibility)
					nodeParams = combinedInput
					flowInput = combinedInput
				}
			} else {
				return nil, fmt.Errorf("expected map[string]interface{}, got %T", input)
			}

			// Overwrite llmParams with the resolved params from the current execution.
			// This ensures that secrets and template variables are correctly processed.
			for k, v := range nodeParams {
				llmParams[k] = v
			}

			// Map the agent's 'prompt' parameter to the underlying LLM's 'question' parameter
			// for clear and direct delegation. This is more robust than relying on fallback logic.
			if prompt, ok := llmParams["prompt"].(string); ok && prompt != "" {
				llmParams["question"] = prompt
				delete(llmParams, "prompt") // Avoid passing both 'question' and 'prompt'
			}

			// Prepare input for the LLM node
			llmInput := map[string]interface{}{
				"params": llmParams,
				"input":  flowInput,
			}
			
			// The agent node should use the same execution pattern as other nodes
			// We need to call the exec function directly since we're in a NodeWrapper
			if wrapper, ok := llmNode.(*NodeWrapper); ok {
				result, err := wrapper.exec(llmInput)
				if err != nil {
					return nil, fmt.Errorf("agent LLM execution failed: %w", err)
				}
				
				// Wrap the result with agent-specific metadata
				if resultMap, ok := result.(map[string]interface{}); ok {
					// Add agent-specific fields
					resultMap["agent_type"] = "delegated_llm"
					resultMap["node_type"] = "agent"
					
					// For backwards compatibility, also provide response field
					if content, ok := resultMap["content"].(string); ok {
						resultMap["response"] = content
					}
					
					return resultMap, nil
				}
				
				return result, nil
			}
			
			return nil, fmt.Errorf("LLM node is not a NodeWrapper")
		},
	}

	// Set the parameters
	wrapper.SetParams(params)

	return wrapper, nil
}
