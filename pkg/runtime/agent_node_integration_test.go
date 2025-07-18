package runtime

import (
"fmt"
"os"
"strings"
"testing"
"time"

"github.com/joho/godotenv"
"github.com/stretchr/testify/assert"
"github.com/tcmartin/flowlib"
)

func TestAgentNodeIntegration(t *testing.T) {
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

	// Create the agent node with HTTP and email tools
	t.Log("Creating agent node...")
	agentNode, err := NewAgentNodeWrapper(map[string]interface{}{
"provider":  "openai",
"api_key":   apiKey,
"model":     "gpt-4-turbo",
"max_steps": 10.0,
"prompt":    fmt.Sprintf("Search for information about '%s' and send an email summarizing the findings to %s. Be concise but informative.", searchQuery, emailRecipient),
"tools": []interface{}{
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
},
})
	assert.NoError(t, err)
	assert.NotNil(t, agentNode)

	// Create a shared context for the flow
	shared := map[string]interface{}{}

	// Create tool handlers
	t.Log("Setting up tool handlers...")
	toolHandlers := map[string]func(params map[string]interface{}) (interface{}, error){
		"search_google": func(params map[string]interface{}) (interface{}, error) {
			query, _ := params["query"].(string)
			
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
		"send_email": func(params map[string]interface{}) (interface{}, error) {
			subject, _ := params["subject"].(string)
			body, _ := params["body"].(string)
			
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

	// Set up the agent node with tool handlers
	t.Log("Setting up agent node with tool handlers...")
	agentNodeWrapper := agentNode.(*NodeWrapper)
	agentParams := agentNodeWrapper.Params()
	agentParams["tool_handlers"] = toolHandlers
	agentNodeWrapper.SetParams(agentParams)

	// Execute the agent node
	t.Log("Starting agent execution...")
	startTime := time.Now()
	
	// Run the agent node
	action, err := agentNode.Run(shared)
	
	// Log execution time
	executionTime := time.Since(startTime)
	t.Logf("Agent execution completed in %v", executionTime)
	
	// Check for errors
	assert.NoError(t, err)
	assert.Equal(t, flowlib.DefaultAction, action)
	
	// Check the result
	result, ok := shared["result"].(map[string]interface{})
	assert.True(t, ok, "Expected result to be a map")
	
	// Log the result for debugging
	t.Logf("Agent response: %s", result["response"])
	t.Logf("Steps taken: %v", result["steps"])
	
	intermediateResults, ok := result["intermediate_results"].([]map[string]interface{})
	assert.True(t, ok, "Expected intermediate_results to be a slice of maps")
	t.Logf("Intermediate results: %d", len(intermediateResults))
	
	// Check that the agent used both tools
	var usedSearch, usedEmail bool
	
	for _, step := range intermediateResults {
		tool, ok := step["tool"].(string)
		if ok {
			if tool == "search_google" {
				usedSearch = true
				t.Log("Agent used search_google tool")
				
				args, ok := step["arguments"].(map[string]interface{})
				if ok {
					t.Logf("Search query: %s", args["query"])
				}
			} else if tool == "send_email" {
				usedEmail = true
				t.Log("Agent used send_email tool")
				
				// Log email details
				args, ok := step["arguments"].(map[string]interface{})
				if ok {
					t.Logf("Email subject: %s", args["subject"])
					
					// Log a preview of the email body
					body, ok := args["body"].(string)
					if ok {
						bodyPreview := body
						if len(bodyPreview) > 100 {
							bodyPreview = bodyPreview[:97] + "..."
						}
						t.Logf("Email body preview: %s", bodyPreview)
					}
				}
			}
		}
	}
	
	// Assert that both tools were used
	assert.True(t, usedSearch, "Agent did not use the search_google tool")
	assert.True(t, usedEmail, "Agent did not use the send_email tool")
	
	// Check that the final response mentions both searching and emailing
	response := result["response"].(string)
	assert.True(t, 
strings.Contains(strings.ToLower(response), "search") || 
strings.Contains(strings.ToLower(response), "information") || 
strings.Contains(strings.ToLower(response), "found"),
"Response does not mention searching")
	
	assert.True(t, 
strings.Contains(strings.ToLower(response), "email") || 
strings.Contains(strings.ToLower(response), "sent") || 
strings.Contains(strings.ToLower(response), "message"),
"Response does not mention sending an email")
}
