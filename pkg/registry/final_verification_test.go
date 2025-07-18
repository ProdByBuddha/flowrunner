package registry

import (
	"testing"
)

func TestFlowVersioningAndMetadata(t *testing.T) {
	// Create a new registry with mock implementations
	mockStore := NewMockFlowStore()
	mockLoader := &MockYAMLLoader{}

	registry := NewFlowRegistry(mockStore, FlowRegistryOptions{
		YAMLLoader: mockLoader,
	})

	// 1. Create a flow
	yamlContent := `
metadata:
  name: Version Test Flow
  description: Test flow for versioning and metadata
  version: 1.0.0
nodes:
  start:
    type: start
  end:
    type: end
`
	flowID, err := registry.Create("test-account", "test-flow", yamlContent)
	if err != nil {
		t.Fatalf("Failed to create flow: %v", err)
	}
	t.Logf("Created flow with ID: %s", flowID)

	// 2. Update metadata
	err = registry.UpdateMetadata("test-account", flowID, FlowMetadata{
		Tags:     []string{"test", "versioning"},
		Category: "testing",
		Status:   "development",
		Custom: map[string]interface{}{
			"owner": "tester",
			"priority": 1,
		},
	})
	if err != nil {
		t.Fatalf("Failed to update metadata: %v", err)
	}
	t.Log("Successfully updated metadata")

	// 3. Create a new version
	updatedYAML := `
metadata:
  name: Version Test Flow
  description: Updated test flow for versioning and metadata
  version: 1.1.0
nodes:
  start:
    type: start
  process:
    type: process
  end:
    type: end
`
	err = registry.Update("test-account", flowID, updatedYAML)
	if err != nil {
		t.Fatalf("Failed to update flow: %v", err)
	}
	t.Log("Successfully created version 1.1.0")

	// 4. List versions
	versions, err := registry.ListVersions("test-account", flowID)
	if err != nil {
		t.Fatalf("Failed to list versions: %v", err)
	}
	t.Logf("Found %d versions", len(versions))
	for i, v := range versions {
		t.Logf("Version %d: %s", i, v.Version)
	}

	// 5. Get specific versions
	v1Content, err := registry.GetVersion("test-account", flowID, "1.0.0")
	if err != nil {
		t.Fatalf("Failed to get version 1.0.0: %v", err)
	}
	t.Logf("Successfully retrieved version 1.0.0: %d bytes", len(v1Content))

	v2Content, err := registry.GetVersion("test-account", flowID, "1.1.0") 
	if err != nil {
		t.Fatalf("Failed to get version 1.1.0: %v", err)
	}
	t.Logf("Successfully retrieved version 1.1.0: %d bytes", len(v2Content))

	// 6. Search by metadata
	results, err := registry.Search("test-account", FlowSearchFilters{
		Tags:     []string{"versioning"},
		Category: "testing",
	})
	if err != nil {
		t.Fatalf("Failed to search flows: %v", err)
	}
	t.Logf("Found %d flows matching metadata criteria", len(results))

	// 7. Verify the metadata was preserved across versions
	if len(results) == 0 {
		t.Fatal("No flows found with the specified metadata")
	}
	
	flow := results[0]
	if flow.Version != "1.1.0" {
		t.Errorf("Expected latest version 1.1.0, got %s", flow.Version)
	}
	
	if len(flow.Tags) != 2 || flow.Tags[0] != "test" || flow.Tags[1] != "versioning" {
		t.Errorf("Expected tags [test, versioning], got %v", flow.Tags)
	}
	
	if flow.Category != "testing" {
		t.Errorf("Expected category 'testing', got %s", flow.Category)
	}
	
	if flow.Status != "development" {
		t.Errorf("Expected status 'development', got %s", flow.Status)
	}
	
	if flow.Custom == nil {
		t.Error("Expected custom fields to be present")
	} else {
		if owner, ok := flow.Custom["owner"].(string); !ok || owner != "tester" {
			t.Errorf("Expected owner 'tester', got %v", flow.Custom["owner"])
		}
	}
}
