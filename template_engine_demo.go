package main

import (
	"fmt"
	"log"

	"github.com/tcmartin/flowrunner/pkg/runtime"
	"github.com/tcmartin/flowrunner/pkg/scripting"
	"github.com/tcmartin/flowrunner/pkg/services"
	"github.com/tcmartin/flowrunner/pkg/storage"
)

func main() {
	fmt.Println("🚀 FlowRunner Template Engine Integration Demo")
	fmt.Println("===============================================")

	// Create in-memory storage for secrets
	storageProvider := storage.NewMemoryProvider()
	storageProvider.Initialize()

	// Create secret vault with encryption
	encryptionKey := []byte("test-encryption-key-32-bytes-123")
	secretVault, err := services.NewSecretVaultService(storageProvider.GetSecretStore(), encryptionKey)
	if err != nil {
		log.Fatal("Failed to create secret vault:", err)
	}

	// Set up test secrets
	accountID := "demo-account"
	secretVault.Set(accountID, "API_KEY", "secret-api-key-abc123")
	secretVault.Set(accountID, "DB_PASSWORD", "super-secret-db-password")
	secretVault.Set(accountID, "WEBHOOK_URL", "https://api.example.com/webhook")

	fmt.Println("\n✅ Secret vault initialized with test secrets")

	// === Demo 1: FlowContext with Secrets and Node Results ===
	fmt.Println("\n📋 Demo 1: FlowContext with Secrets and Node Results")
	fmt.Println("---------------------------------------------------")

	// Create a FlowContext
	flowContext := runtime.NewFlowContext("exec-123", "flow-456", accountID, secretVault)

	// Simulate previous node results (like what would happen in a real flow)
	flowContext.SetNodeResult("http_node", map[string]any{
		"status_code": 200,
		"response_time": "120ms",
		"data": map[string]any{
			"users": []any{
				map[string]any{"id": 1, "name": "Alice", "email": "alice@example.com"},
				map[string]any{"id": 2, "name": "Bob", "email": "bob@example.com"},
				map[string]any{"id": 3, "name": "Charlie", "email": "charlie@example.com"},
			},
			"total_count": 3,
		},
	})

	flowContext.SetNodeResult("transform_node", map[string]any{
		"processed_users": []any{
			map[string]any{"name": "Alice", "email": "alice@example.com", "verified": true},
			map[string]any{"name": "Bob", "email": "bob@example.com", "verified": false},
			map[string]any{"name": "Charlie", "email": "charlie@example.com", "verified": true},
		},
		"verification_summary": map[string]any{
			"total": 3,
			"verified": 2,
			"unverified": 1,
		},
	})

	// Set shared data (flow-level variables)
	flowContext.SetSharedData("request_id", "req-789")
	flowContext.SetSharedData("timestamp", "2025-01-20T10:30:00Z")
	flowContext.SetSharedData("user_agent", "FlowRunner/1.0")

	// Demonstrate accessing secrets
	fmt.Println("\n🔐 Accessing secrets in expressions:")
	testExpression(flowContext, "${secrets.API_KEY}")
	testExpression(flowContext, "${\"Bearer \" + secrets.API_KEY}")
	testExpression(flowContext, "${\"Database URL: postgresql://user:\" + secrets.DB_PASSWORD + \"@localhost/mydb\"}")

	// Demonstrate accessing node results
	fmt.Println("\n📊 Accessing node results in expressions:")
	testExpression(flowContext, "${results.http_node.status_code}")
	testExpression(flowContext, "${results.http_node.data.total_count}")
	testExpression(flowContext, "${results.http_node.data.users[0].name}")
	testExpression(flowContext, "${results.transform_node.verification_summary.verified}")

	// Demonstrate accessing shared data
	fmt.Println("\n🔗 Accessing shared data in expressions:")
	testExpression(flowContext, "${shared.request_id}")
	testExpression(flowContext, "${shared.timestamp}")
	testExpression(flowContext, "${shared.user_agent}")

	// Demonstrate complex expressions
	fmt.Println("\n🧮 Complex expressions combining multiple contexts:")
	testExpression(flowContext, "${\"Request \" + shared.request_id + \" processed \" + results.http_node.data.total_count + \" users\"}")
	testExpression(flowContext, "${\"API Response: \" + results.http_node.status_code + \" (\" + results.http_node.response_time + \") using key \" + secrets.API_KEY}")
	testExpression(flowContext, "${\"Verification: \" + results.transform_node.verification_summary.verified + \"/\" + results.transform_node.verification_summary.total + \" users verified\"}")

	// === Demo 2: Processing Node Parameters ===
	fmt.Println("\n⚙️  Demo 2: Processing Node Parameters with Expressions")
	fmt.Println("----------------------------------------------------")

	// Simulate dynamic node configuration (like what would happen in YAML flows)
	nodeParams := map[string]any{
		"webhook_url": "${secrets.WEBHOOK_URL}",
		"method":      "POST",
		"timeout":     "30s",
		"headers": map[string]any{
			"Authorization":     "${\"Bearer \" + secrets.API_KEY}",
			"X-Request-ID":      "${shared.request_id}",
			"X-Timestamp":       "${shared.timestamp}",
			"Content-Type":      "application/json",
			"User-Agent":        "${shared.user_agent}",
		},
		"payload": map[string]any{
			"request_id":        "${shared.request_id}",
			"timestamp":         "${shared.timestamp}",
			"http_status":       "${results.http_node.status_code}",
			"response_time":     "${results.http_node.response_time}",
			"users_processed":   "${results.http_node.data.total_count}",
			"first_user":        "${results.http_node.data.users[0].name}",
			"verified_count":    "${results.transform_node.verification_summary.verified}",
			"webhook_endpoint":  "${secrets.WEBHOOK_URL + \"/callback\"}",
			"database_config": map[string]any{
				"host":     "localhost",
				"database": "mydb",
				"password": "${secrets.DB_PASSWORD}",
				"ssl":      true,
			},
		},
	}

	fmt.Println("\nOriginal node parameters (with expressions):")
	printMap(nodeParams, "  ")

	processedParams, err := flowContext.ProcessNodeParams(nodeParams)
	if err != nil {
		log.Fatal("Failed to process node parameters:", err)
	}

	fmt.Println("\nProcessed node parameters (expressions resolved):")
	printMap(processedParams, "  ")

	// === Demo 3: Direct SecretAwareExpressionEvaluator Usage ===
	fmt.Println("\n🔧 Demo 3: Direct SecretAwareExpressionEvaluator Usage")
	fmt.Println("---------------------------------------------------")

	evaluator := scripting.NewSecretAwareExpressionEvaluator(secretVault)

	// Simulate the context that would be passed during flow execution
	context := map[string]any{
		"accountID": accountID,
		"_flow_context": map[string]any{
			"node_results": map[string]any{
				"api_call": map[string]any{
					"response": map[string]any{
						"status": "success",
						"data":   []any{"item1", "item2", "item3", "item4", "item5"},
						"metadata": map[string]any{
							"page": 1,
							"per_page": 5,
							"total": 25,
						},
					},
				},
			},
			"shared_data": map[string]any{
				"user_id":   "user-123",
				"session":   "sess-456",
				"operation": "data_fetch",
			},
		},
	}

	fmt.Println("\nDirect evaluator tests:")
	testDirectEvaluator(evaluator, "${secrets.API_KEY}", context)
	testDirectEvaluator(evaluator, "${results.api_call.response.status}", context)
	testDirectEvaluator(evaluator, "${shared.user_id}", context)
	testDirectEvaluator(evaluator, "${\"User \" + shared.user_id + \" in session \" + shared.session + \" got \" + results.api_call.response.data.length + \" items\"}", context)

	// Test EvaluateInObject
	fmt.Println("\nEvaluateInObject test:")
	obj := map[string]any{
		"authentication": map[string]any{
			"token":   "${\"Bearer \" + secrets.API_KEY}",
			"user_id": "${shared.user_id}",
		},
		"request_data": map[string]any{
			"items_count": "${results.api_call.response.data.length}",
			"status":      "${results.api_call.response.status}",
			"page_info":   "${\"Page \" + results.api_call.response.metadata.page + \" of \" + Math.ceil(results.api_call.response.metadata.total / results.api_call.response.metadata.per_page)}",
		},
		"webhook_config": map[string]any{
			"url":     "${secrets.WEBHOOK_URL + \"/\" + shared.operation}",
			"session": "${shared.session}",
		},
	}

	fmt.Println("\nOriginal object:")
	printMap(obj, "  ")

	processedObj, err := evaluator.EvaluateInObject(obj, context)
	if err != nil {
		log.Fatal("Failed to evaluate object:", err)
	}

	fmt.Println("\nProcessed object:")
	printMap(processedObj, "  ")

	fmt.Println("\n🎉 Template Engine Demo Complete!")
	fmt.Println("\n✨ Key Features Demonstrated:")
	fmt.Println("   • Secret access via ${secrets.SECRET_NAME}")
	fmt.Println("   • Node results access via ${results.node_name.field}")
	fmt.Println("   • Shared data access via ${shared.variable}")
	fmt.Println("   • Complex JavaScript expressions with context mixing")
	fmt.Println("   • Dynamic node parameter processing")
	fmt.Println("   • Nested object and array expression evaluation")
	fmt.Println("   • Error handling for missing values")
}

func testExpression(flowContext *runtime.FlowContext, expression string) {
	result, err := flowContext.EvaluateExpression(expression)
	if err != nil {
		fmt.Printf("   ❌ %s → ERROR: %v\n", expression, err)
	} else {
		fmt.Printf("   ✅ %s → %v\n", expression, result)
	}
}

func testDirectEvaluator(evaluator scripting.SecretAwareEvaluator, expression string, context map[string]any) {
	result, err := evaluator.Evaluate(expression, context)
	if err != nil {
		fmt.Printf("   ❌ %s → ERROR: %v\n", expression, err)
	} else {
		fmt.Printf("   ✅ %s → %v\n", expression, result)
	}
}

func printMap(m map[string]any, indent string) {
	for k, v := range m {
		switch val := v.(type) {
		case map[string]any:
			fmt.Printf("%s%s:\n", indent, k)
			printMap(val, indent+"  ")
		case []any:
			fmt.Printf("%s%s: [%d items]\n", indent, k, len(val))
		default:
			fmt.Printf("%s%s: %v\n", indent, k, val)
		}
	}
}
