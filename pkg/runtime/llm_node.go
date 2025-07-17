package runtime

import (
	"context"
	"fmt"
	"time"

	"github.com/tcmartin/flowlib"
	"github.com/tcmartin/flowrunner/pkg/utils"
)

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
			// Get parameters from input
			params, ok := input.(map[string]any)
			if !ok {
				return nil, fmt.Errorf("expected map[string]any, got %T", input)
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

			// Extract messages
			var messages []utils.Message

			// Check if we're using templates
			if templatesParam, ok := params["templates"].([]any); ok {
				// Extract template variables
				variables := make(map[string]any)

				// Add context variables if provided
				if contextParam, ok := params["context"].(map[string]any); ok {
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
			} else if messagesParam, ok := params["messages"]; ok {
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
			} else if promptParam, ok := params["prompt"].(string); ok {
				// Support simple prompt parameter as user message
				messages = []utils.Message{
					{
						Role:    "user",
						Content: promptParam,
					},
				}
			} else if templateParam, ok := params["template"].(string); ok && params["variables"] != nil {
				// Support single template with variables
				variables, ok := params["variables"].(map[string]any)
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
				if roleParam, ok := params["role"].(string); ok {
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

			// Extract temperature
			temperature := 0.7 // Default temperature
			if tempParam, ok := params["temperature"].(float64); ok {
				temperature = tempParam
			}

			// Extract max tokens
			maxTokens := 0 // Default (no limit)
			if tokensParam, ok := params["max_tokens"].(int); ok {
				maxTokens = tokensParam
			}

			// Extract stop sequences
			var stop []string
			if stopParam, ok := params["stop"].([]any); ok {
				for _, s := range stopParam {
					if stopStr, ok := s.(string); ok {
						stop = append(stop, stopStr)
					}
				}
			}

			// Extract functions
			var functions []utils.FunctionDefinition
			if funcsParam, ok := params["functions"].([]any); ok {
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
			if toolsParam, ok := params["tools"].([]any); ok {
				for _, toolInterface := range toolsParam {
					if toolMap, ok := toolInterface.(map[string]any); ok {
						toolType, _ := toolMap["type"].(string)

						if funcMap, ok := toolMap["function"].(map[string]any); ok {
							name, _ := funcMap["name"].(string)
							desc, _ := funcMap["description"].(string)
							params, _ := funcMap["parameters"].(map[string]any)

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

			// Execute request
			ctx := context.Background()
			resp, err := client.Complete(ctx, request)
			if err != nil {
				return nil, fmt.Errorf("LLM request failed: %w", err)
			}

			// Check for errors
			if resp.Error != nil {
				return nil, fmt.Errorf("LLM API error: %s", resp.Error.Message)
			}

			// Extract response
			if len(resp.Choices) == 0 {
				return nil, fmt.Errorf("no choices returned from LLM")
			}

			// Process structured output if requested
			var structuredOutput any
			if resp.Choices[0].Message.Content != "" && params["parse_structured"] == true {
				// Try to parse the response as YAML
				var yamlOutput any
				if err := utils.ParseYAML(resp.Choices[0].Message.Content, &yamlOutput); err == nil {
					structuredOutput = yamlOutput
				}
			}

			// Return response
			result := map[string]any{
				"id":            resp.ID,
				"model":         resp.Model,
				"choices":       resp.Choices,
				"usage":         resp.Usage,
				"content":       resp.Choices[0].Message.Content,
				"finish_reason": resp.Choices[0].FinishReason,
				"raw_response":  resp.RawResponse,
			}

			// Add structured output if available
			if structuredOutput != nil {
				result["structured_output"] = structuredOutput
			}

			return result, nil
		},
	}

	// Set the parameters
	wrapper.SetParams(params)

	return wrapper, nil
}
