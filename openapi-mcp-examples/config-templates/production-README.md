# Production Configuration

Template optimized for production use with security and performance considerations

## Environment Variables

```bash
export N8N_API_KEY="your_value_here"
export N8N_API_URL="your_value_here"
export HOSTINGER_API_TOKEN="your_value_here"
export MCP_RATE_LIMIT=100
```

## Usage

```bash
node examples/mcp-multi-host.js --config examples/config-templates/production.json
```
