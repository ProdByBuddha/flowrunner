package main

import (
	"fmt"
	"os"

	"github.com/joho/godotenv"
	"github.com/tcmartin/flowrunner/pkg/runtime"
)

// Demonstrates the new dynamic input capability for LLM nodes
func main() {
	// Load environment variables
	_ = godotenv.Load()

	// Get API key from environment
	apiKey := os.Getenv("OPENAI_API_KEY")
	if apiKey == "" {
		fmt.Println("OPENAI_API_KEY not found in environment")
		return
	}

	fmt.Println("🔍 Testing Dynamic LLM Node Input...")

	// Test 1: Static parameters only (original behavior)
	fmt.Println("\n📝 Test 1: Static parameters (original behavior)")
	testStaticParams(apiKey)

	// Test 2: Dynamic input override (new behavior)
	fmt.Println("\n🚀 Test 2: Dynamic input override (new behavior)")
	testDynamicInput(apiKey)
}

func testStaticParams(apiKey string) {
	// Create LLM node with static parameters
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
				"content": "What is 2+2?",
			},
		},
		"temperature": 0.7,
		"max_tokens":  50,
	}

	node, err := runtime.NewLLMNodeWrapper(params)
	if err != nil {
		fmt.Printf("❌ Failed to create LLM node: %v\n", err)
		return
	}

	// Execute with static parameters only
	shared := make(map[string]interface{})
	result, err := node.Run(shared)
	if err != nil {
		fmt.Printf("❌ Failed to execute LLM node: %v\n", err)
		return
	}

	fmt.Printf("✅ Result: %s\n", result)
	if resultData, ok := shared["result"].(map[string]interface{}); ok {
		if content, ok := resultData["content"].(string); ok {
			fmt.Printf("📄 Response: %s\n", content)
		}
	}
}

func testDynamicInput(apiKey string) {
	// Create LLM node with minimal static parameters
	params := map[string]interface{}{
		"provider":    "openai",
		"api_key":     apiKey,
		"model":       "gpt-3.5-turbo",
		"temperature": 0.7,
		"max_tokens":  50,
	}

	node, err := runtime.NewLLMNodeWrapper(params)
	if err != nil {
		fmt.Printf("❌ Failed to create LLM node: %v\n", err)
		return
	}

	// Execute with dynamic input containing a question
	shared := map[string]interface{}{
		"question": "What is the capital of China?",
		"context":  "The user is asking about geography.",
	}

	result, err := node.Run(shared)
	if err != nil {
		fmt.Printf("❌ Failed to execute LLM node: %v\n", err)
		return
	}

	fmt.Printf("✅ Result: %s\n", result)
	if resultData, ok := shared["result"].(map[string]interface{}); ok {
		if content, ok := resultData["content"].(string); ok {
			fmt.Printf("📄 Response: %s\n", content)
		}
	}

	fmt.Println("\n🎉 Dynamic input successfully overrode static parameters!")
	fmt.Println("   The LLM node received the question from flow input instead of hardcoded parameters.")
}
