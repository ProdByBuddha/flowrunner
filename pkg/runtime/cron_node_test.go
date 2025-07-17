package runtime

import (
	"context"
	"testing"

	"github.com/alicebob/miniredis/v2"
	"github.com/go-redis/redis/v8"
	"github.com/stretchr/testify/assert"
)

func TestCronNode(t *testing.T) {
	// Start a mock Redis server
	s, err := miniredis.Run()
	if err != nil {
		t.Fatalf("Failed to start miniredis: %v", err)
	}
	defer s.Close()

	// Replace the Redis client with our mock
	originalRedisClient := redisClient
	redisClient = redis.NewClient(&redis.Options{
		Addr: s.Addr(),
	})
	defer func() {
		redisClient = originalRedisClient
	}()

	// Reset initialized flag to force initialization with our mock Redis
	initialized = false

	// Test scheduling a job
	t.Run("Schedule job", func(t *testing.T) {
		params := map[string]interface{}{
			"operation": "schedule",
			"schedule":  "0 */5 * * * *", // Every 5 minutes (with seconds field)
			"flow_id":   "test_flow",
			"node_id":   "test_node",
			"id":        "test_job",
			"payload": map[string]interface{}{
				"key": "value",
			},
		}

		node, err := NewCronNodeWrapper(params)
		assert.NoError(t, err)
		assert.NotNil(t, node)

		result, err := node.(*NodeWrapper).exec(params)
		assert.NoError(t, err)
		assert.NotNil(t, result)

		resultMap, ok := result.(map[string]interface{})
		assert.True(t, ok)
		assert.Equal(t, "test_job", resultMap["job_id"])

		// Verify the job was saved to Redis
		ctx := context.Background()
		exists, err := redisClient.Exists(ctx, "cron:job:test_job").Result()
		assert.NoError(t, err)
		assert.Equal(t, int64(1), exists)
	})

	// Test listing jobs
	t.Run("List jobs", func(t *testing.T) {
		node, err := NewCronNodeWrapper(map[string]interface{}{
			"operation": "list",
		})
		assert.NoError(t, err)
		assert.NotNil(t, node)

		result, err := node.(*NodeWrapper).exec(map[string]interface{}{
			"operation": "list",
		})
		assert.NoError(t, err)
		assert.NotNil(t, result)

		resultMap, ok := result.(map[string]interface{})
		assert.True(t, ok)

		jobs, ok := resultMap["jobs"].([]map[string]interface{})
		assert.True(t, ok)
		assert.GreaterOrEqual(t, len(jobs), 1)

		// Find our test job
		var found bool
		for _, job := range jobs {
			if job["id"] == "test_job" {
				found = true
				break
			}
		}
		assert.True(t, found, "Expected to find job with ID 'test_job'")
	})

	// Test getting a job
	t.Run("Get job", func(t *testing.T) {
		node, err := NewCronNodeWrapper(map[string]interface{}{
			"operation": "get",
			"id":        "test_job",
		})
		assert.NoError(t, err)
		assert.NotNil(t, node)

		result, err := node.(*NodeWrapper).exec(map[string]interface{}{
			"operation": "get",
			"id":        "test_job",
		})
		assert.NoError(t, err)
		assert.NotNil(t, result)

		resultMap, ok := result.(map[string]interface{})
		assert.True(t, ok)
		assert.Equal(t, "test_job", resultMap["id"])
		assert.Equal(t, "test_flow", resultMap["flow_id"])
		assert.Equal(t, "test_node", resultMap["node_id"])

		// Check payload
		payload, ok := resultMap["payload"].(map[string]interface{})
		assert.True(t, ok)
		assert.Equal(t, "value", payload["key"])
	})

	// Test deleting a job
	t.Run("Delete job", func(t *testing.T) {
		node, err := NewCronNodeWrapper(map[string]interface{}{
			"operation": "delete",
			"id":        "test_job",
		})
		assert.NoError(t, err)
		assert.NotNil(t, node)

		result, err := node.(*NodeWrapper).exec(map[string]interface{}{
			"operation": "delete",
			"id":        "test_job",
		})
		assert.NoError(t, err)
		assert.NotNil(t, result)

		resultMap, ok := result.(map[string]interface{})
		assert.True(t, ok)
		assert.Equal(t, true, resultMap["deleted"])
		assert.Equal(t, "test_job", resultMap["id"])

		// Verify the job is deleted by trying to get it
		ctx := context.Background()
		exists, err := redisClient.Exists(ctx, "cron:job:test_job").Result()
		assert.NoError(t, err)
		assert.Equal(t, int64(0), exists)
	})
}
