package runtime

import (
	"fmt"
	"log"
	"time"

	"github.com/robertkrimen/otto"
	"github.com/tcmartin/flowlib"
	"github.com/tcmartin/flowrunner/pkg/utils"
)

// RouterNodeWrapper provides intelligent routing for LLM tool calls
type RouterNodeWrapper struct {
	*NodeWrapper
	toolHelper *ToolExecutionHelper
}

// NewRouterNodeWrapper creates a new router node with tool call support
func NewRouterNodeWrapper(params map[string]interface{}) (flowlib.Node, error) {
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

			log.Printf("[Router] Processing input for tool call detection")

			// Extract tool calls from various possible locations in the input
			var toolCalls []utils.ToolCall
			var foundLocation string
			
			// First check nodeParams.input if it exists (explicit input parameter)
			var inputToCheck map[string]interface{}
			if nodeInput, ok := nodeParams["input"].(map[string]interface{}); ok {
				inputToCheck = nodeInput
			} else {
				inputToCheck = flowInput
			}

			// Check multiple possible locations for tool calls
			if inputToCheck != nil {
				// Check input.tool_calls
				if tcValue, exists := inputToCheck["tool_calls"]; exists {
					if tc, ok := tcValue.([]interface{}); ok && len(tc) > 0 {
						toolCalls = convertToToolCalls(tc)
						foundLocation = "input.tool_calls"
					} else if utilsToolCalls, ok := tcValue.([]utils.ToolCall); ok && len(utilsToolCalls) > 0 {
						toolCalls = utilsToolCalls
						foundLocation = "input.tool_calls"
					}
				}
				// Check input.result.tool_calls
				if len(toolCalls) == 0 {
					if result, ok := inputToCheck["result"].(map[string]interface{}); ok {
						if tc, ok := result["tool_calls"].([]interface{}); ok && len(tc) > 0 {
							toolCalls = convertToToolCalls(tc)
							foundLocation = "input.result.tool_calls"
						}
					}
				}
				// Check input.choices[0].message.tool_calls (OpenAI format)
				if len(toolCalls) == 0 {
					if choices, ok := inputToCheck["choices"].([]interface{}); ok && len(choices) > 0 {
						if choice, ok := choices[0].(map[string]interface{}); ok {
							if message, ok := choice["message"].(map[string]interface{}); ok {
								if tc, ok := message["tool_calls"].([]interface{}); ok && len(tc) > 0 {
									toolCalls = convertToToolCalls(tc)
									foundLocation = "input.choices[0].message.tool_calls"
								}
							}
						}
					}
				}
				// Check input.message.tool_calls
				if len(toolCalls) == 0 {
					if message, ok := inputToCheck["message"].(map[string]interface{}); ok {
						if tc, ok := message["tool_calls"].([]interface{}); ok && len(tc) > 0 {
							toolCalls = convertToToolCalls(tc)
							foundLocation = "input.message.tool_calls"
						}
					}
				}
				// Check input.llm_result.tool_calls
				if len(toolCalls) == 0 {
					if llmResult, ok := inputToCheck["llm_result"].(map[string]interface{}); ok {
						if tc, ok := llmResult["tool_calls"].([]interface{}); ok && len(tc) > 0 {
							toolCalls = convertToToolCalls(tc)
							foundLocation = "input.llm_result.tool_calls"
						}
						// Also check input.llm_result.message.tool_calls
						if len(toolCalls) == 0 {
							if message, ok := llmResult["message"].(map[string]interface{}); ok {
								if tc, ok := message["tool_calls"].([]interface{}); ok && len(tc) > 0 {
									toolCalls = convertToToolCalls(tc)
									foundLocation = "input.llm_result.message.tool_calls"
								}
							}
						}
						// Also check input.llm_result.choices[0].message.tool_calls
						if len(toolCalls) == 0 {
							if choices, ok := llmResult["choices"].([]interface{}); ok && len(choices) > 0 {
								if choice, ok := choices[0].(map[string]interface{}); ok {
									if message, ok := choice["message"].(map[string]interface{}); ok {
										if tc, ok := message["tool_calls"].([]interface{}); ok && len(tc) > 0 {
											toolCalls = convertToToolCalls(tc)
											foundLocation = "input.llm_result.choices[0].message.tool_calls"
										}
									}
								}
							}
						}
					}
				}
			}

			log.Printf("[Router] Found %d tool calls at location: %s", len(toolCalls), foundLocation)

			// If no tool calls found, check if we should execute custom routing logic
			if len(toolCalls) == 0 {
				// Execute custom condition script if provided
				if conditionScript, ok := nodeParams["condition_script"].(string); ok {
					return executeConditionScript(conditionScript, flowInput)
				}
				
				// Default: no tool calls, route to 'output' or 'default'
				log.Printf("[Router] No tool calls found, routing to default/output")
				return map[string]interface{}{
					"route": "output",
					"reason": "no_tool_calls",
				}, nil
			}

			// Process the first tool call (for now, handle one at a time)
			toolCall := toolCalls[0]
			log.Printf("[Router] Processing tool call: %s", toolCall.Function.Name)

			// Determine the route based on the tool name
			// First check if there's a next route configuration in nodeParams
			var route string
			if nextRoutes, ok := nodeParams["next"].(map[string]interface{}); ok {
				if routeName, exists := nextRoutes[toolCall.Function.Name]; exists {
					if routeStr, ok := routeName.(string); ok {
						route = routeStr
					}
				}
			}
			
			// Fallback to hardcoded mappings if no YAML route found
			if route == "" {
				route = determineRouteForTool(toolCall.Function.Name)
			}
			
			// Extract parameters for the target node
			nodeType := getNodeTypeForRoute(route)
			toolParams, err := toolHelper.ExtractToolCallParameters(toolCall, nodeType)
			if err != nil {
				log.Printf("[Router] Failed to extract tool parameters: %v", err)
				return map[string]interface{}{
					"route": "error",
					"error": fmt.Sprintf("Failed to extract tool parameters: %v", err),
				}, nil
			}

			log.Printf("[Router] Routing to '%s' with parameters: %v", route, toolParams)

			// Return the tool execution data that will be stored in shared context
			return map[string]interface{}{
				"route": route,
				"tool_call": toolCall,
				"tool_params": toolParams,
				"tool_name": toolCall.Function.Name,
				"original_input": flowInput,
			}, nil
		},
		post: func(shared, params, result interface{}) (flowlib.Action, error) {
			if resultMap, ok := result.(map[string]interface{}); ok {
				// Store the active tool call in the shared context for downstream nodes
				if sharedMap, ok := shared.(map[string]interface{}); ok {
					if toolCall, exists := resultMap["tool_call"]; exists {
						sharedMap["active_tool_call"] = toolCall
					}
				}

				// Extract the tool name from the result to use as the action for routing
				if toolName, ok := resultMap["tool_name"].(string); ok {
					log.Printf("[Router] Returning action: %s", toolName)
					return flowlib.Action(toolName), nil
				}
			}

			// Fallback to default action if no tool name found
			log.Printf("[Router] No tool name found in result, using default action")
			return flowlib.DefaultAction, nil
		},
	}

	// Set the parameters
	wrapper.SetParams(params)

	return wrapper, nil
}

