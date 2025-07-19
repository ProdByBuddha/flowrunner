package storage

import (
	"flag"
	"fmt"
	"os"
	"strings"
	"sync"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/dynamodb"
	"github.com/aws/aws-sdk-go/service/dynamodb/dynamodbiface"
)

var (
	useRealDynamoDB = flag.Bool("real-dynamodb", false, "Use real DynamoDB for tests instead of mock")
)

// MockDynamoDBAPI implements the dynamodbiface.DynamoDBAPI interface for testing
type MockDynamoDBAPI struct {
	dynamodbiface.DynamoDBAPI
	mu     sync.RWMutex
	tables map[string]*MockTable
}

// MockTable represents a DynamoDB table in memory
type MockTable struct {
	Name         string
	Items        map[string]map[string]*dynamodb.AttributeValue
	Indexes      map[string]*MockIndex
	BillingMode  string
	TableStatus  string
	KeySchema    []*dynamodb.KeySchemaElement
	AttributeDef []*dynamodb.AttributeDefinition
	GSI          []*dynamodb.GlobalSecondaryIndex
}

// MockIndex represents a Global Secondary Index
type MockIndex struct {
	Name      string
	KeySchema []*dynamodb.KeySchemaElement
	Items     map[string]map[string]*dynamodb.AttributeValue
}

// NewMockDynamoDBAPI creates a new mock DynamoDB client
func NewMockDynamoDBAPI() *MockDynamoDBAPI {
	return &MockDynamoDBAPI{
		tables: make(map[string]*MockTable),
	}
}

// CreateTable creates a mock table
func (m *MockDynamoDBAPI) CreateTable(input *dynamodb.CreateTableInput) (*dynamodb.CreateTableOutput, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	tableName := aws.StringValue(input.TableName)
	if _, exists := m.tables[tableName]; exists {
		return nil, fmt.Errorf("table already exists: %s", tableName)
	}

	// Create mock indexes
	indexes := make(map[string]*MockIndex)
	if input.GlobalSecondaryIndexes != nil {
		for _, gsi := range input.GlobalSecondaryIndexes {
			indexName := aws.StringValue(gsi.IndexName)
			indexes[indexName] = &MockIndex{
				Name:      indexName,
				KeySchema: gsi.KeySchema,
				Items:     make(map[string]map[string]*dynamodb.AttributeValue),
			}
		}
	}

	m.tables[tableName] = &MockTable{
		Name:         tableName,
		Items:        make(map[string]map[string]*dynamodb.AttributeValue),
		Indexes:      indexes,
		BillingMode:  aws.StringValue(input.BillingMode),
		TableStatus:  "ACTIVE",
		KeySchema:    input.KeySchema,
		AttributeDef: input.AttributeDefinitions,
		GSI:          input.GlobalSecondaryIndexes,
	}

	return &dynamodb.CreateTableOutput{
		TableDescription: &dynamodb.TableDescription{
			TableName:   input.TableName,
			TableStatus: aws.String("ACTIVE"),
		},
	}, nil
}

// DescribeTable describes a mock table
func (m *MockDynamoDBAPI) DescribeTable(input *dynamodb.DescribeTableInput) (*dynamodb.DescribeTableOutput, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	tableName := aws.StringValue(input.TableName)
	table, exists := m.tables[tableName]
	if !exists {
		// Return AWS-style error for resource not found
		return nil, awserr.New(dynamodb.ErrCodeResourceNotFoundException, fmt.Sprintf("Requested resource not found"), nil)
	}

	return &dynamodb.DescribeTableOutput{
		Table: &dynamodb.TableDescription{
			TableName:               aws.String(table.Name),
			TableStatus:             aws.String(table.TableStatus),
			KeySchema:               table.KeySchema,
			AttributeDefinitions:    table.AttributeDef,
			BillingModeSummary: &dynamodb.BillingModeSummary{
				BillingMode: aws.String(table.BillingMode),
			},
		},
	}, nil
}

