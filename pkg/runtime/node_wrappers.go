package runtime

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/tcmartin/flowlib"
)

// NodeWrapper is a base wrapper for flowlib.Node implementations
type NodeWrapper struct {
	node flowlib.Node
	exec func(input interface{}) (interface{}, error)
	post func(shared, p, e interface{}) (flowlib.Action, error)
}

// SetParams sets the parameters for the node
func (w *NodeWrapper) SetParams(params map[string]interface{}) {
	w.node.SetParams(params)
}

// Params returns the parameters for the node
func (w *NodeWrapper) Params() map[string]interface{} {
	return w.node.Params()
}

// Next sets the next node for the given action
func (w *NodeWrapper) Next(action flowlib.Action, n flowlib.Node) flowlib.Node {
	return w.node.Next(action, n)
}

// Successors returns the successors of the node
func (w *NodeWrapper) Successors() map[flowlib.Action]flowlib.Node {
	return w.node.Successors()
}

// Run executes the node
func (w *NodeWrapper) Run(shared interface{}) (flowlib.Action, error) {
	// Create a custom implementation that calls our exec function
	if w.exec != nil {
		// Get the parameters
		params := w.Params()

		// Execute the function
		result, err := w.exec(params)
		if err != nil {
			return "", err
		}

		// Call the post function if provided
		if w.post != nil {
			return w.post(shared, params, result)
		}

		// Default to the "default" action
		return flowlib.DefaultAction, nil
	}

	// Fall back to the wrapped node's Run method
	return w.node.Run(shared)
}

// NewHTTPRequestNodeWrapper creates a new HTTP request node wrapper
func NewHTTPRequestNodeWrapper(params map[string]interface{}) (flowlib.Node, error) {
	// Create the base node
	baseNode := flowlib.NewNode(3, 1*time.Second)

	// Create the client
	client := &http.Client{}

	// Create the wrapper
	wrapper := &NodeWrapper{
		node: baseNode,
		exec: func(input interface{}) (interface{}, error) {
			// Get parameters from input
			params, ok := input.(map[string]interface{})
			if !ok {
				return nil, fmt.Errorf("expected map[string]interface{}, got %T", input)
			}

			url, ok := params["url"].(string)
			if !ok {
				return nil, fmt.Errorf("url parameter is required")
			}

			method, ok := params["method"].(string)
			if !ok {
				method = "GET"
			}

			// Create request
			req, err := http.NewRequest(method, url, nil)
			if err != nil {
				return nil, fmt.Errorf("failed to create request: %w", err)
			}

			// Add headers
			if headers, ok := params["headers"].(map[string]interface{}); ok {
				for key, value := range headers {
					if strValue, ok := value.(string); ok {
						req.Header.Add(key, strValue)
					}
				}
			}

			// Execute request
			resp, err := client.Do(req)
			if err != nil {
				return nil, fmt.Errorf("request failed: %w", err)
			}
			defer resp.Body.Close()

			// Read the response body
			body, err := io.ReadAll(resp.Body)
			if err != nil {
				return nil, fmt.Errorf("failed to read response body: %w", err)
			}

			// Parse response based on content type
			var parsedBody interface{}
			contentType := resp.Header.Get("Content-Type")
			if strings.Contains(contentType, "application/json") {
				// Parse JSON
				if err := json.Unmarshal(body, &parsedBody); err != nil {
					// If JSON parsing fails, use the raw body
					parsedBody = string(body)
				}
			} else {
				// Use raw body for non-JSON responses
				parsedBody = string(body)
			}

			// Return response
			return map[string]interface{}{
				"status_code": resp.StatusCode,
				"headers":     resp.Header,
				"body":        parsedBody,
			}, nil
		},
	}

	// Set the parameters
	wrapper.SetParams(params)

	return wrapper, nil
}

// NewStoreNodeWrapper creates a new store node wrapper
func NewStoreNodeWrapper(params map[string]interface{}) (flowlib.Node, error) {
	// Create the base node
	baseNode := flowlib.NewNode(1, 0)

	// Create the store
	store := make(map[string]interface{})

	// Create the wrapper
	wrapper := &NodeWrapper{
		node: baseNode,
		exec: func(input interface{}) (interface{}, error) {
			// Get parameters from input
			params, ok := input.(map[string]interface{})
			if !ok {
				return nil, fmt.Errorf("expected map[string]interface{}, got %T", input)
			}

			operation, ok := params["operation"].(string)
			if !ok {
				return nil, fmt.Errorf("operation parameter is required")
			}

			switch operation {
			case "get":
				key, ok := params["key"].(string)
				if !ok {
					return nil, fmt.Errorf("key parameter is required for get operation")
				}
				value, exists := store[key]
				if !exists {
					return nil, fmt.Errorf("key not found: %s", key)
				}
				return value, nil

			case "set":
				key, ok := params["key"].(string)
				if !ok {
					return nil, fmt.Errorf("key parameter is required for set operation")
				}
				value, ok := params["value"]
				if !ok {
					return nil, fmt.Errorf("value parameter is required for set operation")
				}
				store[key] = value
				return value, nil

			case "delete":
				key, ok := params["key"].(string)
				if !ok {
					return nil, fmt.Errorf("key parameter is required for delete operation")
				}
				delete(store, key)
				return nil, nil

			case "list":
				keys := make([]string, 0, len(store))
				for key := range store {
					keys = append(keys, key)
				}
				return keys, nil

			default:
				return nil, fmt.Errorf("unknown operation: %s", operation)
			}
		},
	}

	// Set the parameters
	wrapper.SetParams(params)

	return wrapper, nil
}

// NewDelayNodeWrapper creates a new delay node wrapper
func NewDelayNodeWrapper(params map[string]interface{}) (flowlib.Node, error) {
	// Create the base node
	baseNode := flowlib.NewNode(1, 0)

	// Create the wrapper
	wrapper := &NodeWrapper{
		node: baseNode,
		exec: func(input interface{}) (interface{}, error) {
			// Get parameters from input
			params, ok := input.(map[string]interface{})
			if !ok {
				return nil, fmt.Errorf("expected map[string]interface{}, got %T", input)
			}

			durationStr, ok := params["duration"].(string)
			if !ok {
				return nil, fmt.Errorf("duration parameter is required")
			}

			// Parse duration
			duration, err := time.ParseDuration(durationStr)
			if err != nil {
				return nil, fmt.Errorf("invalid duration: %w", err)
			}

			// Wait
			time.Sleep(duration)

			return input, nil
		},
	}

	// Set the parameters
	wrapper.SetParams(params)

	return wrapper, nil
}

// NewConditionNodeWrapper creates a new condition node wrapper
func NewConditionNodeWrapper(params map[string]interface{}) (flowlib.Node, error) {
	// Create the base node
	baseNode := flowlib.NewNode(1, 0)

	// Create the wrapper
	wrapper := &NodeWrapper{
		node: baseNode,
		exec: func(input interface{}) (interface{}, error) {
			// This is a placeholder - in a real implementation, this would evaluate
			// the condition using the JavaScript engine
			return true, nil
		},
		post: func(shared, p, e interface{}) (flowlib.Action, error) {
			result, ok := e.(bool)
			if !ok {
				return "", fmt.Errorf("expected boolean result, got %T", e)
			}

			if result {
				return "true", nil
			}
			return "false", nil
		},
	}

	// Set the parameters
	wrapper.SetParams(params)

	return wrapper, nil
}
