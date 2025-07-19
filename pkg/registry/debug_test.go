package registry

import (
	"testing"

	"github.com/tcmartin/flowrunner/pkg/storage"
)

// TestFlowCRUDDebug is a simple integration test that verifies all CRUD operations work together
func TestFlowCRUDDebug(t *testing.T) {
	// Use the existing mock store and loader patterns from flow_registry_test.go
	mockStore := NewMockFlowStore()
	mockLoader := &MockYAMLLoader{}

	registry := NewFlowRegistry(mockStore, FlowRegistryOptions{
		YAMLLoader: mockLoader,
	})

	accountID := "test-account"
	flowName := "debug-test-flow"
	flowContent := `metadata:
  name: Debug Test Flow
  description: A flow for debugging CRUD operations
  version: 1.0.0
nodes:
  start:
    type: test
    params:
      message: "Hello from debug test"`

	t.Log("Testing complete flow CRUD cycle...")

	// 1. CREATE
	t.Log("Step 1: Creating flow...")
	flowID, err := registry.Create(accountID, flowName, flowContent)
	if err != nil {
		t.Fatalf("Failed to create flow: %v", err)
	}
	t.Logf("✓ Created flow with ID: %s", flowID)

	// 2. READ (Get specific flow)
	t.Log("Step 2: Reading flow...")
	retrievedContent, err := registry.Get(accountID, flowID)
	if err != nil {
		t.Fatalf("Failed to get flow: %v", err)
	}
	if retrievedContent != flowContent {
		t.Errorf("Expected flow content to match original")
	}
	t.Logf("✓ Retrieved flow content (length: %d chars)", len(retrievedContent))

	// 3. LIST
	t.Log("Step 3: Listing flows...")
	flows, err := registry.List(accountID)
	if err != nil {
		t.Fatalf("Failed to list flows: %v", err)
	}
	if len(flows) == 0 {
		t.Fatal("Expected at least one flow in list")
	}
	t.Logf("✓ Listed %d flow(s)", len(flows))

	// 4. UPDATE
	t.Log("Step 4: Updating flow...")
	updatedContent := `metadata:
  name: Updated Debug Test Flow
  description: An updated flow for debugging CRUD operations
  version: 1.1.0
nodes:
  start:
    type: test
    params:
      message: "Updated message from debug test"`

	err = registry.Update(accountID, flowID, updatedContent)
	if err != nil {
		t.Fatalf("Failed to update flow: %v", err)
	}
	t.Log("✓ Updated flow successfully")

	// Verify update worked
	retrievedUpdatedContent, err := registry.Get(accountID, flowID)
	if err != nil {
		t.Fatalf("Failed to get updated flow: %v", err)
	}
	if retrievedUpdatedContent != updatedContent {
		t.Errorf("Expected updated flow content to match")
	}
	t.Log("✓ Verified flow was updated")

	// 5. DELETE
	t.Log("Step 5: Deleting flow...")
	err = registry.Delete(accountID, flowID)
	if err != nil {
		t.Fatalf("Failed to delete flow: %v", err)
	}
	t.Log("✓ Deleted flow successfully")

	// Verify deletion worked
	_, err = registry.Get(accountID, flowID)
	if err == nil {
		t.Fatal("Expected error when getting deleted flow, but got none")
	}
	t.Log("✓ Verified flow was deleted")

	t.Log("All CRUD operations completed successfully!")
}