// DeleteTable deletes a mock table
func (m *MockDynamoDBAPI) DeleteTable(input *dynamodb.DeleteTableInput) (*dynamodb.DeleteTableOutput, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	tableName := aws.StringValue(input.TableName)
	if _, exists := m.tables[tableName]; !exists {
		return nil, fmt.Errorf("table not found: %s", tableName)
	}

	delete(m.tables, tableName)

	return &dynamodb.DeleteTableOutput{
		TableDescription: &dynamodb.TableDescription{
			TableName:   input.TableName,
			TableStatus: aws.String("DELETING"),
		},
	}, nil
}

// PutItem puts an item in a mock table
func (m *MockDynamoDBAPI) PutItem(input *dynamodb.PutItemInput) (*dynamodb.PutItemOutput, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	tableName := aws.StringValue(input.TableName)
	table, exists := m.tables[tableName]
	if !exists {
		return nil, fmt.Errorf("table not found: %s", tableName)
	}

	// Generate key from primary key attributes
	key := m.generateKey(table.KeySchema, input.Item)
	table.Items[key] = input.Item

	// Update GSI items
	for _, gsi := range table.GSI {
		indexName := aws.StringValue(gsi.IndexName)
		if index, exists := table.Indexes[indexName]; exists {
			gsiKey := m.generateKey(gsi.KeySchema, input.Item)
			index.Items[gsiKey] = input.Item
		}
	}

	return &dynamodb.PutItemOutput{}, nil
}

// GetItem gets an item from a mock table
func (m *MockDynamoDBAPI) GetItem(input *dynamodb.GetItemInput) (*dynamodb.GetItemOutput, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	tableName := aws.StringValue(input.TableName)
	table, exists := m.tables[tableName]
	if !exists {
		return nil, fmt.Errorf("table not found: %s", tableName)
	}

	key := m.generateKey(table.KeySchema, input.Key)
	item, exists := table.Items[key]
	if !exists {
		return &dynamodb.GetItemOutput{}, nil
	}

	return &dynamodb.GetItemOutput{Item: item}, nil
}

// Query queries a mock table or index
func (m *MockDynamoDBAPI) Query(input *dynamodb.QueryInput) (*dynamodb.QueryOutput, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	tableName := aws.StringValue(input.TableName)
	table, exists := m.tables[tableName]
	if !exists {
		return nil, fmt.Errorf("table not found: %s", tableName)
	}

	var items map[string]map[string]*dynamodb.AttributeValue

	// Determine if querying table or index
	if input.IndexName != nil {
		indexName := aws.StringValue(input.IndexName)
		index, exists := table.Indexes[indexName]
		if !exists {
			return nil, fmt.Errorf("index not found: %s", indexName)
		}
		items = index.Items
	} else {
		items = table.Items
	}

	// Simple mock query logic - in a real implementation you'd parse the KeyConditionExpression
	var resultItems []map[string]*dynamodb.AttributeValue

	for _, item := range items {
		// Basic filtering - this is simplified for testing
		resultItems = append(resultItems, item)
	}

	// Apply limit if specified
	if input.Limit != nil {
		limit := int(aws.Int64Value(input.Limit))
		if limit < len(resultItems) {
			resultItems = resultItems[:limit]
		}
	}

	return &dynamodb.QueryOutput{
		Items: resultItems,
		Count: aws.Int64(int64(len(resultItems))),
	}, nil
}

// Scan scans a mock table
func (m *MockDynamoDBAPI) Scan(input *dynamodb.ScanInput) (*dynamodb.ScanOutput, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	tableName := aws.StringValue(input.TableName)
	table, exists := m.tables[tableName]
	if !exists {
		return nil, fmt.Errorf("table not found: %s", tableName)
	}

	var resultItems []map[string]*dynamodb.AttributeValue
	for _, item := range table.Items {
		resultItems = append(resultItems, item)
	}

	// Apply limit if specified
	if input.Limit != nil {
		limit := int(aws.Int64Value(input.Limit))
		if limit < len(resultItems) {
			resultItems = resultItems[:limit]
		}
	}

	return &dynamodb.ScanOutput{
		Items: resultItems,
		Count: aws.Int64(int64(len(resultItems))),
	}, nil
}

