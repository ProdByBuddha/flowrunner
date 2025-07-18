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

func TestLLMNodeWithHTTPAndEmailFlow(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	// Set environment variables directly for testing
	if err := godotenv.Load(); err != nil {
		// Try loading from root directory
		if err := godotenv.Load("../../.env"); err != nil {
			fmt.Printf("Error loading .env file: %v\n", err)
		}
	}
	// Get API key from environment

	apiKey := os.Getenv("OPENAI_API_KEY")
	if apiKey == "" {
		t.Skip("Skipping test because OPENAI_API_KEY environment variable is not set")
	}

	// Get email credentials from environment
	emailUser := os.Getenv("GMAIL_USERNAME")
	emailPass := os.Getenv("GMAIL_PASSWORD")
	emailRecipient := os.Getenv("EMAIL_RECIPIENT")

	// If recipient email is not set, use the Gmail username as the recipient
	if emailRecipient == "" {
		emailRecipient = emailUser
	}
	print(emailUser)

	if emailUser == "" || emailPass == "" {
		t.Skip("Skipping test because email credentials are not set")
	}

	t.Logf("Using email credentials: %s -> %s", emailUser, emailRecipient)

	// Gmail SMTP settings
	smtpHost := "smtp.gmail.com"
	smtpPort := 587

	// Create a search query
	searchQuery := "latest advancements in artificial intelligence 2025"
	t.Logf("Search query: %s", searchQuery)

	// Create a shared context for the flow
	shared := map[string]interface{}{}

	// Create the HTTP node for search
	// Using a mock Google search API URL
	httpNode, err := NewHTTPRequestNodeWrapper(map[string]interface{}{
		"url":    "https://www.googleapis.com/customsearch/v1",
		"method": "GET",
		"params": map[string]interface{}{
			"q":   searchQuery,
			"key": "mock-api-key",
			"cx":  "mock-search-engine-id",
		},
		"headers": map[string]interface{}{
			"Accept": "application/json",
		},
	})
	assert.NoError(t, err)
	assert.NotNil(t, httpNode)

	// Create the email node
	emailNode, err := NewSMTPNodeWrapper(map[string]interface{}{
		"smtp_host": smtpHost,
		"smtp_port": smtpPort,
		"username":  emailUser,
		"password":  emailPass,
		"from":      emailUser,
		"to":        emailRecipient,
		"subject":   fmt.Sprintf("AI Search Results: %s", searchQuery),
		"body":      "This is a placeholder email body. The LLM will replace this with the actual content.",
	})
	assert.NoError(t, err)
	assert.NotNil(t, emailNode)

	// Create the LLM node with a system message that instructs it to process search results
	llmNode, err := NewLLMNodeWrapper(map[string]interface{}{
		"provider":    "openai",
		"api_key":     apiKey,
		"model":       "gpt-4-turbo",
		"temperature": 0.7,
		"messages": []map[string]interface{}{
			{
				"role":    "system",
				"content": "You are a helpful assistant that summarizes search results into concise, informative emails. Your task is to analyze the search results provided and create a well-structured email summary.",
			},
			{
				"role":    "user",
				"content": fmt.Sprintf("I've searched for information about '%s' and need to send an email summarizing the findings to %s. Please create a concise but informative email summary based on these results.", searchQuery, emailRecipient),
			},
		},
	})
	assert.NoError(t, err)
	assert.NotNil(t, llmNode)

	// Mock the LLM node to avoid API calls
	llmNodeWrapper := llmNode.(*NodeWrapper)
	originalLLMExec := llmNodeWrapper.exec
	llmNodeWrapper.exec = func(input interface{}) (interface{}, error) {
		t.Log("Executing LLM node with mock response")

		// Extract the messages to see what search results were passed
		params, ok := input.(map[string]interface{})
		if !ok {
			return nil, fmt.Errorf("expected map[string]interface{}, got %T", input)
		}

		// Log the messages for debugging
		if messages, ok := params["messages"].([]map[string]interface{}); ok && len(messages) >= 2 {
			userMsg := messages[1]
			userContent, _ := userMsg["content"].(string)
			t.Logf("LLM would process: %s", userContent[:100]+"...")
		}

		// Return a mock LLM response
		return map[string]interface{}{
			"content": `Subject: Latest Advancements in Artificial Intelligence 2025 - Research Summary

Dear Recipient,

I've compiled the key findings from my research on the latest advancements in artificial intelligence in 2025:

1. Multimodal Models: Significant progress has been made in developing AI systems that can process and understand multiple types of data simultaneously (text, images, audio). These models are showing unprecedented capabilities in understanding context across different media formats.

2. Quantum Machine Learning: The integration of quantum computing with machine learning algorithms has led to breakthroughs in processing complex datasets at speeds previously thought impossible.

3. Neuromorphic Computing: AI systems modeled after the human brain have achieved new milestones in energy efficiency and adaptive learning capabilities.

4. Ethical Frameworks: Researchers are focusing on developing robust ethical guidelines for autonomous systems, with particular attention to addressing bias in large language models.

5. Healthcare Applications: Medical AI has made remarkable strides in disease diagnosis, drug discovery, and personalized treatment planning, potentially revolutionizing healthcare delivery.

These advancements represent significant steps forward in AI technology and its applications across various industries.

Best regards,
[Your Name]`,
			"model": "gpt-4-turbo-mock",
			"usage": map[string]interface{}{
				"prompt_tokens":     250,
				"completion_tokens": 200,
				"total_tokens":      450,
			},
		}, nil
	}

	// Restore original function after test
	defer func() {
		llmNodeWrapper.exec = originalLLMExec
	}()

	// Mock the HTTP response for the search
	httpNodeWrapper := httpNode.(*NodeWrapper)
	originalExec := httpNodeWrapper.exec
	httpNodeWrapper.exec = func(input interface{}) (interface{}, error) {
		t.Log("Executing HTTP node with mock search results")
		return map[string]interface{}{
			"status_code": 200,
			"body": map[string]interface{}{
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
			},
			"success": true,
		}, nil
	}

	// Mock the email node to avoid sending actual emails during testing
	emailNodeWrapper := emailNode.(*NodeWrapper)
	originalEmailExec := emailNodeWrapper.exec
	emailNodeWrapper.exec = func(input interface{}) (interface{}, error) {
		params, ok := input.(map[string]interface{})
		if !ok {
			return nil, fmt.Errorf("expected map[string]interface{}, got %T", input)
		}

		subject, _ := params["subject"].(string)
		body, _ := params["body"].(string)

		t.Logf("Would send email with subject: %s", subject)
		bodyPreview := body
		if len(bodyPreview) > 100 {
			bodyPreview = bodyPreview[:97] + "..."
		}
		t.Logf("Email body preview: %s", bodyPreview)

		return map[string]interface{}{
			"status":  "sent",
			"from":    emailUser,
			"to":      emailRecipient,
			"subject": subject,
		}, nil
	}

	// Restore original functions after test
	defer func() {
		httpNodeWrapper.exec = originalExec
		emailNodeWrapper.exec = originalEmailExec
	}()

	// Add a custom post-processing function to update the LLM node with search results
	httpNodeWrapper.post = func(shared, params, result interface{}) (flowlib.Action, error) {
		// Extract search results from the HTTP response
		httpResult, ok := result.(map[string]interface{})
		if !ok {
			return "", fmt.Errorf("expected map[string]interface{}, got %T", result)
		}

		// Format search results for the LLM
		var searchResultsText string
		if body, ok := httpResult["body"].(map[string]interface{}); ok {
			if items, ok := body["items"].([]map[string]interface{}); ok {
				for i, item := range items {
					title, _ := item["title"].(string)
					snippet, _ := item["snippet"].(string)
					link, _ := item["link"].(string)

					searchResultsText += fmt.Sprintf("Result %d:\nTitle: %s\nLink: %s\nSnippet: %s\n\n",
						i+1, title, link, snippet)
				}
			}
		}

		// Update the LLM node's messages to include the search results
		llmNodeWrapper := llmNode.(*NodeWrapper)
		llmParams := llmNodeWrapper.Params()
		if messages, ok := llmParams["messages"].([]map[string]interface{}); ok && len(messages) >= 2 {
			// Update the user message to include search results
			userMsg := messages[1]
			userContent, _ := userMsg["content"].(string)
			userMsg["content"] = fmt.Sprintf("%s\n\nHere are the search results:\n\n%s",
				userContent, searchResultsText)

			// Update the messages in the LLM node parameters
			llmNodeWrapper.SetParams(llmParams)
		}

		return "success", nil
	}

	// Create a flow with the nodes
	// HTTP -> LLM -> Email
	httpNode.Next("success", llmNode)
	llmNode.Next(flowlib.DefaultAction, emailNode)

	// Create a flow
	flow := flowlib.NewFlow(httpNode)

	// Execute the flow
	t.Log("Starting flow execution...")
	startTime := time.Now()

	// Run the flow
	result, err := flow.Run(shared)

	// Log execution time
	executionTime := time.Since(startTime)
	t.Logf("Flow execution completed in %v", executionTime)

	// Check for errors
	assert.NoError(t, err)
	assert.Equal(t, flowlib.DefaultAction, result)

	// Check that the shared context has the expected results
	httpResult, ok := shared["http_result"].(map[string]interface{})
	if assert.True(t, ok, "Expected http_result in shared context") {
		assert.Equal(t, 200, httpResult["status_code"])
		assert.True(t, httpResult["success"].(bool))
	}

	llmResult, ok := shared["llm_result"].(map[string]interface{})
	if assert.True(t, ok, "Expected llm_result in shared context") {
		content, ok := llmResult["content"].(string)
		if assert.True(t, ok, "Expected content in llm_result") {
			t.Logf("LLM response: %s", content)
			assert.True(t,
				strings.Contains(strings.ToLower(content), "ai") ||
					strings.Contains(strings.ToLower(content), "artificial intelligence") ||
					strings.Contains(strings.ToLower(content), "2025"),
				"LLM response should mention AI or 2025")
		}
	}

	emailResult, ok := shared["email_result"].(map[string]interface{})
	if assert.True(t, ok, "Expected email_result in shared context") {
		assert.Equal(t, "sent", emailResult["status"])
		assert.Equal(t, emailRecipient, emailResult["to"])
	}
}
