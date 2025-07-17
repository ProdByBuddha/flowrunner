package runtime

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/go-redis/redis/v8"
	"github.com/robfig/cron/v3"
	"github.com/tcmartin/flowlib"
)

// Global cron scheduler
var (
	cronScheduler *cron.Cron
	redisClient   *redis.Client
	initialized   bool
)

// CronJob represents a scheduled job
type CronJob struct {
	ID          string                 `json:"id"`
	Schedule    string                 `json:"schedule"`
	FlowID      string                 `json:"flow_id"`
	NodeID      string                 `json:"node_id"`
	Payload     map[string]interface{} `json:"payload"`
	NextRunTime time.Time              `json:"next_run_time"`
	LastRunTime time.Time              `json:"last_run_time,omitempty"`
	CreatedAt   time.Time              `json:"created_at"`
}

// initCronSystem initializes the cron system
func initCronSystem() error {
	if initialized {
		return nil
	}

	// Create Redis client
	redisClient = redis.NewClient(&redis.Options{
		Addr:     "localhost:6379", // Default Redis address
		Password: "",               // No password
		DB:       0,                // Default DB
	})

	// Test Redis connection
	ctx := context.Background()
	if err := redisClient.Ping(ctx).Err(); err != nil {
		return fmt.Errorf("failed to connect to Redis: %w", err)
	}

	// Create cron scheduler with seconds field
	cronScheduler = cron.New(cron.WithSeconds())
	cronScheduler.Start()

	// Load existing jobs from Redis
	loadExistingJobs(ctx)

	initialized = true
	return nil
}

// loadExistingJobs loads existing jobs from Redis
func loadExistingJobs(ctx context.Context) {
	// Get all job keys
	keys, err := redisClient.Keys(ctx, "cron:job:*").Result()
	if err != nil {
		fmt.Printf("Error loading existing jobs: %v\n", err)
		return
	}

	// Load each job
	for _, key := range keys {
		jobData, err := redisClient.Get(ctx, key).Result()
		if err != nil {
			fmt.Printf("Error loading job %s: %v\n", key, err)
			continue
		}

		var job CronJob
		if err := json.Unmarshal([]byte(jobData), &job); err != nil {
			fmt.Printf("Error unmarshaling job %s: %v\n", key, err)
			continue
		}

		// Schedule the job
		scheduleJob(job)
	}
}

// scheduleJob schedules a job with the cron scheduler
func scheduleJob(job CronJob) (cron.EntryID, error) {
	// Schedule the job
	id, err := cronScheduler.AddFunc(job.Schedule, func() {
		ctx := context.Background()

		// Update last run time
		job.LastRunTime = time.Now()

		// Calculate next run time - try both with and without seconds
		var parser cron.Schedule
		var err error

		// First try with seconds (6 fields)
		parser, err = cron.NewParser(cron.Second | cron.Minute | cron.Hour | cron.Dom | cron.Month | cron.Dow).Parse(job.Schedule)
		if err != nil {
			// Try standard format (5 fields)
			parser, err = cron.ParseStandard(job.Schedule)
			if err == nil {
				job.NextRunTime = parser.Next(time.Now())
			}
		} else {
			job.NextRunTime = parser.Next(time.Now())
		}

		// Save updated job to Redis
		jobData, err := json.Marshal(job)
		if err == nil {
			redisClient.Set(ctx, fmt.Sprintf("cron:job:%s", job.ID), jobData, 0)
		}

		// Execute the job
		// In a real implementation, this would trigger the flow execution
		fmt.Printf("Executing job %s for flow %s, node %s\n", job.ID, job.FlowID, job.NodeID)

		// Store execution record
		execRecord := map[string]interface{}{
			"job_id":      job.ID,
			"flow_id":     job.FlowID,
			"node_id":     job.NodeID,
			"executed_at": time.Now(),
			"payload":     job.Payload,
		}
		execData, err := json.Marshal(execRecord)
		if err == nil {
			redisClient.LPush(ctx, fmt.Sprintf("cron:executions:%s", job.ID), execData)
			// Trim the list to keep only the last 100 executions
			redisClient.LTrim(ctx, fmt.Sprintf("cron:executions:%s", job.ID), 0, 99)
		}
	})

	return id, err
}

