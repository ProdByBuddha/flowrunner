package storage

import (
	"encoding/json"
	"fmt"
	"strconv"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/dynamodb"
	"github.com/aws/aws-sdk-go/service/dynamodb/dynamodbattribute"
	"github.com/aws/aws-sdk-go/service/dynamodb/dynamodbiface"
	"github.com/aws/aws-sdk-go/service/dynamodb/expression"
	"github.com/tcmartin/flowrunner/pkg/auth"
	"github.com/tcmartin/flowrunner/pkg/runtime"
)

// DynamoDBProvider implements the StorageProvider interface using DynamoDB
type DynamoDBProvider struct {
	client         dynamodbiface.DynamoDBAPI
	flowStore      *DynamoDBFlowStore
	secretStore    *DynamoDBSecretStore
	executionStore *DynamoDBExecutionStore
	accountStore   *DynamoDBAccountStore
	tablePrefix    string
}

// DynamoDBProviderConfig contains configuration for the DynamoDB provider
type DynamoDBProviderConfig struct {
	Region      string
	AccessKey   string
	SecretKey   string
	TablePrefix string
	Endpoint    string // Optional, for local DynamoDB
}

// NewDynamoDBProvider creates a new DynamoDB storage provider
func NewDynamoDBProvider(config DynamoDBProviderConfig) (*DynamoDBProvider, error) {
	// Create AWS session
	awsConfig := &aws.Config{
		Region: aws.String(config.Region),
	}

	// Set credentials if provided
	if config.AccessKey != "" && config.SecretKey != "" {
		awsConfig.Credentials = credentials.NewStaticCredentials(
			config.AccessKey,
			config.SecretKey,
			"",
		)
	}

	// Set endpoint for local DynamoDB if provided
	if config.Endpoint != "" {
		awsConfig.Endpoint = aws.String(config.Endpoint)
	}

	// Create session
	sess, err := session.NewSession(awsConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to create AWS session: %w", err)
	}

	// Create DynamoDB client
	client := dynamodb.New(sess)

	// Create provider
	provider := &DynamoDBProvider{
		client:      client,
		tablePrefix: config.TablePrefix,
	}

	// Create stores
	provider.flowStore = NewDynamoDBFlowStore(client, config.TablePrefix)
	provider.secretStore = NewDynamoDBSecretStore(client, config.TablePrefix)
	provider.executionStore = NewDynamoDBExecutionStore(client, config.TablePrefix)
	provider.accountStore = NewDynamoDBAccountStore(client, config.TablePrefix)

	return provider, nil
}

// NewDynamoDBProviderWithClient creates a new DynamoDB storage provider with a custom client
// This is primarily used for testing with mock clients
func NewDynamoDBProviderWithClient(client dynamodbiface.DynamoDBAPI, tablePrefix string) *DynamoDBProvider {
	// Create provider
	provider := &DynamoDBProvider{
		client:      client,
		tablePrefix: tablePrefix,
	}

	// Create stores
	provider.flowStore = NewDynamoDBFlowStore(client, tablePrefix)
	provider.secretStore = NewDynamoDBSecretStore(client, tablePrefix)
	provider.executionStore = NewDynamoDBExecutionStore(client, tablePrefix)
	provider.accountStore = NewDynamoDBAccountStore(client, tablePrefix)

	return provider
}

// Initialize sets up the storage backend
func (p *DynamoDBProvider) Initialize() error {
	// Initialize all stores
	if err := p.flowStore.Initialize(); err != nil {
		return fmt.Errorf("failed to initialize flow store: %w", err)
	}

	if err := p.secretStore.Initialize(); err != nil {
		return fmt.Errorf("failed to initialize secret store: %w", err)
	}

	if err := p.executionStore.Initialize(); err != nil {
		return fmt.Errorf("failed to initialize execution store: %w", err)
	}

	if err := p.accountStore.Initialize(); err != nil {
		return fmt.Errorf("failed to initialize account store: %w", err)
	}

	return nil
}

// Close cleans up resources
func (p *DynamoDBProvider) Close() error {
	// Nothing to close for DynamoDB client
	return nil
}

// GetFlowStore returns a store for flow definitions
func (p *DynamoDBProvider) GetFlowStore() FlowStore {
	return p.flowStore
}

// GetSecretStore returns a store for secrets
func (p *DynamoDBProvider) GetSecretStore() SecretStore {
	return p.secretStore
}

// GetExecutionStore returns a store for execution data
func (p *DynamoDBProvider) GetExecutionStore() ExecutionStore {
	return p.executionStore
}

// GetAccountStore returns a store for account data
func (p *DynamoDBProvider) GetAccountStore() AccountStore {
	return p.accountStore
}

// DynamoDBFlowStore implements the FlowStore interface using DynamoDB
type DynamoDBFlowStore struct {
	client      dynamodbiface.DynamoDBAPI
	tablePrefix string
	tableName   string
}

// NewDynamoDBFlowStore creates a new DynamoDB flow store
func NewDynamoDBFlowStore(client dynamodbiface.DynamoDBAPI, tablePrefix string) *DynamoDBFlowStore {
	return &DynamoDBFlowStore{
		client:      client,
		tablePrefix: tablePrefix,
		tableName:   tablePrefix + "flows",
	}
}

// Initialize creates the DynamoDB tables if they don't exist
func (s *DynamoDBFlowStore) Initialize() error {
	// Initialize main flows table
	if err := s.initializeFlowsTable(); err != nil {
		return err
	}

	// Initialize flow versions table
	if err := s.initializeFlowVersionsTable(); err != nil {
		return err
	}

	return nil
}

// initializeFlowsTable creates the flows table if it doesn't exist
func (s *DynamoDBFlowStore) initializeFlowsTable() error {
	// Check if table exists
	_, err := s.client.DescribeTable(&dynamodb.DescribeTableInput{
		TableName: aws.String(s.tableName),
	})

	if err == nil {
		// Table exists
		return nil
	}

	// Check if error is "table not found"
	if aerr, ok := err.(awserr.Error); ok && aerr.Code() == dynamodb.ErrCodeResourceNotFoundException {
		// Create table
		_, err = s.client.CreateTable(&dynamodb.CreateTableInput{
			TableName: aws.String(s.tableName),
			AttributeDefinitions: []*dynamodb.AttributeDefinition{
				{
					AttributeName: aws.String("AccountID"),
					AttributeType: aws.String("S"),
				},
				{
					AttributeName: aws.String("FlowID"),
					AttributeType: aws.String("S"),
				},
			},
			KeySchema: []*dynamodb.KeySchemaElement{
				{
					AttributeName: aws.String("AccountID"),
					KeyType:       aws.String("HASH"),
				},
				{
					AttributeName: aws.String("FlowID"),
					KeyType:       aws.String("RANGE"),
				},
			},
			BillingMode: aws.String("PAY_PER_REQUEST"),
		})

		if err != nil {
			return fmt.Errorf("failed to create table: %w", err)
		}

		// Wait for table to be created
		err = s.client.WaitUntilTableExists(&dynamodb.DescribeTableInput{
			TableName: aws.String(s.tableName),
		})

		if err != nil {
			return fmt.Errorf("failed to wait for table creation: %w", err)
		}

		return nil
	}

	return fmt.Errorf("failed to check if table exists: %w", err)
}

