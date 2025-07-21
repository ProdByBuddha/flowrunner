package loader

import (
	"github.com/tcmartin/flowlib"
	"github.com/tcmartin/flowrunner/pkg/plugins"
)

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
	Nodes map[string]plugins.NodeDefinition `yaml:"nodes" json:"nodes"`
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
