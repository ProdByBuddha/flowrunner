package api

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/tcmartin/flowrunner/pkg/config"
	"github.com/tcmartin/flowrunner/pkg/registry"
	"github.com/tcmartin/flowrunner/pkg/services"
	"github.com/tcmartin/flowrunner/pkg/storage"
)

func TestSecretHandlers(t *testing.T) {
	// Create test server with secret vault
	server := createTestServerWithSecrets(t)

	// Create test account and get auth header
	accountID, authHeader := createTestAccountAndAuth(t, server)

	t.Run("list_empty_secrets", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, fmt.Sprintf("/api/v1/accounts/%s/secrets", accountID), nil)
		req.Header.Set("Authorization", authHeader)

		w := httptest.NewRecorder()
		server.router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)

		var response SecretListResponse
		err := json.Unmarshal(w.Body.Bytes(), &response)
		require.NoError(t, err)

		assert.Empty(t, response.Secrets)
		assert.Equal(t, 0, response.Total)
	})

	t.Run("create_secret", func(t *testing.T) {
		secretReq := SecretRequest{Value: "test-secret-value"}
		body, _ := json.Marshal(secretReq)

		req := httptest.NewRequest(http.MethodPost, fmt.Sprintf("/api/v1/accounts/%s/secrets/api_key", accountID), bytes.NewBuffer(body))
		req.Header.Set("Authorization", authHeader)
		req.Header.Set("Content-Type", "application/json")

		w := httptest.NewRecorder()
		server.router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusCreated, w.Code)

		var response SecretResponse
		err := json.Unmarshal(w.Body.Bytes(), &response)
		require.NoError(t, err)

		assert.Equal(t, "api_key", response.Key)
		assert.NotZero(t, response.CreatedAt)
		assert.NotZero(t, response.UpdatedAt)
	})

	t.Run("get_secret", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, fmt.Sprintf("/api/v1/accounts/%s/secrets/api_key", accountID), nil)
		req.Header.Set("Authorization", authHeader)

		w := httptest.NewRecorder()
		server.router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)

		var response map[string]interface{}
		err := json.Unmarshal(w.Body.Bytes(), &response)
		require.NoError(t, err)

		assert.Equal(t, "api_key", response["key"])
		assert.Equal(t, "test-secret-value", response["value"])
	})

	t.Run("list_secrets_with_data", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, fmt.Sprintf("/api/v1/accounts/%s/secrets", accountID), nil)
		req.Header.Set("Authorization", authHeader)

		w := httptest.NewRecorder()
		server.router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)

		var response SecretListResponse
		err := json.Unmarshal(w.Body.Bytes(), &response)
		require.NoError(t, err)

		assert.Len(t, response.Secrets, 1)
		assert.Equal(t, 1, response.Total)
		assert.Equal(t, "api_key", response.Secrets[0].Key)
	})

	t.Run("get_secret_keys", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, fmt.Sprintf("/api/v1/accounts/%s/secrets/keys", accountID), nil)
		req.Header.Set("Authorization", authHeader)

		w := httptest.NewRecorder()
		server.router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)

		var response map[string]interface{}
		err := json.Unmarshal(w.Body.Bytes(), &response)
		require.NoError(t, err)

		keys := response["keys"].([]interface{})
		assert.Len(t, keys, 1)
		assert.Equal(t, "api_key", keys[0])
		assert.Equal(t, float64(1), response["total"])
	})

	t.Run("update_secret", func(t *testing.T) {
		secretReq := SecretRequest{Value: "updated-secret-value"}
		body, _ := json.Marshal(secretReq)

		req := httptest.NewRequest(http.MethodPut, fmt.Sprintf("/api/v1/accounts/%s/secrets/api_key", accountID), bytes.NewBuffer(body))
		req.Header.Set("Authorization", authHeader)
		req.Header.Set("Content-Type", "application/json")

		w := httptest.NewRecorder()
		server.router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)

		// Verify the value was updated
		req = httptest.NewRequest(http.MethodGet, fmt.Sprintf("/api/v1/accounts/%s/secrets/api_key", accountID), nil)
		req.Header.Set("Authorization", authHeader)

		w = httptest.NewRecorder()
		server.router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)

		var response map[string]interface{}
		err := json.Unmarshal(w.Body.Bytes(), &response)
		require.NoError(t, err)

		assert.Equal(t, "updated-secret-value", response["value"])
	})

	t.Run("delete_secret", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodDelete, fmt.Sprintf("/api/v1/accounts/%s/secrets/api_key", accountID), nil)
		req.Header.Set("Authorization", authHeader)

		w := httptest.NewRecorder()
		server.router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusNoContent, w.Code)

		// Verify the secret was deleted
		req = httptest.NewRequest(http.MethodGet, fmt.Sprintf("/api/v1/accounts/%s/secrets/api_key", accountID), nil)
		req.Header.Set("Authorization", authHeader)

		w = httptest.NewRecorder()
		server.router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusNotFound, w.Code)
	})
}

