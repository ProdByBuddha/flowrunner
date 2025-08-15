# FlowRunner Development Guidelines

## Code Style and Standards

### Go Conventions
- Follow standard Go formatting with `gofmt`
- Use `golint` and `go vet` for code quality
- Implement proper error handling with wrapped errors
- Use context.Context for cancellation and timeouts
- Follow Go naming conventions (exported vs unexported)

### Package Structure
- Keep packages focused and cohesive
- Avoid circular dependencies
- Use interfaces for abstraction and testability
- Implement dependency injection patterns
- Separate business logic from infrastructure concerns

### Error Handling
```go
// Preferred error handling pattern
if err != nil {
    return fmt.Errorf("failed to execute flow %s: %w", flowID, err)
}
```

## Testing Standards

### Unit Testing
- Test coverage should be >80% for core packages
- Use table-driven tests for multiple scenarios
- Mock external dependencies using interfaces
- Test both success and error paths
- Use testify/assert for cleaner assertions

### Integration Testing
- Test storage backend implementations
- Verify HTTP API endpoints with real requests
- Test flow execution end-to-end
- Use test containers for database testing

### Test Organization
```go
func TestFlowExecution(t *testing.T) {
    tests := []struct {
        name     string
        flow     *Flow
        input    map[string]interface{}
        expected ExecutionResult
        wantErr  bool
    }{
        // test cases
    }
    
    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            // test implementation
        })
    }
}
```

## API Design Principles

### RESTful Design
- Use proper HTTP methods (GET, POST, PUT, DELETE)
- Follow consistent URL patterns: `/api/v1/resource/{id}`
- Return appropriate HTTP status codes
- Use JSON for request/response bodies
- Implement proper error responses with details

### Authentication & Authorization
- Use JWT tokens for stateless authentication
- Implement account-based isolation
- Encrypt sensitive data in storage
- Validate all inputs and sanitize outputs
- Follow security best practices

### Versioning
- Use URL versioning: `/api/v1/`
- Maintain backward compatibility within major versions
- Document breaking changes clearly
- Provide migration guides for major version updates

## Configuration Management

### Environment Variables
- Use clear, prefixed environment variable names: `FLOWRUNNER_*`
- Provide sensible defaults where possible
- Document all configuration options
- Support multiple configuration sources (env, file, flags)

### Secrets Management
- Never commit secrets to version control
- Use encrypted storage for sensitive data
- Implement proper key rotation mechanisms
- Audit access to sensitive configuration

## Performance Guidelines

### Execution Efficiency
- Use goroutines for concurrent node execution
- Implement proper context cancellation
- Avoid blocking operations in hot paths
- Use connection pooling for database access
- Implement caching where appropriate

### Memory Management
- Avoid memory leaks in long-running processes
- Use streaming for large data processing
- Implement proper cleanup in defer statements
- Monitor memory usage in production

## Logging and Monitoring

### Structured Logging
```go
log.WithFields(log.Fields{
    "flow_id":      flowID,
    "execution_id": execID,
    "node_id":      nodeID,
}).Info("Node execution completed")
```

### Metrics and Observability
- Track execution times and success rates
- Monitor resource usage (CPU, memory, connections)
- Implement health check endpoints
- Use distributed tracing for complex flows

## Documentation Standards

### Code Documentation
- Document all exported functions and types
- Include usage examples in godoc comments
- Explain complex algorithms and business logic
- Document configuration options and defaults

### API Documentation
- Maintain OpenAPI/Swagger specifications
- Provide request/response examples
- Document error codes and messages
- Include authentication requirements

## Security Considerations

### Input Validation
- Validate all user inputs
- Sanitize data before storage
- Implement rate limiting on API endpoints
- Use parameterized queries for database access

### Data Protection
- Encrypt sensitive data at rest
- Use TLS for all network communication
- Implement proper session management
- Follow OWASP security guidelines

## Deployment and Operations

### Build Process
- Use multi-stage Docker builds
- Implement proper dependency management
- Create reproducible builds
- Use semantic versioning for releases

### Monitoring and Alerting
- Implement health checks and readiness probes
- Monitor key business metrics
- Set up alerting for critical failures
- Maintain operational runbooks