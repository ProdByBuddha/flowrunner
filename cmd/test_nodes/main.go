package main

import (
	"fmt"
	"os"
	"strings"

	"github.com/joho/godotenv"
	"github.com/tcmartin/flowrunner/pkg/runtime"
)

func main() {
	// Load environment variables from .env file
	// Try loading from current directory first, then from root directory
	if err := godotenv.Load(); err != nil {
		// Try loading from root directory
		if err := godotenv.Load("../../.env"); err != nil {
			fmt.Printf("Error loading .env file: %v\n", err)
		}
	}

	// Parse command line arguments
	args := os.Args[1:]
	if len(args) > 0 {
		// Run specific tests based on arguments
		for _, arg := range args {
			switch arg {
			case "openai":
				fmt.Println("Testing OpenAI LLM node...")
				testOpenAILLM()
			case "anthropic":
				fmt.Println("\nTesting Anthropic LLM node...")
				testAnthropicLLM()
			case "template":
				fmt.Println("\nTesting LLM with template...")
				testLLMWithTemplate()
			case "structured":
				fmt.Println("\nTesting LLM with structured output...")
				testLLMWithStructuredOutput()
			case "email":
				if os.Getenv("GMAIL_USERNAME") != "" && os.Getenv("GMAIL_PASSWORD") != "" {
					fmt.Println("\nTesting Email nodes...")
					testEmailNodes()
				} else {
					fmt.Println("\nSkipping email tests - no credentials provided")
				}
			}
		}
		return
	}

	// Run all tests if no arguments provided
	fmt.Println("Testing OpenAI LLM node...")
	testOpenAILLM()

	fmt.Println("\nTesting Anthropic LLM node...")
	testAnthropicLLM()

	fmt.Println("\nTesting LLM with template...")
	testLLMWithTemplate()

	fmt.Println("\nTesting LLM with structured output...")
	testLLMWithStructuredOutput()

	// Test email nodes if credentials are available
	if os.Getenv("GMAIL_USERNAME") != "" && os.Getenv("GMAIL_PASSWORD") != "" {
		fmt.Println("\nTesting Email nodes...")
		testEmailNodes()
	} else {
		fmt.Println("\nSkipping email tests - no credentials provided")
	}
}

func testOpenAILLM() {
	// Get API key from environment
	apiKey := os.Getenv("OPENAI_API_KEY")
	if apiKey == "" {
		fmt.Println("OPENAI_API_KEY not found in environment")
		return
	}

	// Create LLM node parameters
	params := map[string]interface{}{
		"provider": "openai",
		"api_key":  apiKey,
		"model":    "gpt-3.5-turbo",
		"messages": []map[string]interface{}{
			{
				"role":    "system",
				"content": "You are a helpful assistant. Keep your answers brief.",
			},
			{
				"role":    "user",
				"content": "What is the capital of France?",
			},
		},
		"temperature": 0.7,
		"max_tokens":  100,
	}

	// Create LLM node
	node, err := runtime.NewLLMNodeWrapper(params)
	if err != nil {
		fmt.Printf("Failed to create LLM node: %v\n", err)
		return
	}

	// Create shared context to store results
	shared := make(map[string]interface{})

	// Execute node
	_, runErr := node.Run(shared)
	if runErr != nil {
		fmt.Printf("Failed to execute LLM node: %v\n", runErr)
		return
	}

	// Print result
	fmt.Println("OpenAI Response:")
	if result, ok := shared["result"]; ok {
		if resultMap, ok := result.(map[string]interface{}); ok {
			if content, ok := resultMap["content"].(string); ok {
				fmt.Println(content)
			} else {
				fmt.Printf("%+v\n", resultMap)
			}
		} else {
			fmt.Printf("%+v\n", result)
		}
	} else {
		fmt.Println("No result found in shared context")
	}
}