func TestSecretHandlers_Authorization(t *testing.T) {
	server := createTestServerWithSecrets(t)

	// Create two test accounts
	account1ID, auth1Header := createTestAccountAndAuth(t, server)
	_, auth2Header := createTestAccountAndAuth(t, server)

	// Create a secret for account1
	secretReq := SecretRequest{Value: "account1-secret"}
	body, _ := json.Marshal(secretReq)

	req := httptest.NewRequest(http.MethodPost, fmt.Sprintf("/api/v1/accounts/%s/secrets/test_key", account1ID), bytes.NewBuffer(body))
	req.Header.Set("Authorization", auth1Header)
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()
	server.router.ServeHTTP(w, req)
	assert.Equal(t, http.StatusCreated, w.Code)

	t.Run("account2_cannot_access_account1_secrets", func(t *testing.T) {
		// Try to access account1's secrets with account2's auth
		req := httptest.NewRequest(http.MethodGet, fmt.Sprintf("/api/v1/accounts/%s/secrets/test_key", account1ID), nil)
		req.Header.Set("Authorization", auth2Header)

		w := httptest.NewRecorder()
		server.router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusForbidden, w.Code)
	})

	t.Run("account2_cannot_list_account1_secrets", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, fmt.Sprintf("/api/v1/accounts/%s/secrets", account1ID), nil)
		req.Header.Set("Authorization", auth2Header)

		w := httptest.NewRecorder()
		server.router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusForbidden, w.Code)
	})

	t.Run("account2_cannot_create_secret_for_account1", func(t *testing.T) {
		secretReq := SecretRequest{Value: "malicious-secret"}
		body, _ := json.Marshal(secretReq)

		req := httptest.NewRequest(http.MethodPost, fmt.Sprintf("/api/v1/accounts/%s/secrets/malicious_key", account1ID), bytes.NewBuffer(body))
		req.Header.Set("Authorization", auth2Header)
		req.Header.Set("Content-Type", "application/json")

		w := httptest.NewRecorder()
		server.router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusForbidden, w.Code)
	})

	t.Run("unauthenticated_request", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, fmt.Sprintf("/api/v1/accounts/%s/secrets", account1ID), nil)
		// No authorization header

		w := httptest.NewRecorder()
		server.router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusUnauthorized, w.Code)
	})
}

func TestSecretHandlers_ErrorCases(t *testing.T) {
	server := createTestServerWithSecrets(t)
	accountID, authHeader := createTestAccountAndAuth(t, server)

	t.Run("invalid_json", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPost, fmt.Sprintf("/api/v1/accounts/%s/secrets/test_key", accountID), bytes.NewBufferString("invalid json"))
		req.Header.Set("Authorization", authHeader)
		req.Header.Set("Content-Type", "application/json")

		w := httptest.NewRecorder()
		server.router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("empty_key", func(t *testing.T) {
		secretReq := SecretRequest{Value: "test-value"}
		body, _ := json.Marshal(secretReq)

		req := httptest.NewRequest(http.MethodPost, fmt.Sprintf("/api/v1/accounts/%s/secrets/", accountID), bytes.NewBuffer(body))
		req.Header.Set("Authorization", authHeader)
		req.Header.Set("Content-Type", "application/json")

		w := httptest.NewRecorder()
		server.router.ServeHTTP(w, req)

		// This should result in a 404 because the route won't match
		assert.Equal(t, http.StatusNotFound, w.Code)
	})

	t.Run("get_nonexistent_secret", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, fmt.Sprintf("/api/v1/accounts/%s/secrets/nonexistent", accountID), nil)
		req.Header.Set("Authorization", authHeader)

		w := httptest.NewRecorder()
		server.router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusNotFound, w.Code)
	})

	t.Run("update_nonexistent_secret", func(t *testing.T) {
		secretReq := SecretRequest{Value: "test-value"}
		body, _ := json.Marshal(secretReq)

		req := httptest.NewRequest(http.MethodPut, fmt.Sprintf("/api/v1/accounts/%s/secrets/nonexistent", accountID), bytes.NewBuffer(body))
		req.Header.Set("Authorization", authHeader)
		req.Header.Set("Content-Type", "application/json")

		w := httptest.NewRecorder()
		server.router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusNotFound, w.Code)
	})

	t.Run("delete_nonexistent_secret", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodDelete, fmt.Sprintf("/api/v1/accounts/%s/secrets/nonexistent", accountID), nil)
		req.Header.Set("Authorization", authHeader)

		w := httptest.NewRecorder()
		server.router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusNotFound, w.Code)
	})
}