// convertToToolCalls converts interface{} tool calls to utils.ToolCall structs
func convertToToolCalls(toolCallsInterface []interface{}) []utils.ToolCall {
	var toolCalls []utils.ToolCall
	
	for _, tcInterface := range toolCallsInterface {
		if tcMap, ok := tcInterface.(map[string]interface{}); ok {
			var toolCall utils.ToolCall
			
			// Extract ID
			if id, ok := tcMap["id"].(string); ok {
				toolCall.ID = id
			}
			
			// Extract type
			if tcType, ok := tcMap["type"].(string); ok {
				toolCall.Type = tcType
			}
			
			// Extract function
			if function, ok := tcMap["function"].(map[string]interface{}); ok {
				if name, ok := function["name"].(string); ok {
					toolCall.Function.Name = name
				}
				if args, ok := function["arguments"].(string); ok {
					toolCall.Function.Arguments = args
				}
			} else if function, ok := tcMap["Function"].(map[string]interface{}); ok {
				// Handle capitalized version
				if name, ok := function["Name"].(string); ok {
					toolCall.Function.Name = name
				}
				if args, ok := function["Arguments"].(string); ok {
					toolCall.Function.Arguments = args
				}
			}
			
			toolCalls = append(toolCalls, toolCall)
		}
	}
	
	return toolCalls
}

// determineRouteForTool determines the route name based on tool name
func determineRouteForTool(toolName string) string {
	switch toolName {
	case "get_website", "fetch_url", "scrape_text":
		return "http_tool"
	case "search_web", "google_search", "search_google":
		return "search_tool"
	case "send_email", "send_email_summary":
		return "email_tool"
	default:
		return "unknown_tool"
	}
}

// getNodeTypeForRoute returns the node type for a given route
func getNodeTypeForRoute(route string) string {
	switch route {
	case "http_tool", "search_tool":
		return "http.request"
	case "email_tool":
		return "email.send"
	default:
		return "transform"
	}
}

// executeConditionScript executes a custom JavaScript condition script
func executeConditionScript(script string, input map[string]interface{}) (interface{}, error) {
	// Create JavaScript engine using Otto
	vm := otto.New()

	// Set up console.log for debugging
	vm.Set("console", map[string]interface{}{
		"log": func(args ...interface{}) {
			log.Printf("[Router Script] %v", args...)
		},
	})

	// Set up the context
	vm.Set("input", input)
	
	// Add shared context if available
	if shared, ok := input["shared"]; ok {
		vm.Set("shared", shared)
	}

	// Execute the condition script
	wrappedScript := "(function() {\n" + script + "\n})()"
	result, err := vm.Run(wrappedScript)
	if err != nil {
		return nil, fmt.Errorf("failed to execute condition script: %w", err)
	}

	// Convert result to Go value
	goValue, err := result.Export()
	if err != nil {
		return nil, fmt.Errorf("failed to export JavaScript result: %w", err)
	}

	// If the result is a string, treat it as a route
	if routeStr, ok := goValue.(string); ok {
		return map[string]interface{}{
			"route": routeStr,
			"reason": "custom_script",
		}, nil
	}

	// Otherwise return the raw result
	return goValue, nil
}


