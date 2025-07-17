package scripting

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestJSExpressionEvaluator_JavaScript(t *testing.T) {
	evaluator := NewJSExpressionEvaluator()

	tests := []struct {
		name       string
		expression string
		context    map[string]any
		want       any
		wantErr    bool
	}{
		{
			name:       "Math.random()",
			expression: "${Math.random()}",
			context:    map[string]any{},
			want:       float64(0), // We just check it's a float64, not the exact value
			wantErr:    false,
		},
		{
			name:       "Math.floor()",
			expression: "${Math.floor(3.9)}",
			context:    map[string]any{},
			want:       float64(3),
			wantErr:    false,
		},
		{
			name:       "String manipulation",
			expression: "${'hello'.toUpperCase()}",
			context:    map[string]any{},
			want:       "HELLO",
			wantErr:    false,
		},
		{
			name:       "Context variable in JS expression",
			expression: "${name.toUpperCase()}",
			context:    map[string]any{"name": "john"},
			want:       "JOHN",
			wantErr:    false,
		},
		{
			name:       "JavaScript function",
			expression: "${(function() { return 42; })()}",
			context:    map[string]any{},
			want:       float64(42), // Ensure we expect float64 for numeric results
			wantErr:    false,
		},
		{
			name:       "JavaScript object creation",
			expression: "${JSON.stringify({name: 'John', age: 30})}",
			context:    map[string]any{},
			want:       `{"age":30,"name":"John"}`, // Match the actual JSON string format
			wantErr:    false,
		},
		{
			name:       "JavaScript date manipulation",
			expression: "${new Date(2023, 0, 1).getFullYear()}",
			context:    map[string]any{},
			want:       float64(2023), // Ensure we expect float64 for numeric results
			wantErr:    false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := evaluator.Evaluate(tt.expression, tt.context)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				if tt.name == "Math.random()" {
					// For Math.random(), we just check the type, not the exact value
					assert.IsType(t, tt.want, got)
				} else {
					assert.Equal(t, tt.want, got)
				}
			}
		})
	}
}
