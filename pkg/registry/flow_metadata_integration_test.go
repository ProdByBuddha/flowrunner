package registry

import (
	"testing"
	"time"
)

func TestFlowMetadataManagement(t *testing.T) {
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

	// Test updating flow metadata
	err = registry.UpdateMetadata("account1", flowID, FlowMetadata{
		Tags:     []string{"test", "example", "metadata"},
		Category: "testing",
		Status:   "draft",
		Custom: map[string]interface{}{
			"owner":      "test-user",
			"department": "engineering",
			"priority":   1,
		},
	})
	
	if err != nil {
		t.Errorf("Expected no error updating metadata, got %v", err)
	}

	// Get the flow info to verify the metadata was updated
	flows, err := registry.List("account1")
	if err != nil {
		t.Errorf("Expected no error listing flows, got %v", err)
	}
	
	if len(flows) != 1 {
		t.Errorf("Expected 1 flow, got %d", len(flows))
	} else {
		flow := flows[0]
		
		// Verify tags
		if len(flow.Tags) != 3 {
			t.Errorf("Expected 3 tags, got %d", len(flow.Tags))
		} else {
			expectedTags := []string{"test", "example", "metadata"}
			for i, tag := range expectedTags {
				if flow.Tags[i] != tag {
					t.Errorf("Expected tag %s, got %s", tag, flow.Tags[i])
				}
			}
		}
		
		// Verify category
		if flow.Category != "testing" {
			t.Errorf("Expected category 'testing', got '%s'", flow.Category)
		}
		
		// Verify status
		if flow.Status != "draft" {
			t.Errorf("Expected status 'draft', got '%s'", flow.Status)
		}
		
		// Verify custom fields
		if flow.Custom == nil {
			t.Errorf("Expected custom fields, got nil")
		} else {
			if owner, ok := flow.Custom["owner"].(string); !ok || owner != "test-user" {
				t.Errorf("Expected custom field 'owner' to be 'test-user'")
			}
			
			if department, ok := flow.Custom["department"].(string); !ok || department != "engineering" {
				t.Errorf("Expected custom field 'department' to be 'engineering'")
			}
			
			if priority, ok := flow.Custom["priority"].(int); !ok || priority != 1 {
				t.Errorf("Expected custom field 'priority' to be 1")
			}
		}
	}

	// Test non-existent flow
	err = registry.UpdateMetadata("account1", "non-existent", FlowMetadata{})
	if err == nil {
		t.Error("Expected error updating metadata for non-existent flow, got nil")
	}

	// Test unauthorized access
	err = registry.UpdateMetadata("account2", flowID, FlowMetadata{})
	if err == nil {
		t.Error("Expected error updating metadata with unauthorized access, got nil")
	}
}

