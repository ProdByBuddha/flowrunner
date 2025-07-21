# PostgreSQL WebSocket Integration Tests

This directory contains comprehensive integration tests that demonstrate the FlowRunner system working with PostgreSQL backend and real-time WebSocket updates.

## Test Overview

### TestWebSocketPostgreSQLIntegration_ComplexFlow
A comprehensive test that demonstrates:
- **PostgreSQL Storage Backend**: All data (accounts, secrets, flows, executions) stored in PostgreSQL
- **Parallel Batch Processing**: Processing multiple items concurrently with configurable parallelism
- **Retry Logic**: Automatic retry with exponential backoff for failed operations
- **Conditional Branching**: Dynamic flow routing based on processing results
- **Real-time WebSocket Updates**: Live status updates during execution
- **Concurrent Executions**: Multiple flows running simultaneously

### TestWebSocketPostgreSQLIntegration_SimpleBranching
A simpler test that focuses on:
- Basic branching logic based on data values
- Retry mechanisms with WebSocket monitoring
- PostgreSQL backend integration
- Easier to run and debug

## Prerequisites

### PostgreSQL Setup
1. Install PostgreSQL (version 12 or higher recommended)
2. Create a test database:
   ```sql
   CREATE DATABASE flowrunner_test;
   CREATE USER flowrunner_user WITH PASSWORD 'flowrunner_password';
   GRANT ALL PRIVILEGES ON DATABASE flowrunner_test TO flowrunner_user;
   ```

### Environment Variables
Create a `.env` file in the project root with:
```bash
# PostgreSQL Configuration (standard FlowRunner env vars)
FLOWRUNNER_POSTGRES_HOST=localhost
FLOWRUNNER_POSTGRES_PORT=5432
FLOWRUNNER_POSTGRES_USER=flowrunner_user
FLOWRUNNER_POSTGRES_PASSWORD=flowrunner_password
FLOWRUNNER_POSTGRES_DATABASE=flowrunner_test
FLOWRUNNER_POSTGRES_SSL_MODE=disable
```

**Note**: The tests automatically load the `.env` file using `godotenv.Load()`, so you don't need to export environment variables manually.

## Running the Tests

### Quick Start (Recommended)
```bash
# Use the provided test script
./scripts/test_postgres_integration.sh
```

This script will:
- Load environment variables from `.env` file
- Check PostgreSQL connectivity
- Create test database if needed
- Run both simple and complex tests
- Provide detailed output and summary

### Manual Test Execution

#### Simple Branching Test (Recommended for first run)
```bash
# Just ensure your .env file has PostgreSQL configuration, then:
go test -v ./pkg/api -run TestWebSocketPostgreSQLIntegration_SimpleBranching
```

#### Complex Flow Test (Full feature demonstration)
```bash
# Just ensure your .env file has PostgreSQL configuration, then:
go test -v ./pkg/api -run TestWebSocketPostgreSQLIntegration_ComplexFlow
```

#### Run All PostgreSQL Integration Tests
```bash
# The tests will automatically load .env file
go test -v ./pkg/api -run "TestWebSocketPostgreSQL.*"
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

### 6. PostgreSQL Integration
- All storage operations use PostgreSQL
- Concurrent access handling
- Transaction management
- Connection pooling

## Expected Test Output

### Successful Complex Flow Execution
```
=== RUN   TestWebSocketPostgreSQLIntegration_ComplexFlow
    websocket_postgres_integration_test.go:89: PostgreSQL integration test config: host=localhost, port=5432, user=flowrunner_user, database=flowrunner_test, sslmode=disable
    websocket_postgres_integration_test.go:120: Created complex flow with ID: complex-parallel-batch-flow-1234567890
=== RUN   TestWebSocketPostgreSQLIntegration_ComplexFlow/Concurrent_Execution_1
    websocket_postgres_integration_test.go:456: Execution 1: Status=completed, Updates=8, Duration=2.3s
=== RUN   TestWebSocketPostgreSQLIntegration_ComplexFlow/Concurrent_Execution_2
    websocket_postgres_integration_test.go:456: Execution 2: Status=completed, Updates=7, Duration=2.1s
=== RUN   TestWebSocketPostgreSQLIntegration_ComplexFlow/Concurrent_Execution_3
    websocket_postgres_integration_test.go:456: Execution 3: Status=completed, Updates=9, Duration=2.5s
    websocket_postgres_integration_test.go:465: PostgreSQL Complex Flow Integration Test completed successfully!
--- PASS: TestWebSocketPostgreSQLIntegration_ComplexFlow (7.2s)
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

1. **PostgreSQL Connection Failed**
   ```
   Failed to create PostgreSQL provider: connection refused
   ```
   - Ensure PostgreSQL is running
   - Check host/port configuration
   - Verify user permissions

2. **Database Does Not Exist**
   ```
   Failed to initialize PostgreSQL provider: database "flowrunner_test" does not exist
   ```
   - Create the database as shown in Prerequisites
   - Ensure user has access to the database

3. **Account ID Constraint Violation** (Fixed)
   ```
   null value in column "account_id" of relation "executions" violates not-null constraint
   ```
   - This issue has been fixed in the current implementation
   - The PostgreSQL execution store now properly handles account_id assignment
   - If you still see this error, ensure you're using the latest code

4. **WebSocket Connection Failed**
   ```
   WebSocket connection failed: websocket: bad handshake
   ```
   - Check authentication credentials
   - Verify server is running
   - Check for port conflicts

5. **WebSocket Panic on Connection Close** (Fixed)
   ```
   panic: repeated read on failed websocket connection
   ```
   - This issue has been fixed with improved error handling
   - WebSocket errors now gracefully break the read loop instead of panicking

6. **Flow Execution Timeout**
   ```
   Execution timed out after 60 seconds
   ```
   - Complex flows may take longer
   - Check PostgreSQL performance
   - Increase timeout in test configuration

### Performance Considerations

- **Database Performance**: Ensure PostgreSQL has adequate resources
- **Connection Limits**: PostgreSQL default connection limit may need adjustment
- **Batch Sizes**: Adjust batch sizes based on system capacity
- **Timeout Values**: Set appropriate timeouts for your environment

## Architecture Benefits Demonstrated

1. **Scalability**: PostgreSQL backend supports high concurrency
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

These tests demonstrate that the FlowRunner system is production-ready for enterprise workloads with PostgreSQL backend and real-time monitoring capabilities.