// Package webhooks provides functionality for sending HTTP callbacks.
package webhooks

import (
	"time"
)

// WebhookDispatcher sends HTTP callbacks
type WebhookDispatcher interface {
	// SendFlowCompleted notifies when a flow completes
	SendFlowCompleted(flowID string, executionID string, result map[string]interface{}) error

	// SendNodeCompleted notifies when a node completes
	SendNodeCompleted(flowID string, executionID string, nodeID string, result interface{}) error

	// RegisterWebhook adds a webhook URL for a flow or node
	RegisterWebhook(flowID string, nodeID string, url string) error

	// UnregisterWebhook removes a webhook URL
	UnregisterWebhook(flowID string, nodeID string, url string) error

	// ListWebhooks returns all webhook URLs for a flow or node
	ListWebhooks(flowID string, nodeID string) ([]string, error)
}

// WebhookConfig contains configuration for a webhook
type WebhookConfig struct {
	// URL to send the webhook to
	URL string `json:"url"`

	// Headers to include in the request
	Headers map[string]string `json:"headers,omitempty"`

	// Secret for signing the webhook payload
	Secret string `json:"secret,omitempty"`

	// RetryConfig for failed webhook deliveries
	RetryConfig RetryConfig `json:"retry_config,omitempty"`
}

// RetryConfig contains retry settings for webhook delivery
type RetryConfig struct {
	// MaxRetries is the maximum number of retry attempts
	MaxRetries int `json:"max_retries"`

	// InitialDelay is the initial delay before the first retry
	InitialDelay time.Duration `json:"initial_delay"`

	// MaxDelay is the maximum delay between retries
	MaxDelay time.Duration `json:"max_delay"`

	// BackoffFactor is the multiplier for the delay between retries
	BackoffFactor float64 `json:"backoff_factor"`
}

// WebhookEvent represents an event that triggers a webhook
type WebhookEvent struct {
	// Type of the event
	Type string `json:"type"` // "flow.completed", "node.completed", etc.

	// Timestamp of the event
	Timestamp time.Time `json:"timestamp"`

	// FlowID is the ID of the flow
	FlowID string `json:"flow_id"`

	// ExecutionID is the ID of the execution
	ExecutionID string `json:"execution_id"`

	// NodeID is the ID of the node (if applicable)
	NodeID string `json:"node_id,omitempty"`

	// Data contains event-specific information
	Data map[string]interface{} `json:"data,omitempty"`
}
