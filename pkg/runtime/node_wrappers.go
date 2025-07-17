package runtime

import (
	"fmt"
	"time"

	"github.com/tcmartin/flowlib"
	"github.com/tcmartin/flowrunner/pkg/utils"
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

		// Store the result in the shared context if it's a map
		if sharedMap, ok := shared.(map[string]interface{}); ok {
			sharedMap["result"] = result
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

	// Create HTTP client
	httpClient := utils.NewHTTPClient()

	// Create the wrapper
	wrapper := &NodeWrapper{
		node: baseNode,
		exec: func(input interface{}) (interface{}, error) {
			// Get parameters from input
			params, ok := input.(map[string]interface{})
			if !ok {
				return nil, fmt.Errorf("expected map[string]interface{}, got %T", input)
			}

			// Extract parameters
			url, ok := params["url"].(string)
			if !ok {
				return nil, fmt.Errorf("url parameter is required")
			}

			method, _ := params["method"].(string) // Default is set in HTTPClient

			// Extract headers
			headers := make(map[string]string)
			if headersParam, ok := params["headers"].(map[string]interface{}); ok {
				for key, value := range headersParam {
					if strValue, ok := value.(string); ok {
						headers[key] = strValue
					}
				}
			}

			// Extract body
			var body interface{}
			if bodyParam, ok := params["body"]; ok {
				body = bodyParam
			}

			// Extract timeout
			var timeout time.Duration
			if timeoutParam, ok := params["timeout"].(string); ok {
				if parsedTimeout, err := time.ParseDuration(timeoutParam); err == nil {
					timeout = parsedTimeout
				}
			}

			// Extract authentication
			var auth map[string]interface{}
			if authParam, ok := params["auth"].(map[string]interface{}); ok {
				auth = authParam
			}

			// Create HTTP request
			httpRequest := &utils.HTTPRequest{
				URL:     url,
				Method:  method,
				Headers: headers,
				Body:    body,
				Timeout: timeout,
				Auth:    auth,
			}

			// Execute request
			resp, err := httpClient.Do(httpRequest)
			if err != nil {
				return nil, err
			}

			// Return response
			return map[string]interface{}{
				"status_code": resp.StatusCode,
				"headers":     resp.Headers,
				"body":        resp.Body,
				"raw_body":    string(resp.RawBody),
				"metadata":    resp.Metadata,
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
