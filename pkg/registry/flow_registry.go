package registry

import (
	"errors"
	"fmt"
	"time"

	"github.com/tcmartin/flowrunner/pkg/loader"
	"github.com/tcmartin/flowrunner/pkg/storage"
	"gopkg.in/yaml.v3"
)

// Errors returned by the flow registry
var (
	ErrFlowNotFound      = errors.New("flow not found")
	ErrInvalidYAML       = errors.New("invalid YAML flow definition")
	ErrFlowAlreadyExists = errors.New("flow with this name already exists")
	ErrUnauthorized      = errors.New("unauthorized access to flow")
)

// FlowRegistryService implements the FlowRegistry interface
type FlowRegistryService struct {
	flowStore  storage.FlowStore
	yamlLoader loader.YAMLLoader
}

// NewFlowRegistry creates a new flow registry service
func NewFlowRegistry(flowStore storage.FlowStore, options FlowRegistryOptions) FlowRegistry {
	return &FlowRegistryService{
		flowStore:  flowStore,
		yamlLoader: options.YAMLLoader,
	}
}

// Create stores a new flow definition
func (r *FlowRegistryService) Create(accountID string, name string, yamlContent string) (string, error) {
	// Validate the YAML content
	if err := r.yamlLoader.Validate(yamlContent); err != nil {
		return "", fmt.Errorf("%w: %v", ErrInvalidYAML, err)
	}

	// Parse the YAML to extract metadata
	flowDef := &loader.FlowDefinition{}
	if err := yaml.Unmarshal([]byte(yamlContent), flowDef); err != nil {
		return "", fmt.Errorf("%w: %v", ErrInvalidYAML, err)
	}

	// Generate a unique ID for the flow
	flowID := fmt.Sprintf("%s-%d", name, time.Now().UnixNano())

	// Create flow metadata and save it with the flow definition
	now := time.Now().Unix()
	metadata := storage.FlowMetadata{
		ID:          flowID,
		AccountID:   accountID,
		Name:        flowDef.Metadata.Name,
		Description: flowDef.Metadata.Description,
		Version:     flowDef.Metadata.Version,
		CreatedAt:   now,
		UpdatedAt:   now,
	}

	// Save the flow definition with metadata
	// Note: The FlowStore implementation should handle storing the metadata
	if err := r.flowStore.SaveFlow(accountID, flowID, []byte(yamlContent)); err != nil {
		return "", fmt.Errorf("failed to save flow: %w", err)
	}

	return flowID, nil
}

// Get retrieves a flow definition by ID
func (r *FlowRegistryService) Get(accountID string, id string) (string, error) {
	// Get the flow definition
	flowBytes, err := r.flowStore.GetFlow(accountID, id)
	if err != nil {
		return "", fmt.Errorf("failed to get flow: %w", err)
	}

	return string(flowBytes), nil
}

// List returns all flows for an account
func (r *FlowRegistryService) List(accountID string) ([]FlowInfo, error) {
	// Get all flows with metadata for the account
	metadataList, err := r.flowStore.ListFlowsWithMetadata(accountID)
	if err != nil {
		return nil, fmt.Errorf("failed to list flows: %w", err)
	}

	// Convert to FlowInfo
	flowInfos := make([]FlowInfo, len(metadataList))
	for i, metadata := range metadataList {
		flowInfos[i] = FlowInfo{
			ID:          metadata.ID,
			AccountID:   metadata.AccountID,
			Name:        metadata.Name,
			Description: metadata.Description,
			Version:     metadata.Version,
			CreatedAt:   time.Unix(metadata.CreatedAt, 0),
			UpdatedAt:   time.Unix(metadata.UpdatedAt, 0),
		}
	}

	return flowInfos, nil
}

// Update modifies an existing flow definition
func (r *FlowRegistryService) Update(accountID string, id string, yamlContent string) error {
	// Check if the flow exists and belongs to the account
	_, err := r.flowStore.GetFlow(accountID, id)
	if err != nil {
		return fmt.Errorf("failed to get flow: %w", err)
	}

	// Validate the YAML content
	if err := r.yamlLoader.Validate(yamlContent); err != nil {
		return fmt.Errorf("%w: %v", ErrInvalidYAML, err)
	}

	// Parse the YAML to extract metadata
	flowDef := &loader.FlowDefinition{}
	if err := yaml.Unmarshal([]byte(yamlContent), flowDef); err != nil {
		return fmt.Errorf("%w: %v", ErrInvalidYAML, err)
	}

	// Update flow definition with updated metadata
	// Note: The FlowStore implementation should handle updating the metadata
	if err := r.flowStore.SaveFlow(accountID, id, []byte(yamlContent)); err != nil {
		return fmt.Errorf("failed to update flow: %w", err)
	}

	return nil
}

// Delete removes a flow definition
func (r *FlowRegistryService) Delete(accountID string, id string) error {
	// Check if the flow exists and belongs to the account
	_, err := r.flowStore.GetFlow(accountID, id)
	if err != nil {
		return fmt.Errorf("failed to get flow: %w", err)
	}

	// Delete the flow
	if err := r.flowStore.DeleteFlow(accountID, id); err != nil {
		return fmt.Errorf("failed to delete flow: %w", err)
	}

	return nil
}
