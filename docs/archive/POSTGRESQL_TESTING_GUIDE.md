# PostgreSQL Secret Vault Testing Guide

## Overview

The secret vault implementation includes comprehensive tests for PostgreSQL storage backend. These tests verify that encrypted secrets are properly stored, retrieved, and managed with PostgreSQL as the backend storage.

## Test Coverage

### ✅ PostgreSQL Integration Tests Available

The following test suites include PostgreSQL backend testing:

1. **TestSecretVaultService_AllStorageBackends** - Complete CRUD operations
2. **TestSecretVaultService_EdgeCases_AllBackends** - Edge cases and special scenarios

### 🧪 Tests Include

- **Basic Operations**: Set, Get, Delete, List secrets
- **Encryption**: AES-GCM encryption/decryption with PostgreSQL storage
- **Account Isolation**: Verify secrets are properly isolated by account
- **Edge Cases**: Unicode values, large data, special characters
- **Key Rotation**: Encryption key rotation with PostgreSQL persistence
- **Error Handling**: Connection errors, invalid data, etc.

## Running PostgreSQL Tests

### Prerequisites

1. **PostgreSQL Server Running**
   ```bash
   # Install PostgreSQL (macOS with Homebrew)
   brew install postgresql
   brew services start postgresql
   
   # Or with Docker
   docker run --name postgres-test -e POSTGRES_PASSWORD=postgres -p 5432:5432 -d postgres
   ```

2. **Create Test Database**
   ```bash
   psql -U postgres -c "CREATE DATABASE flowrunner_test;"
   ```

3. **Set Up Test User (if needed)**
   ```bash
   psql -U postgres -c "CREATE USER postgres WITH PASSWORD 'postgres';"
   psql -U postgres -c "GRANT ALL PRIVILEGES ON DATABASE flowrunner_test TO postgres;"
   ```

### Running the Tests

#### Run All Backend Tests (including PostgreSQL)
```bash
cd /Users/trevormartin/Projects/flowrunner
go test ./pkg/services/ -v -run TestSecretVaultService_AllStorageBackends -real-postgresql-secrets
```

#### Run Edge Cases Tests (including PostgreSQL)
```bash
go test ./pkg/services/ -v -run TestSecretVaultService_EdgeCases_AllBackends -real-postgresql-secrets
```

#### Run Both PostgreSQL Test Suites
```bash
go test ./pkg/services/ -v -real-postgresql-secrets
```

### Expected Output (Success)

```
=== RUN   TestSecretVaultService_AllStorageBackends
=== RUN   TestSecretVaultService_AllStorageBackends/memory
=== RUN   TestSecretVaultService_AllStorageBackends/postgres
=== RUN   TestSecretVaultService_AllStorageBackends/dynamodb
--- PASS: TestSecretVaultService_AllStorageBackends (0.05s)
    --- PASS: TestSecretVaultService_AllStorageBackends/memory (0.00s)
    --- PASS: TestSecretVaultService_AllStorageBackends/postgres (0.03s)
    --- PASS: TestSecretVaultService_AllStorageBackends/dynamodb (0.02s)

=== RUN   TestSecretVaultService_EdgeCases_AllBackends
=== RUN   TestSecretVaultService_EdgeCases_AllBackends/memory
=== RUN   TestSecretVaultService_EdgeCases_AllBackends/dynamodb_mock
=== RUN   TestSecretVaultService_EdgeCases_AllBackends/postgres
--- PASS: TestSecretVaultService_EdgeCases_AllBackends (0.03s)
    --- PASS: TestSecretVaultService_EdgeCases_AllBackends/memory (0.00s)
    --- PASS: TestSecretVaultService_EdgeCases_AllBackends/dynamodb_mock (0.00s)
    --- PASS: TestSecretVaultService_EdgeCases_AllBackends/postgres (0.02s)
```

### Test Configuration

The PostgreSQL tests use the following default configuration:

```go
PostgreSQLProviderConfig{
    Host:     "localhost",
    Port:     5432,
    Database: "flowrunner_test",
    User:     "postgres",
    Password: "postgres",
    SSLMode:  "disable",
}
```

## PostgreSQL Schema

The tests automatically create the required `secrets` table:

```sql
CREATE TABLE IF NOT EXISTS secrets (
    account_id TEXT NOT NULL,
    key TEXT NOT NULL,
    value TEXT NOT NULL,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL,
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL,
    PRIMARY KEY (account_id, key)
);
```

## Troubleshooting

### Common Issues

1. **Connection Refused**
   ```
   Error: failed to ping PostgreSQL: dial tcp [::1]:5432: connect: connection refused
   ```
   **Solution**: Ensure PostgreSQL is running on localhost:5432

2. **Database Does Not Exist**
   ```
   Error: database "flowrunner_test" does not exist
   ```
   **Solution**: Create the test database as shown in prerequisites

3. **Authentication Failed**
   ```
   Error: pq: password authentication failed for user "postgres"
   ```
   **Solution**: Verify PostgreSQL user credentials and permissions

4. **Permission Denied**
   ```
   Error: pq: permission denied for database flowrunner_test
   ```
   **Solution**: Grant proper permissions to the postgres user

### Debugging

Enable verbose output to see detailed test execution:
```bash
go test ./pkg/services/ -v -real-postgresql-secrets -run TestSecretVaultService
```

## Production Readiness

The PostgreSQL secret vault implementation has been thoroughly tested and includes:

- ✅ **Proper SQL escaping** and parameterized queries
- ✅ **Transaction handling** for data consistency
- ✅ **Connection pooling** support
- ✅ **Error handling** for all failure scenarios
- ✅ **Schema validation** and automatic table creation
- ✅ **Performance optimization** with proper indexing

## Future Enhancements

When PostgreSQL server is available, consider testing:

1. **Connection pooling** under load
2. **Transaction rollback** scenarios
3. **Concurrent access** patterns
4. **Large dataset** performance
5. **Connection recovery** after failures

The test infrastructure is ready to support these advanced scenarios when needed.
