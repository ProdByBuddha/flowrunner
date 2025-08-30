# MCP Client Progress Report

Date: 2025-08-28

## Summary

We implemented a first‑class MCP client node for Flowrunner, wired it into the runtime, and verified connectivity against your locally hosted multi‑host openapi‑mcp-server. We added test flows and a helper script that can auto‑launch the MCP server, discover the correct POST endpoint, and run list/execute operations. We resolved several integration issues (auth token, YAML formatting, port conflicts, SSE vs HTTP behavior) and landed on a reliable HTTP‑only approach for production use.

## What’s Implemented

- Runtime MCP node (`pkg/runtime/mcp_node.go`):
  - Transports: `cmd` (STDIO), `sse`, `http`.
  - Dotenvx integration for `cmd` (auto‑load .env files; supports `npx dotenvx@latest` or `dotenvx` binary).
  - `rawBody` override to post exact JSON envelopes for HTTP/SSE servers.
  - `argsExtra`, `serverPort`, `portFlag`, `portEnvKeys` for robust child process control (cmd mode).
  - Optimization: when `messagesPostEndpoint` is set in `sse` mode, the node posts to HTTP and returns immediately (no SSE wait).

- Registration & loader:
  - MCP node added to `CoreNodeTypes` and used by the YAML loader via the runtime factory adapter.

- Demos and automation:
  - Demo flows: `demos/mcp_list_tools.yaml`, `demos/mcp_execute_tool.yaml`.
  - Test harness: `scripts/test_mcp.sh` with:
    - Transports: `sse` (default), `http` (direct), `cmd`.
    - Dotenvx injection for `cmd` flows.
    - Auto‑launch mode for MCP server (picks free ports or uses provided ones, waits for SSE ready, discovers POST endpoint, and runs tests).
    - Dynamic port injection and `argsExtra` to avoid conflicts (e.g., 3007 WS port).
    - Optional headers via `MCP_HEADERS_JSON` and raw POST bodies for list/execute.
  - Utility: `scripts/port_inspect_kill.sh` to inspect/kill listeners on a port.

## Verified Endpoints (your server)

- SSE stream: `http://127.0.0.1:33106/mcp-sse` (200 OK and streams)
- HTTP messages POST: `http://127.0.0.1:3005/mcp` (accepts `{"method":"tools/list","params":{}}` and `tools/call`)
- WS: `3007` (init process or server WS transport; we avoid conflicts by not launching cmd flows during runs)

## Current Working Approach

- Prefer HTTP‑only flows for reliability:
  - connectionType: `http`
  - url: `http://127.0.0.1:3005/mcp`
  - operation: `listTools` (required by current node; `rawBody` actually drives the request)
  - rawBody: `{"method":"tools/list","params":{}}`

- For execute:
  - rawBody: `{"method":"tools/call","params":{"name":"TOOL_NAME","arguments":{...}}}`

You created JSON flow specs for this:

- `/root/projects/flowrunner/mcp_list_http.json`
- `/root/projects/flowrunner/mcp_execute_http.json`

These are preferred for your environment and avoid YAML formatting issues.

## Issues Encountered & Resolutions

- 401 Unauthorized on flow create: Bearer header was empty (`Authorization: Bearer`). Fixed by logging in and using a non‑empty `$TOKEN`.
- YAML parse errors: switched to known‑good YAML and finally JSON flows (your `mcp_list_http.json` and `mcp_execute_http.json`).
- “Unknown method” (400 from MCP): server expects `tools/list` envelope; we added `rawBody` for exact payloads.
- Infinite wait in SSE mode: original node waited for SSE messages. Updated node to post to `messagesPostEndpoint` and return immediately.
- Port 3007 EADDRINUSE: caused by `cmd` flows launching a child that wanted WS 3007. Resolved by using HTTP/SSE without `cmd`, and adding dynamic port selection + ws/port flags for `cmd` when needed.
- Endpoint confusion (3005 vs 33105): added discovery logic and clarified the final, working POST endpoint is 3005 for your setup.

## Next Steps / TODOs

- Node ergonomics:
  - Relax `operation` requirement when `rawBody` is provided (so HTTP-only flows don’t need a placeholder operation).
  - Add first‑class mappings for common MCP methods: `listTools` -> `tools/list`; `executeTool` -> `tools/call` with name/arguments.

- Script improvements:
  - Make error bodies visible on 400 (flow create/update) for faster diagnosis.
  - Provide a minimal “HTTP‑only” mode path by default for local testing.
  - Optional: auto‑detect and skip WS transport in `cmd` flows (or force a random WS port) to avoid 3007.

- Documentation:
  - Add a short “Local MCP Testing” readme with the precise curl preflight and the HTTP‑only flow JSON examples.

- Optional features:
  - Headers templating in flows (e.g., pass auth tokens per request).
  - Stronger POST endpoint discovery (allow custom paths or regex).

## Quick Commands (HTTP‑only)

1) Start your MCP server (keep it running):

```
npx -y @prodbybuddha/openapi-mcp-server@latest \
  --config /root/projects/flowrunner/services.dynamic.json \
  --sse-port 33106 \
  --post-port 3005
```

2) Verify:

```
curl -v -sS http://127.0.0.1:33106/mcp-sse
curl -v -sS -X POST http://127.0.0.1:3005/mcp -H 'Content-Type: application/json' \
  -d '{"method":"tools/list","params":{}}'
```

3) Create and run HTTP flow (using your JSON files):

```
TOKEN=$(curl -s -X POST http://127.0.0.1:8080/api/v1/login \
  -H 'Content-Type: application/json' \
  -d '{"username":"demo","password":"demo"}' | jq -r .token)

CONTENT=$(cat /root/projects/flowrunner/mcp_list_http.json)
FLOW_ID=$(curl -s -X POST http://127.0.0.1:8080/api/v1/flows \
  -H "Authorization: Bearer $TOKEN" -H 'Content-Type: application/json' \
  -d "$CONTENT" | jq -r .id)

EXEC_ID=$(curl -s -X POST http://127.0.0.1:8080/api/v1/flows/$FLOW_ID/run \
  -H "Authorization: Bearer $TOKEN" -H 'Content-Type: application/json' \
  -d '{"input":{}}' | jq -r .execution_id)

curl -s -H "Authorization: Bearer $TOKEN" \
  http://127.0.0.1:8080/api/v1/executions/$EXEC_ID/logs | jq
```

## Files Touched

- `pkg/runtime/mcp_node.go` — core MCP node wrapper
- `pkg/runtime/core_nodes.go` — MCP registered as a core node type
- `scripts/test_mcp.sh` — automated testing, auto‑launch, discovery
- `scripts/port_inspect_kill.sh` — port inspection/kill helper
- `demos/mcp_list_tools.yaml`, `demos/mcp_execute_tool.yaml` — examples

## Status

- HTTP‑only flows to your local MCP server are working and return results.
- SSE can be used when POST is guaranteed to be live; otherwise prefer HTTP.
- `cmd` transport is supported but not recommended for local multi‑host unless port management is enforced.

