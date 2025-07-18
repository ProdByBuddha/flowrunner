package registry

import (
	"fmt"
	"testing"
)

func TestFlowVersioningWithDebug(t *testing.T) {
	mockStore := NewMockFlowStore()
	mockLoader := &MockYAMLLoader{}

	registry := NewFlowRegistry(mockStore, FlowRegistryOptions{
		YAMLLoader: mockLoader,
	})

	// Create a flow first
	yamlContent := `
metadata:
  name: Test Flow
  description: A test flow
  version: 1.0.0
nodes:
  start:
    type: test
    params:
      foo: bar
`
	flowID, err := registry.Create("account1", "test-flow", yamlContent)
	if err != nil {
		t.Fatalf("Failed to create flow: %v", err)
	}
	t.Logf("Created flow with ID: %s", flowID)

	// Verify the flow was created with the initial version
	versions, err := registry.ListVersions("account1", flowID)
	if err != nil {
		t.Fatalf("Failed to list versions after creation: %v", err)
	}
	t.Logf("Found %d versions after creation", len(versions))
	for i, v := range versions {
		t.Logf("Version %d: %s", i, v.Version)
	}

	// Update the flow to create a new version
	updatedYAML := `
metadata:
  name: Updated Flow
  description: An updated test flow
  version: 1.1.0
nodes:
  start:
    type: test
    params:
      foo: baz
`
	err = registry.Update("account1", flowID, updatedYAML)
	if err != nil {
		t.Fatalf("Failed to update flow: %v", err)
	}
	t.Log("Updated flow with new version 1.1.0")

	// List versions after update
	versions, err = registry.ListVersions("account1", flowID)
	if err != nil {
		t.Fatalf("Failed to list versions after update: %v", err)
	}
	t.Logf("Found %d versions after update", len(versions))
	for i, v := range versions {
		t.Logf("Version %d: %s", i, v.Version)
	}

	// Test getting each version
	v1YAML, err := registry.GetVersion("account1", flowID, "1.0.0")
	if err != nil {
		t.Fatalf("Failed to get version 1.0.0: %v", err)
	}
	t.Log("Successfully retrieved version 1.0.0")
	
	v2YAML, err := registry.GetVersion("account1", flowID, "1.1.0")
	if err != nil {
		t.Fatalf("Failed to get version 1.1.0: %v", err)
	}
	t.Log("Successfully retrieved version 1.1.0")

	// Verify content of each version
	if v1YAML != yamlContent {
		t.Errorf("Content mismatch for version 1.0.0")
		fmt.Println("Expected:", yamlContent)
		fmt.Println("Got:", v1YAML)
	}

	if v2YAML != updatedYAML {
		t.Errorf("Content mismatch for version 1.1.0")
		fmt.Println("Expected:", updatedYAML)
		fmt.Println("Got:", v2YAML)
	}
}