// initializeFlowVersionsTable creates the flow versions table if it doesn't exist
func (s *DynamoDBFlowStore) initializeFlowVersionsTable() error {
	versionsTableName := s.tableName + "_versions"

	// Check if table exists
	_, err := s.client.DescribeTable(&dynamodb.DescribeTableInput{
		TableName: aws.String(versionsTableName),
	})

	if err == nil {
		// Table exists
		return nil
	}

	// Check if error is "table not found"
	if aerr, ok := err.(awserr.Error); ok && aerr.Code() == dynamodb.ErrCodeResourceNotFoundException {
		// Create table
		_, err = s.client.CreateTable(&dynamodb.CreateTableInput{
			TableName: aws.String(versionsTableName),
			AttributeDefinitions: []*dynamodb.AttributeDefinition{
				{
					AttributeName: aws.String("AccountID"),
					AttributeType: aws.String("S"),
				},
				{
					AttributeName: aws.String("FlowID"),
					AttributeType: aws.String("S"),
				},
				{
					AttributeName: aws.String("Version"),
					AttributeType: aws.String("S"),
				},
			},
			KeySchema: []*dynamodb.KeySchemaElement{
				{
					AttributeName: aws.String("AccountID"),
					KeyType:       aws.String("HASH"),
				},
				{
					AttributeName: aws.String("FlowID"),
					KeyType:       aws.String("RANGE"),
				},
			},
			GlobalSecondaryIndexes: []*dynamodb.GlobalSecondaryIndex{
				{
					IndexName: aws.String("VersionIndex"),
					KeySchema: []*dynamodb.KeySchemaElement{
						{
							AttributeName: aws.String("FlowID"),
							KeyType:       aws.String("HASH"),
						},
						{
							AttributeName: aws.String("Version"),
							KeyType:       aws.String("RANGE"),
						},
					},
					Projection: &dynamodb.Projection{
						ProjectionType: aws.String("ALL"),
					},
				},
			},
			BillingMode: aws.String("PAY_PER_REQUEST"),
		})

		if err != nil {
			return fmt.Errorf("failed to create flow versions table: %w", err)
		}

		// Wait for table to be created
		err = s.client.WaitUntilTableExists(&dynamodb.DescribeTableInput{
			TableName: aws.String(versionsTableName),
		})

		if err != nil {
			return fmt.Errorf("failed to wait for flow versions table creation: %w", err)
		}

		return nil
	}

	return fmt.Errorf("failed to check if flow versions table exists: %w", err)
}

// dynamoDBFlowItem represents a flow item in DynamoDB
type dynamoDBFlowItem struct {
	AccountID   string `json:"AccountID"`
	FlowID      string `json:"FlowID"`
	Definition  string `json:"Definition"`
	Name        string `json:"Name"`
	Description string `json:"Description"`
	Version     string `json:"Version"`
	CreatedAt   int64  `json:"CreatedAt"`
	UpdatedAt   int64  `json:"UpdatedAt"`
}

// SaveFlow persists a flow definition
func (s *DynamoDBFlowStore) SaveFlow(accountID, flowID string, definition []byte) error {
	// Extract metadata from the definition
	var metadata struct {
		Metadata struct {
			Name        string `json:"name"`
			Description string `json:"description"`
			Version     string `json:"version"`
		} `json:"metadata"`
	}

	if err := json.Unmarshal(definition, &metadata); err != nil {
		// If we can't extract metadata, just use empty values
		metadata.Metadata.Name = flowID
		metadata.Metadata.Description = ""
		metadata.Metadata.Version = "1.0.0"
	}

	// Generate a version if not specified
	version := metadata.Metadata.Version
	if version == "" {
		version = fmt.Sprintf("v%d", time.Now().UnixNano())
	}

	// Create flow item
	now := time.Now().Unix()
	item := dynamoDBFlowItem{
		AccountID:   accountID,
		FlowID:      flowID,
		Definition:  string(definition),
		Name:        metadata.Metadata.Name,
		Description: metadata.Metadata.Description,
		Version:     version,
		UpdatedAt:   now,
	}

	// Check if flow already exists
	result, err := s.client.GetItem(&dynamodb.GetItemInput{
		TableName: aws.String(s.tableName),
		Key: map[string]*dynamodb.AttributeValue{
			"AccountID": {
				S: aws.String(accountID),
			},
			"FlowID": {
				S: aws.String(flowID),
			},
		},
	})

	if err != nil {
		return fmt.Errorf("failed to check if flow exists: %w", err)
	}

	if result.Item == nil {
		// New flow
		item.CreatedAt = now
	} else {
		// Existing flow, preserve creation time
		var existingItem dynamoDBFlowItem
		if err := dynamodbattribute.UnmarshalMap(result.Item, &existingItem); err != nil {
			return fmt.Errorf("failed to unmarshal existing flow: %w", err)
		}
		item.CreatedAt = existingItem.CreatedAt
	}

	// Marshal item
	av, err := dynamodbattribute.MarshalMap(item)
	if err != nil {
		return fmt.Errorf("failed to marshal flow item: %w", err)
	}

	// Save flow
	_, err = s.client.PutItem(&dynamodb.PutItemInput{
		TableName: aws.String(s.tableName),
		Item:      av,
	})

	if err != nil {
		return fmt.Errorf("failed to save flow: %w", err)
	}

	// Also save as a version
	versionItem := struct {
		AccountID   string `json:"AccountID"`
		FlowID      string `json:"FlowID"`
		Version     string `json:"Version"`
		Description string `json:"Description"`
		Definition  string `json:"Definition"`
		CreatedAt   int64  `json:"CreatedAt"`
		CreatedBy   string `json:"CreatedBy,omitempty"`
	}{
		AccountID:   accountID,
		FlowID:      flowID,
		Version:     version,
		Description: metadata.Metadata.Description,
		Definition:  string(definition),
		CreatedAt:   now,
	}

	// Marshal version item
	versionAV, err := dynamodbattribute.MarshalMap(versionItem)
	if err != nil {
		return fmt.Errorf("failed to marshal flow version item: %w", err)
	}

	// Save flow version
	_, err = s.client.PutItem(&dynamodb.PutItemInput{
		TableName: aws.String(s.tableName + "_versions"),
		Item:      versionAV,
	})

	if err != nil {
		return fmt.Errorf("failed to save flow version: %w", err)
	}

	return nil
}

// GetFlow retrieves a flow definition
func (s *DynamoDBFlowStore) GetFlow(accountID, flowID string) ([]byte, error) {
	// Get flow
	result, err := s.client.GetItem(&dynamodb.GetItemInput{
		TableName: aws.String(s.tableName),
		Key: map[string]*dynamodb.AttributeValue{
			"AccountID": {
				S: aws.String(accountID),
			},
			"FlowID": {
				S: aws.String(flowID),
			},
		},
	})

	if err != nil {
		return nil, fmt.Errorf("failed to get flow: %w", err)
	}

	if result.Item == nil {
		return nil, ErrFlowNotFound
	}

	// Unmarshal item
	var item dynamoDBFlowItem
	if err := dynamodbattribute.UnmarshalMap(result.Item, &item); err != nil {
		return nil, fmt.Errorf("failed to unmarshal flow item: %w", err)
	}

	return []byte(item.Definition), nil
}

