package runtime

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/tcmartin/flowlib"
	"github.com/tcmartin/flowrunner/pkg/utils"
)

// Helper function to truncate strings for logging
func truncateString(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen] + "..."
}

// Helper function to recursively convert map[interface{}]interface{} to map[string]any
func convertInterfaceMapToStringMap(input map[interface{}]interface{}) map[string]any {
	result := make(map[string]any)
	for k, v := range input {
		keyStr, ok := k.(string)
		if !ok {
			continue
		}
		
		// Recursively convert nested maps
		switch val := v.(type) {
		case map[interface{}]interface{}:
			result[keyStr] = convertInterfaceMapToStringMap(val)
		default:
			result[keyStr] = v
		}
	}
	return result
}

// LLMNodeWrapper is a wrapper for LLM nodes
type LLMNodeWrapper struct {
	*NodeWrapper
	client *utils.LLMClient
}

// NewLLMNodeWrapper creates a new LLM node wrapper
func NewLLMNodeWrapper(params map[string]any) (flowlib.Node, error) {
	// Create the base node
	baseNode := flowlib.NewNode(3, 5*time.Second)

	// Create the wrapper
	wrapper := &NodeWrapper{
		node: baseNode,
		exec: func(input any) (any, error) {
			// Extract parameters and flow input from combined context
			combinedInput, ok := input.(map[string]interface{})
			if !ok {
				return nil, fmt.Errorf("expected map[string]interface{}, got %T", input)
			}

			// Get static node parameters
			params, ok := combinedInput["params"].(map[string]interface{})
			if !ok {
				return nil, fmt.Errorf("expected params to be map[string]interface{}")
			}

			// Get flow input (shared context)
			flowInput, _ := combinedInput["input"].(map[string]interface{})

			// Extract execution context for structured logging
			var executionID, flowID, nodeID string
			if flowInput != nil {
				if execCtx, ok := flowInput["_execution"].(map[string]interface{}); ok {
					if id, ok := execCtx["execution_id"].(string); ok {
						executionID = id
					}
					if fid, ok := execCtx["flow_id"].(string); ok {
						flowID = fid
					}
				}
			}

			// Helper function for structured logging
			logToExecution := func(level, message string, data map[string]interface{}) {
				if executionID != "" {
					// Add standard fields
					if data == nil {
						data = make(map[string]interface{})
					}
					data["node_type"] = "llm"
					if flowID != "" {
						data["flow_id"] = flowID
					}
					if nodeID != "" {
						data["node_id"] = nodeID
					}
					
					// Standard Go log for immediate visibility
					log.Printf("[LLM Node][%s] %s", level, message)
					
					// Check if we have a flow runtime logger in the execution context
					if flowInput != nil {
						if execCtx, ok := flowInput["_execution"].(map[string]interface{}); ok {
							if logger, ok := execCtx["logger"].(func(string, string, string, map[string]interface{})); ok {
								// Call the flow runtime's logging function
								logger(executionID, level, message, data)
							}
						}
					}
				} else {
					// Fallback to standard logging
					log.Printf("[LLM Node][%s] %s", level, message)
				}
			}

			// Convert params to map[string]any for compatibility
			paramsAny := make(map[string]any)
			for k, v := range params {
				paramsAny[k] = v
			}

			// Extract provider
			providerStr, ok := paramsAny["provider"].(string)
			if !ok {
				providerStr = "openai" // Default to OpenAI
			}

			// Log the start of LLM execution
			logToExecution("info", "Starting LLM execution", map[string]interface{}{
				"provider": providerStr,
			})

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
			apiKey, ok := paramsAny["api_key"].(string)
			if !ok {
				logToExecution("error", "api_key parameter is required", nil)
				return nil, fmt.Errorf("api_key parameter is required")
			}

			// Extract model
			model, ok := paramsAny["model"].(string)
			if !ok {
				logToExecution("error", "model parameter is required", nil)
				return nil, fmt.Errorf("model parameter is required")
			}

			logToExecution("info", "LLM configuration set", map[string]interface{}{
				"provider": providerStr,
				"model":    model,
			})

			// Extract messages - check if we should use dynamic input
			var messages []utils.Message

			// Priority order:
			// 1. If flow input contains "conversation_history", use it for multi-turn conversation
			// 2. If flow input contains "question", use it to override the prompt
			// 3. Otherwise use static parameters (templates, messages, prompt)
			
			if flowInput != nil {
				// Check for conversation history first (for multi-turn conversations)
				if convHistory, ok := flowInput["conversation_history"].([]interface{}); ok && len(convHistory) > 0 {
					logToExecution("info", "Using conversation history from flow input", map[string]interface{}{
						"history_length": len(convHistory),
					})
					
					// Convert conversation history to messages
					for _, msgInterface := range convHistory {
						if msgMap, ok := msgInterface.(map[string]interface{}); ok {
							role, _ := msgMap["role"].(string)
							content, _ := msgMap["content"].(string)
							
							msg := utils.Message{
								Role:    role,
								Content: content,
							}
							
							// Handle tool calls if present
							if toolCalls, ok := msgMap["tool_calls"].([]interface{}); ok {
								for _, tcInterface := range toolCalls {
									if tcMap, ok := tcInterface.(map[string]interface{}); ok {
										var toolCall utils.ToolCall
										if id, ok := tcMap["id"].(string); ok {
											toolCall.ID = id
										}
										if tcType, ok := tcMap["type"].(string); ok {
											toolCall.Type = tcType
										}
										if function, ok := tcMap["function"].(map[string]interface{}); ok {
											if name, ok := function["name"].(string); ok {
												toolCall.Function.Name = name
											}
											if args, ok := function["arguments"].(string); ok {
												toolCall.Function.Arguments = args
											}
										}
										msg.ToolCalls = append(msg.ToolCalls, toolCall)
									}
								}
							}
							
							// Handle tool call ID for tool messages
							if toolCallID, ok := msgMap["tool_call_id"].(string); ok {
								msg.ToolCallID = toolCallID
							}
							
							messages = append(messages, msg)
						}
					}
					
					// Add new user message if provided
					if question, ok := flowInput["question"].(string); ok && question != "" {
						messages = append(messages, utils.Message{
							Role:    "user",
							Content: question,
						})
					}
				} else if question, ok := flowInput["question"].(string); ok && question != "" {
					// Use dynamic question from flow input
					logToExecution("info", "Using dynamic input from flow", map[string]interface{}{
						"question_length": len(question),
						"question_preview": truncateString(question, 100),
					})
					messages = []utils.Message{
						{
							Role:    "system",
							Content: "You are a helpful assistant. Keep your answers brief.",
						},
						{
							Role:    "user",
							Content: question,
						},
					}
				} else {
					logToExecution("info", "Flow input present but no 'question' or 'conversation_history' field found, using static parameters", nil)
				}
			} else {
				logToExecution("info", "No flow input available, using static parameters", nil)
			}

			// If no dynamic content was used, fall back to static parameters
			if len(messages) == 0 {

			// Check if we're using templates
			if templatesParam, ok := paramsAny["templates"].([]any); ok {
				// Extract template variables
				variables := make(map[string]any)

				// Add context variables if provided
				if contextParam, ok := paramsAny["context"].(map[string]any); ok {
					for k, v := range contextParam {
						variables[k] = v
					}
				}

				// Process templates
				templateDefs := make([]struct {
					Role     string
					Template string
				}, 0, len(templatesParam))

				for _, tmplInterface := range templatesParam {
					if tmplMap, ok := tmplInterface.(map[string]any); ok {
						role, _ := tmplMap["role"].(string)
						template, _ := tmplMap["template"].(string)

						templateDefs = append(templateDefs, struct {
							Role     string
							Template string
						}{
							Role:     role,
							Template: template,
						})
					}
				}

				// Render templates
				renderedMessages, err := utils.MessagesFromTemplates(templateDefs, variables)
				if err != nil {
					return nil, fmt.Errorf("failed to render templates: %w", err)
				}

				messages = renderedMessages
			} else if messagesParam, ok := paramsAny["messages"]; ok {
				if messagesArray, ok := messagesParam.([]any); ok {
					for _, msgInterface := range messagesArray {
						if msgMap, ok := msgInterface.(map[string]any); ok {
							role, _ := msgMap["role"].(string)
							content, _ := msgMap["content"].(string)

							messages = append(messages, utils.Message{
								Role:    role,
								Content: content,
							})
						}
					}
				} else if messagesArray, ok := messagesParam.([]map[string]any); ok {
					for _, msgMap := range messagesArray {
						role, _ := msgMap["role"].(string)
						content, _ := msgMap["content"].(string)

						messages = append(messages, utils.Message{
							Role:    role,
							Content: content,
						})
					}
				}
			} else if promptParam, ok := paramsAny["prompt"].(string); ok {
				// Support simple prompt parameter as user message
				messages = []utils.Message{
					{
						Role:    "user",
						Content: promptParam,
					},
				}
			} else if templateParam, ok := paramsAny["template"].(string); ok && paramsAny["variables"] != nil {
				// Support single template with variables
				variables, ok := paramsAny["variables"].(map[string]any)
				if !ok {
					return nil, fmt.Errorf("variables must be a map[string]any")
				}

				// Create template
				tmpl, err := utils.NewPromptTemplate(templateParam)
				if err != nil {
					return nil, fmt.Errorf("failed to create template: %w", err)
				}

				// Render template
				content, err := tmpl.Render(variables)
				if err != nil {
					return nil, fmt.Errorf("failed to render template: %w", err)
				}

				// Default to user role if not specified
				role := "user"
				if roleParam, ok := paramsAny["role"].(string); ok {
					role = roleParam
				}

				messages = []utils.Message{
					{
						Role:    role,
						Content: content,
					},
				}
			} else {
				return nil, fmt.Errorf("either messages, prompt, template, or templates parameter is required")
			}
			} // Close the "if len(messages) == 0" block

			// Extract temperature
			temperature := 0.7 // Default temperature
			if tempParam, ok := paramsAny["temperature"].(float64); ok {
				temperature = tempParam
			}

			// Extract max tokens
			maxTokens := 0 // Default (no limit)
			if tokensParam, ok := paramsAny["max_tokens"].(int); ok {
				maxTokens = tokensParam
			}

			// Extract stop sequences
			var stop []string
			if stopParam, ok := paramsAny["stop"].([]any); ok {
				for _, s := range stopParam {
					if stopStr, ok := s.(string); ok {
						stop = append(stop, stopStr)
					}
				}
			}

			// Extract functions
			var functions []utils.FunctionDefinition
			if funcsParam, ok := paramsAny["functions"].([]any); ok {
				for _, funcInterface := range funcsParam {
					if funcMap, ok := funcInterface.(map[string]any); ok {
						name, _ := funcMap["name"].(string)
						desc, _ := funcMap["description"].(string)
						params, _ := funcMap["parameters"].(map[string]any)

						functions = append(functions, utils.FunctionDefinition{
							Name:        name,
							Description: desc,
							Parameters:  params,
						})
					}
				}
			}

			// Extract tools
			var tools []utils.ToolDefinition
			if toolsParam, ok := paramsAny["tools"].([]any); ok {
				for _, toolInterface := range toolsParam {
					// Handle both map[string]any and map[interface{}]interface{} types from YAML
					var toolMap map[string]any
					if tm, ok := toolInterface.(map[string]any); ok {
						toolMap = tm
					} else if tmInterface, ok := toolInterface.(map[interface{}]interface{}); ok {
						toolMap = convertInterfaceMapToStringMap(tmInterface)
					} else {
						continue // Skip if neither type
					}
					
					toolType, _ := toolMap["type"].(string)

					// Handle function map conversion
					var funcMap map[string]any
					if fm, ok := toolMap["function"].(map[string]any); ok {
						funcMap = fm
					} else if fmInterface, ok := toolMap["function"].(map[interface{}]interface{}); ok {
						funcMap = convertInterfaceMapToStringMap(fmInterface)
					}
					
					if funcMap != nil {
						name, _ := funcMap["name"].(string)
						desc, _ := funcMap["description"].(string)
						
						// Handle parameters map conversion
						var params map[string]any
						if p, ok := funcMap["parameters"].(map[string]any); ok {
							params = p
						} else if pInterface, ok := funcMap["parameters"].(map[interface{}]interface{}); ok {
							params = convertInterfaceMapToStringMap(pInterface)
						}

						tools = append(tools, utils.ToolDefinition{
							Type: toolType,
							Function: utils.FunctionDefinition{
								Name:        name,
								Description: desc,
								Parameters:  params,
							},
						})
					}
				}
			}

			// Extract additional options
			options := make(map[string]any)
			if optsParam, ok := params["options"].(map[string]any); ok {
				for k, v := range optsParam {
					options[k] = v
				}
			}

			// Extract response format (for structured output)
			if formatParam, ok := params["response_format"].(map[string]any); ok {
				options["response_format"] = formatParam
			}

			// Create LLM client
			client := utils.NewLLMClient(provider, apiKey, options)

			// Create LLM request
			request := utils.LLMRequest{
				Model:       model,
				Messages:    messages,
				Temperature: temperature,
				MaxTokens:   maxTokens,
				Stop:        stop,
				Functions:   functions,
				Tools:       tools,
				Options:     options,
			}

			log.Printf("[LLM Node] Making LLM request - Model: %s, Messages: %d, Temperature: %.2f, MaxTokens: %d", 
				model, len(messages), temperature, maxTokens)
			log.Printf("[LLM Node] TOOLS COUNT: %d", len(tools))
			if len(tools) > 0 {
				for i, tool := range tools {
					log.Printf("[LLM Node] Tool %d: %s (%s)", i, tool.Function.Name, tool.Function.Description)
				}
			}

			logToExecution("info", "Making LLM API request", map[string]interface{}{
				"model":        model,
				"messages":     len(messages),
				"temperature":  temperature,
				"max_tokens":   maxTokens,
				"provider":     providerStr,
			})

			// Execute request
			ctx := context.Background()
			resp, err := client.Complete(ctx, request)
			if err != nil {
				logToExecution("error", "LLM request failed", map[string]interface{}{
					"error": err.Error(),
					"model": model,
				})
				return nil, fmt.Errorf("LLM request failed: %w", err)
			}

			logToExecution("info", "LLM request completed successfully", map[string]interface{}{
				"model": model,
			})

			// Check for errors
			if resp.Error != nil {
				logToExecution("error", "LLM API error", map[string]interface{}{
					"error": resp.Error.Message,
					"model": model,
				})
				return nil, fmt.Errorf("LLM API error: %s", resp.Error.Message)
			}

			// Extract response
			if len(resp.Choices) == 0 {
				logToExecution("error", "No choices returned from LLM", map[string]interface{}{
					"model": model,
				})
				return nil, fmt.Errorf("no choices returned from LLM")
			}

			content := resp.Choices[0].Message.Content
			logToExecution("info", "LLM response received", map[string]interface{}{
				"content_length":  len(content),
				"finish_reason":   resp.Choices[0].FinishReason,
				"response_id":     resp.ID,
				"model":          resp.Model,
				"content_preview": truncateString(content, 200),
			})

			// Process structured output if requested
			var structuredOutput any
			if resp.Choices[0].Message.Content != "" && params["parse_structured"] == true {
				// Try to parse the response as YAML
				var yamlOutput any
				if err := utils.ParseYAML(resp.Choices[0].Message.Content, &yamlOutput); err == nil {
					structuredOutput = yamlOutput
				}
			}

			// Extract tool calls from the message for easier access
			message := resp.Choices[0].Message
			var hasToolCalls bool
			var toolCalls []utils.ToolCall
			
			if len(message.ToolCalls) > 0 {
				hasToolCalls = true
				toolCalls = message.ToolCalls
				
				log.Printf("[LLM Node] Tool calls detected: %d calls", len(toolCalls))
				for i, call := range toolCalls {
					log.Printf("[LLM Node] Tool call %d: %s with args: %s", i, call.Function.Name, call.Function.Arguments)
				}
			}

			// Return response with consistent structure for tool calls
			result := map[string]any{
				"id":            resp.ID,
				"model":         resp.Model,
				"choices":       resp.Choices,
				"usage":         resp.Usage,
				"content":       resp.Choices[0].Message.Content,
				"finish_reason": resp.Choices[0].FinishReason,
				"raw_response":  resp.RawResponse,
				"has_tool_calls": hasToolCalls,
				"tool_calls":    toolCalls,
				// Add message for compatibility with router logic
				"message": resp.Choices[0].Message,
			}

			// Add structured output if available
			if structuredOutput != nil {
				result["structured_output"] = structuredOutput
				log.Printf("[LLM Node] Structured output parsed successfully")
			}

			log.Printf("[LLM Node] Execution completed successfully - Response ID: %s", resp.ID)
			// Extract keys for debugging
			keys := make([]string, 0, len(result))
			for k := range result {
				keys = append(keys, k)
			}
			log.Printf("[LLM Node] Result keys: %v", keys)
			log.Printf("[LLM Node] Tool calls in result: %d", len(toolCalls))
			log.Printf("[LLM Node] Has tool calls flag: %v", hasToolCalls)
			return result, nil
		},
	}

	// Set the parameters
	wrapper.SetParams(params)

	return wrapper, nil
}
