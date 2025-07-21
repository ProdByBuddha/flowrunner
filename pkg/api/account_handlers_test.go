package api

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gorilla/mux"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/tcmartin/flowrunner/pkg/auth"
	"github.com/tcmartin/flowrunner/pkg/config"
	"github.com/tcmartin/flowrunner/pkg/middleware"
)

// MockAccountService is a mock implementation of auth.AccountService
type MockAccountService struct {
	mock.Mock
}

func (m *MockAccountService) Authenticate(username, password string) (string, error) {
	args := m.Called(username, password)
	return args.String(0), args.Error(1)
}

func (m *MockAccountService) ValidateToken(token string) (string, error) {
	args := m.Called(token)
	return args.String(0), args.Error(1)
}

func (m *MockAccountService) CreateAccount(username, password string) (string, error) {
	args := m.Called(username, password)
	return args.String(0), args.Error(1)
}

func (m *MockAccountService) DeleteAccount(accountID string) error {
	args := m.Called(accountID)
	return args.Error(0)
}

func (m *MockAccountService) GetAccount(accountID string) (auth.Account, error) {
	args := m.Called(accountID)
	return args.Get(0).(auth.Account), args.Error(1)
}

func (m *MockAccountService) ListAccounts() ([]auth.Account, error) {
	args := m.Called()
	return args.Get(0).([]auth.Account), args.Error(1)
}

// MockSecretVault is a mock implementation of auth.ExtendedSecretVault
type MockSecretVault struct {
	mock.Mock
}

func (m *MockSecretVault) Set(accountID string, key string, value string) error {
	args := m.Called(accountID, key, value)
	return args.Error(0)
}

func (m *MockSecretVault) Get(accountID string, key string) (string, error) {
	args := m.Called(accountID, key)
	return args.String(0), args.Error(1)
}

func (m *MockSecretVault) Delete(accountID string, key string) error {
	args := m.Called(accountID, key)
	return args.Error(0)
}

func (m *MockSecretVault) List(accountID string) ([]string, error) {
	args := m.Called(accountID)
	return args.Get(0).([]string), args.Error(1)
}

func (m *MockSecretVault) RotateEncryptionKey(oldKey, newKey []byte) error {
	args := m.Called(oldKey, newKey)
	return args.Error(0)
}

func (m *MockSecretVault) SetStructured(accountID string, secret auth.StructuredSecret) error {
	args := m.Called(accountID, secret)
	return args.Error(0)
}

func (m *MockSecretVault) GetStructured(accountID string, key string) (auth.StructuredSecret, error) {
	args := m.Called(accountID, key)
	return args.Get(0).(auth.StructuredSecret), args.Error(1)
}

func (m *MockSecretVault) GetField(accountID string, key string, field string) (interface{}, error) {
	args := m.Called(accountID, key, field)
	return args.Get(0), args.Error(1)
}

func (m *MockSecretVault) ListByType(accountID string, secretType auth.SecretType) ([]auth.StructuredSecret, error) {
	args := m.Called(accountID, secretType)
	return args.Get(0).([]auth.StructuredSecret), args.Error(1)
}

func (m *MockSecretVault) ListByTags(accountID string, tags []string) ([]auth.StructuredSecret, error) {
	args := m.Called(accountID, tags)
	return args.Get(0).([]auth.StructuredSecret), args.Error(1)
}

func (m *MockSecretVault) Search(accountID string, query auth.SecretQuery) ([]auth.StructuredSecret, error) {
	args := m.Called(accountID, query)
	return args.Get(0).([]auth.StructuredSecret), args.Error(1)
}

func (m *MockSecretVault) UpdateMetadata(accountID string, key string, metadata auth.SecretMetadata) error {
	args := m.Called(accountID, key, metadata)
	return args.Error(0)
}

func (m *MockSecretVault) SetOAuth(accountID string, key string, oauth auth.OAuthSecret, metadata auth.SecretMetadata) error {
	args := m.Called(accountID, key, oauth, metadata)
	return args.Error(0)
}

func (m *MockSecretVault) SetAPIKey(accountID string, key string, apiKey auth.APIKeySecret, metadata auth.SecretMetadata) error {
	args := m.Called(accountID, key, apiKey, metadata)
	return args.Error(0)
}

func (m *MockSecretVault) SetDatabase(accountID string, key string, db auth.DatabaseSecret, metadata auth.SecretMetadata) error {
	args := m.Called(accountID, key, db, metadata)
	return args.Error(0)
}

func (m *MockSecretVault) SetJWT(accountID string, key string, jwt auth.JWTSecret, metadata auth.SecretMetadata) error {
	args := m.Called(accountID, key, jwt, metadata)
	return args.Error(0)
}

func (m *MockSecretVault) SetCustom(accountID string, key string, data map[string]interface{}, metadata auth.SecretMetadata) error {
	args := m.Called(accountID, key, data, metadata)
	return args.Error(0)
}

func (m *MockSecretVault) MarkUsed(accountID string, key string) error {
	args := m.Called(accountID, key)
	return args.Error(0)
}

func (m *MockSecretVault) GetExpiring(accountID string, within time.Duration) ([]auth.StructuredSecret, error) {
	args := m.Called(accountID, within)
	return args.Get(0).([]auth.StructuredSecret), args.Error(1)
}

