#!/bin/bash

# Test script for MCP list-then-execute flow
# This script demonstrates the agent using MCP client repeatedly:
# 1. List available tools
# 2. Execute a specific tool from the list

set -e

# Configuration
FLOWRUNNER_URL="http://127.0.0.1:8080"
MCP_SERVER_URL="http://127.0.0.1:3005/mcp"
MCP_SSE_URL="http://127.0.0.1:33106/mcp-sse"

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

# Check if MCP server is running
check_mcp_server() {
    log "Checking MCP server availability..."
    
    if curl -s -f "$MCP_SERVER_URL" > /dev/null 2>&1; then
        success "MCP server is running at $MCP_SERVER_URL"
    else
        error "MCP server not available at $MCP_SERVER_URL"
        warn "Please start the MCP server first:"
        echo "  npx -y @prodbybuddha/openapi-mcp-server@latest \\"
        echo "    --config /root/projects/flowrunner/services.dynamic.json \\"
        echo "    --sse-port 33106 \\"
        echo "    --post-port 3005"
        exit 1
    fi
}

# Check if FlowRunner is running
check_flowrunner() {
    log "Checking FlowRunner availability..."
    
    if curl -s -f "$FLOWRUNNER_URL/health" > /dev/null 2>&1; then
        success "FlowRunner is running at $FLOWRUNNER_URL"
    else
        error "FlowRunner not available at $FLOWRUNNER_URL"
        warn "Please start FlowRunner first: ./flowrunner"
        exit 1
    fi
}

# Login and get token
get_token() {
    log "Logging in to FlowRunner..."
    
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
        # Show the response for debugging
        curl -s -X POST "$FLOWRUNNER_URL/api/v1/flows" \
            -H "Authorization: Bearer $TOKEN" \
            -H 'Content-Type: application/json' \
            -d "$FLOW_CONTENT" | jq
        exit 1
    fi
    
    success "Created flow: $flow_name (ID: $FLOW_ID)"
}

# Execute flow with input
execute_flow() {
    local input_data="$1"
    local description="$2"
    
    log "Executing flow: $description"
    
    EXEC_ID=$(curl -s -X POST "$FLOWRUNNER_URL/api/v1/flows/$FLOW_ID/run" \
        -H "Authorization: Bearer $TOKEN" \
        -H 'Content-Type: application/json' \
        -d "$input_data" | jq -r .execution_id)
    
    if [ "$EXEC_ID" = "null" ] || [ -z "$EXEC_ID" ]; then
        error "Failed to start execution: $description"
        exit 1
    fi
    
    success "Started execution: $description (ID: $EXEC_ID)"
    
    # Wait for completion
    log "Waiting for execution to complete..."
    local max_wait=30
    local wait_count=0
    
    while [ $wait_count -lt $max_wait ]; do
        STATUS=$(curl -s -H "Authorization: Bearer $TOKEN" \
            "$FLOWRUNNER_URL/api/v1/executions/$EXEC_ID" | jq -r .status)
        
        case "$STATUS" in
            "completed")
                success "Execution completed successfully"
                break
                ;;
            "failed")
                error "Execution failed"
                curl -s -H "Authorization: Bearer $TOKEN" \
                    "$FLOWRUNNER_URL/api/v1/executions/$EXEC_ID/logs" | jq
                exit 1
                ;;
            "running")
                echo -n "."
                ;;
            *)
                warn "Unknown status: $STATUS"
                ;;
        esac
        
        sleep 1
        ((wait_count++))
    done
    
    if [ $wait_count -ge $max_wait ]; then
        error "Execution timed out after ${max_wait}s"
        exit 1
    fi
    
    echo ""
}

# Show execution results
show_results() {
    local description="$1"
    
    log "Results for: $description"
    echo "----------------------------------------"
    
    # Get execution details
    curl -s -H "Authorization: Bearer $TOKEN" \
        "$FLOWRUNNER_URL/api/v1/executions/$EXEC_ID" | jq '.output // .result // .'
    
    echo "----------------------------------------"
    
    # Get execution logs
    log "Execution logs:"
    curl -s -H "Authorization: Bearer $TOKEN" \
        "$FLOWRUNNER_URL/api/v1/executions/$EXEC_ID/logs" | jq -r '.[] | "\(.timestamp) [\(.level)] \(.message)"' 2>/dev/null || \
    curl -s -H "Authorization: Bearer $TOKEN" \
        "$FLOWRUNNER_URL/api/v1/executions/$EXEC_ID/logs" | jq
    
    echo ""
}

# Test different scenarios
test_list_only() {
    log "=== Test 1: List Tools Only ==="
    
    execute_flow '{"input":{}}' "List available tools"
    show_results "List Tools Only"
}

test_list_and_execute() {
    local tool_name="$1"
    local tool_params="$2"
    
    log "=== Test 2: List and Execute Tool ($tool_name) ==="
    
    local input_json="{\"input\":{\"targetTool\":\"$tool_name\",\"toolParameters\":$tool_params}}"
    
    execute_flow "$input_json" "List and execute $tool_name"
    show_results "List and Execute $tool_name"
}

test_invalid_tool() {
    log "=== Test 3: Invalid Tool Name ==="
    
    execute_flow '{"input":{"targetTool":"nonexistent_tool","toolParameters":{}}}' "Try invalid tool"
    show_results "Invalid Tool Test"
}

# Main execution
main() {
    log "Starting MCP List-Then-Execute Flow Test"
    echo "========================================"
    
    # Prerequisites
    check_mcp_server
    check_flowrunner
    get_token
    
    # Create the flow
    create_flow "mcp_list_then_execute.json" "MCP List-Then-Execute"
    
    # Test scenarios
    test_list_only
    
    # Test with a real tool (adjust based on your MCP server's available tools)
    # Common tools might include: get_weather, search_web, etc.
    # For now, let's try a generic tool - you can modify this based on your server
    test_list_and_execute "get_weather" '{"location":"San Francisco"}'
    
    # Test error handling
    test_invalid_tool
    
    success "All tests completed!"
}

# Handle script arguments
case "${1:-}" in
    "check")
        check_mcp_server
        check_flowrunner
        ;;
    "list")
        check_mcp_server
        check_flowrunner
        get_token
        create_flow "mcp_list_then_execute.json" "MCP List-Then-Execute"
        test_list_only
        ;;
    "execute")
        if [ -z "$2" ]; then
            error "Usage: $0 execute <tool_name> [tool_params_json]"
            exit 1
        fi
        tool_name="$2"
        tool_params="${3:-{}}"
        check_mcp_server
        check_flowrunner
        get_token
        create_flow "mcp_list_then_execute.json" "MCP List-Then-Execute"
        test_list_and_execute "$tool_name" "$tool_params"
        ;;
    "")
        main
        ;;
    *)
        echo "Usage: $0 [check|list|execute <tool_name> [params]]"
        echo ""
        echo "Commands:"
        echo "  check                    - Check if services are running"
        echo "  list                     - List available MCP tools"
        echo "  execute <tool> [params]  - List tools then execute specific tool"
        echo "  (no args)               - Run full test suite"
        exit 1
        ;;
esac