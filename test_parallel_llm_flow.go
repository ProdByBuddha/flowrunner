package main

import (
	"fmt"
	"os"
	"sync"

	"github.com/joho/godotenv"
	"github.com/tcmartin/flowrunner/pkg/runtime"
)

// Demonstrates parallel LLM execution with result merging
func main() {
	// Load environment variables
	_ = godotenv.Load()

	// Get API key from environment
	apiKey := os.Getenv("OPENAI_API_KEY")
	if apiKey == "" {
		fmt.Println("OPENAI_API_KEY not found in environment")
		return
	}

	fmt.Println("🔍 Testing Parallel LLM Flow with Branching and Merging...")
	fmt.Println("   This test demonstrates:")
	fmt.Println("   1. Two parallel LLM nodes processing different aspects of a topic")
	fmt.Println("   2. A third LLM node that combines and synthesizes the results")
	fmt.Println("   3. Dynamic input capability across the entire flow")

	// Test the parallel flow
	testParallelLLMFlow(apiKey)
}

func testParallelLLMFlow(apiKey string) {
	// Input topic for analysis
	topic := "Artificial Intelligence in Healthcare"
	
	fmt.Printf("\n📋 Input Topic: %s\n", topic)
	fmt.Println("\n🚀 Starting Parallel LLM Flow...")

	// Create shared result storage
	results := make(map[string]interface{})
	var wg sync.WaitGroup
	var mu sync.Mutex

	// Channel for errors
	errorCh := make(chan error, 2)

	// Branch 1: Technical Analysis LLM
	wg.Add(1)
	go func() {
		defer wg.Done()
		
		fmt.Println("\n🔬 Branch 1: Technical Analysis (Running in parallel)")
		
		technicalResult, err := executeTechnicalAnalysisLLM(apiKey, topic)
		if err != nil {
			errorCh <- fmt.Errorf("technical analysis failed: %w", err)
			return
		}
		
		mu.Lock()
		results["technical_analysis"] = technicalResult
		mu.Unlock()
		
		fmt.Println("✅ Technical Analysis completed")
		if content, ok := technicalResult["content"].(string); ok {
			fmt.Printf("📄 Technical Analysis: %s\n", truncateString(content, 100))
		}
	}()

	// Branch 2: Business Impact LLM
	wg.Add(1)
	go func() {
		defer wg.Done()
		
		fmt.Println("\n💼 Branch 2: Business Impact Analysis (Running in parallel)")
		
		businessResult, err := executeBusinessImpactLLM(apiKey, topic)
		if err != nil {
			errorCh <- fmt.Errorf("business impact analysis failed: %w", err)
			return
		}
		
		mu.Lock()
		results["business_impact"] = businessResult
		mu.Unlock()
		
		fmt.Println("✅ Business Impact Analysis completed")
		if content, ok := businessResult["content"].(string); ok {
			fmt.Printf("📄 Business Impact: %s\n", truncateString(content, 100))
		}
	}()

	// Wait for parallel branches to complete
	wg.Wait()
	close(errorCh)

	// Check for errors
	for err := range errorCh {
		fmt.Printf("❌ Error in parallel execution: %v\n", err)
		return
	}

	// Merge Phase: Combine results with third LLM
	fmt.Println("\n🔄 Merge Phase: Combining results with synthesis LLM...")
	
	synthesisResult, err := executeSynthesisLLM(apiKey, topic, results)
	if err != nil {
		fmt.Printf("❌ Failed to execute synthesis LLM: %v\n", err)
		return
	}

	// Display final results
	fmt.Println("\n🎉 Parallel LLM Flow completed successfully!")
	fmt.Println("\n📊 Final Synthesis Result:")
	if content, ok := synthesisResult["content"].(string); ok {
		fmt.Printf("📄 Combined Analysis: %s\n", content)
	}

	fmt.Println("\n✨ Flow Summary:")
	fmt.Println("   ✅ Two parallel LLM executions completed successfully")
	fmt.Println("   ✅ Results merged and synthesized by third LLM")
	fmt.Println("   ✅ Dynamic input passed through entire flow")
	fmt.Println("   ✅ Demonstrates complex branching/merging workflow")
}

