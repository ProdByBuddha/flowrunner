# hostinger - Workflow Integration

Multi-step workflows using multiple tools for Hostinger cloud hosting and domain management

**Complexity Level:** intermediate
**Generated:** 2025-08-30T00:29:20.884Z

## Examples

### Complete Workflow

Multi-step workflow using hostinger

**Category:** workflow

```json
// Multi-step workflow example
[
  {
    "step": 1,
    "description": "List existing items",
    "request": {
      "jsonrpc": "2.0",
      "id": "step-1",
      "method": "tools/call",
      "params": {
        "name": "hostinger.billing_getCatalogItemListV1",
        "arguments": {
      "category": "VPS",
      "name": ".COM*"
}
      }
    }
  },
  {
    "step": 2,
    "description": "Create new item based on list results",
    "request": {
      "jsonrpc": "2.0",
      "id": "step-2",
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
  },
  {
    "step": 3,
    "description": "Update the created item",
    "request": {
      "jsonrpc": "2.0",
      "id": "step-3",
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
  }
]
```

### Error Handling Workflow

Workflow with proper error handling

**Category:** error-handling

```json
{
  "workflow": "Error Handling Example",
  "steps": [
    {
      "name": "attempt_operation",
      "request": {
        "jsonrpc": "2.0",
        "id": "attempt-1",
        "method": "tools/call",
        "params": {
          "name": "hostinger.billing_getCatalogItemListV1",
          "arguments": {
      "category": "VPS",
      "name": ".COM*"
}
        }
      },
      "error_handling": {
        "on_error": "retry",
        "max_retries": 3,
        "retry_delay": 1000,
        "fallback": {
          "action": "log_and_continue",
          "message": "Operation failed after retries, continuing workflow"
        }
      }
    },
    {
      "name": "handle_success",
      "condition": "previous_step_success",
      "action": "process_result"
    },
    {
      "name": "handle_failure",
      "condition": "previous_step_failed",
      "action": "send_alert"
    }
  ]
}
```

### Conditional Workflow

Workflow with conditional logic

**Category:** conditional

```json
{
  "workflow": "Conditional Logic Example",
  "steps": [
    {
      "name": "check_existing",
      "request": {
        "jsonrpc": "2.0",
        "id": "check-1",
        "method": "tools/call",
        "params": {
          "name": "hostinger.billing_getCatalogItemListV1",
          "arguments": {
      "category": "VPS",
      "name": ".COM*"
}
        }
      }
    },
    {
      "name": "create_if_needed",
      "condition": {
        "if": "response.data.length === 0",
        "then": {
          "request": {
            "jsonrpc": "2.0",
            "id": "create-1",
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
        },
        "else": {
          "action": "log",
          "message": "Items already exist, skipping creation"
        }
      }
    }
  ]
}
```