// NewCronNodeWrapper creates a new cron node wrapper
func NewCronNodeWrapper(params map[string]interface{}) (flowlib.Node, error) {
	// Initialize cron system
	if err := initCronSystem(); err != nil {
		return nil, fmt.Errorf("failed to initialize cron system: %w", err)
	}

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

			// Get operation
			operation, _ := params["operation"].(string)
			if operation == "" {
				operation = "schedule" // Default operation
			}

			ctx := context.Background()

			switch operation {
			case "schedule":
				// Get schedule parameter
				schedule, ok := params["schedule"].(string)
				if !ok {
					return nil, fmt.Errorf("schedule parameter is required for schedule operation")
				}

				// Validate schedule - try both with and without seconds
				var parser cron.Schedule
				var err error

				// First try with seconds (6 fields)
				parser, err = cron.NewParser(cron.Second | cron.Minute | cron.Hour | cron.Dom | cron.Month | cron.Dow).Parse(schedule)
				if err != nil {
					// Try standard format (5 fields)
					parser, err = cron.ParseStandard(schedule)
					if err != nil {
						return nil, fmt.Errorf("invalid cron schedule: %w", err)
					}
				}

				// Get flow ID
				flowID, _ := params["flow_id"].(string)
				if flowID == "" {
					flowID = "current" // Default to current flow
				}

				// Get node ID
				nodeID, _ := params["node_id"].(string)
				if nodeID == "" {
					nodeID = "next" // Default to next node
				}

				// Get payload
				var payload map[string]interface{}
				if payloadParam, ok := params["payload"].(map[string]interface{}); ok {
					payload = payloadParam
				} else {
					payload = make(map[string]interface{})
				}

				// Create job ID
				jobID := fmt.Sprintf("%d", time.Now().UnixNano())
				if id, ok := params["id"].(string); ok && id != "" {
					jobID = id
				}

				// Calculate next run time using the already parsed schedule
				nextRunTime := parser.Next(time.Now())

				// Create job
				job := CronJob{
					ID:          jobID,
					Schedule:    schedule,
					FlowID:      flowID,
					NodeID:      nodeID,
					Payload:     payload,
					NextRunTime: nextRunTime,
					CreatedAt:   time.Now(),
				}

				// Save job to Redis
				jobData, err := json.Marshal(job)
				if err != nil {
					return nil, fmt.Errorf("failed to marshal job: %w", err)
				}

				if err := redisClient.Set(ctx, fmt.Sprintf("cron:job:%s", jobID), jobData, 0).Err(); err != nil {
					return nil, fmt.Errorf("failed to save job to Redis: %w", err)
				}

				// Schedule job
				entryID, err := scheduleJob(job)
				if err != nil {
					return nil, fmt.Errorf("failed to schedule job: %w", err)
				}

				return map[string]interface{}{
					"job_id":        jobID,
					"entry_id":      int(entryID),
					"schedule":      schedule,
					"next_run_time": nextRunTime,
				}, nil

			case "list":
				// Get all job keys
				keys, err := redisClient.Keys(ctx, "cron:job:*").Result()
				if err != nil {
					return nil, fmt.Errorf("failed to list jobs: %w", err)
				}

				// Load each job
				jobs := make([]map[string]interface{}, 0, len(keys))
				for _, key := range keys {
					jobData, err := redisClient.Get(ctx, key).Result()
					if err != nil {
						continue
					}

					var job CronJob
					if err := json.Unmarshal([]byte(jobData), &job); err != nil {
						continue
					}

					jobs = append(jobs, map[string]interface{}{
						"id":            job.ID,
						"schedule":      job.Schedule,
						"flow_id":       job.FlowID,
						"node_id":       job.NodeID,
						"next_run_time": job.NextRunTime,
						"last_run_time": job.LastRunTime,
						"created_at":    job.CreatedAt,
					})
				}

				return map[string]interface{}{
					"jobs": jobs,
				}, nil

			case "get":
				// Get job ID
				jobID, ok := params["id"].(string)
				if !ok {
					return nil, fmt.Errorf("id parameter is required for get operation")
				}

				// Get job from Redis
				jobData, err := redisClient.Get(ctx, fmt.Sprintf("cron:job:%s", jobID)).Result()
				if err != nil {
					return nil, fmt.Errorf("job not found: %w", err)
				}

				var job CronJob
				if err := json.Unmarshal([]byte(jobData), &job); err != nil {
					return nil, fmt.Errorf("failed to unmarshal job: %w", err)
				}

				// Get executions
				executions := make([]map[string]interface{}, 0)
				execData, err := redisClient.LRange(ctx, fmt.Sprintf("cron:executions:%s", jobID), 0, 9).Result()
				if err == nil {
					for _, data := range execData {
						var exec map[string]interface{}
						if err := json.Unmarshal([]byte(data), &exec); err == nil {
							executions = append(executions, exec)
						}
					}
				}

				return map[string]interface{}{
					"id":            job.ID,
					"schedule":      job.Schedule,
					"flow_id":       job.FlowID,
					"node_id":       job.NodeID,
					"payload":       job.Payload,
					"next_run_time": job.NextRunTime,
					"last_run_time": job.LastRunTime,
					"created_at":    job.CreatedAt,
					"executions":    executions,
				}, nil

			case "delete":
				// Get job ID
				jobID, ok := params["id"].(string)
				if !ok {
					return nil, fmt.Errorf("id parameter is required for delete operation")
				}

				// Delete job from Redis
				if err := redisClient.Del(ctx, fmt.Sprintf("cron:job:%s", jobID)).Err(); err != nil {
					return nil, fmt.Errorf("failed to delete job: %w", err)
				}

				// Note: We can't easily remove the job from the cron scheduler
				// In a real implementation, we would need to track the entry IDs

				return map[string]interface{}{
					"deleted": true,
					"id":      jobID,
				}, nil

			default:
				return nil, fmt.Errorf("unknown operation: %s", operation)
			}
		},
	}

	// Set the parameters
	wrapper.SetParams(params)

	return wrapper, nil
}
