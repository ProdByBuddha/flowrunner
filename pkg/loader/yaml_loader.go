package loader

import (
	"fmt"

	"github.com/tcmartin/flowlib"
	"gopkg.in/yaml.v2"
)

// DefaultYAMLLoader implements the YAMLLoader interface
type DefaultYAMLLoader struct {
	nodeFactories map[string]NodeFactory
	evaluator     ExpressionEvaluator
	scriptEngine  ScriptEngine
}

// NewYAMLLoader creates a new YAML loader
func NewYAMLLoader(nodeFactories map[string]NodeFactory, evaluator ExpressionEvaluator, scriptEngine ScriptEngine) YAMLLoader {
	return &DefaultYAMLLoader{
		nodeFactories: nodeFactories,
		evaluator:     evaluator,
		scriptEngine:  scriptEngine,
	}
}

// Parse converts a YAML string into a Flowlib graph
func (l *DefaultYAMLLoader) Parse(yamlContent string) (*flowlib.Flow, error) {
	// First validate the YAML
	if err := l.Validate(yamlContent); err != nil {
		return nil, err
	}

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

	// Validate node types exist in factories
	for nodeName, nodeDef := range flowDef.Nodes {
		if _, exists := l.nodeFactories[nodeDef.Type]; !exists {
			return fmt.Errorf("unknown node type '%s' in node '%s'", nodeDef.Type, nodeName)
		}
	}

	// Validate node references
	for nodeName, nodeDef := range flowDef.Nodes {
		for action, nextNode := range nodeDef.Next {
			if _, exists := flowDef.Nodes[nextNode]; !exists {
				return fmt.Errorf("node '%s' references non-existent node '%s' for action '%s'", nodeName, nextNode, action)
			}
		}
	}

	return nil
}
