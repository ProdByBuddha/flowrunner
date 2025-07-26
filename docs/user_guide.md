# FlowRunner User Guide

This guide provides comprehensive documentation for users of the FlowRunner platform, focusing on creating and running workflows.

## Table of Contents

1. [Introduction](#introduction)
2. [Getting Started](#getting-started)
3. [Flow Definition](#flow-definition)
4. [Core Node Types](#core-node-types)
5. [Flow Execution](#flow-execution)
6. [Advanced Features](#advanced-features)
7. [Troubleshooting](#troubleshooting)

## Introduction

FlowRunner is a lightweight, YAML-driven orchestration service that enables users to define, manage, and trigger workflows without writing Go code. The system provides a RESTful HTTP API for flow management, execution capabilities, multi-tenant account support with secrets management, and extensibility through plugins and inline scripting.

### Key Features

- **YAML-based workflow definitions** with expression support
- **Multiple persistence options** (in-memory, DynamoDB, PostgreSQL)
- **CLI and HTTP API interfaces** for flow management and execution
- **Core node implementations** for common tasks
- **Real-time monitoring** via WebSockets
- **Secure credential storage** with encryption
- **Expression evaluation** in YAML definitions
- **JavaScript scripting** for data transformations and logic

## Getting Started

### Prerequisites

- Go 1.18 or higher
- Access to desired storage backend (in-memory, DynamoDB, PostgreSQL)
- API keys for external services (OpenAI, Anthropic, etc.)

### Installation

1. Clone the repository:
   ```bash
   git clone https://github.com/tcmartin/flowrunner.git
   cd flowrunner
   ```

2. Install dependencies:
   ```bash
   go mod download
   ```

3. Build the project:
   ```bash
   go build -o flowrunner cmd/flowrunner/main.go
   ```

### Configuration

Create a `.env` file in the project root with the following variables:

```
# Server configuration
FLOWRUNNER_SERVER_HOST=localhost
FLOWRUNNER_SERVER_PORT=8080

# Storage configuration
# Options: memory, dynamodb, postgres
FLOWRUNNER_STORAGE_TYPE=memory

# DynamoDB configuration (used when FLOWRUNNER_STORAGE_TYPE=dynamodb)
FLOWRUNNER_DYNAMODB_REGION=us-west-2
FLOWRUNNER_DYNAMODB_ENDPOINT=http://localhost:8000
FLOWRUNNER_DYNAMODB_TABLE_PREFIX=flowrunner_

# PostgreSQL configuration (used when FLOWRUNNER_STORAGE_TYPE=postgres)
FLOWRUNNER_POSTGRES_HOST=localhost
FLOWRUNNER_POSTGRES_PORT=5432
FLOWRUNNER_POSTGRES_DATABASE=flowrunner
FLOWRUNNER_POSTGRES_USER=postgres
FLOWRUNNER_POSTGRES_PASSWORD=postgres
FLOWRUNNER_POSTGRES_SSL_MODE=disable

# Auth configuration
FLOWRUNNER_JWT_SECRET=your-jwt-secret-key
FLOWRUNNER_TOKEN_EXPIRATION=24
FLOWRUNNER_ENCRYPTION_KEY=your-encryption-key

# LLM API Keys
OPENAI_API_KEY=your_openai_api_key
ANTHROPIC_API_KEY=your_anthropic_api_key

# Email Configuration (optional)
GMAIL_USERNAME=your_gmail_username
GMAIL_PASSWORD=your_app_password
EMAIL_RECIPIENT=recipient_email
```

## Flow Definition

Flows in FlowRunner are defined using YAML. A flow consists of a metadata section and a nodes section.

### Basic Structure

```yaml
metadata:
  name: "Example Flow"
  description: "A simple example flow"
  version: "1.0.0"

nodes:
  start:
    type: "http.request"
    params:
      url: "https://api.example.com/data"
      method: "GET"
    next:
      default: "process"
      error: "handleError"
  
  process:
    type: "transform"
    params:
      script: |
        // Transform the data
        return input.data.map(item => ({
          id: item.id,
          name: item.name
        }));
    next:
      default: "end"
  
  handleError:
    type: "notification"
    params:
      channel: "slack"
      message: "Flow failed: ${error.message}"
    next:
      default: "end"
  
  end:
    type: "webhook"
    params:
      url: "https://webhook.example.com/callback"
      method: "POST"
```

### Flow Components

#### Metadata

The metadata section contains information about the flow:

- **name**: The name of the flow (required)
- **description**: A description of the flow (optional)
- **version**: The version of the flow (optional)

#### Nodes

The nodes section defines the nodes in the flow. Each node has:

- **type**: The type of the node (required)
- **params**: Parameters specific to the node type (optional)
- **next**: Defines the next nodes to execute based on the action (optional)
- **batch**: Configuration for batch processing (optional)
- **retry**: Configuration for retrying failed nodes (optional)
- **hooks**: JavaScript hooks for the node (optional)

#### Node Connections

Nodes are connected using the `next` property, which maps actions to node names:

```yaml
next:
  default: "nextNode"    # Execute nextNode on successful execution
  error: "errorNode"     # Execute errorNode on error
  condition1: "node1"    # Custom action names can be defined
  condition2: "node2"    # And mapped to specific nodes
```

The `default` action is used when no specific action is triggered.

#### JavaScript Hooks

Nodes can have JavaScript hooks that execute at different stages:

```yaml
hooks:
  prep: |
    // Runs before node execution
    // Can modify the input
    context.headers = { "Authorization": "Bearer " + secrets.API_KEY };
    return context;
  
  exec: |
    // Runs during node execution
    // Can replace the default node execution
    return input.map(item => ({ id: item.id, name: item.name }));
  
  post: |
    // Runs after node execution
    // Can modify the output
    result.timestamp = new Date().toISOString();
    return result;
```

## Core Node Types

FlowRunner provides several built-in node types for common tasks.

### HTTP Request Node

The HTTP request node makes HTTP requests to external APIs.

```yaml
http_node:
  type: "http.request"
  params:
    url: "https://api.example.com/data"
    method: "POST"
    headers:
      Content-Type: "application/json"
      Authorization: "Bearer ${secrets.API_KEY}"
    body:
      key: "value"
    timeout: "30s"
    follow_redirect: true
    auth:
      username: "user"
      password: "pass"
```

#### Parameters

| Parameter | Type | Required | Description |
|-----------|------|----------|-------------|
| `url` | string | Yes | The URL to request |
| `method` | string | No | HTTP method (default: GET) |
| `headers` | object | No | HTTP headers |
| `body` | any | No | Request body (string, object, or array) |
| `timeout` | string | No | Request timeout (e.g., "30s") |
| `follow_redirect` | boolean | No | Whether to follow redirects |
| `auth` | object | No | Authentication details |

#### Authentication Options

- **Basic Auth**: `auth: { username: "user", password: "pass" }`
- **Bearer Token**: `auth: { token: "your-token" }`
- **API Key**: `auth: { api_key: "your-key", key_name: "X-API-Key" }`

#### Output

```json
{
  "status_code": 200,
  "headers": {
    "Content-Type": ["application/json"]
  },
  "body": {
    "key": "value"
  },
  "metadata": {
    "content_type": "application/json",
    "content_length": 123,
    "request_url": "https://api.example.com/data",
    "request_method": "POST",
    "timing": "0.123s",
    "timing_ms": 123
  }
}
```

### Transform Node

The transform node applies JavaScript transformations to data.

```yaml
transform_node:
  type: "transform"
  params:
    script: |
      // Transform the input data
      return input.items.map(item => ({
        id: item.id,
        name: item.name.toUpperCase(),
        created: new Date(item.created_at).toISOString()
      }));
```

#### Parameters

| Parameter | Type | Required | Description |
|-----------|------|----------|-------------|
| `script` | string | Yes | JavaScript code to execute |

#### JavaScript Environment

The transform node provides:

- `input`: The input data to transform
- `console.log()`: For debugging

#### Output

The return value of the JavaScript code.

### LLM Node

The LLM node interacts with large language models like OpenAI's GPT and Anthropic's Claude.

```yaml
llm_node:
  type: "llm"
  params:
    provider: "openai"
    api_key: "${secrets.OPENAI_API_KEY}"
    model: "gpt-3.5-turbo"
    messages:
      - role: "system"
        content: "You are a helpful assistant."
      - role: "user"
        content: "What is the capital of France?"
    temperature: 0.7
    max_tokens: 100
```

For more details, see the [LLM Node Documentation](llm_node.md).

### Email Nodes

FlowRunner provides nodes for sending and receiving emails.

#### SMTP (Send)

```yaml
email_send_node:
  type: "email.send"
  params:
    smtp_host: "smtp.gmail.com"
    smtp_port: 587
    username: "${secrets.EMAIL_USERNAME}"
    password: "${secrets.EMAIL_PASSWORD}"
    from: "sender@example.com"
    to: "recipient@example.com"
    subject: "Hello from FlowRunner"
    body: "This is a test email."
    html: "<h1>Hello</h1><p>This is a test email.</p>"
```

#### IMAP (Receive)

```yaml
email_receive_node:
  type: "email.receive"
  params:
    imap_host: "imap.gmail.com"
    imap_port: 993
    username: "${secrets.EMAIL_USERNAME}"
    password: "${secrets.EMAIL_PASSWORD}"
    folder: "INBOX"
    limit: 10
    unseen: true
    with_body: true
```

For more details, see the [Email Nodes Documentation](email_nodes.md).

### Store Node

The store node provides in-memory storage for data.

```yaml
store_set_node:
  type: "store"
  params:
    operation: "set"
    key: "user_id"
    value: "12345"

store_get_node:
  type: "store"
  params:
    operation: "get"
    key: "user_id"

store_delete_node:
  type: "store"
  params:
    operation: "delete"
    key: "user_id"
```

#### Operations

- **set**: Store a value
- **get**: Retrieve a value
- **delete**: Delete a value
- **list**: List all keys
- **clear**: Clear all values

### Condition Node

The condition node evaluates conditions and routes flow execution.

```yaml
condition_node:
  type: "condition"
  params:
    conditions:
      - condition: "input.status == 'success'"
        action: "success"
      - condition: "input.status == 'error'"
        action: "error"
    default_action: "unknown"
  next:
    success: "successNode"
    error: "errorNode"
    unknown: "unknownNode"
```

### Delay Node

The delay node pauses flow execution for a specified duration.

```yaml
delay_node:
  type: "delay"
  params:
    duration: "5s"
```

### Wait Node

The wait node waits for an external event before continuing.

```yaml
wait_node:
  type: "wait"
  params:
    timeout: "1h"
    key: "order-123"
```

### Cron Node

The cron node schedules recurring tasks.

```yaml
cron_node:
  type: "cron"
  params:
    schedule: "0 */1 * * *"  # Every hour
    timezone: "UTC"
```

### Webhook Node

The webhook node sends data to a webhook endpoint.

```yaml
webhook_node:
  type: "webhook"
  params:
    url: "https://webhook.example.com/callback"
    method: "POST"
    headers:
      Content-Type: "application/json"
    body:
      result: "${result}"
```

### Database Nodes

#### DynamoDB Node

```yaml
dynamodb_node:
  type: "dynamodb"
  params:
    operation: "put"
    table: "users"
    item:
      id: "user-123"
      name: "John Doe"
      email: "john@example.com"
```

#### PostgreSQL Node

```yaml
postgres_node:
  type: "postgres"
  params:
    operation: "query"
    query: "SELECT * FROM users WHERE id = $1"
    args: ["user-123"]
```

### Agent Node

The agent node executes AI agent workflows.

```yaml
agent_node:
  type: "agent"
  params:
    provider: "openai"
    api_key: "${secrets.OPENAI_API_KEY}"
    model: "gpt-4"
    system_prompt: "You are a helpful assistant."
    user_prompt: "Please analyze this data: ${input.data}"
    tools:
      - type: "function"
        function:
          name: "get_weather"
          description: "Get the current weather"
          parameters:
            type: "object"
            properties:
              location:
                type: "string"
            required: ["location"]
```

## Flow Execution

### Using the CLI

```bash
# Create a new flow
./flowrunner create myflow --file flow.yaml

# List all flows
./flowrunner list

# Run a flow
./flowrunner run flow-id --input input.json

# View execution logs
./flowrunner logs execution-id
```

### Using the API

#### Create a Flow

```bash
curl -X POST http://localhost:8080/api/v1/flows \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer YOUR_TOKEN" \
  -d '{
    "name": "My Flow",
    "definition": "metadata:\n  name: \"My Flow\"\n  description: \"A simple flow\"\n  version: \"1.0.0\"\n\nnodes:\n  start:\n    type: \"http.request\"\n    params:\n      url: \"https://api.example.com/data\"\n      method: \"GET\"\n    next:\n      default: \"end\"\n  \n  end:\n    type: \"webhook\"\n    params:\n      url: \"https://webhook.example.com/callback\"\n      method: \"POST\"\n"
  }'
```

#### Run a Flow

```bash
curl -X POST http://localhost:8080/api/v1/flows/flow-id/run \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer YOUR_TOKEN" \
  -d '{
    "input": {
      "key": "value"
    }
  }'
```

#### Get Execution Status

```bash
curl -X GET http://localhost:8080/api/v1/executions/execution-id \
  -H "Authorization: Bearer YOUR_TOKEN"
```

### WebSocket Monitoring

Connect to the WebSocket endpoint to receive real-time updates:

```javascript
const ws = new WebSocket('ws://localhost:8080/ws/executions/execution-id');

ws.onmessage = (event) => {
  const data = JSON.parse(event.data);
  console.log('Execution update:', data);
};
```

## Advanced Features

### Batch Processing

Batch processing allows nodes to process multiple items in parallel:

```yaml
batch_node:
  type: "http.request"
  params:
    url: "https://api.example.com/data/${item.id}"
    method: "GET"
  batch:
    strategy: "parallel"
    max_parallel: 5
```

#### Batch Strategies

- **serial**: Process items one at a time
- **parallel**: Process all items in parallel
- **worker_pool**: Process items with a limited number of workers

### Retry Configuration

Retry configuration allows nodes to retry on failure:

```yaml
retry_node:
  type: "http.request"
  params:
    url: "https://api.example.com/data"
    method: "GET"
  retry:
    max_retries: 3
    wait: "5s"
```

### Secrets Management

Secrets can be accessed in flow definitions using the `${secrets.KEY}` syntax:

```yaml
http_node:
  type: "http.request"
  params:
    url: "https://api.example.com/data"
    headers:
      Authorization: "Bearer ${secrets.API_KEY}"
```

To manage secrets:

```bash
# Create a secret
curl -X POST http://localhost:8080/api/v1/accounts/account-id/secrets \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer YOUR_TOKEN" \
  -d '{
    "key": "API_KEY",
    "value": "your-api-key"
  }'

# List secrets
curl -X GET http://localhost:8080/api/v1/accounts/account-id/secrets \
  -H "Authorization: Bearer YOUR_TOKEN"

# Delete a secret
curl -X DELETE http://localhost:8080/api/v1/accounts/account-id/secrets/API_KEY \
  -H "Authorization: Bearer YOUR_TOKEN"
```

### Expression Evaluation

FlowRunner supports expression evaluation in YAML definitions:

```yaml
http_node:
  type: "http.request"
  params:
    url: "https://api.example.com/users/${input.user_id}"
    method: "GET"
```

## Troubleshooting

### Common Issues

#### Flow Creation Fails

- Ensure the YAML is valid
- Check that all referenced nodes exist
- Verify that all required parameters are provided

#### Flow Execution Fails

- Check the execution logs for errors
- Verify that all external services are accessible
- Ensure that all required secrets are available

#### Node Execution Fails

- Check the node parameters
- Verify that the node type is supported
- Check for syntax errors in JavaScript hooks

### Debugging

#### Enable Debug Logging

Set the `FLOWRUNNER_LOG_LEVEL` environment variable to `debug`:

```
FLOWRUNNER_LOG_LEVEL=debug
```

#### Use WebSocket Monitoring

Connect to the WebSocket endpoint to receive real-time updates during execution.

#### Check Node Outputs

Use the transform node to log intermediate results:

```yaml
debug_node:
  type: "transform"
  params:
    script: |
      console.log("Debug:", input);
      return input;
```

### Getting Help

If you encounter issues:

1. Check the [documentation](https://github.com/tcmartin/flowrunner)
2. Search for similar issues in the [issue tracker](https://github.com/tcmartin/flowrunner/issues)
3. Create a new issue with detailed information about the problem