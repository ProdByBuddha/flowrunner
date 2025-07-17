// Package runtime provides functionality for executing flows.
package runtime

import (
	"fmt"
	"net/http"
	"time"

	"github.com/tcmartin/flowlib"
)

// CoreNodeTypes returns a map of built-in node types
func CoreNodeTypes() map[string]NodeFactory {
	return map[string]NodeFactory{
		"http.request":  NewHTTPRequestNode,
		"store":         NewStoreNode,
		"transform":     NewTransformNode,
		"condition":     NewConditionNode,
		"delay":         NewDelayNode,
		"llm":           NewLLMNode,
		"email.send":    NewSMTPNode,
		"email.receive": NewIMAPNode,
		"agent":         NewAgentNode,
		"webhook":       NewWebhookNode,
	}
}

// NodeFactory is a function that creates a node
type NodeFactory func(params map[string]interface{}) (flowlib.Node, error)

// HTTPRequestNode makes HTTP requests
type HTTPRequestNode struct {
	*flowlib.NodeWithRetry
	client *http.Client
}

// NewHTTPRequestNode creates a new HTTP request node
func NewHTTPRequestNode(params map[string]interface{}) (flowlib.Node, error) {
	node := &HTTPRequestNode{
		NodeWithRetry: flowlib.NewNode(3, 1*time.Second),
		client:        &http.Client{},
	}
	node.SetParams(params)
	node.execFn = node.Exec
	return node, nil
}

// Exec executes the HTTP request
func (n *HTTPRequestNode) Exec(input interface{}) (interface{}, error) {
	// Get parameters
	params := n.Params()
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
	resp, err := n.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	// TODO: Parse response based on content type

	// Return response
	return map[string]interface{}{
		"status_code": resp.StatusCode,
		"headers":     resp.Header,
		// "body": parsed body
	}, nil
}

// StoreNode provides in-memory data storage
type StoreNode struct {
	*flowlib.NodeWithRetry
	store map[string]interface{}
}

// NewStoreNode creates a new store node
func NewStoreNode(params map[string]interface{}) (flowlib.Node, error) {
	node := &StoreNode{
		NodeWithRetry: flowlib.NewNode(1, 0),
		store:         make(map[string]interface{}),
	}
	node.SetParams(params)
	node.execFn = node.Exec
	return node, nil
}

// Exec executes the store operation
func (n *StoreNode) Exec(input interface{}) (interface{}, error) {
	// Get parameters
	params := n.Params()
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
		value, exists := n.store[key]
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
		n.store[key] = value
		return value, nil

	case "delete":
		key, ok := params["key"].(string)
		if !ok {
			return nil, fmt.Errorf("key parameter is required for delete operation")
		}
		delete(n.store, key)
		return nil, nil

	case "list":
		keys := make([]string, 0, len(n.store))
		for key := range n.store {
			keys = append(keys, key)
		}
		return keys, nil

	default:
		return nil, fmt.Errorf("unknown operation: %s", operation)
	}
}

// TransformNode transforms data
type TransformNode struct {
	*flowlib.NodeWithRetry
}

// NewTransformNode creates a new transform node
func NewTransformNode(params map[string]interface{}) (flowlib.Node, error) {
	node := &TransformNode{
		NodeWithRetry: flowlib.NewNode(1, 0),
	}
	node.SetParams(params)
	node.execFn = node.Exec
	return node, nil
}

// Exec executes the transformation
func (n *TransformNode) Exec(input interface{}) (interface{}, error) {
	// This is a placeholder - in a real implementation, this would use the
	// JavaScript engine to transform the input data
	return input, nil
}

// ConditionNode evaluates a condition
type ConditionNode struct {
	*flowlib.NodeWithRetry
}

// NewConditionNode creates a new condition node
func NewConditionNode(params map[string]interface{}) (flowlib.Node, error) {
	node := &ConditionNode{
		NodeWithRetry: flowlib.NewNode(1, 0),
	}
	node.SetParams(params)
	node.execFn = node.Exec
	node.baseNode.postFn = node.Post
	return node, nil
}

// Exec evaluates the condition
func (n *ConditionNode) Exec(input interface{}) (interface{}, error) {
	// This is a placeholder - in a real implementation, this would evaluate
	// the condition using the JavaScript engine
	return true, nil
}