func TestSecretHandlers_SpecialValues(t *testing.T) {
	server := createTestServerWithSecrets(t)
	accountID, authHeader := createTestAccountAndAuth(t, server)

	testCases := []struct {
		name  string
		key   string
		value string
	}{
		{"empty_value", "empty_key", ""},
		{"unicode_value", "unicode_key", "Hello 世界! 🔐"},
		{"json_value", "json_key", `{"key": "value", "number": 123}`},
		{"multiline_value", "multiline_key", "line1\nline2\nline3"},
		{"special_chars_key", "key-with.special_chars@domain.com", "special-value"},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Create secret
			secretReq := SecretRequest{Value: tc.value}
			body, _ := json.Marshal(secretReq)

			req := httptest.NewRequest(http.MethodPost, fmt.Sprintf("/api/v1/accounts/%s/secrets/%s", accountID, tc.key), bytes.NewBuffer(body))
			req.Header.Set("Authorization", authHeader)
			req.Header.Set("Content-Type", "application/json")

			w := httptest.NewRecorder()
			server.router.ServeHTTP(w, req)

			assert.Equal(t, http.StatusCreated, w.Code)

			// Retrieve and verify
			req = httptest.NewRequest(http.MethodGet, fmt.Sprintf("/api/v1/accounts/%s/secrets/%s", accountID, tc.key), nil)
			req.Header.Set("Authorization", authHeader)

			w = httptest.NewRecorder()
			server.router.ServeHTTP(w, req)

			assert.Equal(t, http.StatusOK, w.Code)

			var response map[string]interface{}
			err := json.Unmarshal(w.Body.Bytes(), &response)
			require.NoError(t, err)

			assert.Equal(t, tc.key, response["key"])
			assert.Equal(t, tc.value, response["value"])
		})
	}
}

// Helper functions

func createTestServerWithSecrets(t *testing.T) *Server {
	// Create in-memory storage
	storageProvider := storage.NewMemoryProvider()
	require.NoError(t, storageProvider.Initialize())

	// Create account service
	accountService := services.NewAccountService(storageProvider.GetAccountStore())

	// Create secret vault with test encryption key
	encryptionKey, err := services.GenerateEncryptionKey()
	require.NoError(t, err)

	secretVault, err := services.NewExtendedSecretVaultService(storageProvider.GetSecretStore(), encryptionKey)
	require.NoError(t, err)

	// Create flow registry (needed for server)
	flowRegistry := registry.NewFlowRegistry(storageProvider.GetFlowStore(), registry.FlowRegistryOptions{})

	// Create test config
	cfg := &config.Config{
		Server: config.ServerConfig{
			Host: "localhost",
			Port: 8080,
		},
	}

	// Create server
	server := NewServer(cfg, flowRegistry, accountService, secretVault)

	return server
}

func createTestAccountAndAuth(t *testing.T, server *Server) (string, string) {
	// Create test account
	username := fmt.Sprintf("testuser-%d", time.Now().UnixNano())
	password := "testpassword"

	accountID, err := server.accountService.CreateAccount(username, password)
	require.NoError(t, err)

	// Get account to retrieve API token
	account, err := server.accountService.GetAccount(accountID)
	require.NoError(t, err)

	return accountID, fmt.Sprintf("Bearer %s", account.APIToken)
}
