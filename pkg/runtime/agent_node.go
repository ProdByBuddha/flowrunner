package runtime

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/tcmartin/flowlib"
	"github.com/tcmartin/flowrunner/pkg/utils"
)

// AgentNodeWrapper is a wrapper for AI agent nodes
type AgentNodeWrapper struct {
	*NodeWrapper
	client *utils.LLMClient
}

// ToolCall represents a tool call from the LLM
type ToolCall struct {
	ID       string `json:"id"`
	Type     string `json:"type"`
	Function struct {
		Name      string `json:"name"`
		Arguments string `json:"arguments"`
	} `json:"function"`
}

// AgentState represents the state of an agent during execution
type AgentState struct {
	Messages            []utils.Message
	Tools               []utils.ToolDefinition
	ToolHandlers        map[string]func(params map[string]interface{}) (interface{}, error)
	MaxSteps            int
	CurrentStep         int
	Thinking            string
	IntermediateResults []map[string]interface{}
}

// CreateAgentNode creates a new AI agent node wrapper
// This is the actual implementation that will be used by the CoreNodeTypes function
func NewAgentNodeWrapper(params map[string]interface{}) (flowlib.Node, error) {
	// Create the base node
	baseNode := flowlib.NewNode(3, 30*time.Second)

	// Create the wrapper
	wrapper := &NodeWrapper{
		node: baseNode,
		exec: func(input interface{}) (interface{}, error) {
			// Handle both old format (direct params) and new format (combined input)
			var params map[string]interface{}
			
			if combinedInput, ok := input.(map[string]interface{}); ok {
				if nodeParams, hasParams := combinedInput["params"]; hasParams {
					// New format: combined input with params and input
					if paramsMap, ok := nodeParams.(map[string]interface{}); ok {
						params = paramsMap
					} else {
						return nil, fmt.Errorf("expected params to be map[string]interface{}")
					}
				} else {
					// Old format: direct params (backwards compatibility)
					params = combinedInput
				}
			} else {
				return nil, fmt.Errorf("expected map[string]interface{}, got %T", input)
			}

			// Extract provider
			providerStr, ok := params["provider"].(string)
			if !ok {
				providerStr = "openai" // Default to OpenAI
			}

			var provider utils.LLMProvider
			switch providerStr {
			case "openai":
				provider = utils.OpenAI
			case "anthropic":
				provider = utils.Anthropic
			default:
				provider = utils.Generic
			}

			// Extract API key
			apiKey, ok := params["api_key"].(string)
			if !ok {
				return nil, fmt.Errorf("api_key parameter is required")
			}

			// Extract model
			model, ok := params["model"].(string)
			if !ok {
				return nil, fmt.Errorf("model parameter is required")
			}

			// Extract prompt
			prompt, ok := params["prompt"].(string)
			if !ok {
				return nil, fmt.Errorf("prompt parameter is required")
			}

			// Extract max steps
			maxSteps := 10 // Default max steps
			if stepsParam, ok := params["max_steps"].(float64); ok {
				maxSteps = int(stepsParam)
			}

			// Extract temperature
			temperature := 0.7 // Default temperature
			if tempParam, ok := params["temperature"].(float64); ok {
				temperature = tempParam
			}

			// Extract tools
			tools := []utils.ToolDefinition{}
			toolHandlers := make(map[string]func(params map[string]interface{}) (interface{}, error))

			if toolsParam, ok := params["tools"].([]interface{}); ok {
				for _, toolInterface := range toolsParam {
					if toolMap, ok := toolInterface.(map[string]interface{}); ok {
						toolType := "function"
						if typeStr, ok := toolMap["type"].(string); ok {
							toolType = typeStr
						}

						if funcMap, ok := toolMap["function"].(map[string]interface{}); ok {
							name, _ := funcMap["name"].(string)
							desc, _ := funcMap["description"].(string)
							params, _ := funcMap["parameters"].(map[string]interface{})

							tools = append(tools, utils.ToolDefinition{
								Type: toolType,
								Function: utils.FunctionDefinition{
									Name:        name,
									Description: desc,
									Parameters:  params,
								},
							})

							// Add default handler that returns an error
							toolHandlers[name] = func(params map[string]interface{}) (interface{}, error) {
								return nil, fmt.Errorf("no handler defined for tool: %s", name)
							}
						}
					}
				}
			}

			// Add built-in tools
			if _, ok := toolHandlers["get_current_time"]; !ok {
				tools = append(tools, utils.ToolDefinition{
					Type: "function",
					Function: utils.FunctionDefinition{
						Name:        "get_current_time",
						Description: "Get the current time",
						Parameters:  map[string]interface{}{},
					},
				})

				toolHandlers["get_current_time"] = func(params map[string]interface{}) (interface{}, error) {
					now := time.Now()
					return map[string]interface{}{
						"time":     now.Format(time.RFC3339),
						"unix":     now.Unix(),
						"year":     now.Year(),
						"month":    now.Month().String(),
						"day":      now.Day(),
						"hour":     now.Hour(),
						"minute":   now.Minute(),
						"second":   now.Second(),
						"weekday":  now.Weekday().String(),
						"timezone": now.Location().String(),
					}, nil
				}
			}

			// Add custom tool handlers
			if handlersParam, ok := params["tool_handlers"].(map[string]interface{}); ok {
				for name, handler := range handlersParam {
					if handlerFunc, ok := handler.(func(params map[string]interface{}) (interface{}, error)); ok {
						toolHandlers[name] = handlerFunc
					}
				}
			}

			// Create LLM client
			client := utils.NewLLMClient(provider, apiKey, nil)

			// Initialize agent state
			state := &AgentState{
				Messages: []utils.Message{
					{
						Role:    "system",
						Content: "You are a helpful AI assistant with reasoning capabilities. You can use tools to help you answer questions.",
					},
					{
						Role:    "user",
						Content: prompt,
					},
				},
				Tools:               tools,
				ToolHandlers:        toolHandlers,
				MaxSteps:            maxSteps,
				CurrentStep:         0,
				IntermediateResults: []map[string]interface{}{},
			}

			// Run the agent
			finalResponse, err := runAgent(client, model, state, temperature)
			if err != nil {
				return nil, fmt.Errorf("agent execution failed: %w", err)
			}

			// Return the final response
			return map[string]interface{}{
				"response":             finalResponse,
				"steps":                state.CurrentStep,
				"thinking":             state.Thinking,
				"intermediate_results": state.IntermediateResults,
				"conversation":         state.Messages,
			}, nil
		},
	}

	// Set the parameters
	wrapper.SetParams(params)

	return wrapper, nil
}

