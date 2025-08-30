#!/bin/bash

# Universal OpenAPI MCP server launcher that loads .env file
# This ensures environment variables are available regardless of shell setup

# Get the directory where this script is located
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

# Load .env file if it exists
if [ -f "$SCRIPT_DIR/.env" ]; then
    echo "Loading environment variables from .env file..."
    set -a  # automatically export all variables
    source "$SCRIPT_DIR/.env"
    set +a  # stop automatically exporting
    echo "Environment variables loaded successfully"
else
    echo "Warning: .env file not found at $SCRIPT_DIR/.env"
fi

# Launch the OpenAPI MCP server with all environment variables available
exec /usr/bin/npx @prodbybuddha/openapi-mcp-server@1.4.6 --config "$SCRIPT_DIR/services.dynamic.json" "$@"