// ListFlows returns all flow IDs for an account
func (s *DynamoDBFlowStore) ListFlows(accountID string) ([]string, error) {
	// Create query expression
	keyCond := expression.Key("AccountID").Equal(expression.Value(accountID))
	expr, err := expression.NewBuilder().WithKeyCondition(keyCond).Build()
	if err != nil {
		return nil, fmt.Errorf("failed to build expression: %w", err)
	}

	// Query flows
	result, err := s.client.Query(&dynamodb.QueryInput{
		TableName:                 aws.String(s.tableName),
		KeyConditionExpression:    expr.KeyCondition(),
		ExpressionAttributeNames:  expr.Names(),
		ExpressionAttributeValues: expr.Values(),
	})

	if err != nil {
		return nil, fmt.Errorf("failed to query flows: %w", err)
	}

	// Extract flow IDs
	flowIDs := make([]string, 0, len(result.Items))
	for _, item := range result.Items {
		flowID := item["FlowID"].S
		if flowID != nil {
			flowIDs = append(flowIDs, *flowID)
		}
	}

	return flowIDs, nil
}

// DeleteFlow removes a flow definition and all its versions
func (s *DynamoDBFlowStore) DeleteFlow(accountID, flowID string) error {
	// Delete flow
	_, err := s.client.DeleteItem(&dynamodb.DeleteItemInput{
		TableName: aws.String(s.tableName),
		Key: map[string]*dynamodb.AttributeValue{
			"AccountID": {
				S: aws.String(accountID),
			},
			"FlowID": {
				S: aws.String(flowID),
			},
		},
		ConditionExpression: aws.String("attribute_exists(AccountID) AND attribute_exists(FlowID)"),
	})

	if err != nil {
		if aerr, ok := err.(awserr.Error); ok && aerr.Code() == dynamodb.ErrCodeConditionalCheckFailedException {
			return ErrFlowNotFound
		}
		return fmt.Errorf("failed to delete flow: %w", err)
	}

	// Delete all versions of the flow
	// First, query to get all versions
	versions, err := s.ListFlowVersions(accountID, flowID)
	if err != nil {
		return fmt.Errorf("failed to list flow versions for deletion: %w", err)
	}

	// Delete each version
	for _, version := range versions {
		_, err := s.client.DeleteItem(&dynamodb.DeleteItemInput{
			TableName: aws.String(s.tableName + "_versions"),
			Key: map[string]*dynamodb.AttributeValue{
				"AccountID": {
					S: aws.String(accountID),
				},
				"FlowID": {
					S: aws.String(flowID),
				},
			},
		})

		if err != nil {
			return fmt.Errorf("failed to delete flow version %s: %w", version, err)
		}
	}

	return nil
}

// GetFlowMetadata retrieves metadata for a flow
func (s *DynamoDBFlowStore) GetFlowMetadata(accountID, flowID string) (FlowMetadata, error) {
	// Get flow
	result, err := s.client.GetItem(&dynamodb.GetItemInput{
		TableName: aws.String(s.tableName),
		Key: map[string]*dynamodb.AttributeValue{
			"AccountID": {
				S: aws.String(accountID),
			},
			"FlowID": {
				S: aws.String(flowID),
			},
		},
		ProjectionExpression: aws.String("FlowID, AccountID, #name, #desc, #ver, CreatedAt, UpdatedAt"),
		ExpressionAttributeNames: map[string]*string{
			"#name": aws.String("Name"),
			"#desc": aws.String("Description"),
			"#ver":  aws.String("Version"),
		},
	})

	if err != nil {
		return FlowMetadata{}, fmt.Errorf("failed to get flow metadata: %w", err)
	}

	if result.Item == nil {
		return FlowMetadata{}, ErrFlowNotFound
	}

	// Unmarshal item
	var item dynamoDBFlowItem
	if err := dynamodbattribute.UnmarshalMap(result.Item, &item); err != nil {
		return FlowMetadata{}, fmt.Errorf("failed to unmarshal flow item: %w", err)
	}

	// Convert to FlowMetadata
	metadata := FlowMetadata{
		ID:          item.FlowID,
		AccountID:   item.AccountID,
		Name:        item.Name,
		Description: item.Description,
		Version:     item.Version,
		CreatedAt:   item.CreatedAt,
		UpdatedAt:   item.UpdatedAt,
	}

	return metadata, nil
}

// ListFlowsWithMetadata returns all flows with metadata for an account
func (s *DynamoDBFlowStore) ListFlowsWithMetadata(accountID string) ([]FlowMetadata, error) {
	// Create query expression
	keyCond := expression.Key("AccountID").Equal(expression.Value(accountID))
	proj := expression.NamesList(
		expression.Name("FlowID"),
		expression.Name("AccountID"),
		expression.Name("Name"),
		expression.Name("Description"),
		expression.Name("Version"),
		expression.Name("CreatedAt"),
		expression.Name("UpdatedAt"),
	)
	expr, err := expression.NewBuilder().WithKeyCondition(keyCond).WithProjection(proj).Build()
	if err != nil {
		return nil, fmt.Errorf("failed to build expression: %w", err)
	}

	// Query flows
	result, err := s.client.Query(&dynamodb.QueryInput{
		TableName:                 aws.String(s.tableName),
		KeyConditionExpression:    expr.KeyCondition(),
		ExpressionAttributeNames:  expr.Names(),
		ExpressionAttributeValues: expr.Values(),
		ProjectionExpression:      expr.Projection(),
	})

	if err != nil {
		return nil, fmt.Errorf("failed to query flows: %w", err)
	}

	// Extract flow metadata
	metadataList := make([]FlowMetadata, 0, len(result.Items))
	for _, item := range result.Items {
		var flowItem dynamoDBFlowItem
		if err := dynamodbattribute.UnmarshalMap(item, &flowItem); err != nil {
			return nil, fmt.Errorf("failed to unmarshal flow item: %w", err)
		}

		metadata := FlowMetadata{
			ID:          flowItem.FlowID,
			AccountID:   flowItem.AccountID,
			Name:        flowItem.Name,
			Description: flowItem.Description,
			Version:     flowItem.Version,
			CreatedAt:   flowItem.CreatedAt,
			UpdatedAt:   flowItem.UpdatedAt,
		}

		metadataList = append(metadataList, metadata)
	}

	return metadataList, nil
}

// DynamoDBSecretStore implements the SecretStore interface using DynamoDB
type DynamoDBSecretStore struct {
	client      dynamodbiface.DynamoDBAPI
	tablePrefix string
	tableName   string
}

// NewDynamoDBSecretStore creates a new DynamoDB secret store
func NewDynamoDBSecretStore(client dynamodbiface.DynamoDBAPI, tablePrefix string) *DynamoDBSecretStore {
	return &DynamoDBSecretStore{
		client:      client,
		tablePrefix: tablePrefix,
		tableName:   tablePrefix + "secrets",
	}
}

// Initialize creates the DynamoDB table if it doesn't exist
func (s *DynamoDBSecretStore) Initialize() error {
	// Check if table exists
	_, err := s.client.DescribeTable(&dynamodb.DescribeTableInput{
		TableName: aws.String(s.tableName),
	})

	if err == nil {
		// Table exists
		return nil
	}

	// Check if error is "table not found"
	if aerr, ok := err.(awserr.Error); ok && aerr.Code() == dynamodb.ErrCodeResourceNotFoundException {
		// Create table
		_, err = s.client.CreateTable(&dynamodb.CreateTableInput{
			TableName: aws.String(s.tableName),
			AttributeDefinitions: []*dynamodb.AttributeDefinition{
				{
					AttributeName: aws.String("AccountID"),
					AttributeType: aws.String("S"),
				},
				{
					AttributeName: aws.String("Key"),
					AttributeType: aws.String("S"),
				},
			},
			KeySchema: []*dynamodb.KeySchemaElement{
				{
					AttributeName: aws.String("AccountID"),
					KeyType:       aws.String("HASH"),
				},
				{
					AttributeName: aws.String("Key"),
					KeyType:       aws.String("RANGE"),
				},
			},
			BillingMode: aws.String("PAY_PER_REQUEST"),
		})

		if err != nil {
			return fmt.Errorf("failed to create table: %w", err)
		}

		// Wait for table to be created
		err = s.client.WaitUntilTableExists(&dynamodb.DescribeTableInput{
			TableName: aws.String(s.tableName),
		})

		if err != nil {
			return fmt.Errorf("failed to wait for table creation: %w", err)
		}

		return nil
	}

	return fmt.Errorf("failed to check if table exists: %w", err)
}

