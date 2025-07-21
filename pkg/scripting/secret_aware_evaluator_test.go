package scripting

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ErrSecretNotFound is returned when a secret is not found
var ErrSecretNotFound = fmt.Errorf("secret not found")

// MockSecretVault is a simple implementation of auth.SecretVault for testing
type MockSecretVault struct {
	secrets map[string]map[string]string
}

func NewMockSecretVault() *MockSecretVault {
	return &MockSecretVault{
		secrets: make(map[string]map[string]string),
	}
}

func (m *MockSecretVault) Set(accountID, key, value string) error {
	if _, ok := m.secrets[accountID]; !ok {
		m.secrets[accountID] = make(map[string]string)
	}
	m.secrets[accountID][key] = value
	return nil
}

func (m *MockSecretVault) Get(accountID, key string) (string, error) {
	if accountSecrets, ok := m.secrets[accountID]; ok {
		if value, ok := accountSecrets[key]; ok {
			return value, nil
		}
	}
	return "", ErrSecretNotFound
}

func (m *MockSecretVault) Delete(accountID, key string) error {
	if accountSecrets, ok := m.secrets[accountID]; ok {
		delete(accountSecrets, key)
	}
	return nil
}

func (m *MockSecretVault) List(accountID string) ([]string, error) {
	if accountSecrets, ok := m.secrets[accountID]; ok {
		keys := make([]string, 0, len(accountSecrets))
		for k := range accountSecrets {
			keys = append(keys, k)
		}
		return keys, nil
	}
	return []string{}, nil
}

// RotateEncryptionKey is a no-op for the mock implementation
func (m *MockSecretVault) RotateEncryptionKey(oldKey, newKey []byte) error {
	// No-op for testing
	return nil
}

func TestSecretAwareExpressionEvaluator_Evaluate(t *testing.T) {
	// Create a mock secret vault
	vault := NewMockSecretVault()

	// Add some test secrets
	err := vault.Set("test-account", "API_KEY", "secret-api-key-123")
	require.NoError(t, err)

	err = vault.Set("test-account", "DB_PASSWORD", "super-secret-password")
	require.NoError(t, err)

	// Create the evaluator with the mock vault
	evaluator := NewSecretAwareExpressionEvaluator(vault)

	tests := []struct {
		name       string
		expression string
		context    map[string]any
		want       any
		wantErr    bool
	}{
		{
			name:       "Access secret directly",
			expression: "${secrets.API_KEY}",
			context:    map[string]any{"accountID": "test-account"},
			want:       "secret-api-key-123",
			wantErr:    false,
		},
		{
			name:       "Use secret in string concatenation",
			expression: "${\"Bearer \" + secrets.API_KEY}",
			context:    map[string]any{"accountID": "test-account"},
			want:       "Bearer secret-api-key-123",
			wantErr:    false,
		},
		{
			name:       "Use secret in object",
			expression: "${JSON.stringify({auth: secrets.API_KEY})}",
			context:    map[string]any{"accountID": "test-account"},
			want:       `{"auth":"secret-api-key-123"}`,
			wantErr:    false,
		},
		{
			name:       "Access non-existent secret",
			expression: "${secrets.NON_EXISTENT}",
			context:    map[string]any{"accountID": "test-account"},
			want:       nil,
			wantErr:    false,
		},
		{
			name:       "No account ID provided",
			expression: "${secrets ? secrets.API_KEY : null}",
			context:    map[string]any{},
			want:       nil,
			wantErr:    false,
		},
		{
			name:       "Wrong account ID",
			expression: "${secrets.API_KEY}",
			context:    map[string]any{"accountID": "wrong-account"},
			want:       nil,
			wantErr:    false,
		},
		{
			name:       "Use secret with other context variables",
			expression: "${name + \" uses \" + secrets.API_KEY}",
			context:    map[string]any{"accountID": "test-account", "name": "John"},
			want:       "John uses secret-api-key-123",
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

func TestSecretAwareExpressionEvaluator_EvaluateInObject(t *testing.T) {
	// Create a mock secret vault
	vault := NewMockSecretVault()

	// Add some test secrets
	err := vault.Set("test-account", "API_KEY", "secret-api-key-123")
	require.NoError(t, err)

	err = vault.Set("test-account", "DB_PASSWORD", "super-secret-password")
	require.NoError(t, err)

	// Create the evaluator with the mock vault
	evaluator := NewSecretAwareExpressionEvaluator(vault)

	tests := []struct {
		name    string
		obj     map[string]any
		context map[string]any
		want    map[string]any
		wantErr bool
	}{
		{
			name: "Object with secrets",
			obj: map[string]any{
				"headers": map[string]any{
					"Authorization": "${\"Bearer \" + secrets.API_KEY}",
				},
				"database": map[string]any{
					"password": "${secrets.DB_PASSWORD}",
				},
			},
			context: map[string]any{"accountID": "test-account"},
			want: map[string]any{
				"headers": map[string]any{
					"Authorization": "Bearer secret-api-key-123",
				},
				"database": map[string]any{
					"password": "super-secret-password",
				},
			},
			wantErr: false,
		},
		{
			name: "Array with secrets",
			obj: map[string]any{
				"credentials": []any{
					"${secrets.API_KEY}",
					"${secrets.DB_PASSWORD}",
				},
			},
			context: map[string]any{"accountID": "test-account"},
			want: map[string]any{
				"credentials": []any{
					"secret-api-key-123",
					"super-secret-password",
				},
			},
			wantErr: false,
		},
		{
			name: "Mix of secrets and context variables",
			obj: map[string]any{
				"user":  "${username}",
				"token": "${secrets.API_KEY}",
			},
			context: map[string]any{
				"accountID": "test-account",
				"username":  "john.doe",
			},
			want: map[string]any{
				"user":  "john.doe",
				"token": "secret-api-key-123",
			},
			wantErr: false,
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
