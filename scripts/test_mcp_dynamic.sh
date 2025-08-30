#!/bin/bash

# Test script for MCP dynamic execution flow
# Demonstrates various execution strategies for MCP tools

set -e

# Configuration
FLOWRUNNER_URL="http://127.0.0.1:8080"
MCP_SERVER_URL="http://127.0.0.1:3005/mcp"

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

log() {
    echo -e "${BLUE}[$(date '+%H:%M:%S')]${NC} $1"
}

success() {
    echo -e "${GREEN}✓${NC} $1"
}

error() {
    echo -e "${RED}✗${NC} $1"
}

warn() {
    echo -e "${YELLOW}⚠${NC} $1"
}

# Get authentication token
get_token() {
    log "Getting authentication token..."
    
    TOKEN=$(curl -s -X POST "$FLOWRUNNER_URL/api/v1/login" \
        -H 'Content-Type: application/json' \
        -d '{"username":"demo","password":"demo"}' | jq -r .token)
    
    if [ "$TOKEN" = "null" ] || [ -z "$TOKEN" ]; then
        error "Failed to get authentication token"
        exit 1
    fi
    
    success "Got authentication token"
}

# Create flow from JSON file
create_flow() {
    local flow_file="$1"
    local flow_name="$2"
    
    log "Creating flow: $flow_name"
    
    if [ ! -f "$flow_file" ]; then
        error "Flow file not found: $flow_file"
        exit 1
    fi
    
    FLOW_CONTENT=$(cat "$flow_file")
    FLOW_ID=$(curl -s -X POST "$FLOWRUNNER_URL/api/v1/flows" \
        -H "Authorization: Bearer $TOKEN" \
        -H 'Content-Type: application/json' \
        -d "$FLOW_CONTENT" | jq -r .id)
    
    if [ "$FLOW_ID" = "null" ] || [ -z "$FLOW_ID" ]; then
        error "Failed to create flow: $flow_name"
        exit 1
    fi
    
    success "Created flow: $flow_name (ID: $FLOW_ID)"
}

# Execute flow and wait for completion
execute_and_wait() {
    local input_data="$1"
    local description="$2"
    local timeout="${3:-60}"
    
    log "Executing: $description"
    
    EXEC_ID=$(curl -s -X POST "$FLOWRUNNER_URL/api/v1/flows/$FLOW_ID/run" \
        -H "Authorization: Bearer $TOKEN" \
        -H 'Content-Type: application/json' \
        -d "$input_data" | jq -r .execution_id)
    
    if [ "$EXEC_ID" = "null" ] || [ -z "$EXEC_ID" ]; then
        error "Failed to start execution: $description"
        return 1
    fi
    
    success "Started execution: $description (ID: $EXEC_ID)"
    
    # Wait for completion
    local wait_count=0
    while [ $wait_count -lt $timeout ]; do
        STATUS=$(curl -s -H "Authorization: Bearer $TOKEN" \
            "$FLOWRUNNER_URL/api/v1/executions/$EXEC_ID" | jq -r .status)
        
        case "$STATUS" in
            "completed")
                success "Execution completed"
                return 0
                ;;
            "failed")
                error "Execution failed"
                return 1
                ;;
            "running")
                echo -n "."
                ;;
        esac
        
        sleep 1
        ((wait_count++))
    done
    
    error "Execution timed out after ${timeout}s"
    return 1
}

# Show execution results
show_results() {
    local description="$1"
    
    log "Results for: $description"
    echo "========================================"
    
    # Get execution output
    curl -s -H "Authorization: Bearer $TOKEN" \
        "$FLOWRUNNER_URL/api/v1/executions/$EXEC_ID" | jq '.output // .result // .'
    
    echo ""
}

# Test strategy: List tools only
test_list_only() {
    log "=== Test 1: List Available Tools ==="
    
    local input='{
        "input": {
            "strategy": "single"
        }
    }'
    
    if execute_and_wait "$input" "List tools only"; then
        show_results "List Tools"
    fi
}

# Test strategy: Execute single tool
test_single_tool() {
    local tool_name="$1"
    local tool_params="$2"
    
    log "=== Test 2: Execute Single Tool ($tool_name) ==="
    
    local input=$(cat <<EOF
{
    "input": {
        "strategy": "single",
        "targetTool": "$tool_name",
        "toolParameters": $tool_params
    }
}
EOF
)
    
    if execute_and_wait "$input" "Execute single tool: $tool_name"; then
        show_results "Single Tool Execution"
    fi
}