// SaveSecret persists a secret
func (s *DynamoDBSecretStore) SaveSecret(secret auth.Secret) error {
	// Marshal secret
	av, err := dynamodbattribute.MarshalMap(secret)
	if err != nil {
		return fmt.Errorf("failed to marshal secret: %w", err)
	}

	// Save secret
	_, err = s.client.PutItem(&dynamodb.PutItemInput{
		TableName: aws.String(s.tableName),
		Item:      av,
	})

	if err != nil {
		return fmt.Errorf("failed to save secret: %w", err)
	}

	return nil
}

// GetSecret retrieves a secret
func (s *DynamoDBSecretStore) GetSecret(accountID, key string) (auth.Secret, error) {
	// Get secret
	result, err := s.client.GetItem(&dynamodb.GetItemInput{
		TableName: aws.String(s.tableName),
		Key: map[string]*dynamodb.AttributeValue{
			"AccountID": {
				S: aws.String(accountID),
			},
			"Key": {
				S: aws.String(key),
			},
		},
	})

	if err != nil {
		return auth.Secret{}, fmt.Errorf("failed to get secret: %w", err)
	}

	if result.Item == nil {
		return auth.Secret{}, ErrSecretNotFound
	}

	// Unmarshal secret
	var secret auth.Secret
	if err := dynamodbattribute.UnmarshalMap(result.Item, &secret); err != nil {
		return auth.Secret{}, fmt.Errorf("failed to unmarshal secret: %w", err)
	}

	return secret, nil
}

// ListSecrets returns all secrets for an account
func (s *DynamoDBSecretStore) ListSecrets(accountID string) ([]auth.Secret, error) {
	// Create query expression
	keyCond := expression.Key("AccountID").Equal(expression.Value(accountID))
	expr, err := expression.NewBuilder().WithKeyCondition(keyCond).Build()
	if err != nil {
		return nil, fmt.Errorf("failed to build expression: %w", err)
	}

	// Query secrets
	result, err := s.client.Query(&dynamodb.QueryInput{
		TableName:                 aws.String(s.tableName),
		KeyConditionExpression:    expr.KeyCondition(),
		ExpressionAttributeNames:  expr.Names(),
		ExpressionAttributeValues: expr.Values(),
	})

	if err != nil {
		return nil, fmt.Errorf("failed to query secrets: %w", err)
	}

	// Extract secrets
	secrets := make([]auth.Secret, 0, len(result.Items))
	for _, item := range result.Items {
		var secret auth.Secret
		if err := dynamodbattribute.UnmarshalMap(item, &secret); err != nil {
			return nil, fmt.Errorf("failed to unmarshal secret: %w", err)
		}
		secrets = append(secrets, secret)
	}

	return secrets, nil
}

// DeleteSecret removes a secret
func (s *DynamoDBSecretStore) DeleteSecret(accountID, key string) error {
	// Delete secret
	_, err := s.client.DeleteItem(&dynamodb.DeleteItemInput{
		TableName: aws.String(s.tableName),
		Key: map[string]*dynamodb.AttributeValue{
			"AccountID": {
				S: aws.String(accountID),
			},
			"Key": {
				S: aws.String(key),
			},
		},
		ConditionExpression: aws.String("attribute_exists(AccountID) AND attribute_exists(#k)"),
		ExpressionAttributeNames: map[string]*string{
			"#k": aws.String("Key"),
		},
	})

	if err != nil {
		if aerr, ok := err.(awserr.Error); ok && aerr.Code() == dynamodb.ErrCodeConditionalCheckFailedException {
			return ErrSecretNotFound
		}
		return fmt.Errorf("failed to delete secret: %w", err)
	}

	return nil
}

// DynamoDBExecutionStore implements the ExecutionStore interface using DynamoDB
type DynamoDBExecutionStore struct {
	client        dynamodbiface.DynamoDBAPI
	tablePrefix   string
	execTableName string
	logsTableName string
}

// SetExecutionAccountID sets the account ID for an execution in its metadata
func (s *DynamoDBExecutionStore) SetExecutionAccountID(executionID, accountID string) error {
	// Create the item directly with the account ID attribute
	_, err := s.client.UpdateItem(&dynamodb.UpdateItemInput{
		TableName: aws.String(s.execTableName),
		Key: map[string]*dynamodb.AttributeValue{
			"ID": {
				S: aws.String(executionID),
			},
		},
		UpdateExpression: aws.String("SET AccountID = :accountID"),
		ExpressionAttributeValues: map[string]*dynamodb.AttributeValue{
			":accountID": {
				S: aws.String(accountID),
			},
		},
	})

	if err != nil {
		return fmt.Errorf("failed to update execution with account ID: %w", err)
	}

	return nil
}

// NewDynamoDBExecutionStore creates a new DynamoDB execution store
func NewDynamoDBExecutionStore(client dynamodbiface.DynamoDBAPI, tablePrefix string) *DynamoDBExecutionStore {
	return &DynamoDBExecutionStore{
		client:        client,
		tablePrefix:   tablePrefix,
		execTableName: tablePrefix + "executions",
		logsTableName: tablePrefix + "execution_logs",
	}
}

// Initialize creates the DynamoDB tables if they don't exist
func (s *DynamoDBExecutionStore) Initialize() error {
	// Initialize executions table
	if err := s.initializeExecutionsTable(); err != nil {
		return err
	}

	// Initialize execution logs table
	if err := s.initializeExecutionLogsTable(); err != nil {
		return err
	}

	return nil
}

// initializeExecutionsTable creates the executions table if it doesn't exist
func (s *DynamoDBExecutionStore) initializeExecutionsTable() error {
	// Check if table exists
	_, err := s.client.DescribeTable(&dynamodb.DescribeTableInput{
		TableName: aws.String(s.execTableName),
	})

	if err == nil {
		// Table exists
		return nil
	}

	// Check if error is "table not found"
	if aerr, ok := err.(awserr.Error); ok && aerr.Code() == dynamodb.ErrCodeResourceNotFoundException {
		// Create table
		_, err = s.client.CreateTable(&dynamodb.CreateTableInput{
			TableName: aws.String(s.execTableName),
			AttributeDefinitions: []*dynamodb.AttributeDefinition{
				{
					AttributeName: aws.String("ID"),
					AttributeType: aws.String("S"),
				},
				{
					AttributeName: aws.String("AccountID"),
					AttributeType: aws.String("S"),
				},
				{
					AttributeName: aws.String("StartTime"),
					AttributeType: aws.String("N"),
				},
			},
			KeySchema: []*dynamodb.KeySchemaElement{
				{
					AttributeName: aws.String("ID"),
					KeyType:       aws.String("HASH"),
				},
			},
			GlobalSecondaryIndexes: []*dynamodb.GlobalSecondaryIndex{
				{
					IndexName: aws.String("AccountIndex"),
					KeySchema: []*dynamodb.KeySchemaElement{
						{
							AttributeName: aws.String("AccountID"),
							KeyType:       aws.String("HASH"),
						},
						{
							AttributeName: aws.String("StartTime"),
							KeyType:       aws.String("RANGE"),
						},
					},
					Projection: &dynamodb.Projection{
						ProjectionType: aws.String("ALL"),
					},
				},
			},
			BillingMode: aws.String("PAY_PER_REQUEST"),
		})

		if err != nil {
			return fmt.Errorf("failed to create executions table: %w", err)
		}

		// Wait for table to be created
		err = s.client.WaitUntilTableExists(&dynamodb.DescribeTableInput{
			TableName: aws.String(s.execTableName),
		})

		if err != nil {
			return fmt.Errorf("failed to wait for executions table creation: %w", err)
		}

		return nil
	}

	return fmt.Errorf("failed to check if executions table exists: %w", err)
}

