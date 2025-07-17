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
			name:       "JavaScript function",
			expression: "${(function() { return 42; })()}",
			context:    map[string]any{},
			want:       float64(42),
			wantErr:    false,
		},
		{
			name:       "JavaScript array manipulation",
			expression: "${(function() { var arr = [1, 2, 3]; var result = []; for(var i=0; i<arr.length; i++) { result.push(arr[i] * 2); } return result.join(', '); })()}",
			context:    map[string]any{},
			want:       "2, 4, 6",
			wantErr:    false,
		},
		{
			name:       "JavaScript with context",
			expression: "${(function() { var result = []; for(var i=0; i<users.length; i++) { if(users[i].age > 30) { result.push(users[i].name); } } return result.join(', '); })()}",
			context: map[string]any{
				"users": []map[string]any{
					{"name": "Alice", "age": 25},
					{"name": "Bob", "age": 35},
					{"name": "Charlie", "age": 45},
				},
			},
			want:    "Bob, Charlie",
			wantErr: false,
		},
		{
			name:       "JavaScript Math functions",
			expression: "${Math.max(10, 5, 20)}",
			context:    map[string]any{},
			want:       float64(20),
			wantErr:    false,
		},
		{
			name:       "JavaScript string manipulation",
			expression: "${name.toUpperCase()}",
			context:    map[string]any{"name": "john"},
			want:       "JOHN",
			wantErr:    false,
		},
		{
			name:       "JavaScript conditional",
			expression: "${age >= 18 ? 'Adult' : 'Minor'}",
			context:    map[string]any{"age": 25},
			want:       "Adult",
			wantErr:    false,
		},
		{
			name:       "JavaScript object creation",
			expression: "${(function() { var obj = {}; obj.name = name; obj.age = age; return JSON.stringify(obj); })()}",
			context:    map[string]any{"name": "John", "age": 30},
			want:       `{"name":"John","age":30}`,
			wantErr:    false,
		},
		{
			name:       "JavaScript date manipulation",
			expression: "${new Date(2023, 0, 1).getFullYear()}",
			context:    map[string]any{},
			want:       float64(2023),
			wantErr:    false,
		},
		{
			name:       "JavaScript error",
			expression: "${nonExistentFunction()}",
			context:    map[string]any{},
			want:       nil,
			wantErr:    true,
		},
		{
			name:       "JavaScript syntax error",
			expression: "${function() { return 42 }",
			context:    map[string]any{},
			want:       nil,
			wantErr:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := evaluator.Evaluate(tt.expression, tt.context)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.want, got)
			}
		})
	}
}

func TestJSExpressionEvaluator_MathRandom(t *testing.T) {
	evaluator := NewJSExpressionEvaluator()

	// Test that Math.random() returns a value between 0 and 1
	result, err := evaluator.Evaluate("${Math.random()}", map[string]any{})
	assert.NoError(t, err)

	// Check that the result is a float64
	value, ok := result.(float64)
	assert.True(t, ok, "Math.random() should return a float64")

	// Check that the value is between 0 and 1
	assert.GreaterOrEqual(t, value, float64(0))
	assert.Less(t, value, float64(1))
}
