# hostinger - Basic Usage

Simple, single-tool usage examples for Hostinger cloud hosting and domain management

**Complexity Level:** beginner
**Generated:** 2025-08-30T00:29:20.884Z

## Examples

### Service Connection

How to connect to hostinger

**Category:** setup

```bash
# Environment setup for hostinger
export HOSTINGER_API_TOKEN="your_hostinger_api_token"

# Start MCP server
node examples/mcp-hostinger-server.js

# Or use multi-host configuration
node examples/mcp-multi-host.js --config services.json

# Test connection
echo '{"jsonrpc":"2.0","id":"1","method":"initialize","params":{"protocolVersion":"2024-11-05","clientInfo":{"name":"test","version":"1.0.0"}}}' | node examples/mcp-hostinger-server.js
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
        "name": "hostinger.billing_getCatalogItemListV1",
        "description": "Get catalog item list",
        "inputSchema": {
        "type": "object",
        "properties": {
                "category": {
                        "type": "string",
                        "description": "Filter catalog items by category",
                        "example": "VPS",
                        "enum": [
                                "DOMAIN",
                                "VPS"
                        ]
                },
                "name": {
                        "type": "string",
                        "description": "Filter catalog items by name. Use `*` for wildcard search, e.g. `.COM*` to find .com domain",
                        "example": ".COM*"
                },
                "bearerToken": {
                        "type": "string",
                        "description": "Bearer token for apiToken"
                }
        },
        "required": [
                "bearerToken"
        ]
}
      },
      {
        "name": "hostinger.billing_createServiceOrderV1",
        "description": "Create service order",
        "inputSchema": {
        "type": "object",
        "properties": {
                "body": {
                        "type": "object",
                        "properties": {
                                "payment_method_id": {
                                        "type": "integer",
                                        "description": "Payment method ID",
                                        "example": 517244
                                },
                                "items": {
                                        "type": "array",
                                        "items": {
                                                "type": "object",
                                                "properties": {
                                                        "item_id": {
                                                                "type": "string",
                                                                "description": "Price Item ID",
                                                                "example": "hostingercom-vps-kvm2-usd-1m"
                                                        },
                                                        "quantity": {
                                                                "type": "integer",
                                                                "example": 1
                                                        }
                                                },
                                                "required": [
                                                        "item_id",
                                                        "quantity"
                                                ]
                                        }
                                },
                                "coupons": {
                                        "type": "array",
                                        "description": "Discount coupon codes",
                                        "items": {
                                                "type": "string",
                                                "example": [
                                                        "Coupon 3"
                                                ]
                                        }
                                }
                        },
                        "required": [
                                "payment_method_id",
                                "items"
                        ]
                },
                "bearerToken": {
                        "type": "string",
                        "description": "Bearer token for apiToken"
                }
        },
        "required": [
                "body",
                "bearerToken"
        ]
}
      },
      {
        "name": "hostinger.billing_setDefaultPaymentMethodV1",
        "description": "Set default payment method",
        "inputSchema": {
        "type": "object",
        "properties": {
                "paymentMethodId": {
                        "type": "integer",
                        "description": "Payment method ID",
                        "example": 9693613
                },
                "bearerToken": {
                        "type": "string",
                        "description": "Bearer token for apiToken"
                }
        },
        "required": [
                "paymentMethodId",
                "bearerToken"
        ]
}
      }
      // ... 91 more tools
    ]
  }
}
```

### List Operation

Basic list operation using billing_getCatalogItemListV1

**Category:** list

```json
{
  "jsonrpc": "2.0",
  "id": "call-billing_getCatalogItemListV1",
  "method": "tools/call",
  "params": {
    "name": "hostinger.billing_getCatalogItemListV1",
    "arguments": {
    "category": "VPS",
    "name": ".COM*"
}
  }
}

// Expected response:
{
  "jsonrpc": "2.0",
  "id": "call-billing_getCatalogItemListV1",
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

Basic create operation using billing_createServiceOrderV1

**Category:** create

```json
{
  "jsonrpc": "2.0",
  "id": "call-billing_createServiceOrderV1",
  "method": "tools/call",
  "params": {
    "name": "hostinger.billing_createServiceOrderV1",
    "arguments": {
    "body": {
        "payment_method_id": 1,
        "items": [
            {
                "item_id": "example-value",
                "quantity": 1
            }
        ],
        "coupons": [
            "example-value"
        ]
    }
}
  }
}

// Expected response:
{
  "jsonrpc": "2.0",
  "id": "call-billing_createServiceOrderV1",
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

Basic other operation using billing_setDefaultPaymentMethodV1

**Category:** other

```json
{
  "jsonrpc": "2.0",
  "id": "call-billing_setDefaultPaymentMethodV1",
  "method": "tools/call",
  "params": {
    "name": "hostinger.billing_setDefaultPaymentMethodV1",
    "arguments": {
    "paymentMethodId": 9693613
}
  }
}

// Expected response:
{
  "jsonrpc": "2.0",
  "id": "call-billing_setDefaultPaymentMethodV1",
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

Basic delete operation using billing_deletePaymentMethodV1

**Category:** delete

```json
{
  "jsonrpc": "2.0",
  "id": "call-billing_deletePaymentMethodV1",
  "method": "tools/call",
  "params": {
    "name": "hostinger.billing_deletePaymentMethodV1",
    "arguments": {
    "paymentMethodId": 9693613
}
  }
}

// Expected response:
{
  "jsonrpc": "2.0",
  "id": "call-billing_deletePaymentMethodV1",
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

Basic update operation using DNS_updateDNSRecordsV1

**Category:** update

```json
{
  "jsonrpc": "2.0",
  "id": "call-DNS_updateDNSRecordsV1",
  "method": "tools/call",
  "params": {
    "name": "hostinger.DNS_updateDNSRecordsV1",
    "arguments": {
    "domain": "mydomain.tld",
    "body": {
        "overwrite": true,
        "zone": [
            {
                "name": "example-value",
                "records": [
                    {
                        "content": "example-value"
                    }
                ],
                "ttl": 1,
                "type": "A"
            }
        ]
    }
}
  }
}

// Expected response:
{
  "jsonrpc": "2.0",
  "id": "call-DNS_updateDNSRecordsV1",
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