// initializeExecutionLogsTable creates the execution logs table if it doesn't exist
func (s *DynamoDBExecutionStore) initializeExecutionLogsTable() error {
	// Check if table exists
	_, err := s.client.DescribeTable(&dynamodb.DescribeTableInput{
		TableName: aws.String(s.logsTableName),
	})

	if err == nil {
		// Table exists
		return nil
	}

	// Check if error is "table not found"
	if aerr, ok := err.(awserr.Error); ok && aerr.Code() == dynamodb.ErrCodeResourceNotFoundException {
		// Create table
		_, err = s.client.CreateTable(&dynamodb.CreateTableInput{
			TableName: aws.String(s.logsTableName),
			AttributeDefinitions: []*dynamodb.AttributeDefinition{
				{
					AttributeName: aws.String("ExecutionID"),
					AttributeType: aws.String("S"),
				},
				{
					AttributeName: aws.String("Timestamp"),
					AttributeType: aws.String("N"),
				},
			},
			KeySchema: []*dynamodb.KeySchemaElement{
				{
					AttributeName: aws.String("ExecutionID"),
					KeyType:       aws.String("HASH"),
				},
				{
					AttributeName: aws.String("Timestamp"),
					KeyType:       aws.String("RANGE"),
				},
			},
			BillingMode: aws.String("PAY_PER_REQUEST"),
		})

		if err != nil {
			return fmt.Errorf("failed to create execution logs table: %w", err)
		}

		// Wait for table to be created
		err = s.client.WaitUntilTableExists(&dynamodb.DescribeTableInput{
			TableName: aws.String(s.logsTableName),
		})

		if err != nil {
			return fmt.Errorf("failed to wait for execution logs table creation: %w", err)
		}

		return nil
	}

	return fmt.Errorf("failed to check if execution logs table exists: %w", err)
}

// SaveExecution persists execution data
func (s *DynamoDBExecutionStore) SaveExecution(execution runtime.ExecutionStatus) error {
	// Get account ID from metadata if available
	accountID := "default-account"
	if execution.Metadata != nil {
		if id, ok := execution.Metadata["account_id"]; ok && id != "" {
			accountID = id
		}
	}

	// Convert time fields to Unix timestamps for DynamoDB
	startTimeUnix := int64(0)
	if !execution.StartTime.IsZero() {
		startTimeUnix = execution.StartTime.Unix()
	}

	endTimeUnix := int64(0)
	if !execution.EndTime.IsZero() {
		endTimeUnix = execution.EndTime.Unix()
	}

	// Create item directly with AttributeValue map to ensure all required fields are set
	av := map[string]*dynamodb.AttributeValue{
		"ID": {
			S: aws.String(execution.ID),
		},
		"AccountID": {
			S: aws.String(accountID),
		},
		"StartTime": {
			N: aws.String(strconv.FormatInt(startTimeUnix, 10)),
		},
	}

	// Add optional fields
	if execution.FlowID != "" {
		av["FlowID"] = &dynamodb.AttributeValue{S: aws.String(execution.FlowID)}
	}

	if execution.Status != "" {
		av["Status"] = &dynamodb.AttributeValue{S: aws.String(execution.Status)}
	}

	if endTimeUnix > 0 {
		av["EndTime"] = &dynamodb.AttributeValue{N: aws.String(strconv.FormatInt(endTimeUnix, 10))}
	}

	if execution.Error != "" {
		av["Error"] = &dynamodb.AttributeValue{S: aws.String(execution.Error)}
	}

	if execution.CurrentNode != "" {
		av["CurrentNode"] = &dynamodb.AttributeValue{S: aws.String(execution.CurrentNode)}
	}

	av["Progress"] = &dynamodb.AttributeValue{N: aws.String(strconv.FormatFloat(execution.Progress, 'f', -1, 64))}

	// Save execution
	_, err := s.client.PutItem(&dynamodb.PutItemInput{
		TableName: aws.String(s.execTableName),
		Item:      av,
	})

	if err != nil {
		return fmt.Errorf("failed to save execution: %w", err)
	}

	return nil
}

// GetExecution retrieves execution data
func (s *DynamoDBExecutionStore) GetExecution(executionID string) (runtime.ExecutionStatus, error) {
	// Get execution
	result, err := s.client.GetItem(&dynamodb.GetItemInput{
		TableName: aws.String(s.execTableName),
		Key: map[string]*dynamodb.AttributeValue{
			"ID": {
				S: aws.String(executionID),
			},
		},
	})

	if err != nil {
		return runtime.ExecutionStatus{}, fmt.Errorf("failed to get execution: %w", err)
	}

	if result.Item == nil {
		return runtime.ExecutionStatus{}, ErrExecutionNotFound
	}

	// Create execution status manually from the DynamoDB item
	execution := runtime.ExecutionStatus{
		ID: executionID,
	}

	// Extract fields from the DynamoDB item
	if v, ok := result.Item["FlowID"]; ok && v.S != nil {
		execution.FlowID = *v.S
	}

	if v, ok := result.Item["Status"]; ok && v.S != nil {
		execution.Status = *v.S
	}

	if v, ok := result.Item["Error"]; ok && v.S != nil {
		execution.Error = *v.S
	}

	if v, ok := result.Item["CurrentNode"]; ok && v.S != nil {
		execution.CurrentNode = *v.S
	}

	if v, ok := result.Item["Progress"]; ok && v.N != nil {
		if progress, err := strconv.ParseFloat(*v.N, 64); err == nil {
			execution.Progress = progress
		}
	}

	// Convert Unix timestamps back to time.Time
	if v, ok := result.Item["StartTime"]; ok && v.N != nil {
		if startTime, err := strconv.ParseInt(*v.N, 10, 64); err == nil {
			execution.StartTime = time.Unix(startTime, 0)
		}
	}

	if v, ok := result.Item["EndTime"]; ok && v.N != nil {
		if endTime, err := strconv.ParseInt(*v.N, 10, 64); err == nil {
			execution.EndTime = time.Unix(endTime, 0)
		}
	}

	// Extract results if available
	if v, ok := result.Item["Results"]; ok && v.M != nil {
		results := make(map[string]interface{})
		if err := dynamodbattribute.UnmarshalMap(v.M, &results); err == nil {
			execution.Results = results
		}
	}

	// Extract metadata if available
	if v, ok := result.Item["Metadata"]; ok && v.M != nil {
		metadata := make(map[string]string)
		if err := dynamodbattribute.UnmarshalMap(v.M, &metadata); err == nil {
			execution.Metadata = metadata
		}
	}

	return execution, nil
}

