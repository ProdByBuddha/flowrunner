package runtime

import (
	"os"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/dynamodb"
	"github.com/joho/godotenv"
	"github.com/stretchr/testify/assert"
)

func TestDynamoDBNode(t *testing.T) {
	// Load .env file from project root
	err := godotenv.Load()
	if err != nil {
		// Try to load from parent directory if not found in current directory
		err = godotenv.Load("../../.env")
		if err != nil {
			// Try to load from the absolute path
			err = godotenv.Load("../.env")
			if err != nil {
				t.Log("Warning: .env file not found, using environment variables")
			}
		}
	}

	// Get AWS credentials and print for debugging
	accessKey := os.Getenv("AWS_ACCESS_KEY_ID")
	secretKey := os.Getenv("AWS_SECRET_ACCESS_KEY")
	t.Logf("AWS_ACCESS_KEY_ID: %s", accessKey)
	if len(secretKey) > 4 {
		t.Logf("AWS_SECRET_ACCESS_KEY: %s...", secretKey[:4])
	} else {
		t.Logf("AWS_SECRET_ACCESS_KEY: %s", secretKey)
	}

	// Skip test if AWS credentials are not set
	if accessKey == "" || secretKey == "" {
		t.Skip("Skipping DynamoDB test as AWS credentials are not set")
	}

	// Use a test-specific table name
	testTableName := "flowrunner_test_" + time.Now().Format("20060102150405")

	// Create the node
	node, err := NewDynamoDBNodeWrapper(map[string]interface{}{
		"region":     "us-east-1",
		"table_name": testTableName,
	})
	assert.NoError(t, err)
	assert.NotNil(t, node)

	// Clean up after the test
	defer func() {
		// Create AWS session
		sess, err := session.NewSession(&aws.Config{
			Region:      aws.String("us-east-1"),
			Credentials: credentials.NewStaticCredentials(accessKey, secretKey, ""),
		})
		if err != nil {
			t.Logf("Failed to create AWS session for cleanup: %v", err)
			return
		}

		// Create DynamoDB client
		client := dynamodb.New(sess)

		// Delete the test table
		_, err = client.DeleteTable(&dynamodb.DeleteTableInput{
			TableName: aws.String(testTableName),
		})
		if err != nil {
			t.Logf("Failed to delete test table: %v", err)
		}
	}()

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

		// Check that we got results
		assert.NotEmpty(t, queryResults)

		// Check for users with age > 25
		var foundAlice, foundCharlie bool
		for _, user := range queryResults {
			if user["value"].(map[string]interface{})["name"] == "Alice" {
				foundAlice = true
			}
			if user["value"].(map[string]interface{})["name"] == "Charlie" {
				foundCharlie = true
			}
		}
		assert.True(t, foundAlice || foundCharlie)
	})

	// Test batch write operation
	t.Run("Batch write operation", func(t *testing.T) {
		// Create batch items
		items := []interface{}{
			map[string]interface{}{
				"key": "batch1",
				"value": map[string]interface{}{
					"name": "Batch Item 1",
					"type": "test",
				},
			},
			map[string]interface{}{
				"key": "batch2",
				"value": map[string]interface{}{
					"name": "Batch Item 2",
					"type": "test",
				},
			},
		}

		// Batch write
		result, err := node.(*NodeWrapper).exec(map[string]interface{}{
			"operation": "batch_write",
			"items":     items,
		})
		assert.NoError(t, err)

		batchResult, ok := result.(map[string]interface{})
		assert.True(t, ok)
		assert.Equal(t, 2, batchResult["written"])

		// Verify items were written
		result, err = node.(*NodeWrapper).exec(map[string]interface{}{
			"operation": "get",
			"key":       "batch1",
		})
		assert.NoError(t, err)
		item1, ok := result.(map[string]interface{})
		assert.True(t, ok)
		assert.Equal(t, "Batch Item 1", item1["name"])

		result, err = node.(*NodeWrapper).exec(map[string]interface{}{
			"operation": "get",
			"key":       "batch2",
		})
		assert.NoError(t, err)
		item2, ok := result.(map[string]interface{})
		assert.True(t, ok)
		assert.Equal(t, "Batch Item 2", item2["name"])
	})
}
