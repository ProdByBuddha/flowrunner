// Package runtime provides functionality for executing flows.
package runtime

import (
	"net/http"
	"time"

	"github.com/tcmartin/flowlib"
)

// CoreNodeTypes returns a map of built-in node types
func CoreNodeTypes() map[string]NodeFactory {
	return map[string]NodeFactory{
		"http.request":  NewHTTPRequestNodeWrapper,
		"store":         NewStoreNodeWrapper,
		"transform":     NewTransformNodeWrapper,
		"condition":     NewConditionNodeWrapper,
		"delay":         NewDelayNodeWrapper,
		"llm":           NewLLMNodeWrapper,
		"email.send":    NewSMTPNodeWrapper,
		"email.receive": NewIMAPNodeWrapper,
		"agent":         NewAgentNodeWrapper,
		"webhook":       NewWebhookNodeWrapper,
	}
}

// NodeFactory is a function that creates a node
type NodeFactory func(params map[string]interface{}) (flowlib.Node, error)

// NewTransformNodeWrapper creates a new transform node wrapper
func NewTransformNodeWrapper(params map[string]interface{}) (flowlib.Node, error) {
	// Create the base node
	baseNode := flowlib.NewNode(1, 0)

	// Create the wrapper
	wrapper := &NodeWrapper{
		node: baseNode,
		exec: func(input interface{}) (interface{}, error) {
			// This is a placeholder - in a real implementation, this would use the
			// JavaScript engine to transform the input data
			return input, nil
		},
	}

	// Set the parameters
	wrapper.SetParams(params)

	return wrapper, nil
}

// NewLLMNodeWrapper creates a new LLM node wrapper
func NewLLMNodeWrapper(params map[string]interface{}) (flowlib.Node, error) {
	// Create the base node
	baseNode := flowlib.NewNode(3, 1*time.Second)

	// Create the wrapper
	wrapper := &NodeWrapper{
		node: baseNode,
		exec: func(input interface{}) (interface{}, error) {
			// This is a placeholder - in a real implementation, this would call
			// an LLM API like OpenAI
			return map[string]interface{}{
				"text": "This is a placeholder response from the LLM node.",
			}, nil
		},
	}

	// Set the parameters
	wrapper.SetParams(params)

	return wrapper, nil
}

// NewSMTPNodeWrapper creates a new SMTP node wrapper
func NewSMTPNodeWrapper(params map[string]interface{}) (flowlib.Node, error) {
	// Create the base node
	baseNode := flowlib.NewNode(3, 1*time.Second)

	// Create the wrapper
	wrapper := &NodeWrapper{
		node: baseNode,
		exec: func(input interface{}) (interface{}, error) {
			// This is a placeholder - in a real implementation, this would send
			// an email using an SMTP client
			return map[string]interface{}{
				"status": "sent",
			}, nil
		},
	}

	// Set the parameters
	wrapper.SetParams(params)

	return wrapper, nil
}

// NewIMAPNodeWrapper creates a new IMAP node wrapper
func NewIMAPNodeWrapper(params map[string]interface{}) (flowlib.Node, error) {
	// Create the base node
	baseNode := flowlib.NewNode(3, 1*time.Second)

	// Create the wrapper
	wrapper := &NodeWrapper{
		node: baseNode,
		exec: func(input interface{}) (interface{}, error) {
			// This is a placeholder - in a real implementation, this would receive
			// emails using an IMAP client
			return []map[string]interface{}{
				{
					"subject": "Test Email",
					"from":    "sender@example.com",
					"body":    "This is a test email.",
				},
			}, nil
		},
	}

	// Set the parameters
	wrapper.SetParams(params)

	return wrapper, nil
}

// NewAgentNodeWrapper creates a new agent node wrapper
func NewAgentNodeWrapper(params map[string]interface{}) (flowlib.Node, error) {
	// Create the base node
	baseNode := flowlib.NewNode(3, 1*time.Second)

	// Create the wrapper
	wrapper := &NodeWrapper{
		node: baseNode,
		exec: func(input interface{}) (interface{}, error) {
			// This is a placeholder - in a real implementation, this would run
			// an AI agent with reasoning capabilities
			return map[string]interface{}{
				"result": "This is a placeholder result from the agent node.",
			}, nil
		},
	}

	// Set the parameters
	wrapper.SetParams(params)

	return wrapper, nil
}

// NewWebhookNodeWrapper creates a new webhook node wrapper
func NewWebhookNodeWrapper(params map[string]interface{}) (flowlib.Node, error) {
	// Create the base node
	baseNode := flowlib.NewNode(3, 1*time.Second)

	// Create the client
	client := &http.Client{}

	// Create the wrapper
	wrapper := &NodeWrapper{
		node: baseNode,
		exec: func(input interface{}) (interface{}, error) {
			// This is a placeholder - in a real implementation, this would send
			// a webhook to a configured URL
			return map[string]interface{}{
				"status": "sent",
			}, nil
		},
	}

	// Set the parameters
	wrapper.SetParams(params)

	return wrapper, nil
}
