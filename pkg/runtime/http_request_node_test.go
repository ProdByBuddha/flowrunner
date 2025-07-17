package runtime

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestEnhancedHTTPRequestNode(t *testing.T) {
	// Create a test server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Set response headers
		w.Header().Set("Content-Type", "application/json")

		// Handle different endpoints
		switch r.URL.Path {
		case "/ping":
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`{"status":"ok"}`))

		case "/echo":
			// Echo back the request body
			var requestBody map[string]interface{}
			if err := json.NewDecoder(r.Body).Decode(&requestBody); err != nil {
				w.WriteHeader(http.StatusBadRequest)
				w.Write([]byte(`{"error":"invalid request body"}`))
				return
			}

			w.WriteHeader(http.StatusOK)
			json.NewEncoder(w).Encode(requestBody)

		case "/auth":
			// Check for authorization header
			authHeader := r.Header.Get("Authorization")
			if authHeader != "Bearer test-token" {
				w.WriteHeader(http.StatusUnauthorized)
				w.Write([]byte(`{"error":"unauthorized"}`))
				return
			}

			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`{"status":"authenticated"}`))

		default:
			w.WriteHeader(http.StatusNotFound)
			w.Write([]byte(`{"error":"not found"}`))
		}
	}))
	defer server.Close()

	// Test basic GET request
	t.Run("Basic GET request", func(t *testing.T) {
		node, err := NewHTTPRequestNodeWrapper(map[string]interface{}{
			"url":    server.URL + "/ping",
			"method": "GET",
		})
		assert.NoError(t, err)
		assert.NotNil(t, node)

		result, err := node.(*NodeWrapper).exec(map[string]interface{}{
			"url":    server.URL + "/ping",
			"method": "GET",
		})
		assert.NoError(t, err)
		assert.NotNil(t, result)

		resultMap, ok := result.(map[string]interface{})
		assert.True(t, ok)
		assert.Equal(t, 200, resultMap["status_code"])
		assert.True(t, resultMap["success"].(bool))
	})

	// Test request with custom headers
	t.Run("Request with custom headers", func(t *testing.T) {
		node, err := NewHTTPRequestNodeWrapper(map[string]interface{}{
			"url":    server.URL + "/ping",
			"method": "GET",
			"headers": map[string]interface{}{
				"User-Agent": "FlowRunner-Test",
			},
		})
		assert.NoError(t, err)
		assert.NotNil(t, node)

		result, err := node.(*NodeWrapper).exec(map[string]interface{}{
			"url":    server.URL + "/ping",
			"method": "GET",
			"headers": map[string]interface{}{
				"User-Agent": "FlowRunner-Test",
			},
		})
		assert.NoError(t, err)
		assert.NotNil(t, result)

		resultMap, ok := result.(map[string]interface{})
		assert.True(t, ok)
		assert.Equal(t, 200, resultMap["status_code"])
	})

	// Test POST request with JSON body
	t.Run("POST request with JSON body", func(t *testing.T) {
		node, err := NewHTTPRequestNodeWrapper(map[string]interface{}{
			"url":    server.URL + "/echo",
			"method": "POST",
			"headers": map[string]interface{}{
				"Content-Type": "application/json",
			},
			"body": map[string]interface{}{
				"message": "Hello, World!",
				"test":    true,
				"number":  42,
			},
		})
		assert.NoError(t, err)
		assert.NotNil(t, node)

		result, err := node.(*NodeWrapper).exec(map[string]interface{}{
			"url":    server.URL + "/echo",
			"method": "POST",
			"headers": map[string]interface{}{
				"Content-Type": "application/json",
			},
			"body": map[string]interface{}{
				"message": "Hello, World!",
				"test":    true,
				"number":  42,
			},
		})
		assert.NoError(t, err)
		assert.NotNil(t, result)

		resultMap, ok := result.(map[string]interface{})
		assert.True(t, ok)
		assert.Equal(t, 200, resultMap["status_code"])

		// Check if the response body contains our echoed data
		body, ok := resultMap["body"].(map[string]interface{})
		if assert.True(t, ok) {
			assert.Equal(t, "Hello, World!", body["message"])
			assert.Equal(t, true, body["test"])
			assert.Equal(t, float64(42), body["number"])
		}
	})

	// Test with bearer token auth
	t.Run("Request with bearer token", func(t *testing.T) {
		node, err := NewHTTPRequestNodeWrapper(map[string]interface{}{
			"url":          server.URL + "/auth",
			"method":       "GET",
			"bearer_token": "test-token",
		})
		assert.NoError(t, err)
		assert.NotNil(t, node)

		result, err := node.(*NodeWrapper).exec(map[string]interface{}{
			"url":          server.URL + "/auth",
			"method":       "GET",
			"bearer_token": "test-token",
		})
		assert.NoError(t, err)
		assert.NotNil(t, result)

		resultMap, ok := result.(map[string]interface{})
		assert.True(t, ok)
		assert.Equal(t, 200, resultMap["status_code"])

		// Check if the response body indicates authentication success
		body, ok := resultMap["body"].(map[string]interface{})
		if assert.True(t, ok) {
			assert.Equal(t, "authenticated", body["status"])
		}
	})

	// Test with timeout
	t.Run("Request with timeout", func(t *testing.T) {
		node, err := NewHTTPRequestNodeWrapper(map[string]interface{}{
			"url":     server.URL + "/ping",
			"method":  "GET",
			"timeout": "5s",
		})
		assert.NoError(t, err)
		assert.NotNil(t, node)

		result, err := node.(*NodeWrapper).exec(map[string]interface{}{
			"url":     server.URL + "/ping",
			"method":  "GET",
			"timeout": "5s",
		})
		assert.NoError(t, err)
		assert.NotNil(t, result)

		resultMap, ok := result.(map[string]interface{})
		assert.True(t, ok)
		assert.Equal(t, 200, resultMap["status_code"])

		// Check if timing information is included
		_, hasTimingMs := resultMap["timing_ms"]
		assert.True(t, hasTimingMs)
	})

	// Test routing based on status code
	t.Run("Routing based on status code", func(t *testing.T) {
		node, err := NewHTTPRequestNodeWrapper(map[string]interface{}{
			"url":    server.URL + "/ping",
			"method": "GET",
		})
		assert.NoError(t, err)
		assert.NotNil(t, node)

		// Execute the node
		result, err := node.(*NodeWrapper).exec(map[string]interface{}{
			"url":    server.URL + "/ping",
			"method": "GET",
		})
		assert.NoError(t, err)

		// Test the post function for routing
		action, err := node.(*NodeWrapper).post(nil, nil, result)
		assert.NoError(t, err)
		assert.Equal(t, "success", action)

		// Test with 404 status code
		result, err = node.(*NodeWrapper).exec(map[string]interface{}{
			"url":    server.URL + "/nonexistent",
			"method": "GET",
		})
		assert.NoError(t, err)

		action, err = node.(*NodeWrapper).post(nil, nil, result)
		assert.NoError(t, err)
		assert.Equal(t, "client_error", action)
	})
}
