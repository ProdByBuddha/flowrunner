package plugins

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/r3labs/sse/v2"
	"github.com/tcmartin/flowlib"
)

// MCPPlugin is a custom node plugin for interacting with an MCP server.
type MCPPlugin struct{}

// Name returns the name of the plugin.
func (p *MCPPlugin) Name() string {
	return "mcp"
}

// Description returns a description of the plugin.
func (p *MCPPlugin) Description() string {
	return "A plugin for interacting with a Model Context Protocol server."
}

// Version returns the version of the plugin.
func (p *MCPPlugin) Version() string {
	return "1.0.0"
}

// Helper function to get a string parameter from the map.
func getStringParam(params map[string]interface{}, name string, required bool) (string, error) {
	val, ok := params[name]
	if !ok {
		if required {
			return "", fmt.Errorf("mcp: '%s' parameter is required", name)
		}
		return "", nil
	}
	strVal, ok := val.(string)
	if !ok {
		return "", fmt.Errorf("mcp: '%s' parameter must be a string", name)
	}
	return strVal, nil
}

// executeCMD runs the command with the given input and returns the output.
func executeCMD(command string, args []string, env []string, inputPayload []byte) ([]byte, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second) // 60 second timeout for the command
	defer cancel()

	cmd := exec.CommandContext(ctx, command, args...)
	cmd.Env = append(os.Environ(), env...)

	stdin, err := cmd.StdinPipe()
	if err != nil {
		return nil, fmt.Errorf("failed to get stdin pipe: %w", err)
	}

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Start(); err != nil {
		return nil, fmt.Errorf("failed to start command: %w", err)
	}

	_, err = stdin.Write(inputPayload)
	if err != nil {
		// It's possible the process exited before we could write.
		// Check if there's already an error from the command.
		cmd.Wait()
		return nil, fmt.Errorf("failed to write to stdin: %w, stderr: %s", err, stderr.String())
	}
	stdin.Close()

	if err := cmd.Wait(); err != nil {
		return nil, fmt.Errorf("command failed: %w, stderr: %s", err, stderr.String())
	}

	if stderr.Len() > 0 {
		// Log stderr but don't necessarily fail, as some tools use it for progress
		fmt.Printf("MCP command stderr: %s", stderr.String())
	}

	return stdout.Bytes(), nil
}

