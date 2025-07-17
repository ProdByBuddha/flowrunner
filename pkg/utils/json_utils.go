// Package utils provides utility functions for the flowrunner.
package utils

import (
	"encoding/json"
	"strings"
)

// ParseJSON parses a JSON string into an interface{}
func ParseJSON(jsonStr string, result any) error {
	// Trim any leading/trailing whitespace
	jsonStr = strings.TrimSpace(jsonStr)

	// Check if the string is wrapped in code blocks (common in LLM responses)
	if strings.HasPrefix(jsonStr, "```json") {
		// Extract the JSON content from the code block
		endIndex := strings.LastIndex(jsonStr, "```")
		if endIndex > 6 { // 6 is the length of "```json"
			jsonStr = strings.TrimSpace(jsonStr[6:endIndex])
		} else {
			// If no closing code block, just remove the opening one
			jsonStr = strings.TrimSpace(jsonStr[6:])
		}
	} else if strings.HasPrefix(jsonStr, "```") {
		// Generic code block without language specification
		endIndex := strings.LastIndex(jsonStr, "```")
		if endIndex > 3 { // 3 is the length of "```"
			jsonStr = strings.TrimSpace(jsonStr[3:endIndex])
		} else {
			// If no closing code block, just remove the opening one
			jsonStr = strings.TrimSpace(jsonStr[3:])
		}
	}

	// Parse the JSON
	return json.Unmarshal([]byte(jsonStr), result)
}
