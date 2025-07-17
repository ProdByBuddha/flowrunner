package loader

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/tcmartin/flowlib"
	"github.com/tcmartin/flowrunner/pkg/scripting"
	"github.com/xeipuuv/gojsonschema"
	"gopkg.in/yaml.v3"
)

// YAMLLoaderImpl implements the YAMLLoader interface
type YAMLLoaderImpl struct {
	flowSchema      *gojsonschema.Schema
	nodeTypeSchemas map[string]*gojsonschema.Schema
}

// NewYAMLLoader creates a new YAML loader
func NewYAMLLoader() (*YAMLLoaderImpl, error) {
	// Load the flow schema
	flowSchemaLoader := gojsonschema.NewStringLoader(FlowSchema)
	flowSchema, err := gojsonschema.NewSchema(flowSchemaLoader)
	if err != nil {
		return nil, fmt.Errorf("failed to load flow schema: %w", err)
	}

	// Load node type schemas
	nodeTypeSchemas := make(map[string]*gojsonschema.Schema)
	for nodeType, schemaStr := range NodeTypeSchemas {
		schemaLoader := gojsonschema.NewStringLoader(schemaStr)
		schema, err := gojsonschema.NewSchema(schemaLoader)
		if err != nil {
			return nil, fmt.Errorf("failed to load schema for node type '%s': %w", nodeType, err)
		}
		nodeTypeSchemas[nodeType] = schema
	}

	return &YAMLLoaderImpl{
		flowSchema:      flowSchema,
		nodeTypeSchemas: nodeTypeSchemas,
	}, nil
}

// ValidationError represents a validation error
type ValidationError struct {
	// Path to the error in the document
	Path string `json:"path"`

	// Message describing the error
	Message string `json:"message"`

	// Type of error
	Type string `json:"type"`

	// Value that caused the error
	Value interface{} `json:"value,omitempty"`
}

// ValidationResult represents the result of validating a flow definition
type ValidationResult struct {
	// Valid indicates whether the validation passed
	Valid bool `json:"valid"`

	// Errors contains validation errors
	Errors []ValidationError `json:"errors,omitempty"`
}

// Validate checks if a YAML string conforms to the schema
func (l *YAMLLoaderImpl) Validate(yamlContent string) error {
	result, err := l.ValidateWithDetails(yamlContent)
	if err != nil {
		return err
	}

	if !result.Valid {
		var errors []string
		for _, err := range result.Errors {
			errors = append(errors, fmt.Sprintf("%s: %s", err.Path, err.Message))
		}
		return fmt.Errorf("invalid flow definition: %s", strings.Join(errors, "; "))
	}

	return nil
}

// ValidateWithDetails checks if a YAML string conforms to the schema and returns detailed errors
func (l *YAMLLoaderImpl) ValidateWithDetails(yamlContent string) (*ValidationResult, error) {
	// Parse YAML to JSON
	var data interface{}
	if err := yaml.Unmarshal([]byte(yamlContent), &data); err != nil {
		return nil, fmt.Errorf("invalid YAML: %w", err)
	}

	// Convert to JSON for schema validation
	jsonData, err := json.Marshal(data)
	if err != nil {
		return nil, fmt.Errorf("failed to convert YAML to JSON: %w", err)
	}

	// Validate against the flow schema
	documentLoader := gojsonschema.NewBytesLoader(jsonData)
	result, err := l.flowSchema.Validate(documentLoader)
	if err != nil {
		return nil, fmt.Errorf("schema validation error: %w", err)
	}

	validationResult := &ValidationResult{
		Valid: result.Valid(),
	}

	if !result.Valid() {
		for _, err := range result.Errors() {
			validationResult.Errors = append(validationResult.Errors, ValidationError{
				Path:    err.Field(),
				Message: err.Description(),
				Type:    err.Type(),
				Value:   err.Value(),
			})
		}
		return validationResult, nil
	}

	// Parse the flow definition
	var flowDef FlowDefinition
	if err := yaml.Unmarshal([]byte(yamlContent), &flowDef); err != nil {
		return nil, fmt.Errorf("failed to parse flow definition: %w", err)
	}

	// Validate each node against its type-specific schema
	for nodeName, nodeDef := range flowDef.Nodes {
		schema, ok := l.nodeTypeSchemas[nodeDef.Type]
		if !ok {
			// Skip validation for unknown node types
			continue
		}

		// Convert node params to JSON
		paramsJSON, err := json.Marshal(nodeDef.Params)
		if err != nil {
			return nil, fmt.Errorf("failed to convert params for node '%s' to JSON: %w", nodeName, err)
		}

		// Validate params against the node type schema
		paramsLoader := gojsonschema.NewBytesLoader(paramsJSON)
		nodeResult, err := schema.Validate(paramsLoader)
		if err != nil {
			return nil, fmt.Errorf("schema validation error for node '%s': %w", nodeName, err)
		}

		if !nodeResult.Valid() {
			validationResult.Valid = false
			for _, err := range nodeResult.Errors() {
				validationResult.Errors = append(validationResult.Errors, ValidationError{
					Path:    fmt.Sprintf("nodes.%s.params.%s", nodeName, err.Field()),
					Message: err.Description(),
					Type:    err.Type(),
					Value:   err.Value(),
				})
			}
		}
	}

	// Validate node connections
	for nodeName, nodeDef := range flowDef.Nodes {
		for action, nextNodeName := range nodeDef.Next {
			if _, ok := flowDef.Nodes[nextNodeName]; !ok {
				validationResult.Valid = false
				validationResult.Errors = append(validationResult.Errors, ValidationError{
					Path:    fmt.Sprintf("nodes.%s.next.%s", nodeName, action),
					Message: fmt.Sprintf("references non-existent node '%s'", nextNodeName),
					Type:    "invalid_reference",
					Value:   nextNodeName,
				})
			}
		}
	}

	// Check for cycles in the graph
	if cycles := l.detectCycles(flowDef); len(cycles) > 0 {
		validationResult.Valid = false
		for _, cycle := range cycles {
			validationResult.Errors = append(validationResult.Errors, ValidationError{
				Path:    "nodes",
				Message: fmt.Sprintf("cycle detected in flow: %s", strings.Join(cycle, " -> ")),
				Type:    "cycle_detected",
				Value:   cycle,
			})
		}
	}

	return validationResult, nil
}

