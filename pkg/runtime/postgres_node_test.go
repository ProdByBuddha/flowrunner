package runtime

import (
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/joho/godotenv"
	"github.com/stretchr/testify/assert"
)

func TestPostgresNode(t *testing.T) {
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

	// Skip test if PostgreSQL credentials are not set
	pgHost := os.Getenv("POSTGRES_HOST")
	pgUser := os.Getenv("POSTGRES_USER")
	pgPassword := os.Getenv("POSTGRES_PASSWORD")
	pgDB := os.Getenv("POSTGRES_DB")

	// Print the credentials for debugging
	t.Logf("POSTGRES_HOST: %s", pgHost)
	t.Logf("POSTGRES_USER: %s", pgUser)
	t.Logf("POSTGRES_DB: %s", pgDB)
	if len(pgPassword) > 0 {
		t.Logf("POSTGRES_PASSWORD: %s...", pgPassword[:1])
	} else {
		t.Logf("POSTGRES_PASSWORD is empty")
	}

	if pgHost == "" || pgUser == "" || pgDB == "" {
		t.Skip("Skipping PostgreSQL test as credentials are not set")
	}

	// Use a test-specific table name
	testTableName := "flowrunner_test_" + time.Now().Format("20060102150405")

	// Print the parameters we're passing to the node
	fmt.Printf("PostgreSQL test parameters: host=%s, user=%s, dbname=%s, table_name=%s\n",
		pgHost, pgUser, pgDB, testTableName)

	// Create the node
	node, err := NewPostgresNodeWrapper(map[string]interface{}{
		"host":       pgHost,
		"user":       pgUser,
		"password":   pgPassword,
		"dbname":     pgDB,
		"table_name": testTableName,
	})
	assert.NoError(t, err)
	assert.NotNil(t, node)

	// Clean up after the test
	defer func() {
		// Drop the test table
		_, err := node.(*NodeWrapper).exec(map[string]interface{}{
			"operation": "execute",
			"query":     "DROP TABLE IF EXISTS " + testTableName,
		})
		if err != nil {
			t.Logf("Failed to drop test table: %v", err)
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
		assert.GreaterOrEqual(t, len(queryResults), 2)

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
	})

	// Test SQL execution
	t.Run("SQL execution", func(t *testing.T) {
		// Drop the test table if it exists
		_, err := node.(*NodeWrapper).exec(map[string]interface{}{
			"operation": "execute",
			"query":     "DROP TABLE IF EXISTS test_sql",
		})
		assert.NoError(t, err)

		// Create a test table
		result, err := node.(*NodeWrapper).exec(map[string]interface{}{
			"operation": "execute",
			"query":     "CREATE TABLE test_sql (id SERIAL PRIMARY KEY, name TEXT, value INTEGER)",
		})
		assert.NoError(t, err)

		// Insert data
		result, err = node.(*NodeWrapper).exec(map[string]interface{}{
			"operation": "execute",
			"query":     "INSERT INTO test_sql (name, value) VALUES ($1, $2), ($3, $4)",
			"args":      []interface{}{"test1", 100, "test2", 200},
		})
		assert.NoError(t, err)
		execResult, ok := result.(map[string]interface{})
		assert.True(t, ok)
		assert.Equal(t, int64(2), execResult["rows_affected"])

		// Query data
		result, err = node.(*NodeWrapper).exec(map[string]interface{}{
			"operation": "execute",
			"query":     "SELECT * FROM test_sql ORDER BY id",
		})
		assert.NoError(t, err)
		rows, ok := result.([]map[string]interface{})
		assert.True(t, ok)
		assert.Equal(t, 2, len(rows))
		assert.Equal(t, "test1", rows[0]["name"])
		assert.Equal(t, int64(100), rows[0]["value"])
		assert.Equal(t, "test2", rows[1]["name"])
		assert.Equal(t, int64(200), rows[1]["value"])
	})

	// Test transaction
	t.Run("Transaction", func(t *testing.T) {
		// Drop the test table if it exists
		_, err := node.(*NodeWrapper).exec(map[string]interface{}{
			"operation": "execute",
			"query":     "DROP TABLE IF EXISTS test_tx",
		})
		assert.NoError(t, err)

		// Create a test table
		_, err = node.(*NodeWrapper).exec(map[string]interface{}{
			"operation": "execute",
			"query":     "CREATE TABLE test_tx (id SERIAL PRIMARY KEY, name TEXT, value INTEGER)",
		})
		assert.NoError(t, err)

		// Execute transaction
		result, err := node.(*NodeWrapper).exec(map[string]interface{}{
			"operation": "transaction",
			"statements": []interface{}{
				map[string]interface{}{
					"query": "INSERT INTO test_tx (name, value) VALUES ($1, $2)",
					"args":  []interface{}{"tx1", 100},
				},
				map[string]interface{}{
					"query": "INSERT INTO test_tx (name, value) VALUES ($1, $2)",
					"args":  []interface{}{"tx2", 200},
				},
				map[string]interface{}{
					"query": "SELECT * FROM test_tx ORDER BY id",
				},
			},
		})
		assert.NoError(t, err)

		txResult, ok := result.(map[string]interface{})
		assert.True(t, ok)
		results, ok := txResult["results"].([]interface{})
		assert.True(t, ok)
		assert.Equal(t, 3, len(results))

		// Check first statement result (INSERT)
		insert1, ok := results[0].(map[string]interface{})
		assert.True(t, ok)
		assert.Equal(t, int64(1), insert1["rows_affected"])

		// Check second statement result (INSERT)
		insert2, ok := results[1].(map[string]interface{})
		assert.True(t, ok)
		assert.Equal(t, int64(1), insert2["rows_affected"])

		// Check third statement result (SELECT)
		select1, ok := results[2].([]map[string]interface{})
		assert.True(t, ok)
		assert.Equal(t, 2, len(select1))
		assert.Equal(t, "tx1", select1[0]["name"])
		assert.Equal(t, int64(100), select1[0]["value"])
		assert.Equal(t, "tx2", select1[1]["name"])
		assert.Equal(t, int64(200), select1[1]["value"])
	})
}