func TestHandleListAccounts(t *testing.T) {
	// Create mock services
	mockAccountService := new(MockAccountService)
	mockSecretVault := new(MockSecretVault)

	// Create test accounts
	now := time.Now()
	accounts := []auth.Account{
		{
			ID:           "acc1",
			Username:     "user1",
			PasswordHash: "hash1",
			APIToken:     "token1",
			CreatedAt:    now,
			UpdatedAt:    now,
		},
		{
			ID:           "acc2",
			Username:     "user2",
			PasswordHash: "hash2",
			APIToken:     "token2",
			CreatedAt:    now,
			UpdatedAt:    now,
		},
	}

	// Set up expectations
	mockAccountService.On("ListAccounts").Return(accounts, nil)

	// Create server with mocks
	server := NewServer(&config.Config{}, nil, mockAccountService, mockSecretVault)

	// Create request
	req, err := http.NewRequest("GET", "/api/v1/accounts", nil)
	assert.NoError(t, err)

	// Set account ID in request context (simulating auth middleware)
	req = req.WithContext(setAccountIDContext(req.Context(), "acc1"))

	// Create response recorder
	rr := httptest.NewRecorder()

	// Call the handler directly
	server.handleListAccounts(rr, req)

	// Check response
	assert.Equal(t, http.StatusOK, rr.Code)

	// Parse response
	var response AccountListResponse
	err = json.Unmarshal(rr.Body.Bytes(), &response)
	assert.NoError(t, err)

	// Verify response
	assert.Equal(t, 2, response.Total)
	assert.Equal(t, "user1", response.Accounts[0].Username)
	assert.Equal(t, "user2", response.Accounts[1].Username)

	// Verify expectations
	mockAccountService.AssertExpectations(t)
}

func TestHandleGetAccount(t *testing.T) {
	// Create mock services
	mockAccountService := new(MockAccountService)
	mockSecretVault := new(MockSecretVault)

	// Create test account
	now := time.Now()
	account := auth.Account{
		ID:           "acc1",
		Username:     "user1",
		PasswordHash: "hash1",
		APIToken:     "token1",
		CreatedAt:    now,
		UpdatedAt:    now,
	}

	// Set up expectations
	mockAccountService.On("GetAccount", "acc1").Return(account, nil)

	// Create server with mocks
	server := NewServer(&config.Config{}, nil, mockAccountService, mockSecretVault)

	// Create request
	req, err := http.NewRequest("GET", "/api/v1/accounts/acc1", nil)
	assert.NoError(t, err)

	// Set account ID in request context (simulating auth middleware)
	req = req.WithContext(setAccountIDContext(req.Context(), "acc1"))

	// Add URL parameters
	vars := map[string]string{
		"id": "acc1",
	}
	req = mux.SetURLVars(req, vars)

	// Create response recorder
	rr := httptest.NewRecorder()

	// Call the handler directly
	server.handleGetAccount(rr, req)

	// Check response
	assert.Equal(t, http.StatusOK, rr.Code)

	// Parse response
	var response AccountResponse
	err = json.Unmarshal(rr.Body.Bytes(), &response)
	assert.NoError(t, err)

	// Verify response
	assert.Equal(t, "acc1", response.ID)
	assert.Equal(t, "user1", response.Username)

	// Verify expectations
	mockAccountService.AssertExpectations(t)
}

func TestHandleDeleteAccount(t *testing.T) {
	// Create mock services
	mockAccountService := new(MockAccountService)
	mockSecretVault := new(MockSecretVault)

	// Set up expectations
	mockAccountService.On("DeleteAccount", "acc1").Return(nil)

	// Create server with mocks
	server := NewServer(&config.Config{}, nil, mockAccountService, mockSecretVault)

	// Create request
	req, err := http.NewRequest("DELETE", "/api/v1/accounts/acc1", nil)
	assert.NoError(t, err)

	// Set account ID in request context (simulating auth middleware)
	req = req.WithContext(setAccountIDContext(req.Context(), "acc1"))

	// Add URL parameters
	vars := map[string]string{
		"id": "acc1",
	}
	req = mux.SetURLVars(req, vars)

	// Create response recorder
	rr := httptest.NewRecorder()

	// Call the handler directly
	server.handleDeleteAccount(rr, req)

	// Check response
	assert.Equal(t, http.StatusNoContent, rr.Code)

	// Verify expectations
	mockAccountService.AssertExpectations(t)
}

// Helper function to set account ID in context for testing
func setAccountIDContext(ctx context.Context, accountID string) context.Context {
	return context.WithValue(ctx, middleware.AccountIDKey, accountID)
}

func TestHandleUpdateAccount(t *testing.T) {
	// Create mock services
	mockAccountService := new(MockAccountService)
	mockSecretVault := new(MockSecretVault)

	// Create server with mocks
	server := NewServer(&config.Config{}, nil, mockAccountService, mockSecretVault)

	// Create request body
	reqBody := AccountRequest{
		Username: "user1",
		Password: "newpassword",
	}
	body, err := json.Marshal(reqBody)
	assert.NoError(t, err)

	// Create request
	req, err := http.NewRequest("PUT", "/api/v1/accounts/acc1", bytes.NewBuffer(body))
	assert.NoError(t, err)

	// Set account ID in request context (simulating auth middleware)
	req = req.WithContext(setAccountIDContext(req.Context(), "acc1"))

	// Add URL parameters
	vars := map[string]string{
		"id": "acc1",
	}
	req = mux.SetURLVars(req, vars)

	// Create response recorder
	rr := httptest.NewRecorder()

	// Call the handler directly
	server.handleUpdateAccount(rr, req)

	// Check response - should be not implemented for now
	assert.Equal(t, http.StatusNotImplemented, rr.Code)
}