// ListExecutions returns all executions for an account
func (s *DynamoDBExecutionStore) ListExecutions(accountID string) ([]runtime.ExecutionStatus, error) {
	// Create query expression
	keyCond := expression.Key("AccountID").Equal(expression.Value(accountID))
	expr, err := expression.NewBuilder().WithKeyCondition(keyCond).Build()
	if err != nil {
		return nil, fmt.Errorf("failed to build expression: %w", err)
	}

	// Query executions
	result, err := s.client.Query(&dynamodb.QueryInput{
		TableName:                 aws.String(s.execTableName),
		IndexName:                 aws.String("AccountIndex"),
		KeyConditionExpression:    expr.KeyCondition(),
		ExpressionAttributeNames:  expr.Names(),
		ExpressionAttributeValues: expr.Values(),
		ScanIndexForward:          aws.Bool(false), // Sort by StartTime descending
	})

	if err != nil {
		return nil, fmt.Errorf("failed to query executions: %w", err)
	}

	// Extract executions
	executions := make([]runtime.ExecutionStatus, 0, len(result.Items))
	for _, item := range result.Items {
		var execItem struct {
			runtime.ExecutionStatus
			StartTimeUnix int64 `json:"StartTime"`
			EndTimeUnix   int64 `json:"EndTime"`
		}
		if err := dynamodbattribute.UnmarshalMap(item, &execItem); err != nil {
			return nil, fmt.Errorf("failed to unmarshal execution: %w", err)
		}

		// Convert Unix timestamps back to time.Time
		execution := execItem.ExecutionStatus
		execution.StartTime = time.Unix(execItem.StartTimeUnix, 0)
		execution.EndTime = time.Unix(execItem.EndTimeUnix, 0)

		executions = append(executions, execution)
	}

	return executions, nil
}

// SaveExecutionLog persists an execution log entry
func (s *DynamoDBExecutionStore) SaveExecutionLog(executionID string, log runtime.ExecutionLog) error {
	// Convert time field to Unix timestamp
	item := struct {
		ExecutionID string `json:"ExecutionID"`
		Timestamp   int64  `json:"Timestamp"`
		NodeID      string `json:"NodeID"`
		Level       string `json:"Level"`
		Message     string `json:"Message"`
		Data        string `json:"Data"`
	}{
		ExecutionID: executionID,
		Timestamp:   log.Timestamp.UnixNano(),
		NodeID:      log.NodeID,
		Level:       log.Level,
		Message:     log.Message,
	}

	// Marshal log data to JSON
	if log.Data != nil {
		dataJSON, err := json.Marshal(log.Data)
		if err != nil {
			return fmt.Errorf("failed to marshal log data: %w", err)
		}
		item.Data = string(dataJSON)
	}

	// Marshal log entry
	av, err := dynamodbattribute.MarshalMap(item)
	if err != nil {
		return fmt.Errorf("failed to marshal log entry: %w", err)
	}

	// Save log entry
	_, err = s.client.PutItem(&dynamodb.PutItemInput{
		TableName: aws.String(s.logsTableName),
		Item:      av,
	})

	if err != nil {
		return fmt.Errorf("failed to save log entry: %w", err)
	}

	return nil
}

// GetExecutionLogs retrieves logs for an execution
func (s *DynamoDBExecutionStore) GetExecutionLogs(executionID string) ([]runtime.ExecutionLog, error) {
	// Create query expression
	keyCond := expression.Key("ExecutionID").Equal(expression.Value(executionID))
	expr, err := expression.NewBuilder().WithKeyCondition(keyCond).Build()
	if err != nil {
		return nil, fmt.Errorf("failed to build expression: %w", err)
	}

	// Query logs
	result, err := s.client.Query(&dynamodb.QueryInput{
		TableName:                 aws.String(s.logsTableName),
		KeyConditionExpression:    expr.KeyCondition(),
		ExpressionAttributeNames:  expr.Names(),
		ExpressionAttributeValues: expr.Values(),
		ScanIndexForward:          aws.Bool(true), // Sort by Timestamp ascending
	})

	if err != nil {
		return nil, fmt.Errorf("failed to query logs: %w", err)
	}

	// Extract logs
	logs := make([]runtime.ExecutionLog, 0, len(result.Items))
	for _, item := range result.Items {
		var logItem struct {
			ExecutionID string `json:"ExecutionID"`
			Timestamp   int64  `json:"Timestamp"`
			NodeID      string `json:"NodeID"`
			Level       string `json:"Level"`
			Message     string `json:"Message"`
			Data        string `json:"Data"`
		}
		if err := dynamodbattribute.UnmarshalMap(item, &logItem); err != nil {
			return nil, fmt.Errorf("failed to unmarshal log entry: %w", err)
		}

		// Create log entry
		log := runtime.ExecutionLog{
			Timestamp: time.Unix(0, logItem.Timestamp),
			NodeID:    logItem.NodeID,
			Level:     logItem.Level,
			Message:   logItem.Message,
		}

		// Unmarshal data if present
		if logItem.Data != "" {
			var data map[string]interface{}
			if err := json.Unmarshal([]byte(logItem.Data), &data); err != nil {
				return nil, fmt.Errorf("failed to unmarshal log data: %w", err)
			}
			log.Data = data
		}

		logs = append(logs, log)
	}

	return logs, nil
}

// DynamoDBAccountStore implements the AccountStore interface using DynamoDB
type DynamoDBAccountStore struct {
	client      dynamodbiface.DynamoDBAPI
	tablePrefix string
	tableName   string
}

// NewDynamoDBAccountStore creates a new DynamoDB account store
func NewDynamoDBAccountStore(client dynamodbiface.DynamoDBAPI, tablePrefix string) *DynamoDBAccountStore {
	return &DynamoDBAccountStore{
		client:      client,
		tablePrefix: tablePrefix,
		tableName:   tablePrefix + "accounts",
	}
}

// Initialize creates the DynamoDB table if it doesn't exist
func (s *DynamoDBAccountStore) Initialize() error {
	// Check if table exists
	_, err := s.client.DescribeTable(&dynamodb.DescribeTableInput{
		TableName: aws.String(s.tableName),
	})

	if err == nil {
		// Table exists
		return nil
	}

	// Check if error is "table not found"
	if aerr, ok := err.(awserr.Error); ok && aerr.Code() == dynamodb.ErrCodeResourceNotFoundException {
		// Create table
		_, err = s.client.CreateTable(&dynamodb.CreateTableInput{
			TableName: aws.String(s.tableName),
			AttributeDefinitions: []*dynamodb.AttributeDefinition{
				{
					AttributeName: aws.String("ID"),
					AttributeType: aws.String("S"),
				},
				{
					AttributeName: aws.String("Username"),
					AttributeType: aws.String("S"),
				},
				{
					AttributeName: aws.String("APIToken"),
					AttributeType: aws.String("S"),
				},
			},
			KeySchema: []*dynamodb.KeySchemaElement{
				{
					AttributeName: aws.String("ID"),
					KeyType:       aws.String("HASH"),
				},
			},
			GlobalSecondaryIndexes: []*dynamodb.GlobalSecondaryIndex{
				{
					IndexName: aws.String("UsernameIndex"),
					KeySchema: []*dynamodb.KeySchemaElement{
						{
							AttributeName: aws.String("Username"),
							KeyType:       aws.String("HASH"),
						},
					},
					Projection: &dynamodb.Projection{
						ProjectionType: aws.String("ALL"),
					},
				},
				{
					IndexName: aws.String("TokenIndex"),
					KeySchema: []*dynamodb.KeySchemaElement{
						{
							AttributeName: aws.String("APIToken"),
							KeyType:       aws.String("HASH"),
						},
					},
					Projection: &dynamodb.Projection{
						ProjectionType: aws.String("ALL"),
					},
				},
			},
			BillingMode: aws.String("PAY_PER_REQUEST"),
		})

		if err != nil {
			return fmt.Errorf("failed to create accounts table: %w", err)
		}

		// Wait for table to be created
		err = s.client.WaitUntilTableExists(&dynamodb.DescribeTableInput{
			TableName: aws.String(s.tableName),
		})

		if err != nil {
			return fmt.Errorf("failed to wait for accounts table creation: %w", err)
		}

		return nil
	}

	return fmt.Errorf("failed to check if accounts table exists: %w", err)
}

