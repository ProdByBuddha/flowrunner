# Flowrunner

Flowrunner is a lightweight, YAML-driven orchestration service built on top of Flowlib that enables users to define, manage, and trigger workflows without writing Go code. The system provides a RESTful HTTP API for flow management, execution capabilities, multi-tenant account support with secrets management, and extensibility through plugins and inline scripting.

## Features

- **YAML-based workflow definitions** with expression support
- **Multiple persistence options** (in-memory, DynamoDB, PostgreSQL)
- **CLI and HTTP API interfaces** for flow management and execution
- **Core node implementations**:
  - HTTP requests
  - Email (SMTP/IMAP)
  - LLM integration (OpenAI, Anthropic)
  - In-memory store
  - AI agent
  - Webhooks
- **Real-time monitoring** via WebSockets
- **Secure credential storage** with encryption
- **Expression evaluation** in YAML definitions
- **JavaScript scripting** for data transformations and logic

## Getting Started

### Prerequisites

- Go 1.18 or higher
- Access to desired storage backend (in-memory, DynamoDB, PostgreSQL)
- API keys for external services (OpenAI, Anthropic, etc.)

### Installation

1. Clone the repository:
   ```bash
   git clone https://github.com/tcmartin/flowrunner.git
   cd flowrunner
   ```

2. Install dependencies:
   ```bash
   go mod download
   ```

3. Build the project:
   ```bash
   go build -o flowrunner cmd/flowrunner/main.go
   ```

### Configuration

Create a `.env` file in the project root with the following variables:

```
# LLM API Keys
OPENAI_API_KEY=your_openai_api_key
ANTHROPIC_API_KEY=your_anthropic_api_key

# Email Configuration (optional)
GMAIL_USERNAME=your_gmail_username
GMAIL_PASSWORD=your_app_password
EMAIL_RECIPIENT=recipient_email

# Storage Configuration
STORAGE_TYPE=memory  # Options: memory, dynamodb, postgres
```

For DynamoDB or PostgreSQL, additional configuration variables are required.

## Usage

### Running the Server

```bash
./flowrunner server --port 8080
```

### Using the CLI

```bash
# Create a new flow
./flowrunner create myflow --file flow.yaml

# List all flows
./flowrunner list

# Run a flow
./flowrunner run flow-id --input input.json

# View execution logs
./flowrunner logs execution-id
```

### API Endpoints

- **Flow Management**:
  - `GET /api/v1/flows` - List flows
  - `POST /api/v1/flows` - Create flow
  - `GET /api/v1/flows/{id}` - Get flow
  - `PUT /api/v1/flows/{id}` - Update flow
  - `DELETE /api/v1/flows/{id}` - Delete flow

- **Flow Execution**:
  - `POST /api/v1/flows/{id}/run` - Run flow
  - `GET /api/v1/executions/{id}` - Get execution status
  - `GET /api/v1/executions/{id}/logs` - Get execution logs
  - `DELETE /api/v1/executions/{id}` - Cancel execution

- **Account Management**:
  - `POST /api/v1/accounts` - Create account
  - `GET /api/v1/accounts/{id}/secrets` - List secrets
  - `POST /api/v1/accounts/{id}/secrets` - Create secret
  - `DELETE /api/v1/accounts/{id}/secrets/{key}` - Delete secret

### WebSocket API

Connect to `/ws/executions/{id}` to receive real-time updates for a flow execution.

## YAML Flow Definition

```yaml
metadata:
  name: "Example Flow"
  description: "A simple example flow"
  version: "1.0.0"

nodes:
  start:
    type: "http.request"
    params:
      url: "https://api.example.com/data"
      method: "GET"
    next:
      default: "process"
      error: "handleError"
    hooks:
      prep: |
        // JavaScript prep hook
        context.headers = { "Authorization": "Bearer " + secrets.API_KEY };
        return context;
  
  process:
    type: "transform"
    params:
      mapping:
        result: "$.data.items"
    next:
      default: "end"
    hooks:
      exec: |
        // JavaScript exec hook
        return input.map(item => ({ id: item.id, name: item.name }));
  
  handleError:
    type: "notification"
    params:
      channel: "slack"
      message: "Flow failed: ${error.message}"
    next:
      default: "end"
  
  end:
    type: "webhook"
    params:
      url: "https://webhook.example.com/callback"
      method: "POST"
```

## Core Node Types

### HTTP Request Node

```yaml
http_node:
  type: "http.request"
  params:
    url: "https://api.example.com/data"
    method: "POST"
    headers:
      Content-Type: "application/json"
    body:
      key: "value"
    timeout: "30s"
```

### LLM Node

```yaml
llm_node:
  type: "llm"
  params:
    provider: "openai"
    model: "gpt-3.5-turbo"
    messages:
      - role: "system"
        content: "You are a helpful assistant."
      - role: "user"
        content: "What is the capital of France?"
    temperature: 0.7
    max_tokens: 100
```

With template support:

```yaml
llm_template_node:
  type: "llm"
  params:
    provider: "openai"
    model: "gpt-3.5-turbo"
    template: "Hello {{.name}}! Can you tell me about {{.topic}}?"
    variables:
      name: "User"
      topic: "quantum computing"
    temperature: 0.7
```

With structured output:

```yaml
llm_structured_node:
  type: "llm"
  params:
    provider: "openai"
    model: "gpt-3.5-turbo"
    messages:
      - role: "system"
        content: "You are a helpful assistant that responds in YAML format."
      - role: "user"
        content: "Give me information about Tokyo in YAML format."
    parse_structured: true
```

### Email Nodes

SMTP (Send):

```yaml
email_send_node:
  type: "email.send"
  params:
    smtp_host: "smtp.gmail.com"
    smtp_port: 587
    username: "your_email@gmail.com"
    password: "your_password"
    from: "your_email@gmail.com"
    to: "recipient@example.com"
    subject: "Hello from Flowrunner"
    body: "This is a test email."
    html: "<h1>Hello</h1><p>This is a test email.</p>"
```

IMAP (Receive):

```yaml
email_receive_node:
  type: "email.receive"
  params:
    imap_host: "imap.gmail.com"
    imap_port: 993
    username: "your_email@gmail.com"
    password: "your_password"
    folder: "INBOX"
    limit: 10
    unseen: true
    with_body: true
```

### Store Node

```yaml
store_node:
  type: "store"
  params:
    operation: "set"
    key: "user_id"
    value: "12345"
```

### Webhook Node

```yaml
webhook_node:
  type: "webhook"
  params:
    url: "https://webhook.example.com/callback"
    method: "POST"
    headers:
      Content-Type: "application/json"
    body:
      result: "${result}"
```

## Testing

Run the test suite:

```bash
go test ./...
```

Test specific node types:

```bash
go run cmd/test_nodes/main.go openai anthropic template structured email
```

## License

[MIT License](LICENSE)

## Contributing

1. Fork the repository
2. Create your feature branch (`git checkout -b feature/amazing-feature`)
3. Commit your changes (`git commit -m 'Add some amazing feature'`)
4. Push to the branch (`git push origin feature/amazing-feature`)
5. Open a Pull Request