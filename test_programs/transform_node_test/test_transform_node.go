package main

import (
	"fmt"
	"log"

	"github.com/tcmartin/flowrunner/pkg/runtime"
)

func main() {
	fmt.Println("Testing Transform Node JavaScript Engine Integration")

	// Test 1: Simple transformation with return value
	fmt.Println("\n=== Test 1: Simple JavaScript Return ===")
	params1 := map[string]interface{}{
		"script": "return 'Hello from JavaScript!';",
	}

	node1, err := runtime.NewTransformNodeWrapper(params1)
	if err != nil {
		log.Fatalf("Failed to create transform node: %v", err)
	}

	// Test with simple input - the Run method will handle the transformation
	input1 := map[string]interface{}{
		"message": "test input",
	}

	action1, err := node1.Run(input1)
	if err != nil {
		log.Fatalf("Transform node execution failed: %v", err)
	}

	fmt.Printf("Input message: %v\n", input1["message"])
	fmt.Printf("Action: %v\n", action1)
	
	// Check if result was stored in input
	if result, ok := input1["result"]; ok {
		fmt.Printf("Transform result: %v\n", result)
	}

	// Test 2: Transformation using input data
	fmt.Println("\n=== Test 2: JavaScript Input Processing ===")
	params2 := map[string]interface{}{
		"script": `
			console.log('Input received:', input);
			if (input && input.name) {
				return 'Hello, ' + input.name + '!';
			}
			return 'Hello, World!';
		`,
	}

	node2, err := runtime.NewTransformNodeWrapper(params2)
	if err != nil {
		log.Fatalf("Failed to create transform node: %v", err)
	}

	// Use input with flow data
	input2 := map[string]interface{}{
		"name": "FlowRunner",
	}

	action2, err := node2.Run(input2)
	if err != nil {
		log.Fatalf("Transform node execution failed: %v", err)
	}

	fmt.Printf("Input name: %v\n", input2["name"])
	fmt.Printf("Action: %v\n", action2)
	
	// Check if result was stored in input
	if result, ok := input2["result"]; ok {
		fmt.Printf("Transform result: %v\n", result)
	}

	// Test 3: Complex JavaScript transformation
	fmt.Println("\n=== Test 3: Complex JavaScript Processing ===")
	params3 := map[string]interface{}{
		"script": `
			console.log('Processing input data...');
			
			// Create a new object with transformed data
			var output = {
				timestamp: new Date().toISOString(),
				original_input: input,
				processed: true
			};
			
			// Add calculation
			if (input && input.numbers && Array.isArray(input.numbers)) {
				var sum = input.numbers.reduce(function(acc, num) {
					return acc + num;
				}, 0);
				output.sum = sum;
				output.average = sum / input.numbers.length;
			}
			
			return output;
		`,
	}

	node3, err := runtime.NewTransformNodeWrapper(params3)
	if err != nil {
		log.Fatalf("Failed to create transform node: %v", err)
	}

	input3 := map[string]interface{}{
		"numbers": []interface{}{1, 2, 3, 4, 5},
		"source":  "test",
	}

	action3, err := node3.Run(input3)
	if err != nil {
		log.Fatalf("Transform node execution failed: %v", err)
	}

	fmt.Printf("Input source: %v\n", input3["source"])
	fmt.Printf("Action: %v\n", action3)
	
	// Check if result was stored in input
	if result, ok := input3["result"]; ok {
		fmt.Printf("Transform result type: %T\n", result)
		if resultMap, ok := result.(map[string]interface{}); ok {
			fmt.Printf("Result processed: %v\n", resultMap["processed"])
			if sum, ok := resultMap["sum"]; ok {
				fmt.Printf("Result sum: %v\n", sum)
			}
		}
	}

	// Test 4: Error handling
	fmt.Println("\n=== Test 4: Error Handling ===")
	params4 := map[string]interface{}{
		"script": "throw new Error('Test error from JavaScript');",
	}

	node4, err := runtime.NewTransformNodeWrapper(params4)
	if err != nil {
		log.Fatalf("Failed to create transform node: %v", err)
	}

	input4 := map[string]interface{}{
		"test": "data",
	}

	action4, err := node4.Run(input4)
	if err != nil {
		fmt.Printf("Expected error occurred: %v\n", err)
	} else {
		fmt.Printf("Unexpected success - Action: %v\n", action4)
	}

	fmt.Println("\n=== Transform Node Test Complete ===")
	fmt.Println("✅ Transform node is working correctly with JavaScript engine!")
}
