# FlowRunner MCP Integration Guide

This guide covers the Model Context Protocol (MCP) integration in FlowRunner, enabling agents to repeatedly use MCP clients for tool discovery and execution.

## Overview

FlowRunner now supports comprehensive MCP integration that allows:

1. **Tool Discovery**: List available MCP tools dynamically
2. **Tool Execution**: Execute specific MCP tools with parameters
3. **Repeated Operations**: Chain tool listing and execution in workflows
4. **Intelligent Agent Behavior**: Natural language requests converted to tool executions

## MCP Node Implementation

The MCP node (`pkg/runtime/mcp_node.go`) supports three transport types:

- **HTTP**: Direct HTTP POST requests to MCP servers
- **SSE**: Server-Sent Events with optional HTTP POST
- **CMD**: STDIO communication with MCP server processes

### Key Features

- **Raw Body Support**: Send exact JSON payloads to MCP servers
- **Template Support**: Dynamic parameter injection using Go templates
- **Multiple Transports**: Flexible connection options
- **Error Handling**: Comprehensive error reporting and recovery
- **Dotenv Integration**: Environment variable loading for CMD transport

## Flow Examples

### 1. Basic List-Then-Execute Flow

**File**: `mcp_list_then_execute.json`

This flow demonstrates the fundamental pattern:
1. List available MCP tools
2. Parse the tool list
3. Execute a specified tool from the list

```bash
# Test the basic flow
./scripts/test_mcp_list_execute.sh
```

**Usage Examples**:
```bash
# List tools only
./scripts/test_mcp_list_execute.sh list

# Execute specific tool
./scripts/test_mcp_list_execute.sh execute get_weather '{"location":"Paris"}'
```

### 2. Dynamic Multi-Tool Execution

**File**: `mcp_dynamic_execution.json`

Advanced flow supporting multiple execution strategies:

- **Single**: Execute one specific tool
- **Multiple**: Execute several named tools
- **All**: Execute all available tools (with limits)
- **Filter**: Execute tools matching a pattern

```bash
# Test dynamic execution
./scripts/test_mcp_dynamic.sh

# Execute all tools (limited to 3)
./scripts/test_mcp_dynamic.sh all

# Filter and execute tools
./scripts/test_mcp_dynamic.sh filter "weather.*"
```

### 3. Intelligent Agent Flow

**File**: `mcp_agent_flow.json`

Natural language processing flow that:
1. Analyzes user requests
2. Selects relevant tools based on keywords
3. Executes tools with inferred parameters
4. Provides natural language responses

```bash
# Test the agent
./scripts/test_mcp_agent.sh

# Custom requests
./scripts/test_mcp_agent.sh custom "Get weather for Tokyo"
./scripts/test_mcp_agent.sh custom "Search for restaurants"
```

## Configuration

### MCP Server Setup

Start your MCP server (example with openapi-mcp-server):

```bash
npx -y @prodbybuddha/openapi-mcp-server@latest \
  --config /root/projects/flowrunner/services.dynamic.json \
  --sse-port 33106 \
  --post-port 3005
```

### FlowRunner Configuration

Ensure FlowRunner is running:

```bash
./flowrunner
```

The server should be accessible at `http://127.0.0.1:8080`

## Flow Input Formats

### Basic List and Execute

```json
{
  "input": {
    "targetTool": "get_weather",
    "toolParameters": {
      "location": "San Francisco"
    }
  }
}
```

### Dynamic Execution Strategies

```json
{
  "input": {
    "strategy": "multiple",
    "targetTools": ["get_weather", "search_web"],
    "toolParameters": {"default": "value"},
    "perToolParams": {
      "get_weather": {"location": "NYC"},
      "search_web": {"query": "restaurants"}
    }
  }
}
```

### Agent Requests

```json
{
  "input": {
    "request": "Get the weather for Paris and search for tourist attractions",
    "parameters": {
      "location": "Paris",
      "query": "tourist attractions"
    }
  }
}
```

## MCP Node Parameters

### Connection Configuration

```yaml
connectionType: "http"  # http, sse, cmd
url: "http://127.0.0.1:3005/mcp"
operation: "listTools"  # listTools, executeTool, or custom
```

### HTTP Transport

```yaml
connectionType: "http"
url: "http://127.0.0.1:3005/mcp"
headers:
  Authorization: "Bearer token"
  Content-Type: "application/json"
timeout: "30s"
```

### Raw Body Override

```yaml
rawBody: '{"method":"tools/list","params":{}}'
# OR with templates
rawBodyTemplate: |
  {
    "method": "tools/call",
    "params": {
      "name": "{{.input.toolName}}",
      "arguments": {{.input.toolParameters}}
    }
  }
```

### CMD Transport

```yaml
connectionType: "cmd"
command: "npx"
args: ["@prodbybuddha/openapi-mcp-server@latest"]
env: "PORT=3005\nMCP_PORT=3005"
dotenvx: true
```

