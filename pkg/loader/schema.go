package loader

// FlowSchema is the JSON schema for flow definitions
const FlowSchema = `
{
  "$schema": "http://json-schema.org/draft-07/schema#",
  "type": "object",
  "required": ["metadata", "nodes"],
  "properties": {
    "metadata": {
      "type": "object",
      "required": ["name"],
      "properties": {
        "name": {
          "type": "string",
          "minLength": 1
        },
        "description": {
          "type": "string"
        },
        "version": {
          "type": "string"
        }
      }
    },
    "nodes": {
      "type": "object",
      "minProperties": 1,
      "additionalProperties": {
        "type": "object",
        "required": ["type"],
        "properties": {
          "type": {
            "type": "string",
            "minLength": 1
          },
          "params": {
            "type": "object"
          },
          "next": {
            "type": "object",
            "additionalProperties": {
              "type": "string"
            }
          },
          "batch": {
            "type": "object",
            "properties": {
              "strategy": {
                "type": "string",
                "enum": ["serial", "async", "parallel", "worker_pool"]
              },
              "max_parallel": {
                "type": "integer",
                "minimum": 1
              }
            }
          },
          "retry": {
            "type": "object",
            "properties": {
              "max_retries": {
                "type": "integer",
                "minimum": 0
              },
              "wait": {
                "type": "string",
                "pattern": "^[0-9]+(ns|us|ms|s|m|h)$"
              }
            }
          },
          "hooks": {
            "type": "object",
            "properties": {
              "prep": {
                "type": "string"
              },
              "exec": {
                "type": "string"
              },
              "post": {
                "type": "string"
              }
            }
          }
        }
      }
    }
  }
}
`
