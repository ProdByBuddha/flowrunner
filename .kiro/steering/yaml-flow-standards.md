# FlowRunner YAML Flow Standards

## YAML Flow Structure

### Required Sections
Every flow YAML must contain these top-level sections:

```yaml
metadata:
  name: "Flow Name"
  description: "Flow description"
  version: "1.0.0"
  
nodes:
  # Node definitions
  
# Optional sections
variables:
  # Global variables
  
secrets:
  # Required secrets list
```

### Metadata Standards
```yaml
metadata:
  name: "descriptive-flow-name"           # kebab-case, descriptive
  description: "Clear description of what this flow does"
  version: "1.0.0"                       # Semantic versioning
  author: "author@example.com"           # Optional
  tags: ["category", "purpose"]          # Optional classification
  created: "2025-01-15T10:00:00Z"       # ISO 8601 timestamp
  updated: "2025-01-15T10:00:00Z"       # ISO 8601 timestamp
```

## Node Definition Standards

### Basic Node Structure
```yaml
nodes:
  node_id:                              # snake_case identifier
    type: "node.type"                   # dot notation for node types
    description: "What this node does"   # Optional but recommended
    params:                             # Node-specific parameters
      key: "value"
    next:                               # Flow control
      default: "next_node_id"
      error: "error_handler_node"
    hooks:                              # Optional JavaScript hooks
      prep: |
        // Preparation logic
      exec: |
        // Execution logic
      post: |
        // Post-processing logic
    timeout: "30s"                      # Optional timeout
    retry:                              # Optional retry configuration
      attempts: 3
      delay: "1s"
```

### Node Naming Conventions
- Use `snake_case` for node IDs
- Use descriptive names that indicate purpose
- Common patterns:
  - `start` - Entry point node
  - `end` - Terminal node
  - `fetch_data` - Data retrieval nodes
  - `process_results` - Data processing nodes
  - `send_notification` - Notification nodes
  - `handle_error` - Error handling nodes

## Node Type Standards

### HTTP Request Nodes
```yaml
api_call:
  type: "http.request"
  description: "Fetch data from external API"
  params:
    url: "https://api.example.com/data"
    method: "GET"                       # GET, POST, PUT, DELETE, PATCH
    headers:
      Authorization: "Bearer ${secrets.API_TOKEN}"
      Content-Type: "application/json"
    body:                               # For POST/PUT requests
      key: "value"
    timeout: "30s"
    follow_redirects: true
  next:
    default: "process_response"
    error: "handle_api_error"
```

### LLM Integration Nodes
```yaml
ai_analysis:
  type: "llm"
  description: "Analyze data using AI"
  params:
    provider: "openai"                  # openai, anthropic
    model: "gpt-4"
    messages:
      - role: "system"
        content: "You are a data analyst."
      - role: "user"
        content: "Analyze this data: ${input.data}"
    temperature: 0.7
    max_tokens: 1000
    parse_structured: true              # Parse YAML/JSON responses
  next:
    default: "use_analysis"
    error: "handle_ai_error"
```

### Email Nodes
```yaml
send_email:
  type: "email.send"
  description: "Send notification email"
  params:
    smtp_host: "smtp.gmail.com"
    smtp_port: 587
    username: "${secrets.EMAIL_USERNAME}"
    password: "${secrets.EMAIL_PASSWORD}"
    from: "noreply@example.com"
    to: "${variables.recipient_email}"
    subject: "Flow Execution Complete"
    body: "Flow ${metadata.name} completed successfully"
    html: "<h1>Success</h1><p>Flow completed</p>"
  next:
    default: "end"
    error: "log_email_error"
```

### Store Operations
```yaml
cache_result:
  type: "store"
  description: "Cache processing result"
  params:
    operation: "set"                    # set, get, delete
    key: "flow_result_${execution.id}"
    value: "${input.processed_data}"
    ttl: "3600s"                       # Optional TTL
  next:
    default: "next_step"
```

### Webhook Nodes
```yaml
notify_webhook:
  type: "webhook"
  description: "Send results to webhook"
  params:
    url: "https://webhook.example.com/callback"
    method: "POST"
    headers:
      Content-Type: "application/json"
      X-Flow-ID: "${execution.flow_id}"
    body:
      status: "completed"
      result: "${input.final_result}"
      timestamp: "${now()}"
  next:
    default: "end"
```

## Expression and Variable Standards

### Variable References
```yaml
# Reference input data
url: "${input.api_endpoint}"

# Reference secrets
token: "${secrets.API_TOKEN}"

# Reference metadata
flow_name: "${metadata.name}"

# Reference execution context
exec_id: "${execution.id}"
account: "${execution.account_id}"

# Reference previous node output
data: "${nodes.fetch_data.output.results}"
```

