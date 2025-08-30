# Local MCP Testing (HTTP-only)

This guide shows a minimal, reliable HTTP-only setup for exercising Flowrunner's MCP client node against a local MCP server (e.g., `openapi-mcp-server`). It avoids SSE/WS complexity and YAML formatting pitfalls by using JSON flow specs with raw request bodies.

## 1) Start your MCP server

Example using `@prodbybuddha/openapi-mcp-server` with explicit SSE and POST ports (SSE is optional here; we only use POST):

```
npx -y @prodbybuddha/openapi-mcp-server@latest \
  --config /root/projects/flowrunner/services.dynamic.json \
  --sse-port 33106 \
  --post-port 3005
```

## 2) Verify HTTP POST is live

```
curl -v -sS -X POST http://127.0.0.1:3005/mcp \
  -H 'Content-Type: application/json' \
  -d '{"method":"tools/list","params":{}}'
```

## 3) Create and run HTTP-only flows

Use the included JSON flows which embed the exact MCP envelopes via `rawBody`:

- `mcp_list_http.json`
- `mcp_execute_http.json`

Quick commands:

```
BASE=http://127.0.0.1:8080
TOKEN=$(curl -s -X POST "$BASE/api/v1/login" \
  -H 'Content-Type: application/json' \
  -d '{"username":"demo","password":"demo"}' | jq -r .token)

FLOW_ID=$(curl -sS -X POST "$BASE/api/v1/flows" \
  -H "Authorization: Bearer $TOKEN" -H 'Content-Type: application/json' \
  -d "$(jq -Rs --arg name 'MCP: List Tools (HTTP)' '{name:$name, content:.}' < mcp_list_http.json)" | jq -r .id)

EXEC_ID=$(curl -sS -X POST "$BASE/api/v1/flows/$FLOW_ID/run" \
  -H "Authorization: Bearer $TOKEN" -H 'Content-Type: application/json' \
  -d '{"input":{}}' | jq -r .execution_id)

curl -sS -H "Authorization: Bearer $TOKEN" \
  "$BASE/api/v1/executions/$EXEC_ID/logs" | jq
```

To execute a specific tool, edit `mcp_execute_http.json` to set the tool `name` and `arguments` in the `rawBody`, then repeat the create/run steps.

## Notes

- The MCP node now accepts flows with only `rawBody` (no `operation` needed). For convenience, two aliases are recognized when you prefer structured params:
  - `operation: "listTools"` → `method: "tools/list"`
  - `operation: "executeTool"` → `method: "tools/call"` (uses `toolName`/`toolParameters`)
- For multi-host or dynamic setups, prefer the HTTP-only path for predictability. SSE is supported but not required when the POST endpoint returns results synchronously.

### Header and Body Templating

- Headers accept values via `headers` that can be full expressions like `${'Bearer ' + input.token}` (when using expression syntax) or via Go templates in `headersTemplate`, e.g.:

```
"headersTemplate": { "Authorization": "Bearer {{.input.token}}" }
```

- Request bodies can be provided as structured `rawBody` objects (which support expression evaluation on fields) or as a string rendered via `rawBodyTemplate` using Go templates for precise interpolation:

```
"rawBodyTemplate": "{\n  \"method\": \"tools/call\",\n  \"params\": {\n    \"name\": \"{{.params.toolName}}\",\n    \"arguments\": { \"id\": \"{{.input.user_id}}\" }\n  }\n}"
```

Use either approach based on your comfort with expressions (`${...}`) vs templates (`{{...}}`).
