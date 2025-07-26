# FlowRunner Storage Configuration Guide

FlowRunner supports multiple storage backends for persisting flows, executions, and other data. This guide provides detailed instructions for configuring and using each supported storage backend.

## Table of Contents

1. [Overview](#overview)
2. [In-Memory Storage](#in-memory-storage)
3. [PostgreSQL Storage](#postgresql-storage)
4. [DynamoDB Storage](#dynamodb-storage)
5. [Storage Migration](#storage-migration)
6. [Best Practices](#best-practices)

## Overview

FlowRunner supports the following storage backends:

- **In-Memory**: Volatile storage for development and testing
- **PostgreSQL**: Relational database storage for production use
- **DynamoDB**: NoSQL database storage for AWS environments

The storage backend is configured using environment variables or a configuration file.

## In-Memory Storage

In-memory storage is the simplest option and is suitable for development and testing. Data is stored in memory and is lost when the server restarts.

### Configuration

```
# .env file
FLOWRUNNER_STORAGE_TYPE=memory
```

### Advantages

- No external dependencies
- Fast performance
- Simple setup

### Limitations

- Data is lost when the server restarts
- Not suitable for production use
- Limited scalability

### Use Cases

- Development and testing
- Demos and presentations
- Single-user environments

## PostgreSQL Storage

PostgreSQL storage provides persistent storage using a PostgreSQL database. This is suitable for production use and supports multi-user environments.

### Prerequisites

- PostgreSQL server (version 10 or higher)
- Database user with CREATE, ALTER, and SELECT privileges

### Configuration

```
# .env file
FLOWRUNNER_STORAGE_TYPE=postgres
FLOWRUNNER_POSTGRES_HOST=localhost
FLOWRUNNER_POSTGRES_PORT=5432
FLOWRUNNER_POSTGRES_DATABASE=flowrunner
FLOWRUNNER_POSTGRES_USER=postgres
FLOWRUNNER_POSTGRES_PASSWORD=postgres
FLOWRUNNER_POSTGRES_SSL_MODE=disable
```

### Database Setup

1. Create a new database:

```sql
CREATE DATABASE flowrunner;
```

2. Create a user (optional):

```sql
CREATE USER flowrunner WITH PASSWORD 'your-password';
GRANT ALL PRIVILEGES ON DATABASE flowrunner TO flowrunner;
```

3. FlowRunner will automatically create the necessary tables on startup.

### Schema

FlowRunner creates the following tables in the PostgreSQL database:

- `accounts`: User accounts
- `flows`: Flow definitions
- `executions`: Flow executions
- `execution_logs`: Execution logs
- `secrets`: Encrypted secrets
- `structured_secrets`: Structured encrypted secrets

### Connection Pooling

FlowRunner uses connection pooling to manage database connections. You can configure the pool size using the following environment variables:

```
FLOWRUNNER_POSTGRES_MAX_CONNECTIONS=10
FLOWRUNNER_POSTGRES_IDLE_CONNECTIONS=5
FLOWRUNNER_POSTGRES_CONNECTION_LIFETIME=1h
```

### SSL Configuration

To enable SSL for PostgreSQL connections:

```
FLOWRUNNER_POSTGRES_SSL_MODE=require
FLOWRUNNER_POSTGRES_SSL_CERT=/path/to/cert.pem
FLOWRUNNER_POSTGRES_SSL_KEY=/path/to/key.pem
FLOWRUNNER_POSTGRES_SSL_ROOT_CERT=/path/to/root.pem
```

SSL modes:

- `disable`: No SSL
- `require`: Always use SSL (skip verification)
- `verify-ca`: Always use SSL (verify server certificate)
- `verify-full`: Always use SSL (verify server certificate and hostname)

### Testing PostgreSQL Configuration

Use the provided script to test your PostgreSQL configuration:

```bash
./scripts/test_postgres_integration.sh
```

This script will:

1. Connect to your PostgreSQL database
2. Create test tables
3. Insert and retrieve test data
4. Clean up test tables

## DynamoDB Storage

DynamoDB storage provides persistent storage using AWS DynamoDB. This is suitable for AWS environments and supports high scalability.

### Prerequisites

- AWS account with DynamoDB access
- AWS credentials with appropriate permissions

### Configuration

```
# .env file
FLOWRUNNER_STORAGE_TYPE=dynamodb
FLOWRUNNER_DYNAMODB_REGION=us-west-2
FLOWRUNNER_DYNAMODB_ENDPOINT=http://localhost:8000
FLOWRUNNER_DYNAMODB_TABLE_PREFIX=flowrunner_
```

For local development, you can use DynamoDB Local:

```
FLOWRUNNER_DYNAMODB_ENDPOINT=http://localhost:8000
```

For production, use the AWS DynamoDB endpoint:

```
FLOWRUNNER_DYNAMODB_ENDPOINT=https://dynamodb.us-west-2.amazonaws.com
```

### AWS Credentials

FlowRunner uses the AWS SDK for Go to connect to DynamoDB. You can provide AWS credentials using:

1. Environment variables:

```
AWS_ACCESS_KEY_ID=your-access-key
AWS_SECRET_ACCESS_KEY=your-secret-key
AWS_SESSION_TOKEN=your-session-token
```

2. AWS credentials file (`~/.aws/credentials`):

```
[default]
aws_access_key_id = your-access-key
aws_secret_access_key = your-secret-key
```

3. IAM roles for EC2 instances or ECS tasks

### Table Structure

FlowRunner creates the following tables in DynamoDB:

- `{prefix}_accounts`: User accounts
- `{prefix}_flows`: Flow definitions
- `{prefix}_executions`: Flow executions
- `{prefix}_execution_logs`: Execution logs
- `{prefix}_secrets`: Encrypted secrets
- `{prefix}_structured_secrets`: Structured encrypted secrets

### Provisioned Throughput

By default, FlowRunner creates DynamoDB tables with on-demand capacity mode. You can configure provisioned throughput using the following environment variables:

```
FLOWRUNNER_DYNAMODB_READ_CAPACITY=5
FLOWRUNNER_DYNAMODB_WRITE_CAPACITY=5
```

### Local DynamoDB Setup

For local development, you can use DynamoDB Local:

1. Download DynamoDB Local:

```bash
wget https://s3.us-west-2.amazonaws.com/dynamodb-local/dynamodb_local_latest.tar.gz
tar -xzf dynamodb_local_latest.tar.gz
```

2. Start DynamoDB Local:

```bash
java -Djava.library.path=./DynamoDBLocal_lib -jar DynamoDBLocal.jar -sharedDb
```

3. Configure FlowRunner to use the local endpoint:

```
FLOWRUNNER_DYNAMODB_ENDPOINT=http://localhost:8000
```

### Testing DynamoDB Configuration

Use the provided script to test your DynamoDB configuration:

```bash
./scripts/test_dynamodb_integration.sh
```

This script will:

1. Connect to your DynamoDB instance
2. Create test tables
3. Insert and retrieve test data
4. Clean up test tables

## Storage Migration

FlowRunner does not currently provide built-in tools for migrating data between storage backends. However, you can use the following approach to migrate data:

1. Export data from the source storage:

```bash
flowrunner export --all --output data.json
```

2. Configure FlowRunner to use the target storage backend.

3. Import data into the target storage:

```bash
flowrunner import --input data.json
```

## Best Practices

### Production Environments

For production environments, we recommend:

1. **PostgreSQL Storage**:
   - Use a managed PostgreSQL service (AWS RDS, Google Cloud SQL, Azure Database for PostgreSQL)
   - Configure appropriate backup and replication
   - Use SSL for secure connections
   - Monitor database performance

2. **DynamoDB Storage**:
   - Use on-demand capacity mode for unpredictable workloads
   - Use provisioned capacity with auto-scaling for predictable workloads
   - Enable point-in-time recovery
   - Monitor throughput and adjust capacity as needed

### Development Environments

For development environments, we recommend:

1. **In-Memory Storage**:
   - Simplest option for local development
   - No external dependencies

2. **Local PostgreSQL**:
   - Use Docker for easy setup:
     ```bash
     docker run -d --name postgres -p 5432:5432 -e POSTGRES_PASSWORD=postgres postgres
     ```

3. **DynamoDB Local**:
   - Use for testing AWS-specific features
   - No AWS account required

### Security Considerations

1. **Database Credentials**:
   - Use environment variables or a secure configuration manager
   - Never hardcode credentials in source code
   - Use least-privilege database users

2. **Encryption**:
   - Enable encryption at rest for PostgreSQL and DynamoDB
   - Use SSL/TLS for PostgreSQL connections
   - Use HTTPS for DynamoDB connections

3. **Secrets**:
   - FlowRunner encrypts secrets before storing them
   - Use a strong encryption key (`FLOWRUNNER_ENCRYPTION_KEY`)
   - Rotate the encryption key periodically

### Performance Optimization

1. **PostgreSQL**:
   - Optimize connection pooling settings
   - Create appropriate indexes
   - Monitor query performance

2. **DynamoDB**:
   - Choose appropriate partition keys
   - Use sparse indexes for efficient queries
   - Monitor throughput and adjust capacity