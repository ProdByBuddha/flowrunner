#!/bin/bash

# Test script for MCP intelligent agent flow

set -e

FLOWRUNNER_URL="http://127.0.0.1:8080"

# Colors
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m'

log() { echo -e "${BLUE}[$(date '+%H:%M:%S')]${NC} $1"; }
success() { echo -e "${GREEN}✓${NC} $1"; }
error() { echo -e "${RED}✗${NC} $1"; }

get_token() {
    TOKEN=$(curl -s -X POST "$FLOWRUNNER_URL/api/v1/login" \
        -H 'Content-Type: application/json' \
        -d '{"username":"demo","password":"demo"}' | jq -r .token)
    
    if [ "$TOKEN" = "null" ] || [ -z "$TOKEN" ]; then
        error "Failed to get token"
        exit 1
    fi
    success "Got token"
}

create_flow() {
    FLOW_CONTENT=$(cat "mcp_agent_flow.json")
    FLOW_ID=$(curl -s -X POST "$FLOWRUNNER_URL/api/v1/flows" \
        -H "Authorization: Bearer $TOKEN" \
        -H 'Content-Type: application/json' \
        -d "$FLOW_CONTENT" | jq -r .id)
    
    if [ "$FLOW_ID" = "null" ] || [ -z "$FLOW_ID" ]; then
        error "Failed to create flow"
        exit 1
    fi
    success "Created flow (ID: $FLOW_ID)"
}

execute_request() {
    local request="$1"
    local params="$2"
    
    log "Processing: \"$request\""
    
    local input_data="{\"input\":{\"request\":\"$request\""
    if [ -n "$params" ]; then
        input_data="$input_data,\"parameters\":$params"
    fi
    input_data="$input_data}}"
    
    EXEC_ID=$(curl -s -X POST "$FLOWRUNNER_URL/api/v1/flows/$FLOW_ID/run" \
        -H "Authorization: Bearer $TOKEN" \
        -H 'Content-Type: application/json' \
        -d "$input_data" | jq -r .execution_id)
    
    if [ "$EXEC_ID" = "null" ] || [ -z "$EXEC_ID" ]; then
        error "Failed to start execution"
        return 1
    fi
    
    # Wait for completion
    local wait_count=0
    while [ $wait_count -lt 60 ]; do
        STATUS=$(curl -s -H "Authorization: Bearer $TOKEN" \
            "$FLOWRUNNER_URL/api/v1/executions/$EXEC_ID" | jq -r .status)
        
        case "$STATUS" in
            "completed") success "Completed"; return 0 ;;
            "failed") error "Failed"; return 1 ;;
            "running") echo -n "." ;;
        esac
        
        sleep 1
        ((wait_count++))
    done
    
    error "Timeout"
    return 1
}

show_response() {
    echo ""
    log "Agent Response:"
    echo "========================"
    
    local summary=$(curl -s -H "Authorization: Bearer $TOKEN" \
        "$FLOWRUNNER_URL/api/v1/executions/$EXEC_ID" | jq -r '.output.summary // .output.error // "No response"')
    
    echo "$summary"
    echo ""
}

# Test cases
test_weather() {
    log "=== Weather Test ==="
    if execute_request "Get weather for New York"; then
        show_response
    fi
}

test_search() {
    log "=== Search Test ==="
    if execute_request "Search for FlowRunner information"; then
        show_response
    fi
}

# Main
case "${1:-}" in
    "weather") get_token; create_flow; test_weather ;;
    "search") get_token; create_flow; test_search ;;
    "custom")
        if [ -z "$2" ]; then
            error "Usage: $0 custom \"<request>\""
            exit 1
        fi
        get_token; create_flow
        if execute_request "$2"; then show_response; fi
        ;;
    *)
        echo "Usage: $0 [weather|search|custom \"<request>\"]"
        exit 1
        ;;
esac