# FlowRunner Architecture Patterns

## Core Architectural Principles

### Separation of Concerns
- **Presentation Layer**: HTTP handlers and CLI commands
- **Business Logic**: Flow execution and management
- **Data Layer**: Storage abstractions and implementations
- **Infrastructure**: Configuration, logging, and external integrations

### Dependency Injection
```go
type FlowService struct {
    storage   Storage
    executor  Executor
    logger    Logger
    validator Validator
}

func NewFlowService(storage Storage, executor Executor, logger Logger, validator Validator) *FlowService {
    return &FlowService{
        storage:   storage,
        executor:  executor,
        logger:    logger,
        validator: validator,
    }
}
```

## Storage Abstraction Pattern

### Interface Definition
```go
type Storage interface {
    CreateFlow(ctx context.Context, flow *Flow) error
    GetFlow(ctx context.Context, id string) (*Flow, error)
    UpdateFlow(ctx context.Context, flow *Flow) error
    DeleteFlow(ctx context.Context, id string) error
    ListFlows(ctx context.Context, accountID string) ([]*Flow, error)
}
```

### Implementation Strategy
- Implement separate packages for each storage backend
- Use factory pattern for storage initialization
- Implement proper connection management and pooling
- Handle storage-specific errors consistently

## Node Registry Pattern

### Plugin Architecture
```go
type Node interface {
    Execute(ctx context.Context, input map[string]interface{}) (map[string]interface{}, error)
    Validate(params map[string]interface{}) error
    GetType() string
}

type NodeRegistry struct {
    nodes map[string]func() Node
}

func (r *NodeRegistry) Register(nodeType string, factory func() Node) {
    r.nodes[nodeType] = factory
}
```

### Node Implementation
- Each node type implements the Node interface
- Use factory functions for node creation
- Implement proper parameter validation
- Handle node-specific errors gracefully##
 Flow Execution Pattern

### Execution Context
```go
type ExecutionContext struct {
    FlowID      string
    ExecutionID string
    AccountID   string
    Input       map[string]interface{}
    Secrets     map[string]string
    Logger      Logger
    Cancel      context.CancelFunc
}
```

### State Management
- Track execution state throughout flow lifecycle
- Implement proper cleanup on cancellation
- Store intermediate results for debugging
- Handle concurrent node execution safely

## Configuration Pattern

### Hierarchical Configuration
```go
type Config struct {
    Server   ServerConfig   `yaml:"server"`
    Storage  StorageConfig  `yaml:"storage"`
    Auth     AuthConfig     `yaml:"auth"`
    Logging  LoggingConfig  `yaml:"logging"`
}
```

### Configuration Sources
1. Default values in code
2. Configuration file (YAML)
3. Environment variables
4. Command-line flags

Priority: CLI flags > Environment > Config file > Defaults

## Error Handling Patterns

### Structured Errors
```go
type FlowError struct {
    Code      string `json:"code"`
    Message   string `json:"message"`
    Details   string `json:"details,omitempty"`
    Cause     error  `json:"-"`
}

func (e *FlowError) Error() string {
    return fmt.Sprintf("%s: %s", e.Code, e.Message)
}
```

### Error Categories
- **ValidationError**: Input validation failures
- **ExecutionError**: Runtime execution failures
- **StorageError**: Data persistence failures
- **AuthError**: Authentication/authorization failures

## HTTP API Patterns

### Handler Structure
```go
type FlowHandler struct {
    service *FlowService
    logger  Logger
}

func (h *FlowHandler) CreateFlow(w http.ResponseWriter, r *http.Request) {
    var req CreateFlowRequest
    if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
        h.writeError(w, http.StatusBadRequest, "invalid request body")
        return
    }
    
    flow, err := h.service.CreateFlow(r.Context(), &req)
    if err != nil {
        h.handleServiceError(w, err)
        return
    }
    
    h.writeJSON(w, http.StatusCreated, flow)
}
```

### Middleware Chain
- Authentication middleware
- Logging middleware
- CORS middleware
- Rate limiting middleware
- Request validation middleware