func executeTechnicalAnalysisLLM(apiKey, topic string) (map[string]interface{}, error) {
	// Create LLM node for technical analysis
	params := map[string]interface{}{
		"provider":    "openai",
		"api_key":     apiKey,
		"model":       "gpt-3.5-turbo",
		"temperature": 0.3, // Lower temperature for technical accuracy
		"max_tokens":  150,
	}

	node, err := runtime.NewLLMNodeWrapper(params)
	if err != nil {
		return nil, fmt.Errorf("failed to create technical analysis LLM node: %w", err)
	}

	// Execute with dynamic input focused on technical aspects
	shared := map[string]interface{}{
		"question": fmt.Sprintf("Provide a technical analysis of %s. Focus on current technologies, algorithms, implementation challenges, and technical innovations. Keep it concise.", topic),
		"context":  "Technical expert analysis for parallel processing workflow",
	}

	result, err := node.Run(shared)
	if err != nil {
		return nil, fmt.Errorf("failed to execute technical analysis LLM: %w", err)
	}
	_ = result // Suppress unused variable warning

	if resultData, ok := shared["result"].(map[string]interface{}); ok {
		return resultData, nil
	}

	return nil, fmt.Errorf("unexpected result format from technical analysis LLM")
}

func executeBusinessImpactLLM(apiKey, topic string) (map[string]interface{}, error) {
	// Create LLM node for business impact analysis
	params := map[string]interface{}{
		"provider":    "openai",
		"api_key":     apiKey,
		"model":       "gpt-3.5-turbo",
		"temperature": 0.5, // Moderate temperature for balanced analysis
		"max_tokens":  150,
	}

	node, err := runtime.NewLLMNodeWrapper(params)
	if err != nil {
		return nil, fmt.Errorf("failed to create business impact LLM node: %w", err)
	}

	// Execute with dynamic input focused on business aspects
	shared := map[string]interface{}{
		"question": fmt.Sprintf("Analyze the business impact of %s. Focus on market opportunities, cost implications, ROI potential, and business transformation aspects. Keep it concise.", topic),
		"context":  "Business strategy analysis for parallel processing workflow",
	}

	result, err := node.Run(shared)
	if err != nil {
		return nil, fmt.Errorf("failed to execute business impact LLM: %w", err)
	}
	_ = result // Suppress unused variable warning

	if resultData, ok := shared["result"].(map[string]interface{}); ok {
		return resultData, nil
	}

	return nil, fmt.Errorf("unexpected result format from business impact LLM")
}

func executeSynthesisLLM(apiKey, topic string, branchResults map[string]interface{}) (map[string]interface{}, error) {
	// Extract content from branch results
	var technicalContent, businessContent string
	
	if techResult, ok := branchResults["technical_analysis"].(map[string]interface{}); ok {
		if content, ok := techResult["content"].(string); ok {
			technicalContent = content
		}
	}
	
	if bizResult, ok := branchResults["business_impact"].(map[string]interface{}); ok {
		if content, ok := bizResult["content"].(string); ok {
			businessContent = content
		}
	}

	// Create LLM node for synthesis
	params := map[string]interface{}{
		"provider":    "openai",
		"api_key":     apiKey,
		"model":       "gpt-3.5-turbo",
		"temperature": 0.7, // Higher temperature for creative synthesis
		"max_tokens":  200,
	}

	node, err := runtime.NewLLMNodeWrapper(params)
	if err != nil {
		return nil, fmt.Errorf("failed to create synthesis LLM node: %w", err)
	}

	// Create a comprehensive prompt that combines both analyses
	synthesisPrompt := fmt.Sprintf(`Synthesize the following two analyses of "%s" into a comprehensive summary:

TECHNICAL ANALYSIS:
%s

BUSINESS IMPACT ANALYSIS:
%s

Provide a unified perspective that combines both technical and business insights, highlighting key connections and overall implications.`, 
		topic, technicalContent, businessContent)

	// Execute with dynamic input that combines both branch results
	shared := map[string]interface{}{
		"question": synthesisPrompt,
		"context":  "Final synthesis combining parallel analysis results",
	}

	result, err := node.Run(shared)
	if err != nil {
		return nil, fmt.Errorf("failed to execute synthesis LLM: %w", err)
	}
	_ = result // Suppress unused variable warning

	if resultData, ok := shared["result"].(map[string]interface{}); ok {
		return resultData, nil
	}

	return nil, fmt.Errorf("unexpected result format from synthesis LLM")
}

// Helper function to truncate strings for display
func truncateString(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen] + "..."
}
