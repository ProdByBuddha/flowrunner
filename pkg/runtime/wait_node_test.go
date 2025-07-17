package runtime

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestWaitNode(t *testing.T) {
	// Test duration wait
	t.Run("Duration wait", func(t *testing.T) {
		node, err := NewWaitNodeWrapper(map[string]interface{}{
			"type":     "duration",
			"duration": "100ms",
		})
		assert.NoError(t, err)
		assert.NotNil(t, node)

		start := time.Now()
		result, err := node.(*NodeWrapper).exec(map[string]interface{}{
			"type":     "duration",
			"duration": "100ms",
		})
		elapsed := time.Since(start)

		assert.NoError(t, err)
		assert.NotNil(t, result)
		assert.GreaterOrEqual(t, elapsed.Milliseconds(), int64(100))

		resultMap, ok := result.(map[string]interface{})
		assert.True(t, ok)
		assert.Equal(t, "duration", resultMap["type"])
		assert.Equal(t, "100ms", resultMap["waited_for"])
	})

	// Test until_time wait (with a time in the past, should not wait)
	t.Run("Until time (past)", func(t *testing.T) {
		pastTime := time.Now().Add(-1 * time.Hour).Format(time.RFC3339)

		node, err := NewWaitNodeWrapper(map[string]interface{}{
			"type": "until_time",
			"time": pastTime,
		})
		assert.NoError(t, err)
		assert.NotNil(t, node)

		start := time.Now()
		result, err := node.(*NodeWrapper).exec(map[string]interface{}{
			"type": "until_time",
			"time": pastTime,
		})
		elapsed := time.Since(start)

		assert.NoError(t, err)
		assert.NotNil(t, result)
		assert.Less(t, elapsed.Milliseconds(), int64(100)) // Should return immediately

		resultMap, ok := result.(map[string]interface{})
		assert.True(t, ok)
		assert.Equal(t, "until_time", resultMap["type"])
		assert.Equal(t, pastTime, resultMap["waited_until"])
	})

	// Test condition wait
	t.Run("Condition wait", func(t *testing.T) {
		node, err := NewWaitNodeWrapper(map[string]interface{}{
			"type":         "condition",
			"max_attempts": float64(3),
			"interval":     "10ms",
		})
		assert.NoError(t, err)
		assert.NotNil(t, node)

		start := time.Now()
		result, err := node.(*NodeWrapper).exec(map[string]interface{}{
			"type":         "condition",
			"max_attempts": float64(3),
			"interval":     "10ms",
		})
		elapsed := time.Since(start)

		assert.NoError(t, err)
		assert.NotNil(t, result)
		assert.GreaterOrEqual(t, elapsed.Milliseconds(), int64(30)) // Should wait for 3 intervals of 10ms

		resultMap, ok := result.(map[string]interface{})
		assert.True(t, ok)
		assert.Equal(t, "condition", resultMap["type"])
		assert.Equal(t, 3, resultMap["attempts"])
		assert.Equal(t, "10ms", resultMap["interval"])
		assert.Equal(t, true, resultMap["completed"])
	})
}
