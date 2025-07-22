package utils

import (
	"testing"
)

func TestProcessTemplate(t *testing.T) {
	// Test data
	variables := map[string]interface{}{
		"result": map[string]interface{}{
			"tool_calls": []interface{}{
				map[string]interface{}{
					"function": map[string]interface{}{
						"arguments": `{"query":"test query"}`,
					},
				},
			},
		},
	}

	// Test cases
	tests := []struct {
		name     string
		template string
		expected string
	}{
		{
			name:     "Simple variable",
			template: "Hello {{result.tool_calls[0].function.arguments}}",
			expected: `Hello {"query":"test query"}`,
		},
		{
			name:     "With fromjson function",
			template: "Query: {{result.tool_calls[0].function.arguments | fromjson | .query}}",
			expected: "Query: test query",
		},
	}

	// Run tests
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := ProcessTemplate(tt.template, variables)
			if err != nil {
				t.Errorf("ProcessTemplate() error = %v", err)
				return
			}
			if result != tt.expected {
				t.Errorf("ProcessTemplate() = %v, want %v", result, tt.expected)
			}
		})
	}
}
