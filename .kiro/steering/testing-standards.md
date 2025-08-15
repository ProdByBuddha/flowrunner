# FlowRunner Testing Standards

## Testing Philosophy

### Test Pyramid
- **Unit Tests (70%)**: Fast, isolated tests for individual functions and methods
- **Integration Tests (20%)**: Tests for component interactions and external dependencies
- **End-to-End Tests (10%)**: Full system tests simulating real user scenarios

### Test-Driven Development
- Write tests before implementation when possible
- Use tests to drive API design and interface definitions
- Maintain high test coverage (>80% for core packages)
- Focus on testing behavior, not implementation details

## Unit Testing Standards

### Test Structure
```go
func TestFlowService_CreateFlow(t *testing.T) {
    // Arrange
    mockStorage := &MockStorage{}
    service := NewFlowService(mockStorage, logger)
    flow := &Flow{Name: "test-flow"}
    
    // Act
    result, err := service.CreateFlow(context.Background(), flow)
    
    // Assert
    assert.NoError(t, err)
    assert.NotNil(t, result)
    assert.Equal(t, "test-flow", result.Name)
}
```

### Table-Driven Tests
```go
func TestValidateFlowYAML(t *testing.T) {
    tests := []struct {
        name    string
        yaml    string
        wantErr bool
        errMsg  string
    }{
        {
            name: "valid flow",
            yaml: `
metadata:
  name: "test"
nodes:
  start:
    type: "http.request"`,
            wantErr: false,
        },
        {
            name:    "missing metadata",
            yaml:    `nodes: {}`,
            wantErr: true,
            errMsg:  "metadata is required",
        },
    }
    
    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            err := ValidateFlowYAML(tt.yaml)
            if tt.wantErr {
                assert.Error(t, err)
                assert.Contains(t, err.Error(), tt.errMsg)
            } else {
                assert.NoError(t, err)
            }
        })
    }
}
```

### Mock Objects
```go
type MockStorage struct {
    flows map[string]*Flow
    calls []string
}

func (m *MockStorage) CreateFlow(ctx context.Context, flow *Flow) error {
    m.calls = append(m.calls, "CreateFlow")
    if m.flows == nil {
        m.flows = make(map[string]*Flow)
    }
    m.flows[flow.ID] = flow
    return nil
}

func (m *MockStorage) AssertCalled(t *testing.T, method string) {
    assert.Contains(t, m.calls, method)
}
```

## Integration Testing

### Database Testing
```go
func TestPostgreSQLStorage_Integration(t *testing.T) {
    if testing.Short() {
        t.Skip("skipping integration test")
    }
    
    // Setup test database
    db := setupTestDB(t)
    defer cleanupTestDB(t, db)
    
    storage := NewPostgreSQLStorage(db)
    
    // Test flow CRUD operations
    flow := &Flow{
        ID:   "test-flow-1",
        Name: "Test Flow",
    }
    
    err := storage.CreateFlow(context.Background(), flow)
    assert.NoError(t, err)
    
    retrieved, err := storage.GetFlow(context.Background(), flow.ID)
    assert.NoError(t, err)
    assert.Equal(t, flow.Name, retrieved.Name)
}
```

### HTTP API Testing
```go
func TestFlowHandler_CreateFlow(t *testing.T) {
    // Setup test server
    server, cleanup := setupTestServer(t)
    defer cleanup()
    
    // Prepare request
    flow := map[string]interface{}{
        "name": "test-flow",
        "yaml": "metadata:\n  name: test",
    }
    body, _ := json.Marshal(flow)
    
    // Make request
    resp, err := http.Post(server.URL+"/api/v1/flows", "application/json", bytes.NewBuffer(body))
    assert.NoError(t, err)
    defer resp.Body.Close()
    
    // Assert response
    assert.Equal(t, http.StatusCreated, resp.StatusCode)
    
    var result map[string]interface{}
    err = json.NewDecoder(resp.Body).Decode(&result)
    assert.NoError(t, err)
    assert.Equal(t, "test-flow", result["name"])
}
```

## End-to-End Testing

### Flow Execution Testing
```go
func TestFlowExecution_EndToEnd(t *testing.T) {
    if testing.Short() {
        t.Skip("skipping e2e test")
    }
    
    // Setup complete system
    server, cleanup := setupE2EEnvironment(t)
    defer cleanup()
    
    // Create a test flow
    flowYAML := `
metadata:
  name: "e2e-test-flow"
nodes:
  start:
    type: "http.request"
    params:
      url: "https://httpbin.org/json"
      method: "GET"
    next:
      default: "end"
  end:
    type: "webhook"
    params:
      url: "` + server.URL + `/webhook"
      method: "POST"
