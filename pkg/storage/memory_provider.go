package storage

import (
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/tcmartin/flowrunner/pkg/auth"
	"github.com/tcmartin/flowrunner/pkg/models"
	"github.com/tcmartin/flowrunner/pkg/runtime"
)

// Errors returned by the in-memory storage provider
var (
	ErrFlowNotFound      = errors.New("flow not found")
	ErrSecretNotFound    = errors.New("secret not found")
	ErrExecutionNotFound = errors.New("execution not found")
	ErrAccountNotFound   = errors.New("account not found")
)

// MemoryProvider implements the StorageProvider interface using in-memory storage
type MemoryProvider struct {
	flowStore      *MemoryFlowStore
	secretStore    *MemorySecretStore
	executionStore *MemoryExecutionStore
	accountStore   *MemoryAccountStore
}

// NewMemoryProvider creates a new in-memory storage provider
func NewMemoryProvider() *MemoryProvider {
	return &MemoryProvider{
		flowStore:      NewMemoryFlowStore(),
		secretStore:    NewMemorySecretStore(),
		executionStore: NewMemoryExecutionStore(),
		accountStore:   NewMemoryAccountStore(),
	}
}

// Initialize sets up the storage backend
func (p *MemoryProvider) Initialize() error {
	// Nothing to initialize for in-memory storage
	return nil
}

// Close cleans up resources
func (p *MemoryProvider) Close() error {
	// Nothing to close for in-memory storage
	return nil
}

// GetFlowStore returns a store for flow definitions
func (p *MemoryProvider) GetFlowStore() FlowStore {
	return p.flowStore
}

// GetSecretStore returns a store for secrets
func (p *MemoryProvider) GetSecretStore() SecretStore {
	return p.secretStore
}

// GetExecutionStore returns a store for execution data
func (p *MemoryProvider) GetExecutionStore() runtime.ExecutionStore {
	return p.executionStore
}

// GetAccountStore returns a store for account data
func (p *MemoryProvider) GetAccountStore() AccountStore {
	return p.accountStore
}

// MemoryFlowStore implements the FlowStore interface using in-memory storage
type MemoryFlowStore struct {
	flows    map[string]map[string][]byte
	metadata map[string]map[string]FlowMetadata
	versions map[string]map[string]map[string]FlowVersion // accountID -> flowID -> version -> FlowVersion
	mu       sync.RWMutex
}

// NewMemoryFlowStore creates a new in-memory flow store
func NewMemoryFlowStore() *MemoryFlowStore {
	return &MemoryFlowStore{
		flows:    make(map[string]map[string][]byte),
		metadata: make(map[string]map[string]FlowMetadata),
		versions: make(map[string]map[string]map[string]FlowVersion),
	}
}

// SaveFlow persists a flow definition
func (s *MemoryFlowStore) SaveFlow(accountID, flowID string, definition []byte) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Create account maps if they don't exist
	if _, ok := s.flows[accountID]; !ok {
		s.flows[accountID] = make(map[string][]byte)
		s.metadata[accountID] = make(map[string]FlowMetadata)
		s.versions[accountID] = make(map[string]map[string]FlowVersion)
	}

	// Create flow versions map if it doesn't exist
	if _, ok := s.versions[accountID][flowID]; !ok {
		s.versions[accountID][flowID] = make(map[string]FlowVersion)
	}

	// Store the flow definition
	s.flows[accountID][flowID] = definition

	// Generate a version number based on timestamp
	version := fmt.Sprintf("v%d", time.Now().UnixNano())

	// Create or update flow version
	flowVersion := FlowVersion{
		FlowID:     flowID,
		Version:    version,
		CreatedAt:  time.Now().Unix(),
		Definition: definition,
	}
	s.versions[accountID][flowID][version] = flowVersion

	// Update or create metadata
	meta, ok := s.metadata[accountID][flowID]
	if !ok {
		// New flow, create metadata
		meta = FlowMetadata{
			ID:        flowID,
			AccountID: accountID,
			CreatedAt: time.Now().Unix(),
		}
	}

	// Update metadata with new version
	meta.Version = version
	meta.UpdatedAt = time.Now().Unix()
	s.metadata[accountID][flowID] = meta

	return nil
}

