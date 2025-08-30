# n8n - Workflow Integration

Multi-step workflows using multiple tools for n8n workflow automation platform

**Complexity Level:** intermediate
**Generated:** 2025-08-30T00:29:20.882Z

## Examples

### Complete Workflow

Multi-step workflow using n8n

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
        "name": "n8n.getCredentialType",
        "arguments": {
      "credentialTypeName": "example-value"
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
  },
  {
    "step": 3,
    "description": "Update the created item",
    "request": {
      "jsonrpc": "2.0",
      "id": "step-3",
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
          "name": "n8n.getCredentialType",
          "arguments": {
      "credentialTypeName": "example-value"
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