// TestFlowCRUDWithDynamoDB tests flow CRUD operations against actual DynamoDB
func TestFlowCRUDWithDynamoDB(t *testing.T) {
	// Skip if DynamoDB is not available
	if testing.Short() {
		t.Skip("Skipping DynamoDB integration test in short mode")
	}

	t.Log("Testing flow CRUD operations with real DynamoDB...")
	t.Log("NOTE: This requires local DynamoDB running on port 8000")

	// Set up DynamoDB provider pointing to local DynamoDB
	config := storage.DynamoDBProviderConfig{
		Region:      "us-west-2",
		TablePrefix: "flowrunner_test_debug_",
		Endpoint:    "http://localhost:8000", // Local DynamoDB
	}

	// Create DynamoDB provider
	provider, err := storage.NewDynamoDBProvider(config)
	if err != nil {
		t.Skipf("Failed to create DynamoDB provider (local DynamoDB may not be running): %v", err)
	}

	// Initialize the provider (creates tables)
	err = provider.Initialize()
	if err != nil {
		t.Skipf("Failed to initialize DynamoDB provider (local DynamoDB may not be running): %v", err)
	}

	t.Log("✓ DynamoDB provider initialized successfully")

	// Get the flow store
	flowStore := provider.GetFlowStore()

	// Create registry with DynamoDB store
	mockLoader := &MockYAMLLoader{}
	registry := NewFlowRegistry(flowStore, FlowRegistryOptions{
		YAMLLoader: mockLoader,
	})

	// Now run the same CRUD test as the mock version
	accountID := "test-account-dynamodb"
	flowName := "dynamodb-test-flow"
	flowContent := `metadata:
  name: DynamoDB Test Flow
  description: A flow for debugging CRUD operations with DynamoDB
  version: 1.0.0
nodes:
  start:
    type: test
    params:
      message: "Hello from DynamoDB debug test"`

	t.Log("Testing complete flow CRUD cycle with DynamoDB...")

	// 1. CREATE
	t.Log("Step 1: Creating flow in DynamoDB...")
	flowID, err := registry.Create(accountID, flowName, flowContent)
	if err != nil {
		t.Fatalf("Failed to create flow: %v", err)
	}
	t.Logf("✓ Created flow with ID: %s", flowID)

	// 2. READ (Get specific flow)
	t.Log("Step 2: Reading flow from DynamoDB...")
	retrievedContent, err := registry.Get(accountID, flowID)
	if err != nil {
		t.Fatalf("Failed to get flow: %v", err)
	}
	if retrievedContent != flowContent {
		t.Errorf("Expected flow content to match original")
	}
	t.Logf("✓ Retrieved flow content (length: %d chars)", len(retrievedContent))

	// 3. LIST
	t.Log("Step 3: Listing flows from DynamoDB...")
	flows, err := registry.List(accountID)
	if err != nil {
		t.Fatalf("Failed to list flows: %v", err)
	}
	if len(flows) == 0 {
		t.Fatal("Expected at least one flow in list")
	}
	t.Logf("✓ Listed %d flow(s)", len(flows))

	// 4. UPDATE
	t.Log("Step 4: Updating flow in DynamoDB...")
	updatedContent := `metadata:
  name: Updated DynamoDB Test Flow
  description: An updated flow for debugging CRUD operations with DynamoDB
  version: 1.1.0
nodes:
  start:
    type: test
    params:
      message: "Updated message from DynamoDB debug test"`

	err = registry.Update(accountID, flowID, updatedContent)
	if err != nil {
		t.Fatalf("Failed to update flow: %v", err)
	}
	t.Log("✓ Updated flow successfully")

	// Verify update worked
	retrievedUpdatedContent, err := registry.Get(accountID, flowID)
	if err != nil {
		t.Fatalf("Failed to get updated flow: %v", err)
	}
	if retrievedUpdatedContent != updatedContent {
		t.Errorf("Expected updated flow content to match")
	}
	t.Log("✓ Verified flow was updated")

	// 5. DELETE
	t.Log("Step 5: Deleting flow from DynamoDB...")
	err = registry.Delete(accountID, flowID)
	if err != nil {
		t.Fatalf("Failed to delete flow: %v", err)
	}
	t.Log("✓ Deleted flow successfully")

	// Verify deletion worked
	_, err = registry.Get(accountID, flowID)
	if err == nil {
		t.Fatal("Expected error when getting deleted flow, but got none")
	}
	t.Log("✓ Verified flow was deleted")

	// Cleanup - Delete test tables
	t.Log("Cleaning up test tables...")
	// Note: In a real test, you might want to clean up the tables
	// For local DynamoDB, we can leave them for debugging

	t.Log("All DynamoDB CRUD operations completed successfully!")
}