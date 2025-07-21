package runtime

import (
	"fmt"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/dynamodb"
	"github.com/aws/aws-sdk-go/service/dynamodb/dynamodbattribute"
	"github.com/aws/aws-sdk-go/service/dynamodb/expression"
	"github.com/tcmartin/flowlib"
)

// DynamoDBManager manages DynamoDB operations
type DynamoDBManager struct {
	client    *dynamodb.DynamoDB
	tableName string
}

// Global DynamoDB manager
var globalDynamoDBManager *DynamoDBManager

// GetDynamoDBManager returns the global DynamoDB manager
func GetDynamoDBManager(config map[string]interface{}) (*DynamoDBManager, error) {
	if globalDynamoDBManager != nil {
		return globalDynamoDBManager, nil
	}

	// Extract AWS configuration
	region, _ := config["region"].(string)
	if region == "" {
		region = "us-east-1" // Default region
	}

	// Create AWS session
	var sess *session.Session
	var err error

	// Check if credentials are provided
	accessKey, hasAccessKey := config["access_key"].(string)
	secretKey, hasSecretKey := config["secret_key"].(string)

	if hasAccessKey && hasSecretKey {
		// Use provided credentials
		sess, err = session.NewSession(&aws.Config{
			Region:      aws.String(region),
			Credentials: credentials.NewStaticCredentials(accessKey, secretKey, ""),
		})
	} else {
		// Use default credential provider chain
		sess, err = session.NewSession(&aws.Config{
			Region: aws.String(region),
		})
	}

	if err != nil {
		return nil, fmt.Errorf("failed to create AWS session: %w", err)
	}

	// Extract table name
	tableName, _ := config["table_name"].(string)
	if tableName == "" {
		tableName = "flowrunner_store" // Default table name
	}

	// Create DynamoDB client
	client := dynamodb.New(sess)

	// Create manager
	globalDynamoDBManager = &DynamoDBManager{
		client:    client,
		tableName: tableName,
	}

	// Ensure table exists
	if err := globalDynamoDBManager.ensureTableExists(); err != nil {
		return nil, fmt.Errorf("failed to ensure table exists: %w", err)
	}

	return globalDynamoDBManager, nil
}

// ensureTableExists creates the DynamoDB table if it doesn't exist
func (dm *DynamoDBManager) ensureTableExists() error {
	// Check if table exists
	_, err := dm.client.DescribeTable(&dynamodb.DescribeTableInput{
		TableName: aws.String(dm.tableName),
	})

	if err == nil {
		// Table exists
		return nil
	}

	// Create table
	_, err = dm.client.CreateTable(&dynamodb.CreateTableInput{
		TableName: aws.String(dm.tableName),
		AttributeDefinitions: []*dynamodb.AttributeDefinition{
			{
				AttributeName: aws.String("key"),
				AttributeType: aws.String("S"),
			},
		},
		KeySchema: []*dynamodb.KeySchemaElement{
			{
				AttributeName: aws.String("key"),
				KeyType:       aws.String("HASH"),
			},
		},
		BillingMode: aws.String("PAY_PER_REQUEST"),
	})

	if err != nil {
		return fmt.Errorf("failed to create table: %w", err)
	}

	// Wait for table to be created
	err = dm.client.WaitUntilTableExists(&dynamodb.DescribeTableInput{
		TableName: aws.String(dm.tableName),
	})

	if err != nil {
		return fmt.Errorf("failed to wait for table creation: %w", err)
	}

	return nil
}