// GetFlow retrieves a flow definition
func (s *MemoryFlowStore) GetFlow(accountID, flowID string) ([]byte, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	// Check if account exists
	accountFlows, ok := s.flows[accountID]
	if !ok {
		return nil, ErrFlowNotFound
	}

	// Check if flow exists
	definition, ok := accountFlows[flowID]
	if !ok {
		return nil, ErrFlowNotFound
	}

	return definition, nil
}

// ListFlows returns all flow IDs for an account
func (s *MemoryFlowStore) ListFlows(accountID string) ([]string, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	// Check if account exists
	accountFlows, ok := s.flows[accountID]
	if !ok {
		return []string{}, nil
	}

	// Get all flow IDs
	flowIDs := make([]string, 0, len(accountFlows))
	for id := range accountFlows {
		flowIDs = append(flowIDs, id)
	}

	return flowIDs, nil
}

// DeleteFlow removes a flow definition and all its versions
func (s *MemoryFlowStore) DeleteFlow(accountID, flowID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Check if account exists
	accountFlows, ok := s.flows[accountID]
	if !ok {
		return ErrFlowNotFound
	}

	// Check if flow exists
	if _, ok := accountFlows[flowID]; !ok {
		return ErrFlowNotFound
	}

	// Delete flow, metadata, and all versions
	delete(accountFlows, flowID)
	delete(s.metadata[accountID], flowID)
	if _, ok := s.versions[accountID]; ok {
		delete(s.versions[accountID], flowID)
	}

	return nil
}

// GetFlowMetadata retrieves metadata for a flow
func (s *MemoryFlowStore) GetFlowMetadata(accountID, flowID string) (FlowMetadata, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	// Check if account exists
	accountMetadata, ok := s.metadata[accountID]
	if !ok {
		return FlowMetadata{}, ErrFlowNotFound
	}

	// Check if flow exists
	metadata, ok := accountMetadata[flowID]
	if !ok {
		return FlowMetadata{}, ErrFlowNotFound
	}

	return metadata, nil
}

// ListFlowsWithMetadata returns all flows with metadata for an account
func (s *MemoryFlowStore) ListFlowsWithMetadata(accountID string) ([]FlowMetadata, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	// Check if account exists
	accountMetadata, ok := s.metadata[accountID]
	if !ok {
		return []FlowMetadata{}, nil
	}

	// Get all flow metadata
	metadataList := make([]FlowMetadata, 0, len(accountMetadata))
	for _, metadata := range accountMetadata {
		metadataList = append(metadataList, metadata)
	}

	return metadataList, nil
}

// MemorySecretStore implements the SecretStore interface using in-memory storage
type MemorySecretStore struct {
	secrets map[string]map[string]auth.Secret
	mu      sync.RWMutex
}

// NewMemorySecretStore creates a new in-memory secret store
func NewMemorySecretStore() *MemorySecretStore {
	return &MemorySecretStore{
		secrets: make(map[string]map[string]auth.Secret),
	}
}

// SaveSecret persists a secret
func (s *MemorySecretStore) SaveSecret(secret auth.Secret) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Create account map if it doesn't exist
	if _, ok := s.secrets[secret.AccountID]; !ok {
		s.secrets[secret.AccountID] = make(map[string]auth.Secret)
	}

	// Store the secret
	s.secrets[secret.AccountID][secret.Key] = secret

	return nil
}

// GetSecret retrieves a secret
func (s *MemorySecretStore) GetSecret(accountID, key string) (auth.Secret, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	// Check if account exists
	accountSecrets, ok := s.secrets[accountID]
	if !ok {
		return auth.Secret{}, ErrSecretNotFound
	}

	// Check if secret exists
	secret, ok := accountSecrets[key]
	if !ok {
		return auth.Secret{}, ErrSecretNotFound
	}

	return secret, nil
}

// ListSecrets returns all secrets for an account
func (s *MemorySecretStore) ListSecrets(accountID string) ([]auth.Secret, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	// Check if account exists
	accountSecrets, ok := s.secrets[accountID]
	if !ok {
		return []auth.Secret{}, nil
	}

	// Get all secrets
	secretList := make([]auth.Secret, 0, len(accountSecrets))
	for _, secret := range accountSecrets {
		secretList = append(secretList, secret)
	}

	return secretList, nil
}

