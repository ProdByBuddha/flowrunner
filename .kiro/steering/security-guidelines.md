# FlowRunner Security Guidelines

## Authentication and Authorization

### JWT Token Security
- Use strong, randomly generated secrets for JWT signing
- Implement token expiration (recommended: 24 hours)
- Support token refresh mechanisms
- Store tokens securely on client side (httpOnly cookies preferred)
- Validate token signature and expiration on every request

### Role-Based Access Control (RBAC)
```go
type Permission string

const (
    FlowRead      Permission = "flows:read"
    FlowWrite     Permission = "flows:write"
    FlowExecute   Permission = "flows:execute"
    FlowDelete    Permission = "flows:delete"
    SecretsRead   Permission = "secrets:read"
    SecretsWrite  Permission = "secrets:write"
    AccountAdmin  Permission = "accounts:admin"
)
```

### Account Isolation
- Enforce strict account-based data isolation
- Validate account ownership on all resource access
- Use account-scoped database queries
- Implement proper tenant separation in multi-tenant deployments

## Input Validation and Sanitization

### YAML Validation
- Parse YAML structure safely
- Validate required fields and node references
- Implement schema validation
- Prevent YAML bombs and excessive nesting

### Parameter Sanitization
- Validate URL formats and prevent SSRF attacks
- Sanitize HTTP headers and request bodies
- Validate email addresses and SMTP settings
- Check LLM parameters for safety

## Secrets Management

### Encryption at Rest
- Use AES-256-GCM for secret encryption
- Generate unique nonces for each encryption
- Store encryption keys securely (environment variables, key management service)
- Implement proper key rotation mechanisms

### Secret Access Control
- Secrets are only accessible within the owning account
- Implement audit logging for secret access
- Support secret rotation and versioning
- Never log secret values in plain text## Netw
ork Security

### TLS Configuration
- Use TLS 1.2 or higher for all communications
- Configure strong cipher suites
- Implement proper certificate validation
- Use HSTS headers for web interfaces

### CORS Configuration
- Restrict allowed origins to known domains
- Limit allowed methods and headers
- Configure appropriate preflight handling
- Set reasonable max age for preflight responses

### Rate Limiting
- Implement per-user and per-IP rate limiting
- Use sliding window or token bucket algorithms
- Return appropriate HTTP 429 responses
- Log rate limit violations for monitoring

## JavaScript Execution Security

### Sandboxing
- Disable dangerous JavaScript functions (require, eval, etc.)
- Set execution timeouts to prevent infinite loops
- Limit memory usage and CPU time
- Validate scripts before execution

### Script Validation
- Check for dangerous patterns and functions
- Validate JavaScript syntax
- Implement content security policies
- Audit script execution and results

## Database Security

### SQL Injection Prevention
- Use parameterized queries exclusively
- Validate input parameters
- Implement proper error handling
- Use least privilege database accounts

### Connection Security
- Use SSL/TLS for database connections
- Configure connection pooling securely
- Implement connection timeouts
- Monitor database access patterns

## Audit Logging

### Security Event Logging
- Log authentication attempts and failures
- Track secret access and modifications
- Monitor flow execution and API calls
- Include relevant context (IP, user, timestamp)

### Log Security
- Store logs securely with appropriate access controls
- Implement log rotation and retention policies
- Protect against log injection attacks
- Consider centralized logging solutions

## Security Headers

### HTTP Security Headers
- X-Content-Type-Options: nosniff
- X-Frame-Options: DENY
- X-XSS-Protection: 1; mode=block
- Strict-Transport-Security with includeSubDomains
- Content-Security-Policy with restrictive policies
- Referrer-Policy: strict-origin-when-cross-origin

## Vulnerability Prevention

### SSRF Prevention
- Validate and sanitize URLs
- Block access to internal IP ranges
- Implement allow-lists for external services
- Monitor outbound network requests

### Path Traversal Prevention
- Sanitize file paths and remove directory traversal sequences
- Use absolute paths and validate against allowed directories
- Implement proper access controls for file operations
- Audit file system access

## Security Configuration

### Environment Variables
```bash
# Encryption and signing keys
FLOWRUNNER_ENCRYPTION_KEY=your-32-byte-encryption-key
FLOWRUNNER_JWT_SECRET=your-jwt-signing-secret

# Database security
FLOWRUNNER_POSTGRES_SSL_MODE=require
FLOWRUNNER_POSTGRES_SSL_CERT=/path/to/client-cert.pem

# Rate limiting
FLOWRUNNER_RATE_LIMIT_REQUESTS=1000
FLOWRUNNER_RATE_LIMIT_WINDOW=3600

# Security features
FLOWRUNNER_ENABLE_AUDIT_LOGGING=true
FLOWRUNNER_JS_EXECUTION_TIMEOUT=5s
FLOWRUNNER_MAX_YAML_SIZE=1048576
```

### Security Checklist
- [ ] Use HTTPS/TLS for all communications
- [ ] Implement proper authentication and authorization
- [ ] Validate and sanitize all inputs
- [ ] Encrypt secrets at rest
- [ ] Use parameterized queries for database access
- [ ] Implement rate limiting
- [ ] Add security headers to HTTP responses
- [ ] Sandbox JavaScript execution
- [ ] Prevent SSRF and path traversal attacks
- [ ] Enable comprehensive audit logging
- [ ] Regular security updates and dependency scanning
- [ ] Implement proper error handling
- [ ] Use secure random number generation
- [ ] Implement session management best practices