# FlowRunner API Reference

This document provides a comprehensive reference for the FlowRunner API.

## Table of Contents

1. [Authentication](#authentication)
2. [Flow Management](#flow-management)
3. [Flow Execution](#flow-execution)
4. [Account Management](#account-management)
5. [Secrets Management](#secrets-management)
6. [WebSocket API](#websocket-api)
7. [Error Handling](#error-handling)

## Authentication

### Login

Authenticate and get a JWT token.

**Endpoint:** `POST /api/v1/auth/login`

**Request:**

```json
{
  "username": "user@example.com",
  "password": "your-password"
}
```

**Response:**

```json
{
  "token": "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9...",
  "expires_at": "2023-01-01T12:00:00Z",
  "account_id": "account-123"
}
```

### Refresh Token

Refresh an existing JWT token.

**Endpoint:** `POST /api/v1/auth/refresh`

**Headers:**

```
Authorization: Bearer your-token
```

**Response:**

```json
{
  "token": "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9...",
  "expires_at": "2023-01-01T12:00:00Z",
  "account_id": "account-123"
}
```

### Logout

Invalidate the current token.

**Endpoint:** `POST /api/v1/auth/logout`

**Headers:**

```
Authorization: Bearer your-token
```

**Response:**

```json
{
  "message": "Logged out successfully"
}
```

## Flow Management

### List Flows

Get a list of flows.

**Endpoint:** `GET /api/v1/flows`

**Headers:**

```
Authorization: Bearer your-token
```

**Query Parameters:**

- `tag` - Filter by tag
- `limit` - Maximum number of flows to return
- `offset` - Offset for pagination

**Response:**

```json
{
  "flows": [
    {
      "id": "flow-123",
      "name": "My Flow",
      "description": "A simple flow",
      "version": "1.0.0",
      "created_at": "2023-01-01T12:00:00Z",
      "updated_at": "2023-01-01T12:00:00Z",
      "tags": ["production"]
    },
    {
      "id": "flow-456",
      "name": "Another Flow",
      "description": "Another flow",
      "version": "1.0.0",
      "created_at": "2023-01-01T12:00:00Z",
      "updated_at": "2023-01-01T12:00:00Z",
      "tags": ["development"]
    }
  ],
  "total": 2,
  "limit": 10,
  "offset": 0
}
```

### Get Flow

Get details of a specific flow.

**Endpoint:** `GET /api/v1/flows/{id}`

**Headers:**

```
Authorization: Bearer your-token
```

**Response:**

```json
{
  "id": "flow-123",
  "name": "My Flow",
  "description": "A simple flow",
  "version": "1.0.0",
  "definition": "metadata:\n  name: \"My Flow\"\n  description: \"A simple flow\"\n  version: \"1.0.0\"\n\nnodes:\n  start:\n    type: \"http.request\"\n    params:\n      url: \"https://api.example.com/data\"\n      method: \"GET\"\n    next:\n      default: \"end\"\n  \n  end:\n    type: \"webhook\"\n    params:\n      url: \"https://webhook.example.com/callback\"\n      method: \"POST\"\n",
  "created_at": "2023-01-01T12:00:00Z",
  "updated_at": "2023-01-01T12:00:00Z",
  "tags": ["production"]
}
```

### Create Flow

Create a new flow.

**Endpoint:** `POST /api/v1/flows`

**Headers:**

```
Authorization: Bearer your-token
Content-Type: application/json
```

**Request:**

```json
{
  "name": "My Flow",
  "description": "A simple flow",
  "definition": "metadata:\n  name: \"My Flow\"\n  description: \"A simple flow\"\n  version: \"1.0.0\"\n\nnodes:\n  start:\n    type: \"http.request\"\n    params:\n      url: \"https://api.example.com/data\"\n      method: \"GET\"\n    next:\n      default: \"end\"\n  \n  end:\n    type: \"webhook\"\n    params:\n      url: \"https://webhook.example.com/callback\"\n      method: \"POST\"\n",
  "tags": ["production"]
}
```

**Response:**

```json
{
  "id": "flow-123",
  "name": "My Flow",
  "description": "A simple flow",
  "version": "1.0.0",
  "created_at": "2023-01-01T12:00:00Z",
  "updated_at": "2023-01-01T12:00:00Z",
  "tags": ["production"]
}
```

### Update Flow

Update an existing flow.

**Endpoint:** `PUT /api/v1/flows/{id}`

**Headers:**

```
Authorization: Bearer your-token
Content-Type: application/json
```

**Request:**

```json
{
  "name": "Updated Flow",
  "description": "An updated flow",
  "definition": "metadata:\n  name: \"Updated Flow\"\n  description: \"An updated flow\"\n  version: \"1.0.1\"\n\nnodes:\n  start:\n    type: \"http.request\"\n    params:\n      url: \"https://api.example.com/data\"\n      method: \"GET\"\n    next:\n      default: \"end\"\n  \n  end:\n    type: \"webhook\"\n    params:\n      url: \"https://webhook.example.com/callback\"\n      method: \"POST\"\n",
  "tags": ["production", "updated"]
}
```

**Response:**

```json
{
  "id": "flow-123",
  "name": "Updated Flow",
  "description": "An updated flow",
  "version": "1.0.1",
  "created_at": "2023-01-01T12:00:00Z",
  "updated_at": "2023-01-01T13:00:00Z",
  "tags": ["production", "updated"]
}
```

### Delete Flow

Delete a flow.

**Endpoint:** `DELETE /api/v1/flows/{id}`

**Headers:**

```
Authorization: Bearer your-token
```

**Response:**

```json
{
  "message": "Flow deleted successfully"
}
```

## Flow Execution

### Run Flow

Execute a flow.

**Endpoint:** `POST /api/v1/flows/{id}/run`

**Headers:**

```
Authorization: Bearer your-token
Content-Type: application/json
```

**Request:**

```json
{
  "input": {
    "key": "value"
  }
}
```

**Response:**

```json
{
  "execution_id": "exec-123",
  "flow_id": "flow-123",
  "status": "running",
  "created_at": "2023-01-01T12:00:00Z"
}
```

### Get Execution

Get details of a specific execution.

**Endpoint:** `GET /api/v1/executions/{id}`

**Headers:**

```
Authorization: Bearer your-token
```

**Response:**

```json
{
  "id": "exec-123",
  "flow_id": "flow-123",
  "status": "completed",
  "input": {
    "key": "value"
  },
  "result": {
    "output": "value"
  },
  "started_at": "2023-01-01T12:00:00Z",
  "completed_at": "2023-01-01T12:00:05Z",
  "duration": 5000
}
```

### List Executions

Get a list of executions.

**Endpoint:** `GET /api/v1/executions`

**Headers:**

```
Authorization: Bearer your-token
```

**Query Parameters:**

- `flow_id` - Filter by flow ID
- `status` - Filter by status (running, completed, failed)
- `limit` - Maximum number of executions to return
- `offset` - Offset for pagination

**Response:**

```json
{
  "executions": [
    {
      "id": "exec-123",
      "flow_id": "flow-123",
      "status": "completed",
      "started_at": "2023-01-01T12:00:00Z",
      "completed_at": "2023-01-01T12:00:05Z",
      "duration": 5000
    },
    {
      "id": "exec-456",
      "flow_id": "flow-123",
      "status": "failed",
      "started_at": "2023-01-01T12:10:00Z",
      "completed_at": "2023-01-01T12:10:02Z",
      "duration": 2000,
      "error": "HTTP request failed with status 500"
    }
  ],
  "total": 2,
  "limit": 10,
  "offset": 0
}
```

### Cancel Execution

Cancel a running execution.

**Endpoint:** `DELETE /api/v1/executions/{id}`

**Headers:**

```
Authorization: Bearer your-token
```

**Response:**

```json
{
  "message": "Execution cancelled successfully"
}
```

### Get Execution Logs

Get logs for a specific execution.

**Endpoint:** `GET /api/v1/executions/{id}/logs`

**Headers:**

```
Authorization: Bearer your-token
```

**Query Parameters:**

- `level` - Filter by log level (debug, info, warn, error)
- `limit` - Maximum number of logs to return
- `offset` - Offset for pagination

**Response:**

```json
{
  "logs": [
    {
      "timestamp": "2023-01-01T12:00:00Z",
      "level": "info",
      "message": "Execution started",
      "node_id": "start",
      "node_type": "http.request"
    },
    {
      "timestamp": "2023-01-01T12:00:01Z",
      "level": "info",
      "message": "HTTP request successful",
      "node_id": "start",
      "node_type": "http.request",
      "data": {
        "status_code": 200,
        "url": "https://api.example.com/data"
      }
    },
    {
      "timestamp": "2023-01-01T12:00:05Z",
      "level": "info",
      "message": "Execution completed",
      "node_id": "end",
      "node_type": "webhook"
    }
  ],
  "total": 3,
  "limit": 10,
  "offset": 0
}
```

## Account Management

### List Accounts

Get a list of accounts.

**Endpoint:** `GET /api/v1/accounts`

**Headers:**

```
Authorization: Bearer your-token
```

**Query Parameters:**

- `role` - Filter by role
- `limit` - Maximum number of accounts to return
- `offset` - Offset for pagination

**Response:**

```json
{
  "accounts": [
    {
      "id": "account-123",
      "name": "My Account",
      "email": "user@example.com",
      "role": "admin",
      "created_at": "2023-01-01T12:00:00Z",
      "updated_at": "2023-01-01T12:00:00Z"
    },
    {
      "id": "account-456",
      "name": "Another Account",
      "email": "another@example.com",
      "role": "user",
      "created_at": "2023-01-01T12:00:00Z",
      "updated_at": "2023-01-01T12:00:00Z"
    }
  ],
  "total": 2,
  "limit": 10,
  "offset": 0
}
```

### Get Account

Get details of a specific account.

**Endpoint:** `GET /api/v1/accounts/{id}`

**Headers:**

```
Authorization: Bearer your-token
```

**Response:**

```json
{
  "id": "account-123",
  "name": "My Account",
  "email": "user@example.com",
  "role": "admin",
  "created_at": "2023-01-01T12:00:00Z",
  "updated_at": "2023-01-01T12:00:00Z"
}
```

### Create Account

Create a new account.

**Endpoint:** `POST /api/v1/accounts`

**Headers:**

```
Authorization: Bearer your-token
Content-Type: application/json
```

**Request:**

```json
{
  "name": "New Account",
  "email": "new@example.com",
  "password": "your-password",
  "role": "user"
}
```

**Response:**

```json
{
  "id": "account-789",
  "name": "New Account",
  "email": "new@example.com",
  "role": "user",
  "created_at": "2023-01-01T12:00:00Z",
  "updated_at": "2023-01-01T12:00:00Z"
}
```

### Update Account

Update an existing account.

**Endpoint:** `PUT /api/v1/accounts/{id}`

**Headers:**

```
Authorization: Bearer your-token
Content-Type: application/json
```

**Request:**

```json
{
  "name": "Updated Account",
  "email": "updated@example.com",
  "role": "admin"
}
```

**Response:**

```json
{
  "id": "account-123",
  "name": "Updated Account",
  "email": "updated@example.com",
  "role": "admin",
  "created_at": "2023-01-01T12:00:00Z",
  "updated_at": "2023-01-01T13:00:00Z"
}
```

### Delete Account

Delete an account.

**Endpoint:** `DELETE /api/v1/accounts/{id}`

**Headers:**

```
Authorization: Bearer your-token
```

**Response:**

```json
{
  "message": "Account deleted successfully"
}
```

## Secrets Management

### List Secrets

Get a list of secrets for an account.

**Endpoint:** `GET /api/v1/accounts/{account_id}/secrets`

**Headers:**

```
Authorization: Bearer your-token
```

**Response:**

```json
{
  "secrets": [
    {
      "key": "API_KEY",
      "created_at": "2023-01-01T12:00:00Z",
      "updated_at": "2023-01-01T12:00:00Z"
    },
    {
      "key": "DATABASE_URL",
      "created_at": "2023-01-01T12:00:00Z",
      "updated_at": "2023-01-01T12:00:00Z"
    }
  ]
}
```

### Create Secret

Create a new secret.

**Endpoint:** `POST /api/v1/accounts/{account_id}/secrets`

**Headers:**

```
Authorization: Bearer your-token
Content-Type: application/json
```

**Request:**

```json
{
  "key": "NEW_API_KEY",
  "value": "your-api-key"
}
```

**Response:**

```json
{
  "key": "NEW_API_KEY",
  "created_at": "2023-01-01T12:00:00Z",
  "updated_at": "2023-01-01T12:00:00Z"
}
```

### Delete Secret

Delete a secret.

**Endpoint:** `DELETE /api/v1/accounts/{account_id}/secrets/{key}`

**Headers:**

```
Authorization: Bearer your-token
```

**Response:**

```json
{
  "message": "Secret deleted successfully"
}
```

## WebSocket API

### Execution Updates

Connect to the WebSocket endpoint to receive real-time updates for a flow execution.

**Endpoint:** `ws://localhost:8080/ws/executions/{execution_id}`

**Headers:**

```
Authorization: Bearer your-token
```

**Messages:**

The server sends JSON messages with execution updates:

```json
{
  "type": "execution_update",
  "execution_id": "exec-123",
  "status": "running",
  "node_id": "start",
  "node_type": "http.request",
  "timestamp": "2023-01-01T12:00:01Z"
}
```

```json
{
  "type": "node_completed",
  "execution_id": "exec-123",
  "node_id": "start",
  "node_type": "http.request",
  "result": {
    "status_code": 200,
    "body": { "key": "value" }
  },
  "timestamp": "2023-01-01T12:00:02Z"
}
```

```json
{
  "type": "execution_completed",
  "execution_id": "exec-123",
  "status": "completed",
  "result": { "output": "value" },
  "timestamp": "2023-01-01T12:00:05Z"
}
```

```json
{
  "type": "execution_failed",
  "execution_id": "exec-123",
  "status": "failed",
  "error": "HTTP request failed with status 500",
  "node_id": "start",
  "node_type": "http.request",
  "timestamp": "2023-01-01T12:00:02Z"
}
```

```json
{
  "type": "log",
  "execution_id": "exec-123",
  "level": "info",
  "message": "HTTP request successful",
  "node_id": "start",
  "node_type": "http.request",
  "data": {
    "status_code": 200,
    "url": "https://api.example.com/data"
  },
  "timestamp": "2023-01-01T12:00:01Z"
}
```

## Error Handling

### Error Response Format

Error responses follow a consistent format:

```json
{
  "error": {
    "code": "invalid_request",
    "message": "The request was invalid",
    "details": {
      "field": "name",
      "reason": "Name is required"
    }
  }
}
```

### Common Error Codes

| Code | Description |
|------|-------------|
| `unauthorized` | Authentication is required or failed |
| `forbidden` | The authenticated user doesn't have permission |
| `not_found` | The requested resource was not found |
| `invalid_request` | The request was invalid |
| `validation_error` | The request failed validation |
| `internal_error` | An internal server error occurred |

### HTTP Status Codes

| Status Code | Description |
|-------------|-------------|
| 200 | OK - The request was successful |
| 201 | Created - A new resource was created |
| 400 | Bad Request - The request was invalid |
| 401 | Unauthorized - Authentication is required |
| 403 | Forbidden - The authenticated user doesn't have permission |
| 404 | Not Found - The requested resource was not found |
| 409 | Conflict - The request conflicts with the current state |
| 422 | Unprocessable Entity - The request failed validation |
| 500 | Internal Server Error - An internal server error occurred |