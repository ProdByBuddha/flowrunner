package scripting

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestJSExpressionEvaluator_Evaluate(t *testing.T) {
	evaluator := NewJSExpressionEvaluator()

	tests := []struct {
		name       string
		expression string
		context    map[string]any
		want       any
		wantErr    bool
	}{
		{
			name:       "Simple variable",
			expression: "${name}",
			context:    map[string]any{"name": "John"},
			want:       "John",
			wantErr:    false,
		},
		{
			name:       "Nested variable",
			expression: "${user.name}",
			context:    map[string]any{"user": map[string]any{"name": "John"}},
			want:       "John",
			wantErr:    false,
		},
		{
			name:       "Non-existent variable",
			expression: "${unknown}",
			context:    map[string]any{"name": "John"},
			want:       nil,
			wantErr:    true,
		},
		{
			name:       "Not an expression",
			expression: "Hello, world!",
			context:    map[string]any{},
			want:       "Hello, world!",
			wantErr:    false,
		},
		{
			name:       "Simple addition",
			expression: "${1 + 2}",
			context:    map[string]any{},
			want:       float64(3),
			wantErr:    false,
		},
		{
			name:       "Simple subtraction",
			expression: "${5 - 3}",
			context:    map[string]any{},
			want:       float64(2),
			wantErr:    false,
		},
		{
			name:       "Simple multiplication",
			expression: "${2 * 3}",
			context:    map[string]any{},
			want:       float64(6),
			wantErr:    false,
		},
		{
			name:       "Simple division",
			expression: "${6 / 2}",
			context:    map[string]any{},
			want:       float64(3),
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
				assert.Equal(t, tt.want, got)
			}
		})
	}
}

func TestJSExpressionEvaluator_EvaluateInObject(t *testing.T) {
	evaluator := NewJSExpressionEvaluator()

	tests := []struct {
		name    string
		obj     map[string]any
		context map[string]any
		want    map[string]any
		wantErr bool
	}{
		{
			name: "Simple object",
			obj: map[string]any{
				"name": "${user}",
				"age":  "${age}",
			},
			context: map[string]any{
				"user": "John",
				"age":  30,
			},
			want: map[string]any{
				"name": "John",
				"age":  30,
			},
			wantErr: false,
		},
		{
			name: "Nested object",
			obj: map[string]any{
				"user": map[string]any{
					"name": "${name}",
					"age":  "${age}",
				},
			},
			context: map[string]any{
				"name": "John",
				"age":  30,
			},
			want: map[string]any{
				"user": map[string]any{
					"name": "John",
					"age":  30,
				},
			},
			wantErr: false,
		},
		{
			name: "Array in object",
			obj: map[string]any{
				"users": []any{
					"${user1}",
					"${user2}",
				},
			},
			context: map[string]any{
				"user1": "John",
				"user2": "Jane",
			},
			want: map[string]any{
				"users": []any{
					"John",
					"Jane",
				},
			},
			wantErr: false,
		},
		{
			name: "Object in array",
			obj: map[string]any{
				"users": []any{
					map[string]any{
						"name": "${name1}",
					},
					map[string]any{
						"name": "${name2}",
					},
				},
			},
			context: map[string]any{
				"name1": "John",
				"name2": "Jane",
			},
			want: map[string]any{
				"users": []any{
					map[string]any{
						"name": "John",
					},
					map[string]any{
						"name": "Jane",
					},
				},
			},
			wantErr: false,
		},
		{
			name: "Expression in key",
			obj: map[string]any{
				"${key}": "value",
			},
			context: map[string]any{
				"key": "name",
			},
			want: map[string]any{
				"name": "value",
			},
			wantErr: false,
		},
		{
			name: "Missing variable",
			obj: map[string]any{
				"name": "${unknown}",
			},
			context: map[string]any{
				"user": "John",
			},
			want:    nil,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := evaluator.EvaluateInObject(tt.obj, tt.context)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.want, got)
			}
		})
	}
}
