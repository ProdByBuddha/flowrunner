# FlowRunner API Design Standards

## RESTful API Principles

### URL Structure
- Use consistent URL patterns: `/api/v{version}/{resource}/{id}`
- Use plural nouns for resources: `/flows`, `/executions`, `/accounts`
- Use kebab-case for multi-word resources: `/flow-templates`
- Avoid deep nesting beyond 2 levels: `/accounts/{id}/secrets/{key}`

### HTTP Methods
- **GET**: Retrieve resources (idempotent, safe)
- **POST**: Create new resources or trigger actions
- **PUT**: Update entire resources (idempotent)
- **PATCH**: Partial resource updates
- **DELETE**: Remove resources (idempotent)

### Status Codes
```
200 OK          - Successful GET, PUT, PATCH
201 Created     - Successful POST (resource created)
202 Accepted    - Async operation started
204 No Content  - Successful DELETE or PUT with no response body
400 Bad Request - Invalid request data
401 Unauthorized - Authentication required
403 Forbidden   - Insufficient permissions
404 Not Found   - Resource doesn't exist
409 Conflict    - Resource conflict (duplicate, version mismatch)
422 Unprocessable Entity - Validation errors
429 Too Many Requests - Rate limit exceeded
500 Internal Server Error - Server error
503 Service Unavailable - Temporary service issue
```

## API Endpoints Specification

### Flow Management
```
GET    /api/v1/flows
GET    /api/v1/flows/{id}
POST   /api/v1/flows
PUT    /api/v1/flows/{id}
PATCH  /api/v1/flows/{id}
DELETE /api/v1/flows/{id}
```

### Flow Execution
```
POST   /api/v1/flows/{id}/run
GET    /api/v1/executions
GET    /api/v1/executions/{id}
DELETE /api/v1/executions/{id}
GET    /api/v1/executions/{id}/logs
GET    /api/v1/executions/{id}/status
```

### Account Management
```
GET    /api/v1/accounts/{id}
POST   /api/v1/accounts
PUT    /api/v1/accounts/{id}
DELETE /api/v1/accounts/{id}
```

### Secrets Management
```
GET    /api/v1/accounts/{id}/secrets
POST   /api/v1/accounts/{id}/secrets
GET    /api/v1/accounts/{id}/secrets/{key}
PUT    /api/v1/accounts/{id}/secrets/{key}
DELETE /api/v1/accounts/{id}/secrets/{key}
```

## Request/Response Standards

### Request Headers
```
Content-Type: application/json
Authorization: Bearer {jwt_token}
X-Request-ID: {unique_request_id}
X-Account-ID: {account_id}
Accept: application/json
```

### Response Headers
```
Content-Type: application/json
X-Request-ID: {request_id}
X-Rate-Limit-Remaining: {count}
X-Rate-Limit-Reset: {timestamp}
Location: {resource_url}  # For 201 Created responses
```

### Standard Response Format
```json
{
  "data": {
    // Resource data or response payload
  },
  "meta": {
    "request_id": "req_123456",
    "timestamp": "2025-01-15T10:00:00Z",
    "version": "v1"
  }
}
```

### Error Response Format
```json
{
  "error": {
    "code": "VALIDATION_ERROR",
    "message": "Request validation failed",
    "details": "The 'name' field is required",
    "field": "name",
    "request_id": "req_123456"
  },
  "meta": {
    "timestamp": "2025-01-15T10:00:00Z",
    "version": "v1"
  }
}
```

## Flow API Examples

### Create Flow
```http
POST /api/v1/flows
Content-Type: application/json
Authorization: Bearer eyJ0eXAiOiJKV1QiLCJhbGciOiJIUzI1NiJ9...

{
  "name": "user-onboarding",
  "description": "Onboard new users",
  "yaml": "metadata:\n  name: user-onboarding\nnodes:\n  start:\n    type: http.request",
  "tags": ["onboarding", "users"],
  "enabled": true
}
```

Response:
```http
HTTP/1.1 201 Created
Content-Type: application/json
Location: /api/v1/flows/flow_123

{
  "data": {
    "id": "flow_123",
    "name": "user-onboarding",
    "description": "Onboard new users",
    "version": "1.0.0",
    "status": "active",
    "created_at": "2025-01-15T10:00:00Z",
    "updated_at": "2025-01-15T10:00:00Z",
    "created_by": "user_456"
  },
  "meta": {
    "request_id": "req_789",
    "timestamp": "2025-01-15T10:00:00Z",
    "version": "v1"
  }
}
```

### Execute Flow
```http
POST /api/v1/flows/flow_123/run
Content-Type: application/json
Authorization: Bearer eyJ0eXAiOiJKV1QiLCJhbGciOiJIUzI1NiJ9...

{
  "input": {
    "user_email": "user@example.com",
    "user_name": "John Doe"
  },
  "async": true,
  "webhook_url": "https://app.example.com/webhooks/flow-complete"
}
```

Response:
```http
HTTP/1.1 202 Accepted
Content-Type: application/json

{
  "data": {
    "execution_id": "exec_789",
    "status": "running",
    "started_at": "2025-01-15T10:00:00Z",
    "estimated_duration": "30s"
  },
  "meta": {
    "request_id": "req_101112",
    "timestamp": "2025-01-15T10:00:00Z",
    "version": "v1"
  }
}
```

### Get Execution Status
```http
GET /api/v1/executions/exec_789
Authorization: Bearer eyJ0eXAiOiJKV1QiLCJhbGciOiJIUzI1NiJ9...
```

