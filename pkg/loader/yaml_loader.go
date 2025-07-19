package loader

import (
	"fmt"

	"github.com/tcmartin/flowlib"
	"gopkg.in/yaml.v2"
)

// DefaultYAMLLoader implements the YAMLLoader interface
type DefaultYAMLLoader struct{}

// NewYAMLLoader creates a new YAML loader
func NewYAMLLoader() YAMLLoader {
	return &DefaultYAMLLoader{}
}

// Parse converts a YAML string into a Flowlib graph
func (l *DefaultYAMLLoader) Parse(yamlContent string) (*flowlib.Flow, error) {
	// For now, we'll just validate the YAML and return a stub flow
	// This is a placeholder implementation
	var flowDef FlowDefinition
	if err := yaml.Unmarshal([]byte(yamlContent), &flowDef); err != nil {
		return nil, fmt.Errorf("failed to parse YAML: %w", err)
	}

	// Create a basic flowlib.Flow
	// This is a stub implementation - in a real implementation,
	// we would convert the YAML definition to a proper flowlib graph
	flow := &flowlib.Flow{
		// Add basic flow properties here
	}

	return flow, nil
}

// Validate checks if a YAML string conforms to the schema
func (l *DefaultYAMLLoader) Validate(yamlContent string) error {
	// Basic YAML validation
	var flowDef FlowDefinition
	if err := yaml.Unmarshal([]byte(yamlContent), &flowDef); err != nil {
		return fmt.Errorf("invalid YAML: %w", err)
	}

	// Basic validation checks
	if flowDef.Metadata.Name == "" {
		return fmt.Errorf("flow name is required")
	}

	if len(flowDef.Nodes) == 0 {
		return fmt.Errorf("flow must have at least one node")
	}

	return nil
}
