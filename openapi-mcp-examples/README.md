# Examples

Quick commands and tips to use the example scripts in this repo.

## Generate Tools JSON

- From URL:
  - `npm run mcp:gen:tools -- --from-url https://api.example.com/openapi.json --out examples/generated/tools.json`
- From file:
  - `npm run mcp:gen:tools -- --from-file ./path/to/openapi.json --out examples/generated/tools.json`

The output file contains serializable tool metadata (no functions) that the server can load offline.

## Generate a Full MCP Server

- From URL:
  - `OPENAPI_BASE_URL=https://api.example.com/v1 npm run mcp:gen:server -- --from-url https://api.example.com/openapi.json --generate-server ./generated-server`
- From file:
  - `OPENAPI_BASE_URL=https://api.example.com/v1 npm run mcp:gen:server -- --from-file ./openapi.json --generate-server ./generated-server`

Notes:
- The script respects `--baseUrl` and the `OPENAPI_BASE_URL` env var. If neither is provided, it uses `servers[0].url` from the spec.
- The generated project includes a minimal HTTP client, input validation (Zod), and stdio transport entry.

## Run Generic OpenAPI Server

- From file: `OPENAPI_SPEC_FILE=./openapi.json npm run mcp:openapi`
- From URL: `OPENAPI_SPEC_URL=https://api.example.com/openapi.json npm run mcp:openapi`
- Override base URL: `OPENAPI_BASE_URL=https://api.example.com/v1`

## HTTP Debug Logging

Enable verbose HTTP logs for both the generic server and generated servers:

- `DEBUG_HTTP=1` — prints outbound request headers and summarized responses, and logs OAuth2 client credentials token exchanges.

Example:

`DEBUG_HTTP=1 OPENAPI_SPEC_FILE=./openapi.json npm run mcp:openapi`

## Docker Server

The project includes a dedicated Docker MCP server for managing containers, images, and more. For details, refer to the "Docker Server" section in the [main README.md](../README.md#docker-server).

