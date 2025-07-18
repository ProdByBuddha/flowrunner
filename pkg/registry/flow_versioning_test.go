package registry

import (
	"testing"
)

func TestFlowVersioning(t *testing.T) {
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
	flowID, _ := registry.Create("account1", "test-flow", yamlContent)

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
	err := registry.Update("account1", flowID, updatedYAML)
	if err != nil {
		t.Errorf("Expected no error on update, got %v", err)
	}

	// Test getting a specific version
	v1YAML, err := registry.GetVersion("account1", flowID, "1.0.0")
	if err != nil {
		t.Errorf("Expected no error getting version 1.0.0, got %v", err)
	}
	if v1YAML != yamlContent {
		t.Errorf("Expected original content for version 1.0.0, got different content")
	}

	v2YAML, err := registry.GetVersion("account1", flowID, "1.1.0")
	if err != nil {
		t.Errorf("Expected no error getting version 1.1.0, got %v", err)
	}
	if v2YAML != updatedYAML {
		t.Errorf("Expected updated content for version 1.1.0, got different content")
	}

	// Test listing versions
	versions, err := registry.ListVersions("account1", flowID)
	if err != nil {
		t.Errorf("Expected no error listing versions, got %v", err)
	}
	if len(versions) != 2 {
		t.Errorf("Expected 2 versions, got %d", len(versions))
	}

	// Verify we have the correct number of versions
	if len(versions) != 2 {
		t.Errorf("Expected 2 versions, got %d", len(versions))
	}
	
	// Check if we have versions with the expected version numbers
	hasV1 := false
	hasV2 := false
	for _, v := range versions {
		if v.Version == "1.0.0" {
			hasV1 = true
		}
		if v.Version == "1.1.0" {
			hasV2 = true
		}
	}
	if !hasV1 || !hasV2 {
		t.Errorf("Missing versions in the version list. Has v1.0.0: %v, Has v1.1.0: %v", hasV1, hasV2)
	}

	// Test getting a non-existent version
	_, err = registry.GetVersion("account1", flowID, "2.0.0")
	if err == nil {
		t.Error("Expected error for non-existent version, got nil")
	}

	// Test getting a version for a non-existent flow
	_, err = registry.GetVersion("account1", "non-existent", "1.0.0")
	if err == nil {
		t.Error("Expected error for non-existent flow, got nil")
	}

	// Test unauthorized access to a version
	_, err = registry.GetVersion("account2", flowID, "1.0.0")
	if err == nil {
		t.Error("Expected error for unauthorized access, got nil")
	}

	// Test update with explicit version
	thirdUpdateYAML := `
metadata:
  name: Third Update Flow
  description: A third update to the flow
  version: 2.0.0
nodes:
  start:
    type: test
    params:
      foo: qux
`
	err = registry.Update("account1", flowID, thirdUpdateYAML)
	if err != nil {
		t.Errorf("Expected no error on update with explicit version, got %v", err)
	}

	// Verify the third version is available
	v3YAML, err := registry.GetVersion("account1", flowID, "2.0.0")
	if err != nil {
		t.Errorf("Expected no error getting version 2.0.0, got %v", err)
	}
	if v3YAML != thirdUpdateYAML {
		t.Errorf("Expected updated content for version 2.0.0, got different content")
	}

	// Verify that latest version is returned by Get
	latestYAML, err := registry.Get("account1", flowID)
	if err != nil {
		t.Errorf("Expected no error getting latest version, got %v", err)
	}
	if latestYAML != thirdUpdateYAML {
		t.Errorf("Expected latest version to be 2.0.0, got different content")
	}

	// Test update without explicit version (should auto-generate)
	noVersionYAML := `
metadata:
  name: Auto Version Flow
  description: A flow without explicit version
nodes:
  start:
    type: test
    params:
      foo: auto
`
	err = registry.Update("account1", flowID, noVersionYAML)
	if err != nil {
		t.Errorf("Expected no error on update without version, got %v", err)
	}

	// Check that all versions are available after multiple updates
	allVersions, err := registry.ListVersions("account1", flowID)
	if err != nil {
		t.Errorf("Expected no error listing all versions, got %v", err)
	}
	if len(allVersions) < 4 {
		t.Errorf("Expected at least 4 versions, got %d", len(allVersions))
	}
}