## Error Handling

All flows include comprehensive error handling:

- **Connection Errors**: MCP server unavailable
- **Tool Not Found**: Requested tool doesn't exist
- **Parameter Errors**: Invalid tool parameters
- **Execution Failures**: Tool execution errors

Example error response:
```json
{
  "success": false,
  "error": "Tool 'nonexistent_tool' not found",
  "availableTools": ["get_weather", "search_web", "get_time"],
  "suggestions": ["get_weather", "search_web"]
}
```

## Testing

### Prerequisites

1. **MCP Server Running**: Start your MCP server first
2. **FlowRunner Running**: Ensure FlowRunner is accessible
3. **Authentication**: Demo credentials (demo/demo) work by default

### Test Scripts

```bash
# Basic list-then-execute
./scripts/test_mcp_list_execute.sh

# Dynamic execution patterns
./scripts/test_mcp_dynamic.sh

# Intelligent agent behavior
./scripts/test_mcp_agent.sh
```

### Manual Testing

```bash
# Check services
curl -s http://127.0.0.1:3005/mcp -X POST \
  -H 'Content-Type: application/json' \
  -d '{"method":"tools/list","params":{}}'

# Login to FlowRunner
TOKEN=$(curl -s -X POST http://127.0.0.1:8080/api/v1/login \
  -H 'Content-Type: application/json' \
  -d '{"username":"demo","password":"demo"}' | jq -r .token)

# Create and run flow
FLOW_ID=$(curl -s -X POST http://127.0.0.1:8080/api/v1/flows \
  -H "Authorization: Bearer $TOKEN" \
  -H 'Content-Type: application/json' \
  -d @mcp_list_then_execute.json | jq -r .id)

EXEC_ID=$(curl -s -X POST http://127.0.0.1:8080/api/v1/flows/$FLOW_ID/run \
  -H "Authorization: Bearer $TOKEN" \
  -H 'Content-Type: application/json' \
  -d '{"input":{"targetTool":"get_weather","toolParameters":{"location":"NYC"}}}' | jq -r .execution_id)

# Check results
curl -s -H "Authorization: Bearer $TOKEN" \
  http://127.0.0.1:8080/api/v1/executions/$EXEC_ID
```

## Advanced Usage

### Custom Tool Selection Logic

Modify the `analyzeRequest` node in `mcp_agent_flow.json` to implement:

- **LLM-based tool selection**: Use an LLM to analyze requests
- **Semantic matching**: Vector similarity for tool selection
- **Context awareness**: Consider previous tool executions
- **User preferences**: Learn from user behavior

### Parallel Tool Execution

Create flows that execute multiple MCP tools in parallel:

```yaml
parallelExecution:
  type: "parallel"
  params:
    branches:
      - nodes: ["executeTool1"]
      - nodes: ["executeTool2"]
    wait_for: "all"
```

### Tool Chaining

Chain tool outputs as inputs to subsequent tools:

```yaml
chainedExecution:
  type: "transform"
  params:
    script: |
      // Use output from previous tool as input to next
      return {
        toolName: "process_data",
        toolParameters: input.previousResult
      };
```

## Troubleshooting

### Common Issues

1. **MCP Server Not Responding**
   - Check if server is running on correct port
   - Verify endpoint URLs
   - Test with curl directly

2. **Authentication Failures**
   - Ensure FlowRunner login works
   - Check token validity
   - Verify bearer token format

3. **Tool Not Found Errors**
   - List available tools first
   - Check tool name spelling
   - Verify MCP server configuration

4. **Parameter Errors**
   - Validate JSON parameter format
   - Check required vs optional parameters
   - Review tool documentation

### Debug Mode

Enable detailed logging by adding debug parameters:

```yaml
params:
  debug: true
  verbose: true
```

### Log Analysis

Check FlowRunner logs for detailed execution traces:

```bash
# View execution logs
curl -s -H "Authorization: Bearer $TOKEN" \
  http://127.0.0.1:8080/api/v1/executions/$EXEC_ID/logs | jq
```

## Next Steps

1. **Enhanced Agent Logic**: Implement LLM-based tool selection
2. **Tool Caching**: Cache tool lists and results
3. **Parallel Execution**: Execute multiple tools concurrently
4. **Tool Composition**: Chain tools for complex workflows
5. **User Learning**: Adapt tool selection based on user feedback

## Files Reference

- **Core Implementation**: `pkg/runtime/mcp_node.go`
- **Basic Flow**: `mcp_list_then_execute.json`
- **Dynamic Flow**: `mcp_dynamic_execution.json`
- **Agent Flow**: `mcp_agent_flow.json`
- **Test Scripts**: `scripts/test_mcp_*.sh`
- **Progress Report**: `MCP_CLIENT_PROGRESS_REPORT.md`

This integration enables FlowRunner to act as an intelligent agent that can discover and use MCP tools dynamically, supporting complex multi-tool workflows and natural language interactions.