// Get retrieves an item from DynamoDB
func (dm *DynamoDBManager) Get(key string) (interface{}, error) {
	// Get item
	result, err := dm.client.GetItem(&dynamodb.GetItemInput{
		TableName: aws.String(dm.tableName),
		Key: map[string]*dynamodb.AttributeValue{
			"key": {
				S: aws.String(key),
			},
		},
	})

	if err != nil {
		return nil, fmt.Errorf("failed to get item: %w", err)
	}

	// Check if item exists
	if result.Item == nil {
		return nil, fmt.Errorf("item not found: %s", key)
	}

	// Check if item has expired
	if ttl, ok := result.Item["ttl"]; ok {
		if ttl.N != nil {
			var expiryTimestamp int64
			if _, err := fmt.Sscanf(*ttl.N, "%d", &expiryTimestamp); err == nil {
				expiryTime := time.Unix(expiryTimestamp, 0)
				if time.Now().After(expiryTime) {
					// Item has expired
					return nil, fmt.Errorf("item not found: %s", key)
				}
			}
		}
	}

	// Unmarshal item
	var item map[string]interface{}
	if err := dynamodbattribute.UnmarshalMap(result.Item, &item); err != nil {
		return nil, fmt.Errorf("failed to unmarshal item: %w", err)
	}

	// Return value
	return item["value"], nil
}

// Set stores an item in DynamoDB
func (dm *DynamoDBManager) Set(key string, value interface{}, ttl time.Duration) error {
	// Create item
	item := map[string]interface{}{
		"key":       key,
		"value":     value,
		"timestamp": time.Now().Unix(),
	}

	// Add TTL if provided
	if ttl > 0 {
		item["ttl"] = time.Now().Add(ttl).Unix()
	}

	// Marshal item
	av, err := dynamodbattribute.MarshalMap(item)
	if err != nil {
		return fmt.Errorf("failed to marshal item: %w", err)
	}

	// Put item
	_, err = dm.client.PutItem(&dynamodb.PutItemInput{
		TableName: aws.String(dm.tableName),
		Item:      av,
	})

	if err != nil {
		return fmt.Errorf("failed to put item: %w", err)
	}

	return nil
}

// Delete removes an item from DynamoDB
func (dm *DynamoDBManager) Delete(key string) (bool, error) {
	// Delete item
	result, err := dm.client.DeleteItem(&dynamodb.DeleteItemInput{
		TableName: aws.String(dm.tableName),
		Key: map[string]*dynamodb.AttributeValue{
			"key": {
				S: aws.String(key),
			},
		},
		ReturnValues: aws.String("ALL_OLD"),
	})

	if err != nil {
		return false, fmt.Errorf("failed to delete item: %w", err)
	}

	// Check if item existed
	return result.Attributes != nil, nil
}

// List returns all keys in DynamoDB
func (dm *DynamoDBManager) List() ([]string, error) {
	// Scan table
	result, err := dm.client.Scan(&dynamodb.ScanInput{
		TableName: aws.String(dm.tableName),
		ExpressionAttributeNames: map[string]*string{
			"#k": aws.String("key"),
		},
		ProjectionExpression: aws.String("#k"),
	})

	if err != nil {
		return nil, fmt.Errorf("failed to scan table: %w", err)
	}

	// Extract keys
	keys := make([]string, 0, len(result.Items))
	for _, item := range result.Items {
		if key, ok := item["key"]; ok && key.S != nil {
			keys = append(keys, *key.S)
		}
	}

	return keys, nil
}

