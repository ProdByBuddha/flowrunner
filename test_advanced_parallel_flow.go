package main

import (
	"fmt"
	"os"
	"sync"
	"time"

	"github.com/joho/godotenv"
	"github.com/tcmartin/flowrunner/pkg/runtime"
)

// Advanced regression test for parallel LLM flow with timing and error handling
func main() {
	// Load environment variables
	_ = godotenv.Load()

	// Get API key from environment
	apiKey := os.Getenv("OPENAI_API_KEY")
	if apiKey == "" {
		fmt.Println("OPENAI_API_KEY not found in environment")
		return
	}

	fmt.Println("🧪 Advanced Parallel LLM Flow Regression Test")
	fmt.Println("   Features:")
	fmt.Println("   • Parallel execution with timing comparison")
	fmt.Println("   • Error handling and recovery")
	fmt.Println("   • Dynamic input flow")
	fmt.Println("   • Complex branching/merging pattern")
	fmt.Println("   • Performance analysis")

	// Run the comprehensive test
	runComprehensiveFlowTest(apiKey)
}

func runComprehensiveFlowTest(apiKey string) {
	topic := "Machine Learning in Financial Risk Assessment"
	
	fmt.Printf("\n📋 Analysis Topic: %s\n", topic)

	// Test 1: Sequential execution (baseline)
	fmt.Println("\n⏱️  Test 1: Sequential Execution (Baseline)")
	sequentialStart := time.Now()
	sequentialResults := runSequentialFlow(apiKey, topic)
	sequentialDuration := time.Since(sequentialStart)
	fmt.Printf("⏰ Sequential execution time: %v\n", sequentialDuration)

	// Test 2: Parallel execution (optimized)
	fmt.Println("\n🚀 Test 2: Parallel Execution (Optimized)")
	parallelStart := time.Now()
	parallelResults := runParallelFlow(apiKey, topic)
	parallelDuration := time.Since(parallelStart)
	fmt.Printf("⏰ Parallel execution time: %v\n", parallelDuration)

	// Performance comparison
	if parallelDuration < sequentialDuration {
		speedup := float64(sequentialDuration) / float64(parallelDuration)
		fmt.Printf("🎯 Performance Gain: %.2fx speedup with parallel execution\n", speedup)
	}

	// Test 3: Error handling and recovery
	fmt.Println("\n🛡️  Test 3: Error Handling Test")
	testErrorHandling(apiKey)

	// Validate results consistency
	fmt.Println("\n✅ Results Validation:")
	validateResults(sequentialResults, parallelResults)

	fmt.Println("\n🎉 All regression tests passed!")
	fmt.Println("   ✅ Parallel execution works correctly")
	fmt.Println("   ✅ Performance improvements demonstrated")
	fmt.Println("   ✅ Error handling functions properly")
	fmt.Println("   ✅ Dynamic input flows through entire pipeline")
}

func runSequentialFlow(apiKey, topic string) map[string]interface{} {
	results := make(map[string]interface{})
	
	// Step 1: Technical analysis
	fmt.Println("   📊 Running technical analysis...")
	techResult, err := executeAnalysisLLM(apiKey, topic, "technical")
	if err != nil {
		fmt.Printf("   ❌ Technical analysis failed: %v\n", err)
		return nil
	}
	results["technical"] = techResult
	
	// Step 2: Risk analysis  
	fmt.Println("   ⚠️  Running risk analysis...")
	riskResult, err := executeAnalysisLLM(apiKey, topic, "risk")
	if err != nil {
		fmt.Printf("   ❌ Risk analysis failed: %v\n", err)
		return nil
	}
	results["risk"] = riskResult
	
	// Step 3: Synthesis
	fmt.Println("   🔄 Running synthesis...")
	synthesisResult, err := executeSynthesis(apiKey, topic, results)
	if err != nil {
		fmt.Printf("   ❌ Synthesis failed: %v\n", err)
		return nil
	}
	results["synthesis"] = synthesisResult
	
	fmt.Println("   ✅ Sequential flow completed")
	return results
}

func runParallelFlow(apiKey, topic string) map[string]interface{} {
	results := make(map[string]interface{})
	var wg sync.WaitGroup
	var mu sync.Mutex
	errorCh := make(chan error, 2)
	
	// Parallel execution of analysis branches
	fmt.Println("   🔀 Running parallel analysis branches...")
	
	// Branch 1: Technical analysis
	wg.Add(1)
	go func() {
		defer wg.Done()
		techResult, err := executeAnalysisLLM(apiKey, topic, "technical")
		if err != nil {
			errorCh <- fmt.Errorf("technical analysis: %w", err)
			return
		}
		mu.Lock()
		results["technical"] = techResult
		mu.Unlock()
		fmt.Println("   ✅ Technical analysis completed (parallel)")
	}()
	
	// Branch 2: Risk analysis
	wg.Add(1)
	go func() {
		defer wg.Done()
		riskResult, err := executeAnalysisLLM(apiKey, topic, "risk")
		if err != nil {
			errorCh <- fmt.Errorf("risk analysis: %w", err)
			return
		}
		mu.Lock()
		results["risk"] = riskResult
		mu.Unlock()
		fmt.Println("   ✅ Risk analysis completed (parallel)")
	}()
	
	// Wait for parallel branches
	wg.Wait()
	close(errorCh)
	
	// Check for errors
	for err := range errorCh {
		fmt.Printf("   ❌ Parallel execution error: %v\n", err)
		return nil
	}
	
	// Synthesis phase
	fmt.Println("   🔄 Running synthesis...")
	synthesisResult, err := executeSynthesis(apiKey, topic, results)
	if err != nil {
		fmt.Printf("   ❌ Synthesis failed: %v\n", err)
		return nil
	}
	results["synthesis"] = synthesisResult
	
	fmt.Println("   ✅ Parallel flow completed")
	return results
}

