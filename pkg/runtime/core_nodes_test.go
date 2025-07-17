package runtime

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestHTTPRequestNode(t *testing.T) {
	// Create a test server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Check request method
		if r.Method != "GET" {
			t.Errorf("Expected method 'GET', got '%s'", r.Method)
		}

		// Check headers
		if r.Header.Get("X-Test-Header") != "test-value" {
			t.Errorf("Expected header 'X-Test-Header' to be 'test-value', got '%s'", r.Header.Get("X-Test-Header"))
		}

		// Write response
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"message":"success"}`))
	}))
	defer server.Close()

	// Create node parameters
	params := map[string]interface{}{
		"url":    server.URL,
		"method": "GET",
		"headers": map[string]interface{}{
			"X-Test-Header": "test-value",
		},
	}

	// Create the node
	node, err := NewHTTPRequestNodeWrapper(params)
	if err != nil {
		t.Fatalf("Failed to create HTTP request node: %v", err)
	}

	// Execute the node
	result, err := node.Run(nil)
	if err != nil {
		t.Fatalf("Failed to execute HTTP request node: %v", err)
	}

	// Check the result
	if result != "default" {
		t.Errorf("Expected result to be 'default', got '%s'", result)
	}
}

func TestStoreNode(t *testing.T) {
	// Create the node
	node, err := NewStoreNodeWrapper(map[string]interface{}{})
	if err != nil {
		t.Fatalf("Failed to create store node: %v", err)
	}

	// Test set operation
	node.SetParams(map[string]interface{}{
		"operation": "set",
		"key":       "test-key",
		"value":     "test-value",
	})

	_, err = node.Run(nil)
	if err != nil {
		t.Fatalf("Failed to execute set operation: %v", err)
	}

	// Test get operation
	node.SetParams(map[string]interface{}{
		"operation": "get",
		"key":       "test-key",
	})

	_, err = node.Run(nil)
	if err != nil {
		t.Fatalf("Failed to execute get operation: %v", err)
	}

	// Test list operation
	node.SetParams(map[string]interface{}{
		"operation": "list",
	})

	_, err = node.Run(nil)
	if err != nil {
		t.Fatalf("Failed to execute list operation: %v", err)
	}

	// Test delete operation
	node.SetParams(map[string]interface{}{
		"operation": "delete",
		"key":       "test-key",
	})

	_, err = node.Run(nil)
	if err != nil {
		t.Fatalf("Failed to execute delete operation: %v", err)
	}

	// Test get after delete (should fail)
	node.SetParams(map[string]interface{}{
		"operation": "get",
		"key":       "test-key",
	})

	_, err = node.Run(nil)
	if err == nil {
		t.Error("Expected error when getting deleted key, got nil")
	}
}

func TestDelayNode(t *testing.T) {
	// Create the node with a short delay
	params := map[string]interface{}{
		"duration": "10ms",
	}

	node, err := NewDelayNodeWrapper(params)
	if err != nil {
		t.Fatalf("Failed to create delay node: %v", err)
	}

	// Measure execution time
	start := time.Now()
	_, err = node.Run(nil)
	if err != nil {
		t.Fatalf("Failed to execute delay node: %v", err)
	}
	elapsed := time.Since(start)

	// Check that the delay was at least the specified duration
	if elapsed < 10*time.Millisecond {
		t.Errorf("Expected delay of at least 10ms, got %v", elapsed)
	}
}

func TestConditionNode(t *testing.T) {
	// Create the node
	node, err := NewConditionNodeWrapper(map[string]interface{}{})
	if err != nil {
		t.Fatalf("Failed to create condition node: %v", err)
	}

	// Execute the node (should return true in our placeholder implementation)
	result, err := node.Run(nil)
	if err != nil {
		t.Fatalf("Failed to execute condition node: %v", err)
	}

	// Check the result
	if result != "true" {
		t.Errorf("Expected result to be 'true', got '%s'", result)
	}
}