// DeleteSecret removes a secret
func (s *MemorySecretStore) DeleteSecret(accountID, key string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Check if account exists
	accountSecrets, ok := s.secrets[accountID]
	if !ok {
		return ErrSecretNotFound
	}

	// Check if secret exists
	if _, ok := accountSecrets[key]; !ok {
		return ErrSecretNotFound
	}

	// Delete secret
	delete(accountSecrets, key)

	return nil
}

// MemoryExecutionStore implements the ExecutionStore interface using in-memory storage
type MemoryExecutionStore struct {
	executions map[string]ExecutionWrapper
	logs       map[string][]models.ExecutionLog
	mu         sync.RWMutex
}

// NewMemoryExecutionStore creates a new in-memory execution store
func NewMemoryExecutionStore() *MemoryExecutionStore {
	return &MemoryExecutionStore{
		executions: make(map[string]ExecutionWrapper),
		logs:       make(map[string][]models.ExecutionLog),
	}
}

// SaveExecution persists execution data
func (s *MemoryExecutionStore) SaveExecution(execution models.ExecutionStatus) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Get existing wrapper if it exists to preserve the account ID
	var accountID string
	if wrapper, ok := s.executions[execution.ID]; ok {
		accountID = wrapper.AccountID
	}

	// Store the execution
	s.executions[execution.ID] = ExecutionWrapper{
		ExecutionStatus: execution,
		AccountID:       accountID,
	}

	return nil
}

// GetExecution retrieves execution data
func (s *MemoryExecutionStore) GetExecution(executionID string) (models.ExecutionStatus, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	// Check if execution exists
	wrapper, ok := s.executions[executionID]
	if !ok {
		return models.ExecutionStatus{}, ErrExecutionNotFound
	}

	return wrapper.ExecutionStatus, nil
}

// ListExecutions returns all executions for an account
func (s *MemoryExecutionStore) ListExecutions(accountID string) ([]models.ExecutionStatus, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	// Get all executions for the account
	executionList := make([]models.ExecutionStatus, 0)
	for _, wrapper := range s.executions {
		if wrapper.AccountID == accountID {
			executionList = append(executionList, wrapper.ExecutionStatus)
		}
	}

	return executionList, nil
}

// SaveExecutionLog persists an execution log entry
func (s *MemoryExecutionStore) SaveExecutionLog(executionID string, log models.ExecutionLog) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Create log array if it doesn't exist
	if _, ok := s.logs[executionID]; !ok {
		s.logs[executionID] = make([]models.ExecutionLog, 0)
	}

	// Store the log
	s.logs[executionID] = append(s.logs[executionID], log)

	return nil
}

// GetExecutionLogs retrieves logs for an execution
func (s *MemoryExecutionStore) GetExecutionLogs(executionID string) ([]models.ExecutionLog, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	// Check if logs exist
	logs, ok := s.logs[executionID]
	if !ok {
		return []models.ExecutionLog{}, nil
	}

	return logs, nil
}

// MemoryAccountStore implements the AccountStore interface using in-memory storage
type MemoryAccountStore struct {
	accounts        map[string]auth.Account
	accountsByName  map[string]string
	accountsByToken map[string]string
	mu              sync.RWMutex
}

// NewMemoryAccountStore creates a new in-memory account store
func NewMemoryAccountStore() *MemoryAccountStore {
	return &MemoryAccountStore{
		accounts:        make(map[string]auth.Account),
		accountsByName:  make(map[string]string),
		accountsByToken: make(map[string]string),
	}
}

// SaveAccount persists an account
func (s *MemoryAccountStore) SaveAccount(account auth.Account) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Store the account
	s.accounts[account.ID] = account
	s.accountsByName[account.Username] = account.ID
	s.accountsByToken[account.APIToken] = account.ID

	return nil
}

// GetAccount retrieves an account
func (s *MemoryAccountStore) GetAccount(accountID string) (auth.Account, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	// Check if account exists
	account, ok := s.accounts[accountID]
	if !ok {
		return auth.Account{}, ErrAccountNotFound
	}

	return account, nil
}