// Query performs a query on DynamoDB
func (dm *DynamoDBManager) Query(filter map[string]interface{}, sortKey string, limit int) ([]map[string]interface{}, error) {
	fmt.Printf("DynamoDB Query - Filter: %v, SortKey: %s, Limit: %d\n", filter, sortKey, limit)

	// Create filter expression
	var filterExpr expression.Expression
	var err error

	if len(filter) > 0 {
		// Build filter expression
		var filterBuilder expression.ConditionBuilder
		first := true

		for key, value := range filter {
			var condition expression.ConditionBuilder

			fmt.Printf("Processing filter key: %s, value: %v\n", key, value)

			// For DynamoDB, we need to access the value inside the item
			attributePath := fmt.Sprintf("value.%s", key)

			// Handle different comparison operators
			if valueMap, ok := value.(map[string]interface{}); ok {
				for op, opValue := range valueMap {
					fmt.Printf("  Operator: %s, Value: %v\n", op, opValue)

					switch op {
					case "$eq":
						condition = expression.Name(attributePath).Equal(expression.Value(opValue))
					case "$ne":
						condition = expression.Name(attributePath).NotEqual(expression.Value(opValue))
					case "$gt":
						condition = expression.Name(attributePath).GreaterThan(expression.Value(opValue))
					case "$gte":
						condition = expression.Name(attributePath).GreaterThanEqual(expression.Value(opValue))
					case "$lt":
						condition = expression.Name(attributePath).LessThan(expression.Value(opValue))
					case "$lte":
						condition = expression.Name(attributePath).LessThanEqual(expression.Value(opValue))
					case "$contains":
						condition = expression.Name(attributePath).Contains(fmt.Sprintf("%v", opValue))
					}
				}
			} else {
				condition = expression.Name(attributePath).Equal(expression.Value(value))
			}

			if first {
				filterBuilder = condition
				first = false
			} else {
				filterBuilder = filterBuilder.And(condition)
			}
		}

		// Build expression
		expr := expression.NewBuilder().WithFilter(filterBuilder)
		filterExpr, err = expr.Build()
		if err != nil {
			return nil, fmt.Errorf("failed to build filter expression: %w", err)
		}
	}

	// Create scan input
	scanInput := &dynamodb.ScanInput{
		TableName: aws.String(dm.tableName),
	}

	// Add filter expression if provided
	if len(filter) > 0 {
		scanInput.FilterExpression = filterExpr.Filter()
		scanInput.ExpressionAttributeNames = filterExpr.Names()
		scanInput.ExpressionAttributeValues = filterExpr.Values()
	}

	// Add limit if provided
	if limit > 0 {
		scanInput.Limit = aws.Int64(int64(limit))
	}

	// Scan table
	result, err := dm.client.Scan(scanInput)
	if err != nil {
		return nil, fmt.Errorf("failed to scan table: %w", err)
	}

	// Unmarshal items
	items := make([]map[string]interface{}, 0, len(result.Items))
	for _, item := range result.Items {
		var itemMap map[string]interface{}
		if err := dynamodbattribute.UnmarshalMap(item, &itemMap); err != nil {
			continue
		}

		// Extract key and value
		key, _ := itemMap["key"].(string)
		value, _ := itemMap["value"]

		// Create result item
		resultItem := map[string]interface{}{
			"key":   key,
			"value": value,
		}

		items = append(items, resultItem)
	}

	return items, nil
}

// BatchWrite performs a batch write operation on DynamoDB
func (dm *DynamoDBManager) BatchWrite(items []map[string]interface{}) error {
	// Split items into batches of 25 (DynamoDB limit)
	batchSize := 25
	for i := 0; i < len(items); i += batchSize {
		end := i + batchSize
		if end > len(items) {
			end = len(items)
		}

		batch := items[i:end]
		writeRequests := make([]*dynamodb.WriteRequest, len(batch))

		for j, item := range batch {
			// Marshal item
			av, err := dynamodbattribute.MarshalMap(item)
			if err != nil {
				return fmt.Errorf("failed to marshal item: %w", err)
			}

			// Create put request
			writeRequests[j] = &dynamodb.WriteRequest{
				PutRequest: &dynamodb.PutRequest{
					Item: av,
				},
			}
		}

		// Perform batch write
		_, err := dm.client.BatchWriteItem(&dynamodb.BatchWriteItemInput{
			RequestItems: map[string][]*dynamodb.WriteRequest{
				dm.tableName: writeRequests,
			},
		})

		if err != nil {
			return fmt.Errorf("failed to batch write items: %w", err)
		}
	}

	return nil
}