func testAnthropicLLM() {
	// Get API key from environment
	apiKey := os.Getenv("ANTHROPIC_API_KEY")
	if apiKey == "" {
		fmt.Println("ANTHROPIC_API_KEY not found in environment")
		return
	}

	// Create LLM node parameters
	params := map[string]interface{}{
		"provider": "anthropic",
		"api_key":  apiKey,
		"model":    "claude-3-haiku-20240307",
		"messages": []map[string]interface{}{
			{
				"role":    "system",
				"content": "You are a helpful assistant. Keep your answers brief.",
			},
			{
				"role":    "user",
				"content": "What is the capital of Italy?",
			},
		},
		"temperature": 0.7,
		"max_tokens":  100,
	}

	// Create LLM node
	node, err := runtime.NewLLMNodeWrapper(params)
	if err != nil {
		fmt.Printf("Failed to create LLM node: %v\n", err)
		return
	}

	// Create shared context to store results
	shared := make(map[string]interface{})

	// Execute node
	_, runErr := node.Run(shared)
	if runErr != nil {
		fmt.Printf("Failed to execute LLM node: %v\n", runErr)
		return
	}

	// Print result
	fmt.Println("Anthropic Response:")
	if result, ok := shared["result"]; ok {
		if resultMap, ok := result.(map[string]interface{}); ok {
			if content, ok := resultMap["content"].(string); ok {
				fmt.Println(content)
			} else {
				fmt.Printf("%+v\n", resultMap)
			}
		} else {
			fmt.Printf("%+v\n", result)
		}
	} else {
		fmt.Println("No result found in shared context")
	}
}

func testEmailNodes() {
	// Get credentials from environment
	username := os.Getenv("GMAIL_USERNAME")
	password := os.Getenv("GMAIL_PASSWORD")

	// Test IMAP node first to check for emails
	fmt.Println("Testing IMAP node...")
	testIMAPNode(username, password)

	// Test SMTP node to send an email
	fmt.Println("\nTesting SMTP node...")
	testSMTPNode(username, password)
}

func testIMAPNode(username, password string) {
	// Create IMAP node parameters
	params := map[string]interface{}{
		"imap_host":    "imap.gmail.com",
		"imap_port":    993,
		"username":     username,
		"password":     password,
		"folder":       "INBOX",
		"limit":        5,
		"unseen":       false,
		"with_body":    true,
		"mark_as_read": false,
	}

	// Create IMAP node
	node, err := runtime.NewIMAPNodeWrapper(params)
	if err != nil {
		fmt.Printf("Failed to create IMAP node: %v\n", err)
		return
	}

	// Create shared context to store results
	shared := make(map[string]interface{})

	// Execute node
	_, runErr := node.Run(shared)
	if runErr != nil {
		fmt.Printf("Failed to execute IMAP node: %v\n", runErr)
		return
	}

	// Print result
	if result, ok := shared["result"]; ok {
		if emails, ok := result.([]map[string]interface{}); ok {
			fmt.Printf("Found %d emails\n", len(emails))
			for i, email := range emails {
				if i >= 3 {
					fmt.Println("... (more emails)")
					break
				}
				fmt.Printf("Email %d: Subject: %s, From: %s\n", i+1, email["subject"], email["from"])
			}
		} else {
			fmt.Printf("Unexpected result type: %T\n", result)
		}
	} else {
		fmt.Println("No result found in shared context")
	}
}