// SaveAccount persists an account
func (s *DynamoDBAccountStore) SaveAccount(account auth.Account) error {
	// Create DynamoDB-specific item structure
	item := struct {
		ID           string `json:"ID"`
		Username     string `json:"Username"`
		PasswordHash string `json:"PasswordHash"`
		APIToken     string `json:"APIToken"`
		CreatedAt    int64  `json:"CreatedAt"`
		UpdatedAt    int64  `json:"UpdatedAt"`
	}{
		ID:           account.ID,
		Username:     account.Username,
		PasswordHash: account.PasswordHash,
		APIToken:     account.APIToken,
		CreatedAt:    account.CreatedAt.Unix(),
		UpdatedAt:    account.UpdatedAt.Unix(),
	}

	// Marshal account
	av, err := dynamodbattribute.MarshalMap(item)
	if err != nil {
		return fmt.Errorf("failed to marshal account: %w", err)
	}

	// Save account
	_, err = s.client.PutItem(&dynamodb.PutItemInput{
		TableName: aws.String(s.tableName),
		Item:      av,
	})

	if err != nil {
		return fmt.Errorf("failed to save account: %w", err)
	}

	return nil
}

// GetAccount retrieves an account
func (s *DynamoDBAccountStore) GetAccount(accountID string) (auth.Account, error) {
	// Get account
	result, err := s.client.GetItem(&dynamodb.GetItemInput{
		TableName: aws.String(s.tableName),
		Key: map[string]*dynamodb.AttributeValue{
			"ID": {
				S: aws.String(accountID),
			},
		},
	})

	if err != nil {
		return auth.Account{}, fmt.Errorf("failed to get account: %w", err)
	}

	if result.Item == nil {
		return auth.Account{}, ErrAccountNotFound
	}

	// Unmarshal account with explicit field mapping
	var item struct {
		ID           string `json:"ID"`
		Username     string `json:"Username"`
		PasswordHash string `json:"PasswordHash"`
		APIToken     string `json:"APIToken"`
		CreatedAt    int64  `json:"CreatedAt"`
		UpdatedAt    int64  `json:"UpdatedAt"`
	}
	if err := dynamodbattribute.UnmarshalMap(result.Item, &item); err != nil {
		return auth.Account{}, fmt.Errorf("failed to unmarshal account: %w", err)
	}

	// Convert to auth.Account with proper timestamp conversion
	account := auth.Account{
		ID:           item.ID,
		Username:     item.Username,
		PasswordHash: item.PasswordHash,
		APIToken:     item.APIToken,
		CreatedAt:    time.Unix(item.CreatedAt, 0),
		UpdatedAt:    time.Unix(item.UpdatedAt, 0),
	}

	return account, nil
}

// GetAccountByUsername retrieves an account by username
func (s *DynamoDBAccountStore) GetAccountByUsername(username string) (auth.Account, error) {
	// Create query expression
	keyCond := expression.Key("Username").Equal(expression.Value(username))
	expr, err := expression.NewBuilder().WithKeyCondition(keyCond).Build()
	if err != nil {
		return auth.Account{}, fmt.Errorf("failed to build expression: %w", err)
	}

	// Query accounts
	result, err := s.client.Query(&dynamodb.QueryInput{
		TableName:                 aws.String(s.tableName),
		IndexName:                 aws.String("UsernameIndex"),
		KeyConditionExpression:    expr.KeyCondition(),
		ExpressionAttributeNames:  expr.Names(),
		ExpressionAttributeValues: expr.Values(),
	})

	if err != nil {
		return auth.Account{}, fmt.Errorf("failed to query accounts: %w", err)
	}

	if len(result.Items) == 0 {
		return auth.Account{}, ErrAccountNotFound
	}

	// Unmarshal account with explicit field mapping
	var item struct {
		ID           string `json:"ID"`
		Username     string `json:"Username"`
		PasswordHash string `json:"PasswordHash"`
		APIToken     string `json:"APIToken"`
		CreatedAt    int64  `json:"CreatedAt"`
		UpdatedAt    int64  `json:"UpdatedAt"`
	}
	if err := dynamodbattribute.UnmarshalMap(result.Items[0], &item); err != nil {
		return auth.Account{}, fmt.Errorf("failed to unmarshal account: %w", err)
	}

	// Convert to auth.Account with proper timestamp conversion
	account := auth.Account{
		ID:           item.ID,
		Username:     item.Username,
		PasswordHash: item.PasswordHash,
		APIToken:     item.APIToken,
		CreatedAt:    time.Unix(item.CreatedAt, 0),
		UpdatedAt:    time.Unix(item.UpdatedAt, 0),
	}

	return account, nil
}

// GetAccountByToken retrieves an account by API token
func (s *DynamoDBAccountStore) GetAccountByToken(token string) (auth.Account, error) {
	// Create query expression
	keyCond := expression.Key("APIToken").Equal(expression.Value(token))
	expr, err := expression.NewBuilder().WithKeyCondition(keyCond).Build()
	if err != nil {
		return auth.Account{}, fmt.Errorf("failed to build expression: %w", err)
	}

	// Query accounts
	result, err := s.client.Query(&dynamodb.QueryInput{
		TableName:                 aws.String(s.tableName),
		IndexName:                 aws.String("TokenIndex"),
		KeyConditionExpression:    expr.KeyCondition(),
		ExpressionAttributeNames:  expr.Names(),
		ExpressionAttributeValues: expr.Values(),
	})

	if err != nil {
		return auth.Account{}, fmt.Errorf("failed to query accounts: %w", err)
	}

	if len(result.Items) == 0 {
		return auth.Account{}, ErrAccountNotFound
	}

	// Unmarshal account with explicit field mapping
	var item struct {
		ID           string `json:"ID"`
		Username     string `json:"Username"`
		PasswordHash string `json:"PasswordHash"`
		APIToken     string `json:"APIToken"`
		CreatedAt    int64  `json:"CreatedAt"`
		UpdatedAt    int64  `json:"UpdatedAt"`
	}
	if err := dynamodbattribute.UnmarshalMap(result.Items[0], &item); err != nil {
		return auth.Account{}, fmt.Errorf("failed to unmarshal account: %w", err)
	}

	// Convert to auth.Account with proper timestamp conversion
	account := auth.Account{
		ID:           item.ID,
		Username:     item.Username,
		PasswordHash: item.PasswordHash,
		APIToken:     item.APIToken,
		CreatedAt:    time.Unix(item.CreatedAt, 0),
		UpdatedAt:    time.Unix(item.UpdatedAt, 0),
	}

	return account, nil
}

