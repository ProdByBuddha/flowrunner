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
func (w *NodeWrapper) Next(action flowlib.Action, n flowlib.Node) {
	w.node.Next(action, n)
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

		// Determine if this is a flow execution or direct node call
		// Flow execution: shared contains flow input data (may have "question", "input", etc.)
		// Direct node call: shared is empty or only contains result storage
		var combinedInput map[string]interface{}
		
		if sharedMap, ok := shared.(map[string]interface{}); ok {
			// Check if this looks like flow input (has question, input data, etc.)
			hasFlowInput := false
			for key := range sharedMap {
				// These are typical flow input keys
				if key == "question" || key == "input" || key == "context" || key == "data" {
					hasFlowInput = true
					break
				}
				// If it only has result keys, it's probably direct node usage
				if key == "result" || key == "llm_result" || key == "http_result" {
					continue
				}
				// If it has other non-result keys, assume it's flow input
				if !strings.HasSuffix(key, "_result") {
					hasFlowInput = true
					break
				}
			}
			
			if hasFlowInput {
				// Flow execution: create combined input format
				combinedInput = map[string]interface{}{
					"params": params,
					"input":  shared,
				}
			} else {
				// Direct node usage: use stored parameters only
				combinedInput = map[string]interface{}{
					"params": params,
					"input":  map[string]interface{}{}, // empty flow input
				}
			}
		} else {
			// Non-map shared context: assume direct node usage
			combinedInput = map[string]interface{}{
				"params": params,
				"input":  map[string]interface{}{},
			}
		}

		// Execute the function
		result, err := w.exec(combinedInput)
		if err != nil {
			return "", err
		}

		// Store the result in the shared context if it's a map
		if sharedMap, ok := shared.(map[string]interface{}); ok {
			// Store the result with a type-specific key
			nodeType := "result"
			if typeParam, ok := params["type"].(string); ok {
				nodeType = typeParam
			} else {
				// Try to determine the node type from the parameters
				if _, ok := params["url"]; ok {
					nodeType = "http"
				} else if _, ok := params["smtp_host"]; ok {
					nodeType = "email"
				} else if _, ok := params["model"]; ok {
					nodeType = "llm"
				} else if _, ok := params["operation"]; ok {
					nodeType = "store"
				}
			}

			// Store the result with the node type as the key
			sharedMap[nodeType+"_result"] = result

			// Also store in the generic "result" key for backward compatibility
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
			// Handle both old format (direct params) and new format (combined input)
			var params map[string]interface{}
			
			if combinedInput, ok := input.(map[string]interface{}); ok {
				if nodeParams, hasParams := combinedInput["params"]; hasParams {
					// New format: combined input with params and input
					if paramsMap, ok := nodeParams.(map[string]interface{}); ok {
						params = paramsMap
					} else {
						return nil, fmt.Errorf("expected params to be map[string]interface{}")
					}
				} else {
					// Old format: direct params (backwards compatibility)
					params = combinedInput
				}
			} else {
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
					} else {
						// Convert non-string values to string
						headers[key] = fmt.Sprintf("%v", value)
					}
				}
			}

			// Extract body based on content type
			var body interface{}
			contentType := ""

			if bodyParam, ok := params["body"]; ok {
				body = bodyParam

				// Check if content type is specified in headers
				if ct, ok := headers["Content-Type"]; ok {
					contentType = ct
				} else if ct, ok := headers["content-type"]; ok {
					contentType = ct
				}

				// If no content type is specified, try to determine it from the body
				if contentType == "" {
					switch bodyParam.(type) {
					case map[string]interface{}, []interface{}:
						contentType = "application/json"
					case string:
						// Check if it looks like JSON
						strBody := bodyParam.(string)
						if len(strBody) > 0 && (strBody[0] == '{' || strBody[0] == '[') {
							contentType = "application/json"
						} else {
							contentType = "text/plain"
						}
					}

					// Set the content type header if determined
					if contentType != "" {
						headers["Content-Type"] = contentType
					}
				}
			}

			// Handle file uploads
			if fileParam, ok := params["file"]; ok {
				// File upload handling would go here
				// For now, we'll just log that it's not fully implemented
				fmt.Println("File upload requested but not fully implemented")

				// If we have a file path, we could read the file and set it as the body
				if filePath, ok := fileParam.(string); ok {
					// In a real implementation, we would read the file and set up multipart form data
					fmt.Printf("Would upload file: %s\n", filePath)
				}
			}

			// Handle form data
			if formData, ok := params["form_data"].(map[string]interface{}); ok {
				// In a real implementation, we would set up form data
				// For now, we'll just convert it to a string representation
				formStr := ""
				for k, v := range formData {
					if formStr != "" {
						formStr += "&"
					}
					formStr += fmt.Sprintf("%s=%v", k, v)
				}
				body = formStr
				if contentType == "" {
					headers["Content-Type"] = "application/x-www-form-urlencoded"
				}
			}

			// Extract timeout
			var timeout time.Duration
			if timeoutParam, ok := params["timeout"].(string); ok {
				if parsedTimeout, err := time.ParseDuration(timeoutParam); err == nil {
					timeout = parsedTimeout
				}
			} else if timeoutNum, ok := params["timeout"].(float64); ok {
				// Handle numeric timeout in seconds
				timeout = time.Duration(timeoutNum * float64(time.Second))
			}

			// Extract authentication
			var auth map[string]interface{}
			if authParam, ok := params["auth"].(map[string]interface{}); ok {
				auth = authParam
			}

			// Handle specific auth types
			if bearerToken, ok := params["bearer_token"].(string); ok {
				if auth == nil {
					auth = make(map[string]interface{})
				}
				auth["token"] = bearerToken
			}

			if apiKey, ok := params["api_key"].(string); ok {
				if auth == nil {
					auth = make(map[string]interface{})
				}
				auth["api_key"] = apiKey

				// Check if key name is provided
				if keyName, ok := params["key_name"].(string); ok {
					auth["key_name"] = keyName
				}
			}

			// Handle basic auth directly
			if username, ok := params["username"].(string); ok {
				if password, ok := params["password"].(string); ok {
					if auth == nil {
						auth = make(map[string]interface{})
					}
					auth["username"] = username
					auth["password"] = password
				}
			}

			// Extract follow redirects option
			followRedirects := true
			if followParam, ok := params["follow_redirects"].(bool); ok {
				followRedirects = followParam
			}

			// Create HTTP request
			httpRequest := &utils.HTTPRequest{
				URL:            url,
				Method:         method,
				Headers:        headers,
				Body:           body,
				Timeout:        timeout,
				Auth:           auth,
				FollowRedirect: followRedirects,
			}

			// Execute request
			resp, err := httpClient.Do(httpRequest)
			if err != nil {
				return nil, err
			}

			// Return response
			result := map[string]interface{}{
				"status_code": resp.StatusCode,
				"headers":     resp.Headers,
				"body":        resp.Body,
				"raw_body":    string(resp.RawBody),
				"metadata":    resp.Metadata,
				"success":     resp.StatusCode >= 200 && resp.StatusCode < 300,
			}

			// Add timing information if available
			if resp.Metadata != nil {
				if timing, ok := resp.Metadata["timing"].(time.Duration); ok {
					result["timing_ms"] = timing.Milliseconds()
				}
			}

			return result, nil
		},
		post: func(shared, p, e interface{}) (flowlib.Action, error) {
			// Get the result
			result, ok := e.(map[string]interface{})
			if !ok {
				return flowlib.DefaultAction, nil
			}

			// Check if we should route based on status code
			if statusCode, ok := result["status_code"].(int); ok {
				// Route based on status code range
				if statusCode >= 200 && statusCode < 300 {
					return "success", nil
				} else if statusCode >= 400 && statusCode < 500 {
					return "client_error", nil
				} else if statusCode >= 500 {
					return "server_error", nil
				}
			}

			return flowlib.DefaultAction, nil
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
			// Handle both old format (direct params) and new format (combined input)
			var params map[string]interface{}
			
			if combinedInput, ok := input.(map[string]interface{}); ok {
				if nodeParams, hasParams := combinedInput["params"]; hasParams {
					// New format: combined input with params and input
					if paramsMap, ok := nodeParams.(map[string]interface{}); ok {
						params = paramsMap
					} else {
						return nil, fmt.Errorf("expected params to be map[string]interface{}")
					}
				} else {
					// Old format: direct params (backwards compatibility)
					params = combinedInput
				}
			} else {
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
			// Handle both old format (direct params) and new format (combined input)
			var params map[string]interface{}
			
			if combinedInput, ok := input.(map[string]interface{}); ok {
				if nodeParams, hasParams := combinedInput["params"]; hasParams {
					// New format: combined input with params and input
					if paramsMap, ok := nodeParams.(map[string]interface{}); ok {
						params = paramsMap
					} else {
						return nil, fmt.Errorf("expected params to be map[string]interface{}")
					}
				} else {
					// Old format: direct params (backwards compatibility)
					params = combinedInput
				}
			} else {
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
			// Handle both old format (direct params) and new format (combined input)
			// For condition node, we don't actually use the params in this placeholder implementation
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
