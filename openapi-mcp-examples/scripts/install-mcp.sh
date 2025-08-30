#!/bin/bash

# Universal MCP installation script
# This script sets up the MCP server configuration for any user

set -e

PROJECT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
MCP_CONFIG_FILE="$HOME/.kiro/settings/mcp.json"

echo "🚀 Setting up MCP server for n8n OpenAPI tools..."

# Create .kiro settings directory if it doesn't exist
mkdir -p "$(dirname "$MCP_CONFIG_FILE")"

# Create or update MCP configuration
cat > "$MCP_CONFIG_FILE" << EOF
{
  "mcpServers": {
    "openapi-mcp-server": {
      "command": "$PROJECT_DIR/start-mcp-server.sh",
      "args": [],
      "env": {},
      "disabled": false,
      "autoApprove": []
    }
  }
}
EOF

# Create .env file from example if it doesn't exist
if [ ! -f "$PROJECT_DIR/.env" ]; then
    if [ -f "$PROJECT_DIR/.env.example" ]; then
        cp "$PROJECT_DIR/.env.example" "$PROJECT_DIR/.env"
        echo "📝 Created .env file from .env.example"
        echo "⚠️  Please edit .env file with your actual API keys"
    else
        echo "⚠️  No .env.example found. Please create a .env file with your API keys"
    fi
else
    echo "✅ .env file already exists"
fi

# Make the start script executable
chmod +x "$PROJECT_DIR/start-mcp-server.sh"

echo "✅ MCP server configuration updated at: $MCP_CONFIG_FILE"
echo "🔧 Project directory: $PROJECT_DIR"
echo ""
echo "Next steps:"
echo "1. Edit your .env file with actual API keys"
echo "2. Restart Kiro or reconnect MCP servers"
echo "3. Check the MCP Server panel in Kiro to verify tools are loaded"