package plugins

// NodeDefinition represents a node in the flow
type NodeDefinition struct {
	// Type of the node
	Type string `yaml:"type" json:"type"`

	// Parameters for the node
	Params map[string]interface{} `yaml:"params" json:"params"`

	// Next nodes to execute based on action
	Next map[string]string `yaml:"next" json:"next"`

	// Batch processing configuration
	Batch BatchDefinition `yaml:"batch" json:"batch,omitempty"`

	// Retry configuration
	Retry RetryDefinition `yaml:"retry" json:"retry,omitempty"`

	// JavaScript hooks for the node
	Hooks NodeHooks `yaml:"hooks" json:"hooks,omitempty"`
}

// BatchDefinition defines the batch processing strategy for a node.
type BatchDefinition struct {
	Strategy    string `yaml:"strategy" json:"strategy,omitempty"`
	MaxParallel int    `yaml:"max_parallel" json:"max_parallel,omitempty"`
}

// RetryDefinition defines the retry strategy for a node.
type RetryDefinition struct {
	MaxRetries int    `yaml:"max_retries" json:"max_retries,omitempty"`
	Wait       string `yaml:"wait" json:"wait,omitempty"`
}

// NodeHooks contains JavaScript code to execute at different stages
type NodeHooks struct {
	// Prep hook runs before node execution
	Prep string `yaml:"prep" json:"prep,omitempty"`

	// Exec hook runs during node execution
	Exec string `yaml:"exec" json:"exec,omitempty"`

	// Post hook runs after node execution
	Post string `yaml:"post" json:"post,omitempty"`
}