// runAgent runs the agent until it reaches a conclusion or exceeds max steps
func runAgent(client *utils.LLMClient, model string, state *AgentState, temperature float64) (string, error) {
	ctx := context.Background()

	for state.CurrentStep < state.MaxSteps {
		state.CurrentStep++

		// Create LLM request
		request := utils.LLMRequest{
			Model:       model,
			Messages:    state.Messages,
			Temperature: temperature,
			Tools:       state.Tools,
		}

		// Execute request
		resp, err := client.Complete(ctx, request)
		if err != nil {
			return "", fmt.Errorf("LLM request failed: %w", err)
		}

		// Check for errors
		if resp.Error != nil {
			return "", fmt.Errorf("LLM API error: %s", resp.Error.Message)
		}

		// Extract response
		if len(resp.Choices) == 0 {
			return "", fmt.Errorf("no choices returned from LLM")
		}

		// Get the assistant's message
		assistantMessage := resp.Choices[0].Message

		// Add the assistant's message to the conversation
		state.Messages = append(state.Messages, assistantMessage)

		// Check if the assistant wants to use a tool by examining the raw response
		if resp.RawResponse != nil {
			// Try to extract tool calls from the raw response
			var toolCalls []ToolCall

			// Check if there are tool calls in the raw response
			if choices, ok := resp.RawResponse["choices"].([]interface{}); ok && len(choices) > 0 {
				if choice, ok := choices[0].(map[string]interface{}); ok {
					if message, ok := choice["message"].(map[string]interface{}); ok {
						if toolCallsRaw, ok := message["tool_calls"].([]interface{}); ok {
							// Parse tool calls
							for _, tc := range toolCallsRaw {
								if tcMap, ok := tc.(map[string]interface{}); ok {
									var toolCall ToolCall

									// Extract ID
									if id, ok := tcMap["id"].(string); ok {
										toolCall.ID = id
									}

									// Extract type
									if typ, ok := tcMap["type"].(string); ok {
										toolCall.Type = typ
									}

									// Extract function
									if function, ok := tcMap["function"].(map[string]interface{}); ok {
										if name, ok := function["name"].(string); ok {
											toolCall.Function.Name = name
										}
										if args, ok := function["arguments"].(string); ok {
											toolCall.Function.Arguments = args
										}
									}

									toolCalls = append(toolCalls, toolCall)
								}
							}
						}
					}
				}
			}

			// Process tool calls if any
			if len(toolCalls) > 0 {
				// Process each tool call
				for _, toolCall := range toolCalls {
					// Parse the tool call arguments
					var args map[string]interface{}
					if err := json.Unmarshal([]byte(toolCall.Function.Arguments), &args); err != nil {
						return "", fmt.Errorf("failed to parse tool arguments: %w", err)
					}

					// Get the tool handler
					handler, ok := state.ToolHandlers[toolCall.Function.Name]
					if !ok {
						// If no handler is found, return an error message
						toolResult := map[string]interface{}{
							"error": fmt.Sprintf("Tool not found: %s", toolCall.Function.Name),
						}

						// Add the tool result to the conversation
						state.Messages = append(state.Messages, utils.Message{
							Role:    "tool",
							Content: fmt.Sprintf("%v", toolResult),
						})

						// Add to intermediate results
						state.IntermediateResults = append(state.IntermediateResults, map[string]interface{}{
							"step":      state.CurrentStep,
							"tool":      toolCall.Function.Name,
							"arguments": args,
							"result":    toolResult,
							"error":     true,
						})

						continue
					}

					// Execute the tool
					result, err := handler(args)
					if err != nil {
						// If the tool execution fails, return an error message
						toolResult := map[string]interface{}{
							"error": fmt.Sprintf("Tool execution failed: %s", err.Error()),
						}

						// Add the tool result to the conversation
						state.Messages = append(state.Messages, utils.Message{
							Role:    "tool",
							Content: fmt.Sprintf("%v", toolResult),
						})

						// Add to intermediate results
						state.IntermediateResults = append(state.IntermediateResults, map[string]interface{}{
							"step":      state.CurrentStep,
							"tool":      toolCall.Function.Name,
							"arguments": args,
							"result":    toolResult,
							"error":     true,
						})

						continue
					}

					// Convert the result to a string
					var resultStr string
					switch r := result.(type) {
					case string:
						resultStr = r
					case []byte:
						resultStr = string(r)
					default:
						// Try to marshal the result to JSON
						resultBytes, err := json.Marshal(result)
						if err != nil {
							resultStr = fmt.Sprintf("%v", result)
						} else {
							resultStr = string(resultBytes)
						}
					}

					// Add the tool result to the conversation
					state.Messages = append(state.Messages, utils.Message{
						Role:    "tool",
						Content: resultStr,
					})

					// Add to intermediate results
					state.IntermediateResults = append(state.IntermediateResults, map[string]interface{}{
						"step":      state.CurrentStep,
						"tool":      toolCall.Function.Name,
						"arguments": args,
						"result":    result,
						"error":     false,
					})
				}

				// Continue to the next step
				continue
			}
		}

		// If we reach here, the assistant provided a final answer
		return assistantMessage.Content, nil
	}

	// If we reach here, we've exceeded the maximum number of steps
	return "", fmt.Errorf("exceeded maximum number of steps (%d)", state.MaxSteps)
}

// summarizeStep creates a summary of the assistant's message
func summarizeStep(message utils.Message) string {
	// Truncate long messages
	content := message.Content
	if len(content) > 100 {
		content = content[:97] + "..."
	}
	return fmt.Sprintf("Response: %s", content)
}