// detectCycles finds cycles in the flow graph
func (l *YAMLLoaderImpl) detectCycles(flowDef FlowDefinition) [][]string {
	var cycles [][]string
	visited := make(map[string]bool)
	path := make(map[string]bool)

	var dfs func(node string, currentPath []string) bool
	dfs = func(node string, currentPath []string) bool {
		if path[node] {
			// Found a cycle
			cycleStart := -1
			for i, n := range currentPath {
				if n == node {
					cycleStart = i
					break
				}
			}
			if cycleStart >= 0 {
				cycle := append(currentPath[cycleStart:], node)
				cycles = append(cycles, cycle)
			}
			return true
		}

		if visited[node] {
			return false
		}

		visited[node] = true
		path[node] = true
		currentPath = append(currentPath, node)

		nodeDef, ok := flowDef.Nodes[node]
		if ok {
			for _, nextNode := range nodeDef.Next {
				if dfs(nextNode, currentPath) {
					// Continue searching for more cycles
				}
			}
		}

		path[node] = false
		return false
	}

	// Start DFS from each node
	for nodeName := range flowDef.Nodes {
		if !visited[nodeName] {
			dfs(nodeName, []string{})
		}
	}

	return cycles
}

// Parse converts a YAML string into a Flowlib graph
func (l *YAMLLoaderImpl) Parse(yamlContent string) (*flowlib.Flow, error) {
	return l.ParseWithContext(yamlContent, nil)
}

// ParseWithContext converts a YAML string into a Flowlib graph with the given context
func (l *YAMLLoaderImpl) ParseWithContext(yamlContent string, context map[string]interface{}) (*flowlib.Flow, error) {
	// Validate the YAML first
	if err := l.Validate(yamlContent); err != nil {
		return nil, err
	}

	// Parse the flow definition
	var flowDef FlowDefinition
	if err := yaml.Unmarshal([]byte(yamlContent), &flowDef); err != nil {
		return nil, fmt.Errorf("failed to parse flow definition: %w", err)
	}

	// Create nodes
	nodes := make(map[string]flowlib.Node)
	for nodeName, nodeDef := range flowDef.Nodes {
		node, err := l.createNode(nodeName, nodeDef, context)
		if err != nil {
			return nil, fmt.Errorf("failed to create node '%s': %w", nodeName, err)
		}
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
		return nil, fmt.Errorf("no nodes defined in flow")
	}

	return flowlib.NewFlow(startNode), nil
}

// createNode creates a Flowlib node from a node definition
func (l *YAMLLoaderImpl) createNode(name string, def NodeDefinition, context map[string]interface{}) (flowlib.Node, error) {
	// This is a simplified implementation - in a real implementation, we would
	// use a factory pattern to create nodes based on their type
	node := flowlib.NewNode(1, 0)

	// Apply context to parameters if provided
	params := def.Params
	if context != nil {
		// Create an expression evaluator
		evaluator := scripting.NewSimpleExpressionEvaluator()

		// Evaluate expressions in parameters
		evaluatedParams, err := evaluator.EvaluateInObject(params, context)
		if err != nil {
			return nil, fmt.Errorf("failed to evaluate expressions in node parameters: %w", err)
		}

		params = evaluatedParams
	}

	node.SetParams(params)

	// In a real implementation, we would set up the node's exec function
	// based on its type and hooks
	node.execFn = func(input interface{}) (interface{}, error) {
		// If there's an exec hook, we would execute it here
		if def.Hooks.Exec != "" {
			// In a real implementation, this would use the JavaScript engine
			return input, nil
		}

		return input, nil
	}

	// Set up prep function if there's a prep hook
	if def.Hooks.Prep != "" {
		node.baseNode.prepFn = func(shared interface{}) (interface{}, error) {
			// In a real implementation, this would use the JavaScript engine
			return nil, nil
		}
	}

	// Set up post function if there's a post hook
	if def.Hooks.Post != "" {
		node.baseNode.postFn = func(shared, p, e interface{}) (flowlib.Action, error) {
			// In a real implementation, this would use the JavaScript engine
			return flowlib.DefaultAction, nil
		}
	}

	return node, nil
}
