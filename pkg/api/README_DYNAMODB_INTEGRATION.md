# DynamoDB WebSocket Integration Tests

This directory contains comprehensive integration tests that demonstrate the FlowRunner system working with DynamoDB backend and real-time WebSocket updates.

## Test Overview

### TestWebSocketDynamoDBIntegration_ComplexFlow
A comprehensive test that demonstrates:
- **DynamoDB Storage Backend**: All data (accounts, secrets, flows, executions) stored in DynamoDB
- **Parallel Batch Processing**: Processing multiple items concurrently with configurable parallelism
- **Retry Logic**: Automatic retry with exponential backoff for failed operations
- **Conditional Branching**: Dynamic flow routing based on processing results
- **Real-time WebSocket Updates**: Live status updates during execution
- **Concurrent Executions**: Multiple flows running simultaneously

### TestWebSocketDynamoDBIntegration_SimpleBranching
A simpler test that focuses on:
- Basic branching logic based on data values
- Retry mechanisms with WebSocket monitoring
- DynamoDB backend integration
- Easier to run and debug

## Prerequisites

### Local DynamoDB Setup
1. Install and run local DynamoDB:
   ```bash
   # Using Docker
   docker run -p 8000:8000 amazon/dynamodb-local
   
   # Or download and run the JAR file
   java -Djava.library.path=./DynamoDBLocal_lib -jar DynamoDBLocal.jar -sharedDb
   ```

2. The tests will automatically create the required tables with the specified prefix.

### Environment Variables
Create a `.env` file in the project root with:
```bash
# DynamoDB Configuration
FLOWRUNNER_DYNAMODB_ENDPOINT=http://localhost:8000
FLOWRUNNER_DYNAMODB_REGION=us-east-1
FLOWRUNNER_DYNAMODB_TABLE_PREFIX=flowrunner_test_
```

**Note**: The tests automatically load the `.env` file using `godotenv.Load()`, so you don't need to export environment variables manually.

## Running the Tests

### Quick Start (Recommended)
```bash
# Use the provided test script
./scripts/test_dynamodb_integration.sh
```

This script will:
- Load environment variables from `.env` file
- Check DynamoDB connectivity
- Create test tables if needed
- Run both simple and complex tests
- Provide detailed output and summary

### Manual Test Execution

#### Simple Branching Test (Recommended for first run)
```bash
# Just ensure your .env file has DynamoDB configuration, then:
go test -v ./pkg/api -run TestWebSocketDynamoDBIntegration_SimpleBranching
```

#### Complex Flow Test (Full feature demonstration)
```bash
# Just ensure your .env file has DynamoDB configuration, then:
go test -v ./pkg/api -run TestWebSocketDynamoDBIntegration_ComplexFlow
```

#### Run All DynamoDB Integration Tests
```bash
# The tests will automatically load .env file
go test -v ./pkg/api -run "TestWebSocketDynamoDB.*"
```

## Test Features Demonstrated

### 1. Complex Flow Structure
The complex flow demonstrates a real-world scenario:
```
Data Preparation → Parallel Processing → Result Analysis → Branching → Final Summary
                     ↓ (on failure)
                 Retry Handler → Error Recovery
```

### 2. Parallel Batch Processing
- Configurable batch sizes and parallelism
- Timeout handling for long-running operations
- Failure isolation (one batch failure doesn't stop others)

### 3. Retry Logic
- Exponential backoff retry strategy
- Per-node retry configuration
- Retry exhaustion handling

### 4. Conditional Branching
The flow branches based on success rates:
- **High Success (≥80%)**: Proceed to next stage
- **Partial Success (50-79%)**: Review and retry strategy
- **Low Success (<50%)**: Escalate to admin

### 5. WebSocket Real-time Updates
- Live execution status updates
- Progress tracking
- Current node information
- Error notifications

### 6. DynamoDB Integration
- All storage operations use DynamoDB
- Concurrent access handling
- Transaction management
- Connection pooling

## Expected Test Output

### Successful Complex Flow Execution
```
=== RUN   TestWebSocketDynamoDBIntegration_ComplexFlow
    websocket_dynamodb_integration_test.go:89: DynamoDB integration test config: endpoint=http://localhost:8000, region=us-east-1, tablePrefix=flowrunner_test_
    websocket_dynamodb_integration_test.go:120: Created complex flow with ID: complex-parallel-batch-flow-1234567890
=== RUN   TestWebSocketDynamoDBIntegration_ComplexFlow/Concurrent_Execution_1
    websocket_dynamodb_integration_test.go:456: Execution 1: Status=completed, Updates=8, Duration=2.3s
=== RUN   TestWebSocketDynamoDBIntegration_ComplexFlow/Concurrent_Execution_2
    websocket_dynamodb_integration_test.go:456: Execution 2: Status=completed, Updates=7, Duration=2.1s
=== RUN   TestWebSocketDynamoDBIntegration_ComplexFlow/Concurrent_Execution_3
    websocket_dynamodb_integration_test.go:456: Execution 3: Status=completed, Updates=9, Duration=2.5s
    websocket_dynamodb_integration_test.go:465: DynamoDB Complex Flow Integration Test completed successfully!
--- PASS: TestWebSocketDynamoDBIntegration_ComplexFlow (7.2s)
```

### WebSocket Update Examples
```
Status Update: running, Progress: 10.0%, Node: prepare_data
Status Update: running, Progress: 30.0%, Node: parallel_processor
Status Update: running, Progress: 60.0%, Node: analyze_results
Status Update: running, Progress: 80.0%, Node: high_success_handler
Status Update: completed, Progress: 100.0%, Node: final_summary
```

## Troubleshooting

### Common Issues

1. **DynamoDB Connection Failed**
   ```
   Failed to create DynamoDB provider: connection refused
   ```
   - Ensure local DynamoDB is running
   - Check endpoint configuration
   - Verify port is accessible

2. **Tables Do Not Exist**
   ```
   Failed to initialize DynamoDB provider: table does not exist
   ```
   - The provider should create tables automatically
   - Ensure you have proper permissions
   - Check table prefix configuration

3. **WebSocket Connection Failed**
   ```
   WebSocket connection failed: websocket: bad handshake
   ```
   - Check authentication credentials
   - Verify server is running
   - Check for port conflicts

4. **Flow Execution Timeout**
   ```
   Execution timed out after 60 seconds
   ```
   - Complex flows may take longer
   - Check DynamoDB performance
   - Increase timeout in test configuration

### Performance Considerations

- **Local DynamoDB Performance**: Local DynamoDB has different performance characteristics than AWS DynamoDB
- **Connection Limits**: Adjust connection settings based on your environment
- **Batch Sizes**: Adjust batch sizes based on system capacity
- **Timeout Values**: Set appropriate timeouts for your environment

## Architecture Benefits Demonstrated

1. **Scalability**: DynamoDB backend supports high concurrency
2. **Reliability**: Retry logic and error handling ensure robustness
3. **Observability**: Real-time WebSocket updates provide visibility
4. **Flexibility**: Conditional branching enables complex business logic
5. **Performance**: Parallel processing maximizes throughput

## Next Steps

After running these tests successfully, you can:
1. Modify the flow definitions to test your specific use cases
2. Adjust batch sizes and parallelism for your workload
3. Add custom node types for your business logic
4. Implement custom retry strategies
5. Add monitoring and alerting based on WebSocket updates

These tests demonstrate that the FlowRunner system is production-ready for enterprise workloads with DynamoDB backend and real-time monitoring capabilities.