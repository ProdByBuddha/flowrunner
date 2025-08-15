package runtime

import (
	"encoding/json"
	"fmt"
	"log"
	"time"

	"github.com/robertkrimen/otto"
	"github.com/tcmartin/flowlib"
	"github.com/tcmartin/flowrunner/pkg/utils"
)

// ResponseFormatterNodeWrapper formats tool execution results for LLM consumption
type ResponseFormatterNodeWrapper struct {
	*NodeWrapper
	toolHelper *ToolExecutionHelper
}

// NewResponseFormatterNodeWrapper creates a new response formatter node
func NewResponseFormatterNodeWrapper(params map[string]interface{}) (flowlib.Node, error) {
	// Create the base node
	baseNode := flowlib.NewNode(1, 5*time.Second)

	// Create tool helper
	toolHelper := NewToolExecutionHelper()

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

			log.Printf("[Response Formatter] Processing tool execution result")

			// Execute custom script if provided
			if script, ok := nodeParams["script"].(string); ok {
				return executeToolResponseScript(script, flowInput)
			}

			// Default behavior: format tool response
			return formatToolResponse(toolHelper, flowInput)
		},
	}

	// Set the parameters
	wrapper.SetParams(params)

	return wrapper, nil
}

// executeToolResponseScript executes a custom JavaScript script for tool response formatting
func executeToolResponseScript(script string, input map[string]interface{}) (interface{}, error) {
	// Create JavaScript engine using Otto
	vm := otto.New()

	// Set up console.log for debugging
	vm.Set("console", map[string]interface{}{
		"log": func(args ...interface{}) {
			log.Printf("[Response Script] %v", args...)
		},
	})

	// Set up the context
	vm.Set("input", input)
	
	// Add shared context if available
	if shared, ok := input["shared"]; ok {
		vm.Set("shared", shared)
	}

	// Add secrets context if available
	if secrets, ok := input["secrets"]; ok {
		vm.Set("secrets", secrets)
	}

	// Execute the script
	wrappedScript := "(function() {\n" + script + "\n})()"
	result, err := vm.Run(wrappedScript)
	if err != nil {
		return nil, fmt.Errorf("failed to execute tool response script: %w", err)
	}

	// Convert result to Go value
	goValue, err := result.Export()
	if err != nil {
		return nil, fmt.Errorf("failed to export JavaScript result: %w", err)
	}

	return goValue, nil
}

// formatToolResponse formats a tool execution result into a conversation message
func formatToolResponse(toolHelper *ToolExecutionHelper, input map[string]interface{}) (interface{}, error) {
	// Extract tool call information
	var toolCall utils.ToolCall
	if tc, ok := input["tool_call"].(map[string]interface{}); ok {
		// Convert map to ToolCall struct
		if tcBytes, err := json.Marshal(tc); err == nil {
			json.Unmarshal(tcBytes, &toolCall)
		}
	}

	// Extract tool execution result
	var toolResult interface{}
	var toolError error
	
	if err, ok := input["error"].(string); ok && err != "" {
		toolError = fmt.Errorf("%s", err)
	} else {
		toolResult = input
	}

	// Format the result using the tool helper
	formattedResult := toolHelper.FormatToolResult(toolCall, toolResult, toolError)

	// Create tool response message
	toolResponseMsg := map[string]interface{}{
		"role":        "tool",
		"name":        toolCall.Function.Name,
		"content":     formattedResult,
		"tool_call_id": toolCall.ID,
	}

	log.Printf("[Response Formatter] Formatted tool response: %s", formattedResult[:min(len(formattedResult), 100)])

	return map[string]interface{}{
		"tool_response": toolResponseMsg,
		"formatted_result": formattedResult,
		"tool_name": toolCall.Function.Name,
		"original_input": input,
	}, nil
}

// min returns the minimum of two integers
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}