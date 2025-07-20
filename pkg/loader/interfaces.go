// Package loader provides functionality for loading and parsing YAML flow definitions.
package loader

import (
	"github.com/tcmartin/flowlib"
)

// NodeFactory is a function type for creating nodes
type NodeFactory func(params map[string]interface{}) (flowlib.Node, error)

// ExpressionEvaluator evaluates expressions in context
type ExpressionEvaluator interface {
	Evaluate(expression string, context map[string]interface{}) (interface{}, error)
	EvaluateInObject(obj map[string]interface{}, context map[string]interface{}) (map[string]interface{}, error)
}

// ScriptEngine executes scripts
type ScriptEngine interface {
	Execute(script string, context map[string]interface{}) (interface{}, error)
}

// YAMLLoader parses YAML flow definitions into Flowlib graph structures.
type YAMLLoader interface {
	// Parse converts a YAML string into a Flowlib graph
	Parse(yamlContent string) (*flowlib.Flow, error)

	// Validate checks if a YAML string conforms to the schema
	Validate(yamlContent string) error
}

// FlowDefinition represents a parsed flow definition from YAML
type FlowDefinition struct {
	// Metadata about the flow
	Metadata FlowMetadata `yaml:"metadata" json:"metadata"`

	// Nodes in the flow
	Nodes map[string]NodeDefinition `yaml:"nodes" json:"nodes"`
}

// FlowMetadata contains information about the flow
type FlowMetadata struct {
	// Name of the flow
	Name string `yaml:"name" json:"name"`

	// Description of the flow
	Description string `yaml:"description" json:"description"`

	// Version of the flow
	Version string `yaml:"version" json:"version"`
}

// NodeDefinition represents a node in the flow
type NodeDefinition struct {
	// Type of the node
	Type string `yaml:"type" json:"type"`

	// Parameters for the node
	Params map[string]interface{} `yaml:"params" json:"params"`

	// Next nodes to execute based on action
	Next map[string]string `yaml:"next" json:"next"`

	// JavaScript hooks for the node
	Hooks NodeHooks `yaml:"hooks" json:"hooks,omitempty"`
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
