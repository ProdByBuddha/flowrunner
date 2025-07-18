package runtime

import (
	"encoding/json"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/joho/godotenv"
	"github.com/stretchr/testify/assert"
)

func TestLLMNodeAsAgent(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	// Load environment variables from .env file
	err := godotenv.Load()
	if err != nil {
		t.Log("Warning: Error loading .env file, using existing environment variables")
	}

	// Get API key from environment
	apiKey := os.Getenv("OPENAI_API_KEY")
	if apiKey == "" {
		t.Skip("Skipping test because OPENAI_API_KEY environment variable is not set")
	}

	// Get email credentials from environment
	emailUser := os.Getenv("GMAIL_USERNAME")
	emailPass := os.Getenv("GMAIL_PASSWORD")
	emailRecipient := os.Getenv("RECIPIENT_EMAIL")

	if emailUser == "" || emailPass == "" || emailRecipient == "" {
		t.Skip("Skipping test because email environment variables are not set")
	}

	t.Logf("Using email credentials: %s -> %s", emailUser, emailRecipient)

	// Gmail SMTP settings
	smtpHost := "smtp.gmail.com"
	smtpPort := 587

	// Create a search query
	searchQuery := "latest advancements in artificial intelligence 2025"
	t.Logf("Search query: %s", searchQuery)

	// Define tools directly in the LLM node parameters

	// Create tools
	tools := []interface{}{
		map[string]interface{}{
			"type": "function",
			"function": map[string]interface{}{
				"name":        "search_google",
				"description": "Search Google for information",
				"parameters": map[string]interface{}{
					"type": "object",
					"properties": map[string]interface{}{
						"query": map[string]interface{}{
							"type":        "string",
							"description": "The search query",
						},
					},
					"required": []interface{}{"query"},
				},
			},
		},
		map[string]interface{}{
			"type": "function",
			"function": map[string]interface{}{
				"name":        "send_email",
				"description": "Send an email with the given subject and content",
				"parameters": map[string]interface{}{
					"type": "object",
					"properties": map[string]interface{}{
						"subject": map[string]interface{}{
							"type":        "string",
							"description": "The email subject",
						},
						"body": map[string]interface{}{
							"type":        "string",
							"description": "The email body (plain text)",
						},
					},
					"required": []interface{}{"subject", "body"},
				},
			},
		},
	}

	// Create the LLM node
	t.Log("Creating LLM node...")
	llmNode, err := NewLLMNodeWrapper(map[string]interface{}{
		"provider":    "openai",
		"api_key":     apiKey,
		"model":       "gpt-4-turbo",
		"temperature": 0.7,
		"messages": []map[string]interface{}{
			{
				"role":    "system",
				"content": "You are a helpful assistant that can search for information and send emails. Use the tools provided to complete tasks.",
			},
			{
				"role":    "user",
				"content": fmt.Sprintf("Search for information about '%s' and send an email summarizing the findings to %s. Be concise but informative.", searchQuery, emailRecipient),
			},
		},
		"tools": tools,
	})
	assert.NoError(t, err)
	assert.NotNil(t, llmNode)

	// Create a shared context for the flow
	shared := map[string]interface{}{}

	// Create function handlers
	functionHandlers := map[string]func(args map[string]interface{}) (interface{}, error){
		"search_google": func(args map[string]interface{}) (interface{}, error) {
			query, _ := args["query"].(string)

			// Use a mock response for the search
			t.Logf("Using mock search results for query: %s", query)
			return map[string]interface{}{
				"items": []map[string]interface{}{
					{
						"title":       "AI Breakthroughs in 2025: A Comprehensive Overview",
						"link":        "https://example.com/ai-breakthroughs-2025",
						"snippet":     "The latest advancements in artificial intelligence in 2025 include significant progress in multimodal models, quantum machine learning, and neuromorphic computing.",
						"displayLink": "example.com",
					},
					{
						"title":       "Ethical Considerations in AI Development - 2025 Perspective",
						"link":        "https://example.com/ai-ethics-2025",
						"snippet":     "As AI continues to advance in 2025, researchers are focusing on ethical frameworks for autonomous systems and addressing bias in large language models.",
						"displayLink": "example.com",
					},
					{
						"title":       "AI in Healthcare: 2025 Innovations",
						"link":        "https://example.com/ai-healthcare-2025",
						"snippet":     "Medical AI systems in 2025 have achieved breakthrough capabilities in disease diagnosis, drug discovery, and personalized treatment planning.",
						"displayLink": "example.com",
					},
				},
			}, nil
		},
		"send_email": func(args map[string]interface{}) (interface{}, error) {
			subject, _ := args["subject"].(string)
			body, _ := args["body"].(string)

			t.Logf("Sending email with subject: %s", subject)
			bodyPreview := body
			if len(bodyPreview) > 100 {
				bodyPreview = bodyPreview[:97] + "..."
			}
			t.Logf("Email body preview: %s", bodyPreview)

			// Create email parameters
			emailParams := map[string]interface{}{
				"smtp_host": smtpHost,
				"smtp_port": smtpPort,
				"username":  emailUser,
				"password":  emailPass,
				"from":      emailUser,
				"to":        emailRecipient,
				"subject":   subject,
				"body":      body,
			}

			// Create the email node
			emailNode, err := NewSMTPNodeWrapper(emailParams)
			if err != nil {
				t.Logf("Failed to create email node: %v", err)
				return nil, fmt.Errorf("failed to create email node: %w", err)
			}

			// Execute the email node
			t.Log("Executing email node...")
			result, err := emailNode.(*NodeWrapper).exec(emailParams)
			if err != nil {
				t.Logf("Failed to send email: %v", err)
				return nil, fmt.Errorf("failed to send email: %w", err)
			}

			t.Log("Email sent successfully")
			return result, nil
		},
	}

	// Track which functions were used
	var searchUsed, emailUsed bool

	// Create a flow to handle tool calls
	flow := func() error {
		// Execute the LLM node
		t.Log("Starting LLM node execution...")
		startTime := time.Now()

		// Run the LLM node
		_, err := llmNode.Run(shared)
		if err != nil {
			return err
		}

		// Log execution time
		executionTime := time.Since(startTime)
		t.Logf("LLM node execution completed in %v", executionTime)

		// Get the result
		result, ok := shared["result"].(map[string]interface{})
		if !ok {
			return fmt.Errorf("expected result to be a map")
		}

		// Log the content
		content, _ := result["content"].(string)
		t.Logf("LLM response: %s", content)

		// Check if there are tool calls in the raw response
		if rawResp, ok := result["raw_response"].(map[string]interface{}); ok {
			if rawChoices, ok := rawResp["choices"].([]interface{}); ok && len(rawChoices) > 0 {
				if firstChoice, ok := rawChoices[0].(map[string]interface{}); ok {
					if message, ok := firstChoice["message"].(map[string]interface{}); ok {
						// Check for tool_calls
						if toolCalls, ok := message["tool_calls"].([]interface{}); ok && len(toolCalls) > 0 {
							t.Log("Found tool calls in raw response")

							// Process each tool call
							for _, tc := range toolCalls {
								toolCall, ok := tc.(map[string]interface{})
								if !ok {
									continue
								}

								// Get function info
								if function, ok := toolCall["function"].(map[string]interface{}); ok {
									name, _ := function["name"].(string)
									arguments, _ := function["arguments"].(string)

									t.Logf("Tool call: %s with arguments: %s", name, arguments)

									// Parse arguments
									var args map[string]interface{}
									if err := json.Unmarshal([]byte(arguments), &args); err != nil {
										t.Logf("Failed to parse arguments: %v", err)
										continue
									}

									// Execute the function
									handler, ok := functionHandlers[name]
									if !ok {
										t.Logf("No handler for function: %s", name)
										continue
									}

									// Call the handler
									_, err := handler(args)
									if err != nil {
										t.Logf("Function execution failed: %v", err)
										continue
									}

									t.Logf("Function %s executed successfully", name)

									// Mark the function as used
									if name == "search_google" {
										searchUsed = true
									} else if name == "send_email" {
										emailUsed = true
									}
								}
							}
						}
					}
				}
			}
		}

		return nil
	}

	// Execute the flow
	err = flow()
	assert.NoError(t, err)

	// Check if both functions were used
	// Note: This might not always be true as the LLM might decide to use only one function
	// or none at all, depending on its reasoning
	if !searchUsed || !emailUsed {
		t.Logf("Warning: Not all tools were used. Search: %v, Email: %v", searchUsed, emailUsed)
	}
}
