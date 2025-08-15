package runtime

import (
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestStoreNode(t *testing.T) {
	// Use a test-specific file path
	testFilePath := "test_store.json"

	// Clean up after the test
	defer os.Remove(testFilePath)

	// Create the node
	node, err := NewStoreNodeWrapper(map[string]interface{}{
		"file_path": testFilePath,
		"auto_save": true,
	})
	assert.NoError(t, err)
	assert.NotNil(t, node)

	// Test basic operations
	t.Run("Basic operations", func(t *testing.T) {
		// Test set operation
		result, err := node.(*NodeWrapper).exec(map[string]interface{}{
			"operation": "set",
			"key":       "test-key",
			"value":     "test-value",
		})
		assert.NoError(t, err)
		assert.Equal(t, "test-value", result)

		// Test get operation
		result, err = node.(*NodeWrapper).exec(map[string]interface{}{
			"operation": "get",
			"key":       "test-key",
		})
		assert.NoError(t, err)
		assert.Equal(t, "test-value", result)

		// Test list operation
		result, err = node.(*NodeWrapper).exec(map[string]interface{}{
			"operation": "list",
		})
		assert.NoError(t, err)
		keys, ok := result.([]string)
		assert.True(t, ok)
		assert.Contains(t, keys, "test-key")

		// Test delete operation
		result, err = node.(*NodeWrapper).exec(map[string]interface{}{
			"operation": "delete",
			"key":       "test-key",
		})
		assert.NoError(t, err)
		deleteResult, ok := result.(map[string]interface{})
		assert.True(t, ok)
		assert.Equal(t, true, deleteResult["deleted"])

		// Test get after delete (should fail)
		_, err = node.(*NodeWrapper).exec(map[string]interface{}{
			"operation": "get",
			"key":       "test-key",
		})
		assert.Error(t, err)
	})

	// Test TTL functionality
	t.Run("TTL functionality", func(t *testing.T) {
		// Set a key with a short TTL
		_, err := node.(*NodeWrapper).exec(map[string]interface{}{
			"operation": "set",
			"key":       "ttl-key",
			"value":     "ttl-value",
			"ttl":       "100ms",
		})
		assert.NoError(t, err)

		// Verify the key exists
		result, err := node.(*NodeWrapper).exec(map[string]interface{}{
			"operation": "get",
			"key":       "ttl-key",
		})
		assert.NoError(t, err)
		assert.Equal(t, "ttl-value", result)

		// Wait for TTL to expire
		time.Sleep(200 * time.Millisecond)

		// Verify the key no longer exists
		_, err = node.(*NodeWrapper).exec(map[string]interface{}{
			"operation": "get",
			"key":       "ttl-key",
		})
		assert.Error(t, err)
	})

	// Test increment operation
	t.Run("Increment operation", func(t *testing.T) {
		// Set initial value
		_, err := node.(*NodeWrapper).exec(map[string]interface{}{
			"operation": "set",
			"key":       "counter",
			"value":     10,
		})
		assert.NoError(t, err)

		// Increment the value
		result, err := node.(*NodeWrapper).exec(map[string]interface{}{
			"operation": "increment",
			"key":       "counter",
			"amount":    5.0,
		})
		assert.NoError(t, err)

		incrementResult, ok := result.(map[string]interface{})
		assert.True(t, ok)
		assert.Equal(t, 15.0, incrementResult["new_value"])

		// Verify the incremented value
		result, err = node.(*NodeWrapper).exec(map[string]interface{}{
			"operation": "get",
			"key":       "counter",
		})
		assert.NoError(t, err)
		assert.Equal(t, 15.0, result)
	})

	// Test append operation
	t.Run("Append operation", func(t *testing.T) {
		// Set initial array
		_, err := node.(*NodeWrapper).exec(map[string]interface{}{
			"operation": "set",
			"key":       "array",
			"value":     []interface{}{"item1", "item2"},
		})
		assert.NoError(t, err)

		// Append to the array
		result, err := node.(*NodeWrapper).exec(map[string]interface{}{
			"operation": "append",
			"key":       "array",
			"value":     "item3",
		})
		assert.NoError(t, err)

		appendResult, ok := result.(map[string]interface{})
		assert.True(t, ok)
		assert.Equal(t, "item3", appendResult["appended"])

		// Verify the appended value
		result, err = node.(*NodeWrapper).exec(map[string]interface{}{
			"operation": "get",
			"key":       "array",
		})
		assert.NoError(t, err)
		array, ok := result.([]interface{})
		assert.True(t, ok)
		assert.Equal(t, 3, len(array))
		assert.Equal(t, "item3", array[2])
	})

	// Test query operation
	t.Run("Query operation", func(t *testing.T) {
		// Set up some test data
		_, err := node.(*NodeWrapper).exec(map[string]interface{}{
			"operation": "set",
			"key":       "user1",
			"value": map[string]interface{}{
				"name":  "Alice",
				"age":   30,
				"roles": []interface{}{"admin", "user"},
			},
		})
		assert.NoError(t, err)

		_, err = node.(*NodeWrapper).exec(map[string]interface{}{
			"operation": "set",
			"key":       "user2",
			"value": map[string]interface{}{
				"name":  "Bob",
				"age":   25,
				"roles": []interface{}{"user"},
			},
		})
		assert.NoError(t, err)

		_, err = node.(*NodeWrapper).exec(map[string]interface{}{
			"operation": "set",
			"key":       "user3",
			"value": map[string]interface{}{
				"name":  "Charlie",
				"age":   35,
				"roles": []interface{}{"user"},
			},
		})
		assert.NoError(t, err)

		// Query with filter
		result, err := node.(*NodeWrapper).exec(map[string]interface{}{
			"operation": "query",
			"filter": map[string]interface{}{
				"age": map[string]interface{}{
					"$gt": 25,
				},
			},
		})
		assert.NoError(t, err)

		queryResults, ok := result.([]map[string]interface{})
		assert.True(t, ok)
		assert.Equal(t, 2, len(queryResults))

		// Check that we got the right users
		var foundAlice, foundCharlie bool
		for _, user := range queryResults {
			if user["name"] == "Alice" {
				foundAlice = true
			}
			if user["name"] == "Charlie" {
				foundCharlie = true
			}
		}
		assert.True(t, foundAlice)
		assert.True(t, foundCharlie)

		// Query with sort
		result, err = node.(*NodeWrapper).exec(map[string]interface{}{
			"operation": "query",
			"sort":      "age",
		})
		assert.NoError(t, err)

		sortedResults, ok := result.([]map[string]interface{})
		assert.True(t, ok)
		assert.Equal(t, 3, len(sortedResults))
		assert.Equal(t, "Bob", sortedResults[0]["name"])
		assert.Equal(t, "Alice", sortedResults[1]["name"])
		assert.Equal(t, "Charlie", sortedResults[2]["name"])

		// Query with limit
		result, err = node.(*NodeWrapper).exec(map[string]interface{}{
			"operation": "query",
			"sort":      "age",
			"limit":     2.0,
		})
		assert.NoError(t, err)

		limitedResults, ok := result.([]map[string]interface{})
		assert.True(t, ok)
		assert.Equal(t, 2, len(limitedResults))
		assert.Equal(t, "Bob", limitedResults[0]["name"])
		assert.Equal(t, "Alice", limitedResults[1]["name"])
	})

	// Test persistence
	t.Run("Persistence", func(t *testing.T) {
		// Set a key
		_, err := node.(*NodeWrapper).exec(map[string]interface{}{
			"operation": "set",
			"key":       "persistent-key",
			"value":     "persistent-value",
		})
		assert.NoError(t, err)

		// Save explicitly
		_, err = node.(*NodeWrapper).exec(map[string]interface{}{
			"operation": "save",
		})
		assert.NoError(t, err)

		// Create a new node with the same file path
		newNode, err := NewStoreNodeWrapper(map[string]interface{}{
			"file_path": testFilePath,
		})
		assert.NoError(t, err)

		// Load explicitly
		_, err = newNode.(*NodeWrapper).exec(map[string]interface{}{
			"operation": "load",
		})
		assert.NoError(t, err)

		// Verify the key exists in the new node
		result, err := newNode.(*NodeWrapper).exec(map[string]interface{}{
			"operation": "get",
			"key":       "persistent-key",
		})
		assert.NoError(t, err)
		assert.Equal(t, "persistent-value", result)
	})
}