// ListAccounts returns all accounts
func (s *DynamoDBAccountStore) ListAccounts() ([]auth.Account, error) {
	// Scan accounts
	result, err := s.client.Scan(&dynamodb.ScanInput{
		TableName: aws.String(s.tableName),
	})

	if err != nil {
		return nil, fmt.Errorf("failed to scan accounts: %w", err)
	}

	// Extract accounts
	accounts := make([]auth.Account, 0, len(result.Items))
	for _, item := range result.Items {
		var accItem struct {
			ID           string `json:"ID"`
			Username     string `json:"Username"`
			PasswordHash string `json:"PasswordHash"`
			APIToken     string `json:"APIToken"`
			CreatedAt    int64  `json:"CreatedAt"`
			UpdatedAt    int64  `json:"UpdatedAt"`
		}
		if err := dynamodbattribute.UnmarshalMap(item, &accItem); err != nil {
			return nil, fmt.Errorf("failed to unmarshal account: %w", err)
		}

		// Convert to auth.Account with proper timestamp conversion
		account := auth.Account{
			ID:           accItem.ID,
			Username:     accItem.Username,
			PasswordHash: accItem.PasswordHash,
			APIToken:     accItem.APIToken,
			CreatedAt:    time.Unix(accItem.CreatedAt, 0),
			UpdatedAt:    time.Unix(accItem.UpdatedAt, 0),
		}

		accounts = append(accounts, account)
	}

	return accounts, nil
}

// DeleteAccount removes an account
func (s *DynamoDBAccountStore) DeleteAccount(accountID string) error {
	// Delete account
	_, err := s.client.DeleteItem(&dynamodb.DeleteItemInput{
		TableName: aws.String(s.tableName),
		Key: map[string]*dynamodb.AttributeValue{
			"ID": {
				S: aws.String(accountID),
			},
		},
		ConditionExpression: aws.String("attribute_exists(ID)"),
	})

	if err != nil {
		if aerr, ok := err.(awserr.Error); ok && aerr.Code() == dynamodb.ErrCodeConditionalCheckFailedException {
			return ErrAccountNotFound
		}
		return fmt.Errorf("failed to delete account: %w", err)
	}

	return nil
}

// SaveFlowVersion persists a new version of a flow definition
func (s *DynamoDBFlowStore) SaveFlowVersion(accountID, flowID string, definition []byte, version string) error {
	// Extract metadata from the definition
	var metadata struct {
		Metadata struct {
			Name        string `json:"name"`
			Description string `json:"description"`
			Version     string `json:"version"`
		} `json:"metadata"`
	}

	if err := json.Unmarshal(definition, &metadata); err != nil {
		// If we can't extract metadata, just use empty values
		metadata.Metadata.Name = flowID
		metadata.Metadata.Description = ""
	}

	// First, update the main flow record with the new version
	now := time.Now().Unix()
	item := dynamoDBFlowItem{
		AccountID:   accountID,
		FlowID:      flowID,
		Definition:  string(definition),
		Name:        metadata.Metadata.Name,
		Description: metadata.Metadata.Description,
		Version:     version,
		UpdatedAt:   now,
	}

	// Check if flow already exists
	result, err := s.client.GetItem(&dynamodb.GetItemInput{
		TableName: aws.String(s.tableName),
		Key: map[string]*dynamodb.AttributeValue{
			"AccountID": {
				S: aws.String(accountID),
			},
			"FlowID": {
				S: aws.String(flowID),
			},
		},
	})

	if err != nil {
		return fmt.Errorf("failed to check if flow exists: %w", err)
	}

	if result.Item == nil {
		return ErrFlowNotFound
	}

	// Existing flow, preserve creation time
	var existingItem dynamoDBFlowItem
	if err := dynamodbattribute.UnmarshalMap(result.Item, &existingItem); err != nil {
		return fmt.Errorf("failed to unmarshal existing flow: %w", err)
	}
	item.CreatedAt = existingItem.CreatedAt

	// Marshal item
	av, err := dynamodbattribute.MarshalMap(item)
	if err != nil {
		return fmt.Errorf("failed to marshal flow item: %w", err)
	}

	// Update flow
	_, err = s.client.PutItem(&dynamodb.PutItemInput{
		TableName: aws.String(s.tableName),
		Item:      av,
	})

	if err != nil {
		return fmt.Errorf("failed to update flow: %w", err)
	}

	// Then, store the version in the flow_versions table
	versionItem := struct {
		AccountID   string `json:"AccountID"`
		FlowID      string `json:"FlowID"`
		Version     string `json:"Version"`
		Description string `json:"Description"`
		Definition  string `json:"Definition"`
		CreatedAt   int64  `json:"CreatedAt"`
		CreatedBy   string `json:"CreatedBy,omitempty"`
	}{
		AccountID:   accountID,
		FlowID:      flowID,
		Version:     version,
		Description: metadata.Metadata.Description,
		Definition:  string(definition),
		CreatedAt:   now,
	}

	// Marshal version item
	versionAV, err := dynamodbattribute.MarshalMap(versionItem)
	if err != nil {
		return fmt.Errorf("failed to marshal flow version item: %w", err)
	}

	// Save flow version
	_, err = s.client.PutItem(&dynamodb.PutItemInput{
		TableName: aws.String(s.tableName + "_versions"),
		Item:      versionAV,
	})

	if err != nil {
		return fmt.Errorf("failed to save flow version: %w", err)
	}

	return nil
}

// GetFlowVersion retrieves a specific version of a flow definition
func (s *DynamoDBFlowStore) GetFlowVersion(accountID, flowID, version string) ([]byte, error) {
	// Get flow version
	result, err := s.client.GetItem(&dynamodb.GetItemInput{
		TableName: aws.String(s.tableName + "_versions"),
		Key: map[string]*dynamodb.AttributeValue{
			"AccountID": {
				S: aws.String(accountID),
			},
			"FlowID": {
				S: aws.String(flowID),
			},
			"Version": {
				S: aws.String(version),
			},
		},
	})

	if err != nil {
		return nil, fmt.Errorf("failed to get flow version: %w", err)
	}

	if result.Item == nil {
		return nil, ErrFlowNotFound
	}

	// Unmarshal item
	var item struct {
		Definition string `json:"Definition"`
	}
	if err := dynamodbattribute.UnmarshalMap(result.Item, &item); err != nil {
		return nil, fmt.Errorf("failed to unmarshal flow version item: %w", err)
	}

	return []byte(item.Definition), nil
}

// ListFlowVersions returns all versions of a flow
func (s *DynamoDBFlowStore) ListFlowVersions(accountID, flowID string) ([]string, error) {
	// Create query expression
	keyCond := expression.Key("AccountID").Equal(expression.Value(accountID)).
		And(expression.Key("FlowID").Equal(expression.Value(flowID)))
	expr, err := expression.NewBuilder().WithKeyCondition(keyCond).Build()
	if err != nil {
		return nil, fmt.Errorf("failed to build expression: %w", err)
	}

	// Query flow versions
	result, err := s.client.Query(&dynamodb.QueryInput{
		TableName:                 aws.String(s.tableName + "_versions"),
		KeyConditionExpression:    expr.KeyCondition(),
		ExpressionAttributeNames:  expr.Names(),
		ExpressionAttributeValues: expr.Values(),
	})

	if err != nil {
		return nil, fmt.Errorf("failed to query flow versions: %w", err)
	}

	// Extract versions
	versions := make([]string, 0, len(result.Items))
	for _, item := range result.Items {
		version := item["Version"].S
		if version != nil {
			versions = append(versions, *version)
		}
	}

	return versions, nil
}

// SaveFlowVersion persists a new version of a flow definition
