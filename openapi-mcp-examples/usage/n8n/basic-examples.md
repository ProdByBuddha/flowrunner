# n8n - Basic Usage

Simple, single-tool usage examples for n8n workflow automation platform

**Complexity Level:** beginner
**Generated:** 2025-08-30T00:29:20.881Z

## Examples

### Service Connection

How to connect to n8n

**Category:** setup

```bash
# Environment setup for n8n
export N8N_API_KEY="your_n8n_api_key"
export N8N_API_URL="your_n8n_api_url"

# Start MCP server
node examples/mcp-n8n-server.js

# Or use multi-host configuration
node examples/mcp-multi-host.js --config services.json

# Test connection
echo '{"jsonrpc":"2.0","id":"1","method":"initialize","params":{"protocolVersion":"2024-11-05","clientInfo":{"name":"test","version":"1.0.0"}}}' | node examples/mcp-n8n-server.js
```

### List Available Tools

Get all available tools for this service

**Category:** discovery

```json
{
  "jsonrpc": "2.0",
  "id": "list-tools",
  "method": "tools/list",
  "params": {}
}

// Expected response:
{
  "jsonrpc": "2.0",
  "id": "list-tools",
  "result": {
    "tools": [
      {
        "name": "n8n.generateAudit",
        "description": "Generate an audit",
        "inputSchema": {
        "type": "object",
        "properties": {
                "body": {
                        "type": "object",
                        "properties": {
                                "additionalOptions": {
                                        "type": "object",
                                        "properties": {
                                                "daysAbandonedWorkflow": {
                                                        "type": "integer",
                                                        "description": "Days for a workflow to be considered abandoned if not executed"
                                                },
                                                "categories": {
                                                        "type": "array",
                                                        "items": {
                                                                "type": "string",
                                                                "enum": [
                                                                        "credentials",
                                                                        "database",
                                                                        "nodes",
                                                                        "filesystem",
                                                                        "instance"
                                                                ]
                                                        }
                                                }
                                        }
                                }
                        }
                },
                "X-N8N-API-KEY": {
                        "type": "string",
                        "description": "API Key for ApiKeyAuth"
                }
        },
        "required": []
}
      },
      {
        "name": "n8n.createCredential",
        "description": "Create a credential",
        "inputSchema": {
        "type": "object",
        "properties": {
                "body": {
                        "type": "object",
                        "properties": {
                                "id": {
                                        "type": "string",
                                        "example": "R2DjclaysHbqn778"
                                },
                                "name": {
                                        "type": "string",
                                        "example": "Joe's Github Credentials"
                                },
                                "type": {
                                        "type": "string",
                                        "example": "github"
                                },
                                "data": {
                                        "type": "object",
                                        "example": {
                                                "token": "ada612vad6fa5df4adf5a5dsf4389adsf76da7s"
                                        }
                                },
                                "createdAt": {
                                        "type": "string",
                                        "example": "2022-04-29T11:02:29.842Z",
                                        "format": "date-time"
                                },
                                "updatedAt": {
                                        "type": "string",
                                        "example": "2022-04-29T11:02:29.842Z",
                                        "format": "date-time"
                                }
                        },
                        "required": [
                                "name",
                                "type",
                                "data"
                        ]
                },
                "X-N8N-API-KEY": {
                        "type": "string",
                        "description": "API Key for ApiKeyAuth"
                }
        },
        "required": [
                "body"
        ]
}
      },
      {
        "name": "n8n.deleteCredential",
        "description": "Delete credential by ID",
        "inputSchema": {
        "type": "object",
        "properties": {
                "id": {
                        "type": "string",
                        "description": "The credential ID that needs to be deleted"
                },
                "X-N8N-API-KEY": {
                        "type": "string",
                        "description": "API Key for ApiKeyAuth"
                }
        },
        "required": [
                "id"
        ]
}
      }
      // ... 37 more tools
    ]
  }
}
```

### Other Operation

Basic other operation using generateAudit

**Category:** other

```json
{
  "jsonrpc": "2.0",
  "id": "call-generateAudit",
  "method": "tools/call",
  "params": {
    "name": "n8n.generateAudit",
    "arguments": {
    "body": {
        "additionalOptions": {
            "daysAbandonedWorkflow": 1,
            "categories": [
                "credentials"
            ]
        }
    }
}
  }
}

// Expected response:
{
  "jsonrpc": "2.0",
  "id": "call-generateAudit",
  "result": {
    "content": [
      {
        "type": "text",
        "text": "Operation completed successfully"
      }
    ]
  }
}
```

### Create Operation

Basic create operation using createCredential

**Category:** create

```json
{
  "jsonrpc": "2.0",
  "id": "call-createCredential",
  "method": "tools/call",
  "params": {
    "name": "n8n.createCredential",
    "arguments": {
    "body": {
        "id": "example-value",
        "name": "example-value",
        "type": "example-value",
        "data": {},
        "createdAt": "2024-01-01T00:00:00Z",
        "updatedAt": "2024-01-01T00:00:00Z"
    }
}
  }
}

// Expected response:
{
  "jsonrpc": "2.0",
  "id": "call-createCredential",
  "result": {
    "content": [
      {
        "type": "text",
        "text": "Operation completed successfully"
      }
    ]
  }
}
```

### Delete Operation

Basic delete operation using deleteCredential

**Category:** delete

```json
{
  "jsonrpc": "2.0",
  "id": "call-deleteCredential",
  "method": "tools/call",
  "params": {
    "name": "n8n.deleteCredential",
    "arguments": {
    "id": "example-value"
}
  }
}

// Expected response:
{
  "jsonrpc": "2.0",
  "id": "call-deleteCredential",
  "result": {
    "content": [
      {
        "type": "text",
        "text": "Operation completed successfully"
      }
    ]
  }
}
```

### List Operation

Basic list operation using getCredentialType

**Category:** list

```json
{
  "jsonrpc": "2.0",
  "id": "call-getCredentialType",
  "method": "tools/call",
  "params": {
    "name": "n8n.getCredentialType",
    "arguments": {
    "credentialTypeName": "example-value"
}
  }
}

// Expected response:
{
  "jsonrpc": "2.0",
  "id": "call-getCredentialType",
  "result": {
    "content": [
      {
        "type": "text",
        "text": "Operation completed successfully"
      }
    ]
  }
}
```

### Update Operation

Basic update operation using updateTag

**Category:** update

```json
{
  "jsonrpc": "2.0",
  "id": "call-updateTag",
  "method": "tools/call",
  "params": {
    "name": "n8n.updateTag",
    "arguments": {
    "id": "example-value",
    "body": {
        "id": "example-value",
        "name": "example-value",
        "createdAt": "2024-01-01T00:00:00Z",
        "updatedAt": "2024-01-01T00:00:00Z"
    }
}
  }
}

// Expected response:
{
  "jsonrpc": "2.0",
  "id": "call-updateTag",
  "result": {
    "content": [
      {
        "type": "text",
        "text": "Operation completed successfully"
      }
    ]
  }
}
```

