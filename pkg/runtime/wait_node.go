package runtime

import (
	"fmt"
	"time"

	"github.com/tcmartin/flowlib"
)

// NewWaitNodeWrapper creates a new wait node wrapper
// This node is more advanced than the delay node and can handle different types of wait conditions
func NewWaitNodeWrapper(params map[string]interface{}) (flowlib.Node, error) {
	// Create the base node
	baseNode := flowlib.NewNode(3, 1*time.Second)

	// Create the wrapper
	wrapper := &NodeWrapper{
		node: baseNode,
		exec: func(input interface{}) (interface{}, error) {
			// Get parameters from input
			params, ok := input.(map[string]interface{})
			if !ok {
				return nil, fmt.Errorf("expected map[string]interface{}, got %T", input)
			}

			// Check the wait type
			waitType, _ := params["type"].(string)
			switch waitType {
			case "duration", "": // Default is duration
				// Get duration parameter
				durationStr, ok := params["duration"].(string)
				if !ok {
					return nil, fmt.Errorf("duration parameter is required for duration wait type")
				}

				// Parse duration
				duration, err := time.ParseDuration(durationStr)
				if err != nil {
					return nil, fmt.Errorf("invalid duration: %w", err)
				}

				// Wait
				time.Sleep(duration)

				return map[string]interface{}{
					"waited_for": durationStr,
					"type":       "duration",
				}, nil

			case "until_time":
				// Get time parameter
				timeStr, ok := params["time"].(string)
				if !ok {
					return nil, fmt.Errorf("time parameter is required for until_time wait type")
				}

				// Parse time
				targetTime, err := time.Parse(time.RFC3339, timeStr)
				if err != nil {
					return nil, fmt.Errorf("invalid time format, expected RFC3339 (e.g., 2006-01-02T15:04:05Z): %w", err)
				}

				// Calculate wait duration
				now := time.Now()
				if targetTime.After(now) {
					waitDuration := targetTime.Sub(now)
					time.Sleep(waitDuration)
				}

				return map[string]interface{}{
					"waited_until": timeStr,
					"type":         "until_time",
				}, nil

			case "condition":
				// This would use the JavaScript engine to evaluate a condition
				// For now, we'll just implement a simple polling mechanism

				// Get condition parameters
				maxAttempts := 10
				if maxParam, ok := params["max_attempts"].(float64); ok {
					maxAttempts = int(maxParam)
				}

				interval := time.Second
				if intervalStr, ok := params["interval"].(string); ok {
					if parsedInterval, err := time.ParseDuration(intervalStr); err == nil {
						interval = parsedInterval
					}
				}

				// Timeout is not used in this simple implementation
				// but would be used in a real implementation to limit the total wait time
				_ = time.Minute * 5
				if timeoutStr, ok := params["timeout"].(string); ok {
					if _, err := time.ParseDuration(timeoutStr); err != nil {
						return nil, fmt.Errorf("invalid timeout format: %w", err)
					}
				}

				// For demonstration, we'll just wait for the specified number of attempts
				for i := 0; i < maxAttempts; i++ {
					// In a real implementation, we would evaluate the condition here
					// For now, we'll just sleep for the interval
					time.Sleep(interval)
				}

				return map[string]interface{}{
					"attempts":  maxAttempts,
					"interval":  interval.String(),
					"type":      "condition",
					"completed": true,
				}, nil

			default:
				return nil, fmt.Errorf("unknown wait type: %s", waitType)
			}
		},
	}

	// Set the parameters
	wrapper.SetParams(params)

	return wrapper, nil
}
