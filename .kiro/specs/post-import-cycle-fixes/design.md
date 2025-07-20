# Post Import Cycle Fixes - Design Document

## Overview

This design addresses the systematic cleanup needed after resolving the import cycle between pkg/runtime, pkg/registry, and pkg/storage. The solution involves updating type references, fixing function signatures, implementing missing methods, and resolving test conflicts.

## Architecture

The fixes are organized into several independent modules that can be addressed in parallel:

1. **Type Reference Updates** - Update all packages to use pkg/types
2. **Interface Completeness** - Ensure all interfaces are fully implemented
3. **Function Signature Fixes** - Correct parameter mismatches
4. **Test Organization** - Resolve conflicts in test files
5. **Error Handling Standardization** - Use consistent error handling patterns

## Components and Interfaces

### Type Reference Updates

**Component:** TypeReferenceUpdater
- **Purpose:** Systematically update all type references from runtime.* to types.*
- **Files to Update:**
  - pkg/api/websocket.go
  - pkg/api/server.go
  - pkg/cli/execution.go
  - Any other files with runtime.ExecutionLog or runtime.ExecutionStatus references

**Implementation Strategy:**
- Search and replace runtime.ExecutionLog with types.ExecutionLog
- Search and replace runtime.ExecutionStatus with types.ExecutionStatus
- Update import statements to include pkg/types
- Remove unused runtime imports

### Interface Completeness

**Component:** FlowRuntimeInterface
- **Current Issue:** Missing ListExecutions method in interface
- **Solution:** Add ListExecutions method to FlowRuntime interface
- **Files to Update:**
  - pkg/runtime/interfaces.go
  - Any mock implementations

**Component:** FlowRegistryInterface
- **Current Issue:** Mock implementations missing GetVersion method
- **Solution:** Add GetVersion method to mock implementations
- **Files to Update:**
  - Test files with MockFlowRegistry

### Function Signature Fixes

**Component:** ConstructorSignatures
- **registry.NewFlowRegistry:** Update to accept FlowRegistryOptions
- **storage.NewDynamoDBProvider:** Update to accept DynamoDBProviderConfig
- **progress.NewProgressTracker:** Update to accept required parameters
- **config.Load:** Use correct configuration loading approach

**Implementation Strategy:**
- Check current function signatures in their respective packages
- Update all call sites to match expected signatures
- Ensure all required parameters are provided

### Error Handling Standardization

**Component:** ErrorHandling
- **Current Issue:** References to undefined error functions
- **Solution:** Use standard Go error handling or implement missing functions
- **Files to Update:**
  - pkg/api/server.go
  - Any files using errors.NewFlowRunnerError or errors.WriteErrorResponse

**Implementation Options:**
1. Implement missing error functions in pkg/errors
2. Replace with standard Go error handling
3. Use existing error handling patterns from other parts of the codebase

### Test Organization

**Component:** TestFileOrganization
- **Current Issue:** Multiple main functions in root directory
- **Solution:** Move test files to appropriate packages or rename them
- **Files to Address:**
  - test_dynamodb_*.go files
  - test_flow_management*.go files

**Implementation Strategy:**
- Move files to cmd/ subdirectories
- Rename files to avoid conflicts
- Use build tags to separate test executables

## Data Models

### Updated Type References

```go
// Before (causing import cycle)
import "github.com/tcmartin/flowrunner/pkg/runtime"
func handleLog(log runtime.ExecutionLog) { ... }

// After (using shared types)
import "github.com/tcmartin/flowrunner/pkg/types"
func handleLog(log types.ExecutionLog) { ... }
```

### Complete Interface Definitions

```go
// FlowRuntime interface with all required methods
type FlowRuntime interface {
    Execute(accountID string, flowID string, input map[string]interface{}) (string, error)
    GetStatus(executionID string) (types.ExecutionStatus, error)
    GetLogs(executionID string) ([]types.ExecutionLog, error)
    SubscribeToLogs(executionID string) (<-chan types.ExecutionLog, error)
    Cancel(executionID string) error
    ListExecutions(accountID string) ([]types.ExecutionStatus, error) // Missing method
}
```

## Error Handling

### Standardized Error Responses

```go
// Option 1: Implement missing error functions
func NewFlowRunnerError(code string, message string) error {
    return fmt.Errorf("[%s] %s", code, message)
}

func WriteErrorResponse(w http.ResponseWriter, err error, statusCode int) {
    w.WriteHeader(statusCode)
    json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
}

// Option 2: Use standard error handling
func handleError(w http.ResponseWriter, err error, statusCode int) {
    http.Error(w, err.Error(), statusCode)
}
```

## Testing Strategy

### Test File Organization

1. **Move executable test files** to cmd/ subdirectories
2. **Use build tags** to separate test executables from package tests
3. **Rename conflicting files** to avoid global variable conflicts

### Mock Implementation Updates

1. **Add missing methods** to mock implementations
2. **Ensure interface compliance** for all mocks
3. **Update test expectations** to match new signatures

### Validation Testing

1. **YAML loader validation** - Implement proper error checking
2. **Type conversion testing** - Ensure all type updates work correctly
3. **Interface completeness testing** - Verify all methods are implemented

## Implementation Priority

1. **High Priority:** Type reference updates (blocks compilation)
2. **High Priority:** Function signature fixes (blocks compilation)
3. **Medium Priority:** Interface completeness (blocks some tests)
4. **Medium Priority:** Error handling standardization (improves reliability)
5. **Low Priority:** Test file organization (improves test execution)

## Dependencies

- All fixes depend on the successful resolution of the import cycle (already completed)
- Type reference updates must be completed before interface fixes
- Function signature fixes can be done in parallel with type updates
- Test organization can be done independently of other fixes