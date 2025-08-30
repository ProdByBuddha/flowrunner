# Multi-Service MCP Host

Run multiple OpenAPI services in one MCP process. Tools are namespaced as `<service>.<toolName>`.

- Start (stdio): `node examples/mcp-multi-host.js --config ./examples/services.example.json`
- Start (HTTP): `PORT=3005 node examples/mcp-multi-host.js --config ./examples/services.example.json --transport http`
- Start (WS):   `PORT=3007 node examples/mcp-multi-host.js --config ./examples/services.example.json --transport ws`
- Start (SSE):  `PORT=3006 node examples/mcp-multi-host.js --config ./examples/services.example.json --transport sse`

## Config format

See `examples/services.example.json`. Each service:
- `name`: prefix for tool names
- `type`: `openapi`
- `specFile` or `specUrl`: OpenAPI source
- `baseUrl`: API base URL
- `auth`: `{ kind: 'bearer'|'header'|'apiKey', name?, env?, value?, in? }`
- `filters` (optional): same include/exclude + regex filters as generator CLI

## Hardening & logging
- `OPENAPI_MCP_ALLOWED_METHODS`, `OPENAPI_MCP_ALLOWED_PATHS`
- `OPENAPI_MCP_RATE_LIMIT`, `OPENAPI_MCP_RATE_BURST`, `OPENAPI_MCP_RATE_WINDOW_MS`
- `OPENAPI_MCP_CONCURRENCY`, `OPENAPI_MCP_CONCURRENCY_PER_PATH`
- `DEBUG_HTTP=1` to enable request/response summaries (redacted)

### Environment presets

- Development (more permissive, more logs):
  - `OPENAPI_MCP_ALLOWED_METHODS=GET,POST,PUT,PATCH,DELETE`
  - `OPENAPI_MCP_ALLOWED_PATHS=*`
  - `OPENAPI_MCP_RATE_LIMIT=120 OPENAPI_MCP_RATE_BURST=60 OPENAPI_MCP_RATE_WINDOW_MS=60000`
  - `OPENAPI_MCP_CONCURRENCY=16 OPENAPI_MCP_CONCURRENCY_PER_PATH=8`
  - `DEBUG_HTTP=1`

- Production (stricter, safer defaults):
  - `OPENAPI_MCP_ALLOWED_METHODS=GET,POST`
  - `OPENAPI_MCP_ALLOWED_PATHS=/workflows*,/executions*,/api/*` (tighten for your APIs)
  - `OPENAPI_MCP_RATE_LIMIT=60 OPENAPI_MCP_RATE_BURST=30 OPENAPI_MCP_RATE_WINDOW_MS=60000`
  - `OPENAPI_MCP_CONCURRENCY=8 OPENAPI_MCP_CONCURRENCY_PER_PATH=4`
  - No `DEBUG_HTTP` (omit or set to `0`)

## Dev tips
- Use `nodemon` for reload: `npx nodemon examples/mcp-multi-host.js -- --config ./examples/services.example.json`
- Keep secrets in `.env` and export as needed (`N8N_API_KEY`, `HOSTINGER_API_TOKEN`, etc.)

## Cursor/Kiro MCP config snippet (stdio)

```
{
  "clients": [
    {
      "name": "multi-host",
      "command": "node",
      "args": ["examples/mcp-multi-host.js", "--config", "examples/services.example.json"],
      "env": {
        "N8N_API_KEY": "${env:N8N_API_KEY}",
        "HOSTINGER_API_TOKEN": "${env:HOSTINGER_API_TOKEN}"
      }
    }
  ]
}
```

- For HTTP/WS, point your MCP proxy to `http://localhost:3005/mcp` or `ws://localhost:3007/mcp` accordingly.


## Docker notes

- Multi-host can ingest the Docker Engine OpenAPI (YAML) via `specUrl` and talk to the Engine over TCP (e.g., `DOCKER_API_HOST=http://localhost:2375`).
- Unix sockets (`/var/run/docker.sock`) are not directly accessible via HTTP transports; use the dedicated Docker MCP server (`examples/mcp-docker-server.js`) for local unix-socket access and CLI wrappers.
- For safety, keep TCP disabled unless protected by network policy/TLS, and avoid exposing your daemon publicly. Prefer local unix sockets.