func TestFlowSearch(t *testing.T) {
	mockStore := NewMockFlowStore()
	mockLoader := &MockYAMLLoader{}

	registry := NewFlowRegistry(mockStore, FlowRegistryOptions{
		YAMLLoader: mockLoader,
	})

	// Create several flows with different metadata
	flows := []struct {
		name        string
		description string
		tags        []string
		category    string
		status      string
	}{
		{
			name:        "Flow 1",
			description: "A test flow for searching",
			tags:        []string{"test", "search", "example"},
			category:    "testing",
			status:      "draft",
		},
		{
			name:        "Flow 2",
			description: "A production flow for searching",
			tags:        []string{"production", "search", "example"},
			category:    "production",
			status:      "published",
		},
		{
			name:        "Flow 3",
			description: "An archived test flow",
			tags:        []string{"test", "archive", "example"},
			category:    "testing",
			status:      "archived",
		},
	}

	for _, flow := range flows {
		yamlContent := `
metadata:
  name: ` + flow.name + `
  description: ` + flow.description + `
  version: 1.0.0
nodes:
  start:
    type: test
`
		flowID, err := registry.Create("account1", flow.name, yamlContent)
		if err != nil {
			t.Fatalf("Failed to create flow '%s': %v", flow.name, err)
		}

		// Add metadata to the flow
		err = registry.UpdateMetadata("account1", flowID, FlowMetadata{
			Tags:     flow.tags,
			Category: flow.category,
			Status:   flow.status,
		})
		
		if err != nil {
			t.Fatalf("Failed to update metadata for flow '%s': %v", flow.name, err)
		}
	}

	// Test searching by name
	nameResults, err := registry.Search("account1", FlowSearchFilters{
		NameContains: "Flow 2",
	})
	
	if err != nil {
		t.Errorf("Expected no error searching by name, got %v", err)
	}
	
	if len(nameResults) != 1 {
		t.Errorf("Expected 1 result searching by name, got %d", len(nameResults))
	} else if nameResults[0].Name != "Flow 2" {
		t.Errorf("Expected flow name 'Flow 2', got '%s'", nameResults[0].Name)
	}

	// Test searching by description
	descResults, err := registry.Search("account1", FlowSearchFilters{
		DescriptionContains: "production",
	})
	
	if err != nil {
		t.Errorf("Expected no error searching by description, got %v", err)
	}
	
	if len(descResults) != 1 {
		t.Errorf("Expected 1 result searching by description, got %d", len(descResults))
	} else if descResults[0].Name != "Flow 2" {
		t.Errorf("Expected flow name 'Flow 2', got '%s'", descResults[0].Name)
	}

	// Test searching by tag
	tagResults, err := registry.Search("account1", FlowSearchFilters{
		Tags: []string{"archive"},
	})
	
	if err != nil {
		t.Errorf("Expected no error searching by tag, got %v", err)
	}
	
	if len(tagResults) != 1 {
		t.Errorf("Expected 1 result searching by tag, got %d", len(tagResults))
	} else if tagResults[0].Name != "Flow 3" {
		t.Errorf("Expected flow name 'Flow 3', got '%s'", tagResults[0].Name)
	}

	// Test searching by category
	categoryResults, err := registry.Search("account1", FlowSearchFilters{
		Category: "testing",
	})
	
	if err != nil {
		t.Errorf("Expected no error searching by category, got %v", err)
	}
	
	if len(categoryResults) != 2 {
		t.Errorf("Expected 2 results searching by category, got %d", len(categoryResults))
	}

	// Test searching by status
	statusResults, err := registry.Search("account1", FlowSearchFilters{
		Status: "published",
	})
	
	if err != nil {
		t.Errorf("Expected no error searching by status, got %v", err)
	}
	
	if len(statusResults) != 1 {
		t.Errorf("Expected 1 result searching by status, got %d", len(statusResults))
	} else if statusResults[0].Name != "Flow 2" {
		t.Errorf("Expected flow name 'Flow 2', got '%s'", statusResults[0].Name)
	}

	// Test searching by creation time
	past := time.Now().Add(24 * time.Hour) // 1 day in the future
	createdAfterResults, err := registry.Search("account1", FlowSearchFilters{
		CreatedAfter: &past,
	})
	
	if err != nil {
		t.Errorf("Expected no error searching by creation time, got %v", err)
	}
	
	if len(createdAfterResults) != 0 {
		t.Errorf("Expected 0 results searching by future creation time, got %d", len(createdAfterResults))
	}

	// Test searching with combined filters
	combinedResults, err := registry.Search("account1", FlowSearchFilters{
		Tags:     []string{"test"},
		Status:   "draft",
		Category: "testing",
	})
	
	if err != nil {
		t.Errorf("Expected no error searching with combined filters, got %v", err)
	}
	
	if len(combinedResults) != 1 {
		t.Errorf("Expected 1 result searching with combined filters, got %d", len(combinedResults))
	} else if combinedResults[0].Name != "Flow 1" {
		t.Errorf("Expected flow name 'Flow 1', got '%s'", combinedResults[0].Name)
	}

	// Test pagination
	paginationResults, err := registry.Search("account1", FlowSearchFilters{
		Page:     1,
		PageSize: 2,
	})
	
	if err != nil {
		t.Errorf("Expected no error searching with pagination, got %v", err)
	}
	
	if len(paginationResults) != 2 {
		t.Errorf("Expected 2 results on first page, got %d", len(paginationResults))
	}

	// Test second page
	page2Results, err := registry.Search("account1", FlowSearchFilters{
		Page:     2,
		PageSize: 2,
	})
	
	if err != nil {
		t.Errorf("Expected no error searching second page, got %v", err)
	}
	
	if len(page2Results) != 1 {
		t.Errorf("Expected 1 result on second page, got %d", len(page2Results))
	}
}
