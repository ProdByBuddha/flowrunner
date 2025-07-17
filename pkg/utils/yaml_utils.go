// Package utils provides utility functions for the flowrunner.
package utils

import (
	"strings"

	"gopkg.in/yaml.v3"
)

// ParseYAML parses a YAML string into an interface{}
func ParseYAML(yamlStr string, result any) error {
	// Trim any leading/trailing whitespace
	yamlStr = strings.TrimSpace(yamlStr)

	// Check if the string is wrapped in code blocks (common in LLM responses)
	if strings.HasPrefix(yamlStr, "```yaml") || strings.HasPrefix(yamlStr, "```yml") {
		// Extract the YAML content from the code block
		endIndex := strings.LastIndex(yamlStr, "```")
		if endIndex > 7 { // 7 is the length of "```yaml" or "```yml"
			yamlStr = strings.TrimSpace(yamlStr[strings.Index(yamlStr, "\n"):endIndex])
		} else {
			// If no closing code block, just remove the opening one
			yamlStr = strings.TrimSpace(yamlStr[strings.Index(yamlStr, "\n"):])
		}
	} else if strings.HasPrefix(yamlStr, "```") {
		// Generic code block without language specification
		endIndex := strings.LastIndex(yamlStr, "```")
		if endIndex > 3 { // 3 is the length of "```"
			yamlStr = strings.TrimSpace(yamlStr[3:endIndex])
		} else {
			// If no closing code block, just remove the opening one
			yamlStr = strings.TrimSpace(yamlStr[3:])
		}
	}

	// Parse the YAML
	return yaml.Unmarshal([]byte(yamlStr), result)
}