// BatchWriteItem performs batch write on mock table
func (m *MockDynamoDBAPI) BatchWriteItem(input *dynamodb.BatchWriteItemInput) (*dynamodb.BatchWriteItemOutput, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	for tableName, writeRequests := range input.RequestItems {
		table, exists := m.tables[tableName]
		if !exists {
			return nil, fmt.Errorf("table not found: %s", tableName)
		}

		for _, writeRequest := range writeRequests {
			if writeRequest.PutRequest != nil {
				key := m.generateKey(table.KeySchema, writeRequest.PutRequest.Item)
				table.Items[key] = writeRequest.PutRequest.Item
			}
			if writeRequest.DeleteRequest != nil {
				key := m.generateKey(table.KeySchema, writeRequest.DeleteRequest.Key)
				delete(table.Items, key)
			}
		}
	}

	return &dynamodb.BatchWriteItemOutput{}, nil
}

// WaitUntilTableExists waits for table to exist (mock always returns immediately)
func (m *MockDynamoDBAPI) WaitUntilTableExists(input *dynamodb.DescribeTableInput) error {
	return nil // Mock tables are immediately available
}

// WaitUntilTableNotExists waits for table to not exist (mock always returns immediately)
func (m *MockDynamoDBAPI) WaitUntilTableNotExists(input *dynamodb.DescribeTableInput) error {
	return nil // Mock tables are immediately gone
}

// generateKey generates a composite key from key schema and item attributes
func (m *MockDynamoDBAPI) generateKey(keySchema []*dynamodb.KeySchemaElement, item map[string]*dynamodb.AttributeValue) string {
	var keyParts []string
	for _, keyElement := range keySchema {
		attrName := aws.StringValue(keyElement.AttributeName)
		if attr, exists := item[attrName]; exists {
			if attr.S != nil {
				keyParts = append(keyParts, aws.StringValue(attr.S))
			} else if attr.N != nil {
				keyParts = append(keyParts, aws.StringValue(attr.N))
			}
		}
	}
	return strings.Join(keyParts, "#")
}

// DeleteItem deletes an item from a mock table
func (m *MockDynamoDBAPI) DeleteItem(input *dynamodb.DeleteItemInput) (*dynamodb.DeleteItemOutput, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	tableName := aws.StringValue(input.TableName)
	table, exists := m.tables[tableName]
	if !exists {
		return nil, fmt.Errorf("table not found: %s", tableName)
	}

	key := m.generateKey(table.KeySchema, input.Key)

	// Check if item exists (for condition expressions)
	_, exists = table.Items[key]
	if !exists && input.ConditionExpression != nil {
		// Simple condition check - item must exist
		return nil, fmt.Errorf("conditional check failed")
	}

	delete(table.Items, key)

	// Remove from GSI items as well
	for _, gsi := range table.GSI {
		indexName := aws.StringValue(gsi.IndexName)
		if index, exists := table.Indexes[indexName]; exists {
			delete(index.Items, key)
		}
	}

	return &dynamodb.DeleteItemOutput{}, nil
}

// Helper function for tests to get DynamoDB client (mock or real)
func GetTestDynamoDBClient() (dynamodbiface.DynamoDBAPI, error) {
	if *useRealDynamoDB {
		// Use real DynamoDB
		accessKey := os.Getenv("AWS_ACCESS_KEY_ID")
		secretKey := os.Getenv("AWS_SECRET_ACCESS_KEY")
		endpoint := os.Getenv("DYNAMODB_ENDPOINT")

		awsConfig := &aws.Config{
			Region: aws.String("us-east-1"),
		}

		if accessKey != "" && secretKey != "" {
			awsConfig.Credentials = credentials.NewStaticCredentials(accessKey, secretKey, "")
		}

		if endpoint != "" {
			awsConfig.Endpoint = aws.String(endpoint)
		}

		sess, err := session.NewSession(awsConfig)
		if err != nil {
			return nil, fmt.Errorf("failed to create AWS session: %w", err)
		}

		return dynamodb.New(sess), nil
	}

	// Use mock DynamoDB
	return NewMockDynamoDBAPI(), nil
}
