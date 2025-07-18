package registry

import (
	"strings"
	"testing"
)

// We need to add these methods to the MockFlowStore
func TestMissingMethodImplementations(t *testing.T) {
	mockStore := NewMockFlowStore()
	
	// Verify UpdateFlowMetadata works
	metadata := mockStore.metadata["account1"]["flow1"]
	metadata.Tags = []string{"test"}
	err := mockStore.UpdateFlowMetadata("account1", "flow1", metadata)
	if err == nil {
		// Should fail as account1/flow1 doesn't exist yet
		t.Error("Expected error updating non-existent flow")
	}
	
	// Create the flow first
	mockStore.flows["account1"] = make(map[string][]byte)
	mockStore.metadata["account1"] = make(map[string]storage.FlowMetadata)
	mockStore.flows["account1"]["flow1"] = []byte("test")
	mockStore.metadata["account1"]["flow1"] = storage.FlowMetadata{
		ID: "flow1",
		AccountID: "account1",
	}
	
	// Now try updating
	err = mockStore.UpdateFlowMetadata("account1", "flow1", metadata)
	if err != nil {
		t.Errorf("Failed to update metadata: %v", err)
	}
	
	// Test SearchFlows
	filters := map[string]interface{}{
		"tags": []string{"test"},
	}
	
	results, err := mockStore.SearchFlows("account1", filters)
	if err != nil {
		t.Errorf("Failed to search flows: %v", err)
	}
	
	if len(results) != 1 {
		t.Errorf("Expected 1 result, got %d", len(results))
	}
}