// NewDynamoDBNodeWrapper creates a new DynamoDB node wrapper
func NewDynamoDBNodeWrapper(params map[string]interface{}) (flowlib.Node, error) {
	// Create the base node
	baseNode := flowlib.NewNode(3, 1*time.Second)

	// Create the wrapper
	wrapper := &NodeWrapper{
		node: baseNode,
		exec: func(input interface{}) (interface{}, error) {
			// Handle both old format (direct params) and new format (combined input)
			var params map[string]interface{}
			
			if combinedInput, ok := input.(map[string]interface{}); ok {
				if nodeParams, hasParams := combinedInput["params"]; hasParams {
					// New format: combined input with params and input
					if paramsMap, ok := nodeParams.(map[string]interface{}); ok {
						params = paramsMap
					} else {
						return nil, fmt.Errorf("expected params to be map[string]interface{}")
					}
				} else {
					// Old format: direct params (backwards compatibility)
					params = combinedInput
				}
			} else {
				return nil, fmt.Errorf("expected map[string]interface{}, got %T", input)
			}

			// Get DynamoDB manager
			manager, err := GetDynamoDBManager(params)
			if err != nil {
				return nil, fmt.Errorf("failed to get DynamoDB manager: %w", err)
			}

			// Get operation
			operation, ok := params["operation"].(string)
			if !ok {
				return nil, fmt.Errorf("operation parameter is required")
			}

			switch operation {
			case "get":
				// Get key
				key, ok := params["key"].(string)
				if !ok {
					return nil, fmt.Errorf("key parameter is required for get operation")
				}

				// Get item
				value, err := manager.Get(key)
				if err != nil {
					return nil, err
				}

				return value, nil

			case "set":
				// Get key
				key, ok := params["key"].(string)
				if !ok {
					return nil, fmt.Errorf("key parameter is required for set operation")
				}

				// Get value
				value, ok := params["value"]
				if !ok {
					return nil, fmt.Errorf("value parameter is required for set operation")
				}

				// Get TTL
				var ttl time.Duration
				if ttlStr, ok := params["ttl"].(string); ok {
					var err error
					ttl, err = time.ParseDuration(ttlStr)
					if err != nil {
						return nil, fmt.Errorf("invalid ttl format: %w", err)
					}
				}

				// Set item
				if err := manager.Set(key, value, ttl); err != nil {
					return nil, err
				}

				return value, nil

			case "delete":
				// Get key
				key, ok := params["key"].(string)
				if !ok {
					return nil, fmt.Errorf("key parameter is required for delete operation")
				}

				// Delete item
				exists, err := manager.Delete(key)
				if err != nil {
					return nil, err
				}

				return map[string]interface{}{
					"deleted": exists,
					"key":     key,
				}, nil

			case "list":
				// List keys
				keys, err := manager.List()
				if err != nil {
					return nil, err
				}

				return keys, nil

			case "query":
				// Get filter
				var filter map[string]interface{}
				if filterParam, ok := params["filter"].(map[string]interface{}); ok {
					filter = filterParam
				} else {
					filter = make(map[string]interface{})
				}

				// Get sort
				var sortField string
				if sortParam, ok := params["sort"].(string); ok {
					sortField = sortParam
				}

				// Get limit
				limit := 0
				if limitParam, ok := params["limit"].(float64); ok {
					limit = int(limitParam)
				}

				// Query items
				results, err := manager.Query(filter, sortField, limit)
				if err != nil {
					return nil, err
				}

				return results, nil

			case "batch_write":
				// Get items
				items, ok := params["items"].([]interface{})
				if !ok {
					return nil, fmt.Errorf("items parameter is required for batch_write operation")
				}

				// Convert items to maps
				itemMaps := make([]map[string]interface{}, 0, len(items))
				for _, item := range items {
					if itemMap, ok := item.(map[string]interface{}); ok {
						itemMaps = append(itemMaps, itemMap)
					}
				}

				// Batch write items
				if err := manager.BatchWrite(itemMaps); err != nil {
					return nil, err
				}

				return map[string]interface{}{
					"written": len(itemMaps),
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