Response:
```http
HTTP/1.1 200 OK
Content-Type: application/json

{
  "data": {
    "id": "exec_789",
    "flow_id": "flow_123",
    "status": "completed",
    "started_at": "2025-01-15T10:00:00Z",
    "completed_at": "2025-01-15T10:00:25Z",
    "duration": "25s",
    "input": {
      "user_email": "user@example.com",
      "user_name": "John Doe"
    },
    "output": {
      "user_id": "user_456",
      "onboarding_complete": true
    },
    "nodes_executed": 5,
    "nodes_failed": 0
  },
  "meta": {
    "request_id": "req_131415",
    "timestamp": "2025-01-15T10:00:30Z",
    "version": "v1"
  }
}
```

## Pagination Standards

### Query Parameters
```
?page=1&limit=20&sort=created_at&order=desc
```

### Response Format
```json
{
  "data": [
    // Array of resources
  ],
  "pagination": {
    "page": 1,
    "limit": 20,
    "total": 150,
    "pages": 8,
    "has_next": true,
    "has_prev": false
  },
  "meta": {
    "request_id": "req_161718",
    "timestamp": "2025-01-15T10:00:00Z",
    "version": "v1"
  }
}
```

### Link Headers
```
Link: <https://api.example.com/v1/flows?page=2&limit=20>; rel="next",
      <https://api.example.com/v1/flows?page=8&limit=20>; rel="last"
```

## Filtering and Search

### Query Parameters
```
GET /api/v1/flows?status=active&tags=onboarding&search=user&created_after=2025-01-01
```

### Supported Operators
- `eq` (equals): `status=active`
- `ne` (not equals): `status[ne]=inactive`
- `gt` (greater than): `created_at[gt]=2025-01-01`
- `gte` (greater than or equal): `created_at[gte]=2025-01-01`
- `lt` (less than): `created_at[lt]=2025-12-31`
- `lte` (less than or equal): `created_at[lte]=2025-12-31`
- `in` (in array): `status[in]=active,pending`
- `contains`: `name[contains]=user`

## Authentication and Authorization

### JWT Token Structure
```json
{
  "sub": "user_123",
  "account_id": "account_456",
  "roles": ["admin", "flow_manager"],
  "permissions": ["flows:read", "flows:write", "executions:read"],
  "exp": 1642694400,
  "iat": 1642608000
}
```

### Permission-Based Access Control
```
flows:read          - Read flow definitions
flows:write         - Create/update flows
flows:delete        - Delete flows
flows:execute       - Execute flows
executions:read     - View execution status and logs
executions:cancel   - Cancel running executions
secrets:read        - List secrets (not values)
secrets:write       - Create/update secrets
secrets:delete      - Delete secrets
accounts:admin      - Full account management
```

## Rate Limiting

### Rate Limit Headers
```
X-Rate-Limit-Limit: 1000
X-Rate-Limit-Remaining: 999
X-Rate-Limit-Reset: 1642694400
X-Rate-Limit-Window: 3600
```

### Rate Limit Response
```http
HTTP/1.1 429 Too Many Requests
Content-Type: application/json
Retry-After: 3600

{
  "error": {
    "code": "RATE_LIMIT_EXCEEDED",
    "message": "Rate limit exceeded",
    "details": "You have exceeded the rate limit of 1000 requests per hour",
    "retry_after": 3600
  }
}
```

## WebSocket API

### Connection
```
ws://localhost:8080/ws/executions/{execution_id}
Authorization: Bearer {jwt_token}
```

### Message Format
```json
{
  "type": "execution_update",
  "data": {
    "execution_id": "exec_789",
    "status": "running",
    "current_node": "process_data",
    "progress": 0.6,
    "timestamp": "2025-01-15T10:00:15Z"
  }
}
```

### Message Types
- `execution_started`: Execution began
- `execution_update`: Status or progress update
- `node_started`: Node execution started
- `node_completed`: Node execution completed
- `node_failed`: Node execution failed
- `execution_completed`: Execution finished successfully
- `execution_failed`: Execution failed
- `execution_cancelled`: Execution was cancelled

## API Versioning

### URL Versioning
- Current: `/api/v1/`
- Future: `/api/v2/`

### Version Support Policy
- Support current and previous major version
- Deprecation notices 6 months before removal
- Clear migration documentation

### Deprecation Headers
```
Deprecation: true
Sunset: Wed, 15 Jan 2026 10:00:00 GMT
Link: <https://docs.example.com/migration-v2>; rel="successor-version"
```

## Error Handling Standards

### Error Codes
```
VALIDATION_ERROR        - Request validation failed
AUTHENTICATION_ERROR    - Invalid or missing authentication
AUTHORIZATION_ERROR     - Insufficient permissions
RESOURCE_NOT_FOUND     - Requested resource doesn't exist
RESOURCE_CONFLICT      - Resource conflict (duplicate, etc.)
EXECUTION_ERROR        - Flow execution failed
STORAGE_ERROR          - Database or storage error
EXTERNAL_SERVICE_ERROR - External API error
RATE_LIMIT_EXCEEDED    - Too many requests
INTERNAL_ERROR         - Unexpected server error
```

### Validation Error Details
```json
{
  "error": {
    "code": "VALIDATION_ERROR",
    "message": "Request validation failed",
    "details": [
      {
        "field": "name",
        "code": "REQUIRED",
        "message": "Name is required"
      },
      {
        "field": "yaml",
        "code": "INVALID_YAML",
        "message": "Invalid YAML syntax at line 5"
      }
    ]
  }
}
```