// Post returns the appropriate action based on the condition result
func (n *ConditionNode) Post(shared, p, e interface{}) (flowlib.Action, error) {
	result, ok := e.(bool)
	if !ok {
		return "", fmt.Errorf("expected boolean result, got %T", e)
	}

	if result {
		return "true", nil
	}
	return "false", nil
}

// DelayNode waits for a specified duration
type DelayNode struct {
	*flowlib.NodeWithRetry
}

// NewDelayNode creates a new delay node
func NewDelayNode(params map[string]interface{}) (flowlib.Node, error) {
	node := &DelayNode{
		NodeWithRetry: flowlib.NewNode(1, 0),
	}
	node.SetParams(params)
	node.execFn = node.Exec
	return node, nil
}

// Exec waits for the specified duration
func (n *DelayNode) Exec(input interface{}) (interface{}, error) {
	// Get parameters
	params := n.Params()
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
}

// LLMNode makes calls to LLM APIs
type LLMNode struct {
	*flowlib.NodeWithRetry
}

// NewLLMNode creates a new LLM node
func NewLLMNode(params map[string]interface{}) (flowlib.Node, error) {
	node := &LLMNode{
		NodeWithRetry: flowlib.NewNode(3, 1*time.Second),
	}
	node.SetParams(params)
	node.execFn = node.Exec
	return node, nil
}

// Exec makes an LLM API call
func (n *LLMNode) Exec(input interface{}) (interface{}, error) {
	// This is a placeholder - in a real implementation, this would call
	// an LLM API like OpenAI
	return map[string]interface{}{
		"text": "This is a placeholder response from the LLM node.",
	}, nil
}

// SMTPNode sends emails
type SMTPNode struct {
	*flowlib.NodeWithRetry
}

// NewSMTPNode creates a new SMTP node
func NewSMTPNode(params map[string]interface{}) (flowlib.Node, error) {
	node := &SMTPNode{
		NodeWithRetry: flowlib.NewNode(3, 1*time.Second),
	}
	node.SetParams(params)
	node.execFn = node.Exec
	return node, nil
}

// Exec sends an email
func (n *SMTPNode) Exec(input interface{}) (interface{}, error) {
	// This is a placeholder - in a real implementation, this would send
	// an email using an SMTP client
	return map[string]interface{}{
		"status": "sent",
	}, nil
}

// IMAPNode receives emails
type IMAPNode struct {
	*flowlib.NodeWithRetry
}

// NewIMAPNode creates a new IMAP node
func NewIMAPNode(params map[string]interface{}) (flowlib.Node, error) {
	node := &IMAPNode{
		NodeWithRetry: flowlib.NewNode(3, 1*time.Second),
	}
	node.SetParams(params)
	node.execFn = node.Exec
	return node, nil
}

// Exec receives emails
func (n *IMAPNode) Exec(input interface{}) (interface{}, error) {
	// This is a placeholder - in a real implementation, this would receive
	// emails using an IMAP client
	return []map[string]interface{}{
		{
			"subject": "Test Email",
			"from":    "sender@example.com",
			"body":    "This is a test email.",
		},
	}, nil
}

// AgentNode implements an AI agent
type AgentNode struct {
	*flowlib.NodeWithRetry
}

// NewAgentNode creates a new agent node
func NewAgentNode(params map[string]interface{}) (flowlib.Node, error) {
	node := &AgentNode{
		NodeWithRetry: flowlib.NewNode(3, 1*time.Second),
	}
	node.SetParams(params)
	node.execFn = node.Exec
	return node, nil
}

// Exec runs the agent
func (n *AgentNode) Exec(input interface{}) (interface{}, error) {
	// This is a placeholder - in a real implementation, this would run
	// an AI agent with reasoning capabilities
	return map[string]interface{}{
		"result": "This is a placeholder result from the agent node.",
	}, nil
}

// WebhookNode sends a webhook
type WebhookNode struct {
	*flowlib.NodeWithRetry
	client *http.Client
}

// NewWebhookNode creates a new webhook node
func NewWebhookNode(params map[string]interface{}) (flowlib.Node, error) {
	node := &WebhookNode{
		NodeWithRetry: flowlib.NewNode(3, 1*time.Second),
		client:        &http.Client{},
	}
	node.SetParams(params)
	node.execFn = node.Exec
	return node, nil
}

// Exec sends a webhook
func (n *WebhookNode) Exec(input interface{}) (interface{}, error) {
	// This is a placeholder - in a real implementation, this would send
	// a webhook to a configured URL
	return map[string]interface{}{
		"status": "sent",
	}, nil
}
