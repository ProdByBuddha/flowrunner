// Package loader provides functionality for loading and parsing YAML flow definitions.
package loader

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/tcmartin/flowlib"
	"github.com/xeipuuv/gojsonschema"
	"gopkg.in/yaml.v3"
)

// YAMLLoaderImpl implements the YAMLLoader interface
type YAMLLoaderImpl struct {
	// NodeFactories maps node types to factory functions
	NodeFactories map[string]NodeFactory

	// ExpressionEvaluator for evaluating expressions in YAML
	ExpressionEvaluator ExpressionEvaluator

	// ScriptEngine for executing JavaScript hooks
	ScriptEngine ScriptEngine
}

// NodeFactory creates a flowlib.Node from parameters
type NodeFactory func(params map[string]interface{}) (flowlib.Node, error)

// ExpressionEvaluator evaluates expressions in YAML
type ExpressionEvaluator interface {
	Evaluate(expression string, context map[string]interface{}) (interface{}, error)
	EvaluateInObject(obj map[string]interface{}, context map[string]interface{}) (map[string]interface{}, error)
}

// ScriptEngine executes JavaScript code
type ScriptEngine interface {
	Execute(script string, context map[string]interface{}) (interface{}, error)
	ExecuteWithTimeout(ctx context.Context, script string, context map[string]interface{}, timeout time.Duration) (interface{}, error)
}

// NewYAMLLoader creates a new YAML loader with the given node factories
func NewYAMLLoader(nodeFactories map[string]NodeFactory, evaluator ExpressionEvaluator, engine ScriptEngine) YAMLLoader {
	return &YAMLLoaderImpl{
		NodeFactories:       nodeFactories,
		ExpressionEvaluator: evaluator,
		ScriptEngine:        engine,
	}
}

// Parse converts a YAML string into a Flowlib graph
func (l *YAMLLoaderImpl) Parse(yamlContent string) (*flowlib.Flow, error) {
	// Parse the YAML into a FlowDefinition
	var flowDef FlowDefinition
	if err := yaml.Unmarshal([]byte(yamlContent), &flowDef); err != nil {
		return nil, fmt.Errorf("failed to parse YAML: %w", err)
	}

	// Validate the flow definition
	if err := l.Validate(yamlContent); err != nil {
		return nil, err
	}

	// Create nodes
	nodes := make(map[string]flowlib.Node)
	for nodeName, nodeDef := range flowDef.Nodes {
		// Get the factory for this node type
		factory, ok := l.NodeFactories[nodeDef.Type]
		if !ok {
			return nil, fmt.Errorf("unknown node type: %s", nodeDef.Type)
		}

		// Create the node
		node, err := factory(nodeDef.Params)
		if err != nil {
			return nil, fmt.Errorf("failed to create node '%s': %w", nodeName, err)
		}

		// Store the node
		nodes[nodeName] = node
	}

	// Connect nodes
	for nodeName, nodeDef := range flowDef.Nodes {
		node := nodes[nodeName]
		for action, nextNodeName := range nodeDef.Next {
			nextNode, ok := nodes[nextNodeName]
			if !ok {
				return nil, fmt.Errorf("node '%s' references non-existent node '%s'", nodeName, nextNodeName)
			}
			node.Next(action, nextNode)
		}
	}

	// Find the start node (first node in the definition)
	var startNode flowlib.Node
	for _, node := range nodes {
		startNode = node
		break
	}

	if startNode == nil {
		return nil, fmt.Errorf("no nodes found in flow definition")
	}

	// Create the flow
	flow := flowlib.NewFlow(startNode)

	return flow, nil
}

// Validate checks if a YAML string conforms to the schema
func (l *YAMLLoaderImpl) Validate(yamlContent string) error {
	// Parse the YAML into JSON for schema validation
	var flowDef map[string]interface{}
	if err := yaml.Unmarshal([]byte(yamlContent), &flowDef); err != nil {
		return fmt.Errorf("failed to parse YAML: %w", err)
	}

	// Convert to JSON
	jsonData, err := json.Marshal(flowDef)
	if err != nil {
		return fmt.Errorf("failed to convert to JSON: %w", err)
	}

	// Load the schema
	schemaLoader := gojsonschema.NewStringLoader(FlowSchema)
	documentLoader := gojsonschema.NewBytesLoader(jsonData)

	// Validate
	result, err := gojsonschema.Validate(schemaLoader, documentLoader)
	if err != nil {
		return fmt.Errorf("schema validation error: %w", err)
	}

	if !result.Valid() {
		// Collect validation errors
		var errMsg string
		for i, err := range result.Errors() {
			if i > 0 {
				errMsg += "; "
			}
			errMsg += err.String()
		}
		return fmt.Errorf("invalid flow definition: %s", errMsg)
	}

	// Additional validation
	var typedFlowDef FlowDefinition
	if err := yaml.Unmarshal([]byte(yamlContent), &typedFlowDef); err != nil {
		return fmt.Errorf("failed to parse YAML: %w", err)
	}

	// Check that all referenced nodes exist
	for nodeName, nodeDef := range typedFlowDef.Nodes {
		for _, nextNode := range nodeDef.Next {
			if nextNode != "" {
				if _, exists := typedFlowDef.Nodes[nextNode]; !exists {
					return fmt.Errorf("node '%s' references non-existent node '%s'", nodeName, nextNode)
				}
			}
		}

		// Check that the node type is registered
		if l.NodeFactories != nil {
			if _, ok := l.NodeFactories[nodeDef.Type]; !ok {
				return fmt.Errorf("unknown node type: %s", nodeDef.Type)
			}
		}
	}

	return nil
}
