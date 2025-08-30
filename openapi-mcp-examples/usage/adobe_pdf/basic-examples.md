# adobe_pdf - Basic Usage

Simple, single-tool usage examples for Adobe PDF Services for document processing

**Complexity Level:** beginner
**Generated:** 2025-08-30T00:29:20.885Z

## Examples

### Service Connection

How to connect to adobe_pdf

**Category:** setup

```bash
# Environment setup for adobe_pdf
export ADOBE_ACCESS_TOKEN="your_adobe_access_token"

# Start MCP server
node examples/mcp-adobe_pdf-server.js

# Or use multi-host configuration
node examples/mcp-multi-host.js --config services.json

# Test connection
echo '{"jsonrpc":"2.0","id":"1","method":"initialize","params":{"protocolVersion":"2024-11-05","clientInfo":{"name":"test","version":"1.0.0"}}}' | node examples/mcp-adobe_pdf-server.js
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
        "name": "adobe_pdf.createAsset",
        "description": "Create an asset placeholder for upload",
        "inputSchema": {
        "type": "object",
        "properties": {
                "x-api-key": {
                        "type": "string"
                },
                "body": {
                        "type": "object",
                        "properties": {
                                "mediaType": {
                                        "type": "string",
                                        "example": "application/pdf"
                                }
                        }
                },
                "bearerToken": {
                        "type": "string",
                        "description": "Bearer token for bearerAuth"
                }
        },
        "required": [
                "x-api-key",
                "body",
                "bearerToken"
        ]
}
      },
      {
        "name": "adobe_pdf.combinePDF",
        "description": "Combine multiple PDF assets into one PDF",
        "inputSchema": {
        "type": "object",
        "properties": {
                "x-api-key": {
                        "type": "string"
                },
                "body": {
                        "type": "object",
                        "properties": {
                                "inputs": {
                                        "type": "array",
                                        "items": {
                                                "type": "object",
                                                "properties": {
                                                        "assetID": {
                                                                "type": "string"
                                                        }
                                                }
                                        }
                                },
                                "options": {
                                        "type": "object"
                                }
                        },
                        "required": [
                                "inputs"
                        ]
                },
                "bearerToken": {
                        "type": "string",
                        "description": "Bearer token for bearerAuth"
                }
        },
        "required": [
                "x-api-key",
                "body",
                "bearerToken"
        ]
}
      },
      {
        "name": "adobe_pdf.ocrPDF",
        "description": "OCR a PDF asset",
        "inputSchema": {
        "type": "object",
        "properties": {
                "x-api-key": {
                        "type": "string"
                },
                "body": {
                        "type": "object",
                        "properties": {
                                "input": {
                                        "type": "object",
                                        "properties": {
                                                "assetID": {
                                                        "type": "string"
                                                }
                                        }
                                }
                        }
                },
                "bearerToken": {
                        "type": "string",
                        "description": "Bearer token for bearerAuth"
                }
        },
        "required": [
                "x-api-key",
                "body",
                "bearerToken"
        ]
}
      }
      // ... 4 more tools
    ]
  }
}
```

### Create Operation

Basic create operation using createAsset

**Category:** create

```json
{
  "jsonrpc": "2.0",
  "id": "call-createAsset",
  "method": "tools/call",
  "params": {
    "name": "adobe_pdf.createAsset",
    "arguments": {
    "body": {
        "mediaType": "example-value"
    }
}
  }
}

// Expected response:
{
  "jsonrpc": "2.0",
  "id": "call-createAsset",
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

### Other Operation

Basic other operation using combinePDF

**Category:** other

```json
{
  "jsonrpc": "2.0",
  "id": "call-combinePDF",
  "method": "tools/call",
  "params": {
    "name": "adobe_pdf.combinePDF",
    "arguments": {
    "body": {
        "inputs": [
            {
                "assetID": "example-value"
            }
        ],
        "options": {}
    }
}
  }
}

// Expected response:
{
  "jsonrpc": "2.0",
  "id": "call-combinePDF",
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

Basic list operation using getJob

**Category:** list

```json
{
  "jsonrpc": "2.0",
  "id": "call-getJob",
  "method": "tools/call",
  "params": {
    "name": "adobe_pdf.getJob",
    "arguments": {
    "jobId": "example-value"
}
  }
}

// Expected response:
{
  "jsonrpc": "2.0",
  "id": "call-getJob",
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

