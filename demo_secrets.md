# FlowRunner Secret Management Demo

This document demonstrates the comprehensive secret management capabilities implemented for FlowRunner, including encrypted storage, structured secrets, and HTTP API access.

## Features Implemented

✅ **Core Secret Vault with Encryption**
- AES-GCM 256-bit encryption for all secret values
- Per-account isolation with validation
- Key rotation capabilities
- Basic CRUD operations via HTTP API

✅ **Structured Secrets Support**
- OAuth credentials (client ID, secret, tokens, etc.)
- API keys with headers and configuration
- Database connections with SSL options
- JWT tokens with claims and metadata
- Custom structured data with arbitrary fields

✅ **HTTP API Endpoints**
- RESTful endpoints for all secret operations
- Type-specific endpoints for structured secrets
- Field-level access with dot notation
- Search and filtering capabilities
- Metadata management (tags, descriptions, expiration)

✅ **Security Features**
- Account-based access control
- Bearer token authentication
- Encrypted storage of all secret values
- Sensitive field masking in logs and responses
- Key rotation support for encryption keys

## API Examples

### Basic Secret Operations

```bash
# Create a simple secret
curl -X POST "http://localhost:8080/api/v1/accounts/123/secrets/api_key" \
  -H "Authorization: Bearer your-token" \
  -H "Content-Type: application/json" \
  -d '{"value": "sk-1234567890abcdef"}'

# Get a secret
curl -X GET "http://localhost:8080/api/v1/accounts/123/secrets/api_key" \
  -H "Authorization: Bearer your-token"

# List all secrets (metadata only)
curl -X GET "http://localhost:8080/api/v1/accounts/123/secrets" \
  -H "Authorization: Bearer your-token"

# Delete a secret
curl -X DELETE "http://localhost:8080/api/v1/accounts/123/secrets/api_key" \
  -H "Authorization: Bearer your-token"
```

### Structured Secret Operations

#### OAuth Credentials
```bash
# Create OAuth secret
curl -X POST "http://localhost:8080/api/v1/accounts/123/oauth-secrets/google_oauth" \
  -H "Authorization: Bearer your-token" \
  -H "Content-Type: application/json" \
  -d '{
    "description": "Google OAuth for email integration",
    "tags": ["oauth", "google", "email"],
    "client_id": "123456789.apps.googleusercontent.com",
    "client_secret": "GOCSPX-abcdef123456",
    "token_url": "https://oauth2.googleapis.com/token",
    "auth_url": "https://accounts.google.com/o/oauth2/auth",
    "redirect_url": "https://myapp.com/oauth/callback",
    "scopes": ["email", "profile"],
    "access_token": "ya29.abcdef123456",
    "refresh_token": "1//0abcdef123456",
    "token_expires_at": "2025-07-19T12:00:00Z"
  }'

# Get OAuth secret
curl -X GET "http://localhost:8080/api/v1/accounts/123/oauth-secrets/google_oauth" \
  -H "Authorization: Bearer your-token"
```

#### API Key with Headers
```bash
# Create API key secret
curl -X POST "http://localhost:8080/api/v1/accounts/123/api-key-secrets/stripe_api" \
  -H "Authorization: Bearer your-token" \
  -H "Content-Type: application/json" \
  -d '{
    "description": "Stripe payment API key",
    "tags": ["api-key", "stripe", "payments"],
    "key": "sk_live_abcdef123456",
    "header_name": "Authorization",
    "prefix": "Bearer ",
    "base_url": "https://api.stripe.com",
    "rate_limit": {
      "requests_per_second": 100,
      "requests_per_hour": 1000
    }
  }'
```

#### Database Connection
```bash
# Create database secret
curl -X POST "http://localhost:8080/api/v1/accounts/123/database-secrets/prod_db" \
  -H "Authorization: Bearer your-token" \
  -H "Content-Type: application/json" \
  -d '{
    "description": "Production PostgreSQL database",
    "tags": ["database", "postgresql", "production"],
    "type": "postgresql",
    "host": "prod-db.example.com",
    "port": 5432,
    "database": "myapp_prod",
    "username": "app_user",
    "password": "secure_password_123",
    "ssl_mode": "require",
    "max_conns": 20,
    "conn_timeout": "30s"
  }'
```

### Advanced Operations

#### Search and Filter
```bash
# Search secrets by type
curl -X GET "http://localhost:8080/api/v1/accounts/123/structured-secrets?type=oauth" \
  -H "Authorization: Bearer your-token"

# Search secrets with tags
curl -X GET "http://localhost:8080/api/v1/accounts/123/structured-secrets?tag=production&tag=database" \
  -H "Authorization: Bearer your-token"

# Advanced search with POST
curl -X POST "http://localhost:8080/api/v1/accounts/123/structured-secrets/search" \
  -H "Authorization: Bearer your-token" \
  -H "Content-Type: application/json" \
  -d '{
    "type": "api_key",
    "tags": ["production"],
    "description_contains": "stripe"
  }'
```

#### Field-level Access
```bash
# Get specific field from structured secret
curl -X GET "http://localhost:8080/api/v1/accounts/123/structured-secrets/google_oauth/field/access_token" \
  -H "Authorization: Bearer your-token"

# Returns: {"field": "access_token", "value": "ya29.abcdef123456"}
```

#### Metadata Management
```bash
# Update secret metadata only (without changing the secret data)
curl -X PATCH "http://localhost:8080/api/v1/accounts/123/structured-secrets/google_oauth/metadata" \
  -H "Authorization: Bearer your-token" \
  -H "Content-Type: application/json" \
  -d '{
    "description": "Updated Google OAuth for email and calendar",
    "tags": ["oauth", "google", "email", "calendar"],
    "expires_at": "2025-12-31T23:59:59Z"
  }'
```

## YAML Flow Integration

Secrets can be referenced in YAML flows using the secret reference syntax:

```yaml
# Basic secret reference
api_key: 
  secret_key: "stripe_api"

# Field-specific reference for structured secrets
oauth_client_id:
  secret_key: "google_oauth"
  field: "client_id"

# With default value
database_host:
  secret_key: "prod_db"
  field: "host"
  default: "localhost"
```

## Security Model

- **Account Isolation**: All secrets belong to specific accounts and are only accessible by them
- **Encryption at Rest**: All secret values are encrypted using AES-GCM with 256-bit keys
- **Authentication Required**: All API endpoints require valid bearer token authentication
- **Future RBAC Ready**: Architecture supports role-based access control for team scenarios
- **Audit Trail**: Metadata tracks creation, modification, and last accessed times
- **Key Rotation**: Supports rotating encryption keys while preserving existing secrets

## Storage Backend Support

The secret vault works with all FlowRunner storage backends:
- **Memory**: For development and testing
- **PostgreSQL**: For production deployments
- **DynamoDB**: For AWS cloud deployments

## Configuration

Add to your FlowRunner configuration:

```yaml
auth:
  encryption_key: "0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef"  # 64-char hex string (256 bits)
```

Or set via environment variable:
```bash
export FLOWRUNNER_ENCRYPTION_KEY="0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef"
```

The secret management system is now fully integrated and ready for production use!