// executeHTTP sends a request to the given URL and returns the response.
func executeHTTP(url string, headers map[string]string, timeout time.Duration, inputPayload []byte) ([]byte, error) {
	client := &http.Client{
		Timeout: timeout,
	}

	req, err := http.NewRequest("POST", url, bytes.NewBuffer(inputPayload))
	if err != nil {
		return nil, fmt.Errorf("failed to create http request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	for k, v := range headers {
		req.Header.Set(k, v)
	}

	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("http request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("http request returned non-200 status: %d, body: %s", resp.StatusCode, string(body))
	}

	return io.ReadAll(resp.Body)
}

// executeSSE sends a request and listens for an SSE event.
func executeSSE(url, postEndpoint string, headers map[string]string, timeout time.Duration, inputPayload []byte) ([]byte, error) {
	// If there's a separate post endpoint, send the request there first.
	if postEndpoint != "" {
		postHeaders := make(map[string]string)
		for k, v := range headers {
			postHeaders[k] = v
		}
		postHeaders["Content-Type"] = "application/json"

		// This is a fire-and-forget post.
		_, err := executeHTTP(postEndpoint, postHeaders, timeout, inputPayload)
		if err != nil {
			return nil, fmt.Errorf("sse post to messages endpoint failed: %w", err)
		}
	}

	client := sse.NewClient(url)
	// TODO: This library doesn't easily support custom headers on the initial connection.
	// This is a limitation for SSE connections that require authentication headers.

	var eventData []byte
	errChan := make(chan error, 1)

	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	go func() {
		// The library will attempt to reconnect on its own, so we use a context to control it.
		err := client.SubscribeWithContext(ctx, "message", func(msg *sse.Event) {
			if len(msg.Data) > 0 {
				eventData = msg.Data
				cancel() // Got the message, stop subscribing.
			}
		})
		if err != nil && ctx.Err() == nil {
			errChan <- err
		}
	}()

	select {
	case <-ctx.Done():
		if ctx.Err() == context.DeadlineExceeded {
			return nil, fmt.Errorf("sse request timed out after %v", timeout)
		}
		// Context was canceled, likely because we received our message.
		if eventData == nil {
			return nil, fmt.Errorf("sse connection closed without receiving a message")
		}
		return eventData, nil
	case err := <-errChan:
		return nil, fmt.Errorf("sse subscription failed: %w", err)
	}
}

// CreateNode creates a new instance of the MCP node.
func (p *MCPPlugin) CreateNode(params map[string]interface{}) (flowlib.Node, error) {
	connectionType, _ := getStringParam(params, "connectionType", false)
	if connectionType == "" {
		connectionType = "cmd" // Default connection type
	}

	operation, err := getStringParam(params, "operation", true)
	if err != nil {
		return nil, err
	}

	node := flowlib.NewNode(0, 0)

	node.SetExecFn(func(input interface{}) (interface{}, error) {
		// Extract parameters within the execution function to allow dynamic inputs
		// CMD/STDIO params
		command, _ := getStringParam(params, "command", false)
		argsStr, _ := getStringParam(params, "args", false)
		envStr, _ := getStringParam(params, "env", false)
		var env []string
		if envStr != "" {
			if err := json.Unmarshal([]byte(envStr), &env); err == nil {
				// Fallback to splitting by newline for backward compatibility
				env = strings.Split(envStr, "\n")
			}
		}

		// HTTP/SSE params
		url, _ := getStringParam(params, "url", false)
		messagesPostEndpoint, _ := getStringParam(params, "messagesPostEndpoint", false)
		var headers map[string]string
		if h, ok := params["headers"]; ok {
			if headerMap, ok := h.(map[string]string); ok {
				headers = headerMap
			} else if headerMap, ok := h.(map[interface{}]interface{}); ok {
				headers = make(map[string]string)
				for k, v := range headerMap {
					headers[fmt.Sprintf("%v", k)] = fmt.Sprintf("%v", v)
				}
			}
		}
		var timeoutVal int
		if t, ok := params["timeout"].(float64); ok { // JSON numbers are float64
			timeoutVal = int(t)
		} else {
			timeoutVal = 60000 // default 60s
		}
		timeout := time.Duration(timeoutVal) * time.Millisecond

		// Operation-specific params
		resourceUri, _ := getStringParam(params, "resourceUri", false)
		toolName, _ := getStringParam(params, "toolName", false)
		toolParamsStr, _ := getStringParam(params, "toolParameters", false)
		promptName, _ := getStringParam(params, "promptName", false)

		// Construct the request payload based on the operation
		requestPayload := map[string]interface{}{
			"method": operation,
			"params": map[string]interface{}{},
		}
		opParams := requestPayload["params"].(map[string]interface{})
		switch operation {
		case "readResource":
			opParams["uri"] = resourceUri
		case "executeTool":
			opParams["name"] = toolName
			var toolArgs interface{}
			// The tool parameters can be a JSON string or an object from a previous node
			if toolParamsStr != "" {
				if err := json.Unmarshal([]byte(toolParamsStr), &toolArgs); err == nil {
					opParams["arguments"] = toolArgs
				} else {
					opParams["arguments"] = toolParamsStr // Pass as string if not valid JSON
				}
			} else if input != nil {
				// If toolParameters is empty, use the input from the previous node
				opParams["arguments"] = input
			}
		case "getPrompt":
			opParams["name"] = promptName
		}

		var resultData interface{}
		var execErr error

		reqBytes, err := json.Marshal(requestPayload)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal request payload: %w", err)
		}

		switch connectionType {
		case "cmd":
			if command == "" {
				return nil, fmt.Errorf("mcp: 'command' parameter is required for 'cmd' connection type")
			}
			args := strings.Fields(argsStr)

			outputBytes, err := executeCMD(command, args, env, reqBytes)
			if err != nil {
				execErr = fmt.Errorf("cmd execution failed: %w", err)
			} else if err := json.Unmarshal(outputBytes, &resultData); err != nil {
				resultData = string(outputBytes) // Return as raw string if not JSON
			}
		case "http":
			if url == "" {
				return nil, fmt.Errorf("mcp: 'url' parameter is required for 'http' connection type")
			}
			outputBytes, err := executeHTTP(url, headers, timeout, reqBytes)
			if err != nil {
				execErr = fmt.Errorf("http execution failed: %w", err)
			} else if err := json.Unmarshal(outputBytes, &resultData); err != nil {
				resultData = string(outputBytes)
			}
		case "sse":
			if url == "" {
				return nil, fmt.Errorf("mcp: 'url' parameter is required for 'sse' connection type")
			}
			outputBytes, err := executeSSE(url, messagesPostEndpoint, headers, timeout, reqBytes)
			if err != nil {
				execErr = fmt.Errorf("sse execution failed: %w", err)
			} else if err := json.Unmarshal(outputBytes, &resultData); err != nil {
				resultData = string(outputBytes)
			}
		default:
			execErr = fmt.Errorf("unsupported connection type: '%s'", connectionType)
		}

		if execErr != nil {
			return nil, execErr
		}

		return map[string]interface{}{"result": resultData}, nil
	})

	return node, nil
}