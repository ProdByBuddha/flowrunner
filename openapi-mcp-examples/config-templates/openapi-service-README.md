# Generic OpenAPI Configuration

Template for Generic OpenAPI service integration

## Environment Variables

```bash
export OPENAPI_SPEC_URL="your_value_here"
export OPENAPI_SPEC_FILE="your_value_here"
export OPENAPI_BASE_URL="your_value_here"
export OPENAPI_API_KEY="your_value_here"
export OPENAPI_BEARER_TOKEN="your_value_here"
export OPENAPI_BASIC_USER="your_value_here"
export OPENAPI_BASIC_PASS="your_value_here"
export OPENAPI_MCP_ALLOWED_METHODS="your_value_here"
export OPENAPI_MCP_RATE_LIMIT="your_value_here"
```

## Usage

```bash
node examples/mcp-multi-host.js --config examples/config-templates/openapi-service.json
```