### Built-in Functions
```yaml
# Date/time functions
timestamp: "${now()}"
formatted_date: "${format_date(now(), 'YYYY-MM-DD')}"

# String functions
upper_name: "${upper(input.name)}"
trimmed: "${trim(input.text)}"

# JSON path expressions
first_item: "${input.items[0]}"
filtered: "${input.items[?(@.status == 'active')]}"
```

### Global Variables
```yaml
variables:
  api_base_url: "https://api.example.com"
  default_timeout: "30s"
  max_retries: 3
  notification_email: "admin@example.com"

nodes:
  api_call:
    type: "http.request"
    params:
      url: "${variables.api_base_url}/data"
      timeout: "${variables.default_timeout}"
```

## Flow Control Patterns

### Conditional Routing
```yaml
check_status:
  type: "condition"
  params:
    condition: "${input.status == 'success'}"
  next:
    true: "handle_success"
    false: "handle_failure"
    default: "handle_unknown"
```

### Parallel Execution
```yaml
parallel_tasks:
  type: "parallel"
  description: "Execute multiple tasks concurrently"
  params:
    branches:
      - nodes: ["fetch_user_data", "process_user"]
      - nodes: ["fetch_order_data", "process_order"]
    wait_for: "all"                     # all, any, first
  next:
    default: "merge_results"
    error: "handle_parallel_error"
```

### Loop Constructs
```yaml
process_items:
  type: "loop"
  description: "Process each item in the list"
  params:
    items: "${input.item_list}"
    max_iterations: 100
    nodes:
      - "validate_item"
      - "transform_item"
      - "store_item"
  next:
    default: "finalize_processing"
    error: "handle_loop_error"
```

## Error Handling Standards

### Error Node Pattern
```yaml
handle_api_error:
  type: "error_handler"
  description: "Handle API call failures"
  params:
    log_level: "error"
    message: "API call failed: ${error.message}"
    notify: true
    notification:
      type: "email"
      recipient: "${variables.admin_email}"
  next:
    default: "cleanup"
```

### Retry Configuration
```yaml
unreliable_api:
  type: "http.request"
  params:
    url: "https://unreliable-api.com/data"
  retry:
    attempts: 3
    delay: "2s"
    backoff: "exponential"              # linear, exponential
    max_delay: "30s"
    on_error: ["timeout", "5xx"]        # Retry conditions
  next:
    default: "process_data"
    error: "handle_final_failure"
```

## JavaScript Hook Standards

### Hook Types and Usage
```yaml
data_processor:
  type: "transform"
  hooks:
    prep: |
      // Prepare context before execution
      context.start_time = Date.now();
      context.headers = {
        'X-Request-ID': generateUUID(),
        'Authorization': 'Bearer ' + secrets.API_TOKEN
      };
      return context;
    
    exec: |
      // Main processing logic
      const processed = input.items.map(item => ({
        id: item.id,
        name: item.name.toUpperCase(),
        processed_at: new Date().toISOString()
      }));
      return { processed_items: processed };
    
    post: |
      // Post-processing cleanup
      console.log(`Processed ${output.processed_items.length} items`);
      return output;
```

### JavaScript Best Practices
- Keep hooks concise and focused
- Use proper error handling with try/catch
- Return appropriate data structures
- Avoid side effects when possible
- Use meaningful variable names
- Add comments for complex logic

## Validation and Schema

### Required Validations
- All node IDs must be unique within a flow
- All referenced node IDs in `next` must exist
- All required parameters for node types must be present
- Variable and secret references must be valid
- YAML syntax must be valid

### Schema Validation Example
```yaml
# This will fail validation - missing required params
invalid_node:
  type: "http.request"
  # Missing required 'url' parameter
  params:
    method: "GET"

# This will pass validation
valid_node:
  type: "http.request"
  params:
    url: "https://api.example.com"
    method: "GET"
```

## Documentation Standards

### Flow Documentation
```yaml
metadata:
  name: "user-onboarding-flow"
  description: |
    Comprehensive user onboarding flow that:
    1. Validates user data
    2. Creates user account
    3. Sends welcome email
    4. Triggers downstream systems
  version: "2.1.0"
  documentation:
    requirements:
      - "Valid user email address"
      - "User consent for communications"
    outputs:
      - "User ID in system"
      - "Onboarding completion status"
    dependencies:
      - "User management API"
      - "Email service"
    examples:
      input: |
        {
          "email": "user@example.com",
          "name": "John Doe",
          "consent": true
        }
      output: |
        {
          "user_id": "12345",
          "status": "completed",
          "welcome_sent": true
        }
```

### Node Documentation
```yaml
validate_user:
  type: "validation"
  description: |
    Validates user input data including:
    - Email format validation
    - Required field presence
    - Business rule compliance
  params:
    schema:
      type: "object"
      required: ["email", "name"]
      properties:
        email:
          type: "string"
          format: "email"
        name:
          type: "string"
          minLength: 2
```