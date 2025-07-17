package loader

// FlowSchema is the JSON Schema for flow definitions
const FlowSchema = `{
  "$schema": "http://json-schema.org/draft-07/schema#",
  "title": "Flow Definition",
  "description": "A flow definition for the flowrunner service",
  "type": "object",
  "required": ["metadata", "nodes"],
  "properties": {
    "metadata": {
      "type": "object",
      "required": ["name"],
      "properties": {
        "name": {
          "type": "string",
          "description": "The name of the flow"
        },
        "description": {
          "type": "string",
          "description": "A description of the flow"
        },
        "version": {
          "type": "string",
          "description": "The version of the flow",
          "pattern": "^\\d+\\.\\d+\\.\\d+$"
        }
      }
    },
    "nodes": {
      "type": "object",
      "description": "The nodes in the flow",
      "minProperties": 1,
      "additionalProperties": {
        "$ref": "#/definitions/node"
      }
    }
  },
  "definitions": {
    "node": {
      "type": "object",
      "required": ["type"],
      "properties": {
        "type": {
          "type": "string",
          "description": "The type of the node"
        },
        "params": {
          "type": "object",
          "description": "Parameters for the node"
        },
        "next": {
          "type": "object",
          "description": "Next nodes to execute based on action",
          "additionalProperties": {
            "type": "string"
          }
        },
        "hooks": {
          "type": "object",
          "description": "JavaScript hooks for the node",
          "properties": {
            "prep": {
              "type": "string",
              "description": "JavaScript code to run before node execution"
            },
            "exec": {
              "type": "string",
              "description": "JavaScript code to run during node execution"
            },
            "post": {
              "type": "string",
              "description": "JavaScript code to run after node execution"
            }
          }
        }
      }
    }
  }
}`

// NodeTypeSchemas defines JSON schemas for specific node types
var NodeTypeSchemas = map[string]string{
	"http.request": `{
    "type": "object",
    "required": ["url"],
    "properties": {
      "url": {
        "type": "string",
        "format": "uri"
      },
      "method": {
        "type": "string",
        "enum": ["GET", "POST", "PUT", "DELETE", "PATCH", "HEAD", "OPTIONS"]
      },
      "headers": {
        "type": "object",
        "additionalProperties": {
          "type": "string"
        }
      },
      "body": {
        "type": ["string", "object", "null"]
      },
      "timeout": {
        "type": "string",
        "pattern": "^\\d+[smh]$"
      }
    }
  }`,
	"transform": `{
    "type": "object",
    "properties": {
      "mapping": {
        "type": "object"
      }
    }
  }`,
	"condition": `{
    "type": "object",
    "required": ["expression"],
    "properties": {
      "expression": {
        "type": "string"
      }
    }
  }`,
	"delay": `{
    "type": "object",
    "required": ["duration"],
    "properties": {
      "duration": {
        "type": "string",
        "pattern": "^\\d+[smh]$"
      }
    }
  }`,
	"llm": `{
    "type": "object",
    "required": ["model"],
    "properties": {
      "model": {
        "type": "string"
      },
      "messages": {
        "type": "array",
        "items": {
          "type": "object",
          "required": ["role", "content"],
          "properties": {
            "role": {
              "type": "string",
              "enum": ["system", "user", "assistant"]
            },
            "content": {
              "type": "string"
            }
          }
        }
      },
      "temperature": {
        "type": "number",
        "minimum": 0,
        "maximum": 2
      },
      "max_tokens": {
        "type": "integer",
        "minimum": 1
      },
      "structured_output": {
        "type": "object"
      }
    }
  }`,
	"email.send": `{
    "type": "object",
    "required": ["to", "subject", "body"],
    "properties": {
      "to": {
        "type": "string",
        "format": "email"
      },
      "cc": {
        "type": ["string", "array"],
        "items": {
          "type": "string",
          "format": "email"
        }
      },
      "bcc": {
        "type": ["string", "array"],
        "items": {
          "type": "string",
          "format": "email"
        }
      },
      "subject": {
        "type": "string"
      },
      "body": {
        "type": "string"
      },
      "html": {
        "type": "boolean"
      },
      "attachments": {
        "type": "array",
        "items": {
          "type": "object",
          "required": ["filename", "content"],
          "properties": {
            "filename": {
              "type": "string"
            },
            "content": {
              "type": "string"
            },
            "content_type": {
              "type": "string"
            }
          }
        }
      }
    }
  }`,
	"email.receive": `{
    "type": "object",
    "properties": {
      "filter": {
        "type": "object",
        "properties": {
          "from": {
            "type": "string"
          },
          "subject": {
            "type": "string"
          },
          "since": {
            "type": "string",
            "format": "date-time"
          },
          "unseen": {
            "type": "boolean"
          }
        }
      },
      "limit": {
        "type": "integer",
        "minimum": 1
      },
      "mark_as_read": {
        "type": "boolean"
      }
    }
  }`,
	"store": `{
    "type": "object",
    "required": ["operation"],
    "properties": {
      "operation": {
        "type": "string",
        "enum": ["get", "set", "delete", "list"]
      },
      "key": {
        "type": "string"
      },
      "value": {}
    }
  }`,
	"agent": `{
    "type": "object",
    "required": ["task"],
    "properties": {
      "task": {
        "type": "string"
      },
      "model": {
        "type": "string"
      },
      "tools": {
        "type": "array",
        "items": {
          "type": "object",
          "required": ["name", "description"],
          "properties": {
            "name": {
              "type": "string"
            },
            "description": {
              "type": "string"
            },
            "parameters": {
              "type": "object"
            }
          }
        }
      },
      "memory": {
        "type": "boolean"
      }
    }
  }`,
	"webhook": `{
    "type": "object",
    "required": ["url"],
    "properties": {
      "url": {
        "type": "string",
        "format": "uri"
      },
      "method": {
        "type": "string",
        "enum": ["GET", "POST", "PUT", "DELETE", "PATCH"]
      },
      "headers": {
        "type": "object",
        "additionalProperties": {
          "type": "string"
        }
      },
      "payload": {
        "type": ["string", "object"]
      }
    }
  }`,
}
