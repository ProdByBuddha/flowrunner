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
	if result != "success" {
		t.Errorf("Expected result to be 'success', got '%s'", result)
	}
}

// Store node test moved to store_node_test.go for more comprehensive testing

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
	// Create the node with a simple JavaScript condition script
	config := map[string]interface{}{
		"condition_script": "return true;",
	}
	node, err := NewConditionNodeWrapper(config)
	if err != nil {
		t.Fatalf("Failed to create condition node: %v", err)
	}

	// Execute the node
	result, err := node.Run(nil)
	if err != nil {
		t.Fatalf("Failed to execute condition node: %v", err)
	}

	// Check the result
	if result != "true" {
		t.Errorf("Expected result to be 'true', got '%s'", result)
	}
}