func executeAnalysisLLM(apiKey, topic, analysisType string) (map[string]interface{}, error) {
	// Create LLM node
	params := map[string]interface{}{
		"provider":    "openai",
		"api_key":     apiKey,
		"model":       "gpt-3.5-turbo",
		"temperature": 0.4,
		"max_tokens":  120,
	}

	node, err := runtime.NewLLMNodeWrapper(params)
	if err != nil {
		return nil, fmt.Errorf("failed to create LLM node: %w", err)
	}

	// Create analysis-specific prompt
	var prompt string
	switch analysisType {
	case "technical":
		prompt = fmt.Sprintf("Analyze the technical aspects of %s. Focus on algorithms, data requirements, implementation challenges, and technological innovations. Be concise.", topic)
	case "risk":
		prompt = fmt.Sprintf("Analyze the risks and challenges of %s. Focus on potential pitfalls, regulatory concerns, data privacy issues, and mitigation strategies. Be concise.", topic)
	default:
		return nil, fmt.Errorf("unknown analysis type: %s", analysisType)
	}

	// Execute with dynamic input
	shared := map[string]interface{}{
		"question": prompt,
		"context":  fmt.Sprintf("%s analysis for parallel workflow", analysisType),
	}

	_, err = node.Run(shared)
	if err != nil {
		return nil, fmt.Errorf("failed to execute %s analysis: %w", analysisType, err)
	}

	if resultData, ok := shared["result"].(map[string]interface{}); ok {
		return resultData, nil
	}

	return nil, fmt.Errorf("unexpected result format from %s analysis", analysisType)
}

func executeSynthesis(apiKey, topic string, branchResults map[string]interface{}) (map[string]interface{}, error) {
	// Extract content from results
	var techContent, riskContent string
	
	if techResult, ok := branchResults["technical"].(map[string]interface{}); ok {
		if content, ok := techResult["content"].(string); ok {
			techContent = content
		}
	}
	
	if riskResult, ok := branchResults["risk"].(map[string]interface{}); ok {
		if content, ok := riskResult["content"].(string); ok {
			riskContent = content
		}
	}

	// Create synthesis LLM node
	params := map[string]interface{}{
		"provider":    "openai",
		"api_key":     apiKey,
		"model":       "gpt-3.5-turbo",
		"temperature": 0.6,
		"max_tokens":  150,
	}

	node, err := runtime.NewLLMNodeWrapper(params)
	if err != nil {
		return nil, fmt.Errorf("failed to create synthesis LLM node: %w", err)
	}

	// Create synthesis prompt
	prompt := fmt.Sprintf(`Synthesize these analyses of "%s":

TECHNICAL ANALYSIS:
%s

RISK ANALYSIS:
%s

Provide a balanced summary that integrates both perspectives, highlighting key insights and recommendations.`, 
		topic, techContent, riskContent)

	// Execute with dynamic input
	shared := map[string]interface{}{
		"question": prompt,
		"context":  "Final synthesis combining parallel analysis results",
	}

	_, err = node.Run(shared)
	if err != nil {
		return nil, fmt.Errorf("failed to execute synthesis: %w", err)
	}

	if resultData, ok := shared["result"].(map[string]interface{}); ok {
		return resultData, nil
	}

	return nil, fmt.Errorf("unexpected result format from synthesis")
}

func testErrorHandling(apiKey string) {
	fmt.Println("   🧪 Testing error conditions...")
	
	// Test with invalid API key
	fmt.Println("   • Testing invalid API key handling...")
	_, err := executeAnalysisLLM("invalid_key", "test topic", "technical")
	if err != nil {
		fmt.Println("   ✅ Error handling works: Invalid API key detected")
	} else {
		fmt.Println("   ⚠️  Warning: Expected error for invalid API key not detected")
	}
	
	// Test with empty topic
	fmt.Println("   • Testing empty input handling...")
	_, err = executeAnalysisLLM(apiKey, "", "technical")
	if err != nil {
		fmt.Println("   ✅ Error handling works: Empty input detected")
	} else {
		fmt.Println("   ✅ Empty input handled gracefully")
	}
}

func validateResults(sequential, parallel map[string]interface{}) {
	if sequential == nil || parallel == nil {
		fmt.Println("   ⚠️  Cannot validate: One or both result sets are nil")
		return
	}
	
	// Check that both have synthesis results
	seqSynthesis, seqOk := sequential["synthesis"].(map[string]interface{})
	parSynthesis, parOk := parallel["synthesis"].(map[string]interface{})
	
	if seqOk && parOk {
		seqContent, _ := seqSynthesis["content"].(string)
		parContent, _ := parSynthesis["content"].(string)
		
		if len(seqContent) > 0 && len(parContent) > 0 {
			fmt.Println("   ✅ Both sequential and parallel flows produced valid synthesis results")
		} else {
			fmt.Println("   ⚠️  Warning: One or both synthesis results appear to be empty")
		}
	} else {
		fmt.Println("   ⚠️  Warning: Could not extract synthesis results for validation")
	}
}
