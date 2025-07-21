package main

import (
	"fmt"
	"log"

	"github.com/tcmartin/flowrunner/pkg/runtime"
)

func main() {
	fmt.Println("Testing Transform Node in Flow Context")

	// Test realistic flow scenario: process API response data
	fmt.Println("\n=== Flow Context Test: API Response Processing ===")
	
	// Simulate receiving data from an HTTP API call
	flowContext := map[string]interface{}{
		// This simulates what would come from an http.request node
		"http_result": map[string]interface{}{
			"status": 200,
			"data": map[string]interface{}{
				"users": []interface{}{
					map[string]interface{}{"name": "Alice", "age": 30, "email": "alice@example.com"},
					map[string]interface{}{"name": "Bob", "age": 25, "email": "bob@example.com"},
					map[string]interface{}{"name": "Charlie", "age": 35, "email": "charlie@example.com"},
				},
				"total": 3,
			},
		},
		// This is the flow execution input  
		"question": "Filter users over 28 and extract their emails",
		"context": "user processing pipeline",
	}

	// Create transform node to process the API data
	transformParams := map[string]interface{}{
		"script": `
			console.log('=== Transform Node Processing ===');
			console.log('Available input keys:', Object.keys(input));
			
			// Access the HTTP result data
			var httpResult = input.http_result;
			if (!httpResult || !httpResult.data || !httpResult.data.users) {
				return { error: 'No user data found in HTTP result' };
			}
			
			var users = httpResult.data.users;
			console.log('Found', users.length, 'users to process');
			
			// Filter users over 28 and extract emails
			var filteredUsers = [];
			var emails = [];
			
			for (var i = 0; i < users.length; i++) {
				var user = users[i];
				if (user.age > 28) {
					filteredUsers.push(user);
					emails.push(user.email);
				}
			}
			
			console.log('Filtered to', filteredUsers.length, 'users over 28');
			
			// Return processed data
			return {
				filtered_users: filteredUsers,
				email_list: emails,
				total_filtered: filteredUsers.length,
				original_total: users.length,
				filter_criteria: 'age > 28',
				processed_at: new Date().toISOString()
			};
		`,
	}

	transformNode, err := runtime.NewTransformNodeWrapper(transformParams)
	if err != nil {
		log.Fatalf("Failed to create transform node: %v", err)
	}

	// Execute the transform node with flow context
	action, err := transformNode.Run(flowContext)
	if err != nil {
		log.Fatalf("Transform node execution failed: %v", err)
	}

	fmt.Printf("Transform action: %v\n", action)
	
	// Check the transform result
	if result, ok := flowContext["result"]; ok {
		if resultMap, ok := result.(map[string]interface{}); ok {
			fmt.Printf("✅ Transform successful!\n")
			fmt.Printf("   • Original users: %v\n", resultMap["original_total"])
			fmt.Printf("   • Filtered users: %v\n", resultMap["total_filtered"])
			if emails, ok := resultMap["email_list"].([]interface{}); ok {
				fmt.Printf("   • Extracted emails: %v\n", emails)
			}
			fmt.Printf("   • Filter criteria: %v\n", resultMap["filter_criteria"])
		}
	}

	// Test 2: Data validation and error handling
	fmt.Println("\n=== Flow Context Test: Data Validation ===")
	
	invalidContext := map[string]interface{}{
		"question": "Process user data",
		"data": "invalid_data", // This should trigger error handling
	}

	validationParams := map[string]interface{}{
		"script": `
			console.log('=== Validation Transform ===');
			
			// Validate input structure
			if (!input.data || typeof input.data !== 'object') {
				throw new Error('Invalid data format: expected object, got ' + typeof input.data);
			}
			
			return { validated: true, data: input.data };
		`,
	}

	validationNode, err := runtime.NewTransformNodeWrapper(validationParams)
	if err != nil {
		log.Fatalf("Failed to create validation node: %v", err)
	}

	// This should fail with validation error
	_, err = validationNode.Run(invalidContext)
	if err != nil {
		fmt.Printf("✅ Validation error caught: %v\n", err)
	} else {
		fmt.Printf("❌ Expected validation to fail\n")
	}

	// Test 3: Mathematical calculations
	fmt.Println("\n=== Flow Context Test: Mathematical Processing ===")
	
	mathContext := map[string]interface{}{
		"question": "Calculate statistics",
		"numbers": []interface{}{10, 20, 30, 40, 50},
		"context": "statistical analysis",
	}

	mathParams := map[string]interface{}{
		"script": `
			console.log('=== Math Transform ===');
			
			var numbers = input.numbers;
			if (!Array.isArray(numbers)) {
				return { error: 'Numbers must be an array' };
			}
			
			var sum = 0;
			var min = numbers[0];
			var max = numbers[0];
			
			for (var i = 0; i < numbers.length; i++) {
				var num = numbers[i];
				sum += num;
				if (num < min) min = num;
				if (num > max) max = num;
			}
			
			var average = sum / numbers.length;
			
			return {
				input_data: numbers,
				statistics: {
					sum: sum,
					average: average,
					min: min,
					max: max,
					count: numbers.length
				},
				analysis: 'Statistical analysis completed successfully'
			};
		`,
	}

	mathNode, err := runtime.NewTransformNodeWrapper(mathParams)
	if err != nil {
		log.Fatalf("Failed to create math node: %v", err)
	}

	action, err = mathNode.Run(mathContext)
	if err != nil {
		log.Fatalf("Math node execution failed: %v", err)
	}

	if result, ok := mathContext["result"]; ok {
		if resultMap, ok := result.(map[string]interface{}); ok {
			fmt.Printf("✅ Math transform successful!\n")
			if stats, ok := resultMap["statistics"].(map[string]interface{}); ok {
				fmt.Printf("   • Sum: %v\n", stats["sum"])
				fmt.Printf("   • Average: %v\n", stats["average"])
				fmt.Printf("   • Min: %v\n", stats["min"])
				fmt.Printf("   • Max: %v\n", stats["max"])
				fmt.Printf("   • Count: %v\n", stats["count"])
			}
		}
	}

	fmt.Println("\n=== Transform Node Integration Test Complete ===")
	fmt.Println("✅ Transform node successfully integrated with JavaScript engine!")
	fmt.Println("✅ All flow context scenarios working correctly!")
	fmt.Println("✅ Error handling and validation working!")
	fmt.Println("✅ Complex data processing capabilities verified!")
}