`
    
    // Create flow via API
    flowID := createFlowViaAPI(t, server, flowYAML)
    
    // Execute flow
    executionID := executeFlowViaAPI(t, server, flowID, map[string]interface{}{})
    
    // Wait for completion and verify results
    execution := waitForExecution(t, server, executionID, 30*time.Second)
    assert.Equal(t, "completed", execution.Status)
    assert.NoError(t, execution.Error)
}
```

## Test Utilities and Helpers

### Test Database Setup
```go
func setupTestDB(t *testing.T) *sql.DB {
    dbURL := os.Getenv("TEST_DATABASE_URL")
    if dbURL == "" {
        t.Skip("TEST_DATABASE_URL not set")
    }
    
    db, err := sql.Open("postgres", dbURL)
    require.NoError(t, err)
    
    // Run migrations
    err = runMigrations(db)
    require.NoError(t, err)
    
    return db
}

func cleanupTestDB(t *testing.T, db *sql.DB) {
    _, err := db.Exec("TRUNCATE TABLE flows, executions, secrets")
    require.NoError(t, err)
    db.Close()
}
```

### Test Server Setup
```go
func setupTestServer(t *testing.T) (*httptest.Server, func()) {
    storage := NewMemoryStorage()
    executor := NewFlowExecutor(storage)
    service := NewFlowService(storage, executor, testLogger)
    handler := NewFlowHandler(service, testLogger)
    
    server := httptest.NewServer(handler)
    
    cleanup := func() {
        server.Close()
    }
    
    return server, cleanup
}
```

## Performance Testing

### Benchmark Tests
```go
func BenchmarkFlowExecution(b *testing.B) {
    storage := NewMemoryStorage()
    executor := NewFlowExecutor(storage)
    
    flow := createTestFlow()
    input := map[string]interface{}{"test": "data"}
    
    b.ResetTimer()
    for i := 0; i < b.N; i++ {
        _, err := executor.Execute(context.Background(), flow, input)
        if err != nil {
            b.Fatal(err)
        }
    }
}
```

### Load Testing
```go
func TestConcurrentFlowExecution(t *testing.T) {
    const numGoroutines = 100
    const numExecutions = 10
    
    storage := NewMemoryStorage()
    executor := NewFlowExecutor(storage)
    flow := createTestFlow()
    
    var wg sync.WaitGroup
    errors := make(chan error, numGoroutines*numExecutions)
    
    for i := 0; i < numGoroutines; i++ {
        wg.Add(1)
        go func() {
            defer wg.Done()
            for j := 0; j < numExecutions; j++ {
                _, err := executor.Execute(context.Background(), flow, map[string]interface{}{})
                if err != nil {
                    errors <- err
                }
            }
        }()
    }
    
    wg.Wait()
    close(errors)
    
    for err := range errors {
        t.Errorf("execution failed: %v", err)
    }
}
```

## Test Configuration

### Test Environment Variables
```bash
# Test database configuration
TEST_DATABASE_URL=postgres://user:pass@localhost/flowrunner_test

# Test service endpoints
TEST_OPENAI_API_KEY=test-key
TEST_ANTHROPIC_API_KEY=test-key

# Test flags
INTEGRATION_TESTS=true
E2E_TESTS=true
```

### CI/CD Integration
```yaml
# .github/workflows/test.yml
name: Tests
on: [push, pull_request]

jobs:
  unit-tests:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v2
      - uses: actions/setup-go@v2
        with:
          go-version: 1.24
      - run: go test -short ./...
  
  integration-tests:
    runs-on: ubuntu-latest
    services:
      postgres:
        image: postgres:13
        env:
          POSTGRES_PASSWORD: postgres
        options: >-
          --health-cmd pg_isready
          --health-interval 10s
          --health-timeout 5s
          --health-retries 5
    steps:
      - uses: actions/checkout@v2
      - uses: actions/setup-go@v2
        with:
          go-version: 1.24
      - run: go test -tags=integration ./...
```

## Test Data Management

### Test Fixtures
```go
func loadTestFlow(t *testing.T, filename string) *Flow {
    data, err := os.ReadFile(filepath.Join("testdata", filename))
    require.NoError(t, err)
    
    var flow Flow
    err = yaml.Unmarshal(data, &flow)
    require.NoError(t, err)
    
    return &flow
}
```

### Golden Files
```go
func TestFlowYAMLGeneration(t *testing.T) {
    flow := createTestFlow()
    generated, err := flow.ToYAML()
    require.NoError(t, err)
    
    goldenFile := "testdata/expected_flow.yaml"
    if *update {
        err = os.WriteFile(goldenFile, generated, 0644)
        require.NoError(t, err)
    }
    
    expected, err := os.ReadFile(goldenFile)
    require.NoError(t, err)
    
    assert.Equal(t, string(expected), string(generated))
}
```