// GetAccountByUsername retrieves an account by username
func (s *MemoryAccountStore) GetAccountByUsername(username string) (auth.Account, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	// Check if username exists
	accountID, ok := s.accountsByName[username]
	if !ok {
		return auth.Account{}, ErrAccountNotFound
	}

	// Get account
	account, ok := s.accounts[accountID]
	if !ok {
		return auth.Account{}, ErrAccountNotFound
	}

	return account, nil
}

// GetAccountByToken retrieves an account by API token
func (s *MemoryAccountStore) GetAccountByToken(token string) (auth.Account, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	// Check if token exists
	accountID, ok := s.accountsByToken[token]
	if !ok {
		return auth.Account{}, ErrAccountNotFound
	}

	// Get account
	account, ok := s.accounts[accountID]
	if !ok {
		return auth.Account{}, ErrAccountNotFound
	}

	return account, nil
}

// ListAccounts returns all accounts
func (s *MemoryAccountStore) ListAccounts() ([]auth.Account, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	// Get all accounts
	accountList := make([]auth.Account, 0, len(s.accounts))
	for _, account := range s.accounts {
		accountList = append(accountList, account)
	}

	return accountList, nil
}

// DeleteAccount removes an account
func (s *MemoryAccountStore) DeleteAccount(accountID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Check if account exists
	account, ok := s.accounts[accountID]
	if !ok {
		return ErrAccountNotFound
	}

	// Delete account
	delete(s.accounts, accountID)
	delete(s.accountsByName, account.Username)
	delete(s.accountsByToken, account.APIToken)

	return nil
}

// SaveFlowVersion persists a new version of a flow definition
func (s *MemoryFlowStore) SaveFlowVersion(accountID, flowID string, definition []byte, version string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Create account maps if they don't exist
	if _, ok := s.flows[accountID]; !ok {
		s.flows[accountID] = make(map[string][]byte)
		s.metadata[accountID] = make(map[string]FlowMetadata)
		s.versions[accountID] = make(map[string]map[string]FlowVersion)
	}

	// Create flow versions map if it doesn't exist
	if _, ok := s.versions[accountID][flowID]; !ok {
		s.versions[accountID][flowID] = make(map[string]FlowVersion)
	}

	// Store the flow definition as the current version
	s.flows[accountID][flowID] = definition

	// Create or update flow version
	flowVersion := FlowVersion{
		FlowID:     flowID,
		Version:    version,
		CreatedAt:  time.Now().Unix(),
		Definition: definition,
	}
	s.versions[accountID][flowID][version] = flowVersion

	// Update metadata
	meta, ok := s.metadata[accountID][flowID]
	if !ok {
		// New flow, create metadata
		meta = FlowMetadata{
			ID:        flowID,
			AccountID: accountID,
			CreatedAt: time.Now().Unix(),
		}
	}

	// Update metadata with new version
	meta.Version = version
	meta.UpdatedAt = time.Now().Unix()
	s.metadata[accountID][flowID] = meta

	return nil
}

// GetFlowVersion retrieves a specific version of a flow definition
func (s *MemoryFlowStore) GetFlowVersion(accountID, flowID, version string) ([]byte, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	// Check if account exists
	if _, ok := s.versions[accountID]; !ok {
		return nil, ErrFlowNotFound
	}

	// Check if flow exists
	if _, ok := s.versions[accountID][flowID]; !ok {
		return nil, ErrFlowNotFound
	}

	// Check if version exists
	flowVersion, ok := s.versions[accountID][flowID][version]
	if !ok {
		return nil, ErrFlowNotFound
	}

	return flowVersion.Definition, nil
}

// ListFlowVersions returns all versions of a flow
func (s *MemoryFlowStore) ListFlowVersions(accountID, flowID string) ([]string, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	// Check if account exists
	if _, ok := s.versions[accountID]; !ok {
		return []string{}, nil
	}

	// Check if flow exists
	if _, ok := s.versions[accountID][flowID]; !ok {
		return []string{}, nil
	}

	// Get all versions
	versions := make([]string, 0, len(s.versions[accountID][flowID]))
	for version := range s.versions[accountID][flowID] {
		versions = append(versions, version)
	}

	return versions, nil
}