# Test strategy: Execute multiple specific tools
test_multiple_tools() {
    log "=== Test 3: Execute Multiple Specific Tools ==="
    
    local input='{
        "input": {
            "strategy": "multiple",
            "targetTools": ["get_weather", "search_web", "get_time"],
            "toolParameters": {"default": "test"},
            "perToolParams": {
                "get_weather": {"location": "San Francisco"},
                "search_web": {"query": "FlowRunner MCP"},
                "get_time": {"timezone": "UTC"}
            }
        }
    }'
    
    if execute_and_wait "$input" "Execute multiple tools" 90; then
        show_results "Multiple Tools Execution"
    fi
}

# Test strategy: Execute all tools (limited)
test_all_tools() {
    log "=== Test 4: Execute All Tools (Limited) ==="
    
    local input='{
        "input": {
            "strategy": "all",
            "maxExecutions": 3,
            "toolParameters": {"test": true}
        }
    }'
    
    if execute_and_wait "$input" "Execute all tools (max 3)" 120; then
        show_results "All Tools Execution"
    fi
}

# Test strategy: Filter and execute tools
test_filter_tools() {
    local filter="$1"
    
    log "=== Test 5: Filter and Execute Tools (filter: $filter) ==="
    
    local input=$(cat <<EOF
{
    "input": {
        "strategy": "filter",
        "toolFilter": "$filter",
        "maxExecutions": 2,
        "toolParameters": {"filtered": true}
    }
}
EOF
)
    
    if execute_and_wait "$input" "Filter and execute tools" 90; then
        show_results "Filtered Tools Execution"
    fi
}

# Test error handling
test_error_handling() {
    log "=== Test 6: Error Handling ==="
    
    local input='{
        "input": {
            "strategy": "single",
            "targetTool": "nonexistent_tool",
            "toolParameters": {}
        }
    }'
    
    if execute_and_wait "$input" "Test error handling"; then
        show_results "Error Handling Test"
    fi
}

# Main test suite
run_full_suite() {
    log "Starting MCP Dynamic Execution Test Suite"
    echo "=========================================="
    
    get_token
    create_flow "mcp_dynamic_execution.json" "MCP Dynamic Execution"
    
    # Run all tests
    test_list_only
    echo ""
    
    # Adjust these based on your MCP server's available tools
    test_single_tool "get_weather" '{"location": "New York"}'
    echo ""
    
    test_multiple_tools
    echo ""
    
    test_all_tools
    echo ""
    
    test_filter_tools "get.*"
    echo ""
    
    test_error_handling
    echo ""
    
    success "All tests completed!"
}

# Handle command line arguments
case "${1:-}" in
    "list")
        get_token
        create_flow "mcp_dynamic_execution.json" "MCP Dynamic Execution"
        test_list_only
        ;;
    "single")
        if [ -z "$2" ]; then
            error "Usage: $0 single <tool_name> [tool_params_json]"
            exit 1
        fi
        tool_name="$2"
        tool_params="${3:-{}}"
        get_token
        create_flow "mcp_dynamic_execution.json" "MCP Dynamic Execution"
        test_single_tool "$tool_name" "$tool_params"
        ;;
    "multiple")
        get_token
        create_flow "mcp_dynamic_execution.json" "MCP Dynamic Execution"
        test_multiple_tools
        ;;
    "all")
        get_token
        create_flow "mcp_dynamic_execution.json" "MCP Dynamic Execution"
        test_all_tools
        ;;
    "filter")
        if [ -z "$2" ]; then
            error "Usage: $0 filter <filter_pattern>"
            exit 1
        fi
        filter="$2"
        get_token
        create_flow "mcp_dynamic_execution.json" "MCP Dynamic Execution"
        test_filter_tools "$filter"
        ;;
    "error")
        get_token
        create_flow "mcp_dynamic_execution.json" "MCP Dynamic Execution"
        test_error_handling
        ;;
    "")
        run_full_suite
        ;;
    *)
        echo "Usage: $0 [command] [args...]"
        echo ""
        echo "Commands:"
        echo "  list                     - List available MCP tools"
        echo "  single <tool> [params]   - Execute single tool"
        echo "  multiple                 - Execute multiple predefined tools"
        echo "  all                      - Execute all tools (limited)"
        echo "  filter <pattern>         - Filter and execute matching tools"
        echo "  error                    - Test error handling"
        echo "  (no args)               - Run full test suite"
        echo ""
        echo "Examples:"
        echo "  $0 single get_weather '{\"location\":\"Paris\"}'"
        echo "  $0 filter 'weather|time'"
        exit 1
        ;;
esac