func testSMTPNode(username, password string) {
	// Create a test email to yourself
	recipient := os.Getenv("EMAIL_RECIPIENT")
	if recipient == "" {
		recipient = username // Send to self for testing
	}

	// Create SMTP node parameters
	params := map[string]interface{}{
		"smtp_host": "smtp.gmail.com",
		"smtp_port": 587,
		"username":  username,
		"password":  password,
		"from":      username,
		"to":        []string{recipient},
		"subject":   "Test Email from Flowrunner",
		"body":      "This is a test email sent from the Flowrunner SMTP node.",
		"html":      "<h1>Test Email</h1><p>This is a <b>test email</b> sent from the Flowrunner SMTP node.</p>",
	}

	// Create SMTP node
	node, err := runtime.NewSMTPNodeWrapper(params)
	if err != nil {
		fmt.Printf("Failed to create SMTP node: %v\n", err)
		return
	}

	// Ask for confirmation before sending
	fmt.Printf("About to send a test email to %s. Continue? (y/n): ", strings.Split(recipient, "@")[0])
	var response string
	fmt.Scanln(&response)
	if strings.ToLower(response) != "y" {
		fmt.Println("Email sending cancelled")
		return
	}

	// Create shared context to store results
	shared := make(map[string]interface{})

	// Execute node
	_, runErr := node.Run(shared)
	if runErr != nil {
		fmt.Printf("Failed to execute SMTP node: %v\n", runErr)
		return
	}

	// Print result
	fmt.Println("Email sent successfully!")
	if result, ok := shared["result"]; ok {
		if resultMap, ok := result.(map[string]interface{}); ok {
			fmt.Printf("Status: %s\n", resultMap["status"])
		}
	}
}
func testLLMWithTemplate() {
	// Get API key from environment
	apiKey := os.Getenv("OPENAI_API_KEY")
	if apiKey == "" {
		fmt.Println("OPENAI_API_KEY not found in environment")
		return
	}

	// Create LLM node parameters with template
	params := map[string]interface{}{
		"provider": "openai",
		"api_key":  apiKey,
		"model":    "gpt-3.5-turbo",
		"template": "Hello {{.name}}! Can you tell me about the capital of {{.country}}?",
		"variables": map[string]interface{}{
			"name":    "User",
			"country": "Japan",
		},
		"temperature": 0.7,
		"max_tokens":  100,
	}

	// Create LLM node
	node, err := runtime.NewLLMNodeWrapper(params)
	if err != nil {
		fmt.Printf("Failed to create LLM node: %v\n", err)
		return
	}

	// Create shared context to store results
	shared := make(map[string]interface{})

	// Execute node
	_, runErr := node.Run(shared)
	if runErr != nil {
		fmt.Printf("Failed to execute LLM node: %v\n", runErr)
		return
	}

	// Print result
	fmt.Println("Template-based LLM Response:")
	if result, ok := shared["result"]; ok {
		if resultMap, ok := result.(map[string]interface{}); ok {
			if content, ok := resultMap["content"].(string); ok {
				fmt.Println(content)
			} else {
				fmt.Printf("%+v\n", resultMap)
			}
		} else {
			fmt.Printf("%+v\n", result)
		}
	} else {
		fmt.Println("No result found in shared context")
	}
}

func testLLMWithStructuredOutput() {
	// Get API key from environment
	apiKey := os.Getenv("OPENAI_API_KEY")
	if apiKey == "" {
		fmt.Println("OPENAI_API_KEY not found in environment")
		return
	}

	// Create LLM node parameters with structured output request
	params := map[string]interface{}{
		"provider": "openai",
		"api_key":  apiKey,
		"model":    "gpt-3.5-turbo",
		"messages": []map[string]interface{}{
			{
				"role":    "system",
				"content": "You are a helpful assistant that responds in YAML format.",
			},
			{
				"role":    "user",
				"content": "Give me information about Tokyo in YAML format with the following fields: name, country, population, landmarks (as a list).",
			},
		},
		"temperature":      0.7,
		"max_tokens":       200,
		"parse_structured": true,
	}

	// Create LLM node
	node, err := runtime.NewLLMNodeWrapper(params)
	if err != nil {
		fmt.Printf("Failed to create LLM node: %v\n", err)
		return
	}

	// Create shared context to store results
	shared := make(map[string]interface{})

	// Execute node
	_, runErr := node.Run(shared)
	if runErr != nil {
		fmt.Printf("Failed to execute LLM node: %v\n", runErr)
		return
	}

	// Print result
	fmt.Println("Structured Output LLM Response:")
	if result, ok := shared["result"]; ok {
		if resultMap, ok := result.(map[string]interface{}); ok {
			// Print raw content
			if content, ok := resultMap["content"].(string); ok {
				fmt.Println("Raw content:")
				fmt.Println(content)
			}

			// Print structured output if available
			if structuredOutput, ok := resultMap["structured_output"]; ok {
				fmt.Println("\nParsed structured output:")
				fmt.Printf("%+v\n", structuredOutput)
			} else {
				fmt.Println("\nNo structured output available")
			}
		} else {
			fmt.Printf("%+v\n", result)
		}
	} else {
		fmt.Println("No result found in shared context")
	}
}
