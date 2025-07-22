package utils

import (
	"encoding/json"
	"fmt"
	"regexp"
	"strings"
)

// ProcessTemplate processes a template string with variables
// Supports simple variable substitution and basic functions like fromjson
func ProcessTemplate(template string, variables map[string]interface{}) (string, error) {
	// Debug log
	fmt.Printf("ProcessTemplate: Processing template '%s' with variables: %+v\n", template, variables)

	// Match {{variable}} or {{variable | function | .property}}
	re := regexp.MustCompile(`{{([^}]+)}}`)

	result := re.ReplaceAllStringFunc(template, func(match string) string {
		// Extract the expression inside {{}}
		expr := match[2 : len(match)-2]
		expr = strings.TrimSpace(expr)

		// Split by pipe to handle functions
		parts := strings.Split(expr, "|")

		// Get the initial variable
		varPath := strings.TrimSpace(parts[0])
		fmt.Printf("ProcessTemplate: Getting nested value for path '%s'\n", varPath)
		value := getNestedValue(variables, varPath)

		// Process functions in order
		for i := 1; i < len(parts); i++ {
			funcName := strings.TrimSpace(parts[i])

			// Handle fromjson function
			if funcName == "fromjson" {
				if strValue, ok := value.(string); ok {
					var jsonValue interface{}
					if err := json.Unmarshal([]byte(strValue), &jsonValue); err != nil {
						return fmt.Sprintf("ERROR: %v", err)
					}
					value = jsonValue
				} else {
					return "ERROR: fromjson requires string input"
				}
			} else if strings.HasPrefix(funcName, ".") {
				// Handle property access (.property)
				propName := funcName[1:]
				if mapValue, ok := value.(map[string]interface{}); ok {
					value = mapValue[propName]
				} else {
					return fmt.Sprintf("ERROR: cannot access property %s", propName)
				}
			}
		}

		// Convert final value to string
		return fmt.Sprintf("%v", value)
	})

	return result, nil
}

// getNestedValue retrieves a nested value from a map using dot notation
// e.g. "input.result.tool_calls[0].function.arguments"
func getNestedValue(data map[string]interface{}, path string) interface{} {
	parts := strings.Split(path, ".")
	fmt.Printf("getNestedValue: Path parts: %v\n", parts)

	var current interface{} = data

	for i, part := range parts {
		fmt.Printf("getNestedValue: Processing part %d: '%s'\n", i, part)

		// Handle array indexing with [index]
		indexMatch := regexp.MustCompile(`^(.+)\[(\d+)\]$`).FindStringSubmatch(part)

		if indexMatch != nil {
			// We have an array index
			arrayName := indexMatch[1]
			indexStr := indexMatch[2]

			// First get the array
			if currentMap, ok := current.(map[string]interface{}); ok {
				current = currentMap[arrayName]
			} else {
				return nil
			}

			// Then access the index
			if array, ok := current.([]interface{}); ok {
				index := 0
				fmt.Sscanf(indexStr, "%d", &index)
				if index >= 0 && index < len(array) {
					current = array[index]
				} else {
					return nil
				}
			} else {
				return nil
			}
		} else {
			// Regular property access
			if currentMap, ok := current.(map[string]interface{}); ok {
				current = currentMap[part]
			} else {
				return nil
			}
		}
	}

	return current
}
