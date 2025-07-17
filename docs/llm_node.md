# LLM Node Documentation

The LLM (Large Language Model) node in Flowrunner provides a unified interface for interacting with various language model providers. It supports multiple providers, template-based prompts, and structured output parsing.

## Supported Providers

- **OpenAI** - GPT models (gpt-3.5-turbo, gpt-4, etc.)
- **Anthropic** - Claude models (claude-3-haiku, claude-3-opus, etc.)
- **Generic** - Custom API endpoints that follow a similar interface

## Basic Usage

### Simple Message-Based Prompt

```yaml
llm_node:
  type: "llm"
  params:
    provider: "openai"
    api_key: "${secrets.OPENAI_API_KEY}"
    model: "gpt-3.5-turbo"
    messages:
      - role: "system"
        content: "You are a helpful assistant. Keep your answers brief."
      - role: "user"
        content: "What is the capital of France?"
    temperature: 0.7
    max_tokens: 100
```

### Single Prompt

```yaml
llm_node:
  type: "llm"
  params:
    provider: "openai"
    api_key: "${secrets.OPENAI_API_KEY}"
    model: "gpt-3.5-turbo"
    prompt: "What is the capital of France?"
    temperature: 0.7
    max_tokens: 100
```

## Template Support

### Single Template

```yaml
llm_template_node:
  type: "llm"
  params:
    provider: "openai"
    api_key: "${secrets.OPENAI_API_KEY}"
    model: "gpt-3.5-turbo"
    template: "Hello {{.name}}! Can you tell me about the capital of {{.country}}?"
    variables:
      name: "User"
      country: "Japan"
    temperature: 0.7
    max_tokens: 100
```

### Multiple Templates

```yaml
llm_templates_node:
  type: "llm"
  params:
    provider: "openai"
    api_key: "${secrets.OPENAI_API_KEY}"
    model: "gpt-3.5-turbo"
    templates:
      - role: "system"
        template: "You are a helpful assistant specialized in {{.topic}}."
      - role: "user"
        template: "Tell me about {{.subject}} in {{.country}}."
    context:
      topic: "geography"
      subject: "the capital city"
      country: "France"
    temperature: 0.7
    max_tokens: 100
```

## Structured Output

```yaml
llm_structured_node:
  type: "llm"
  params:
    provider: "openai"
    api_key: "${secrets.OPENAI_API_KEY}"
    model: "gpt-3.5-turbo"
    messages:
      - role: "system"
        content: "You are a helpful assistant that responds in YAML format."
      - role: "user"
        content: "Give me information about Tokyo in YAML format with the following fields: name, country, population, landmarks (as a list)."
    parse_structured: true
    temperature: 0.7
    max_tokens: 200
```

## Function Calling

```yaml
llm_function_node:
  type: "llm"
  params:
    provider: "openai"
    api_key: "${secrets.OPENAI_API_KEY}"
    model: "gpt-3.5-turbo"
    messages:
      - role: "user"
        content: "What's the weather like in San Francisco?"
    functions:
      - name: "get_weather"
        description: "Get the current weather in a given location"
        parameters:
          type: "object"
          properties:
            location:
              type: "string"
              description: "The city and state, e.g. San Francisco, CA"
            unit:
              type: "string"
              enum: ["celsius", "fahrenheit"]
          required: ["location"]
    temperature: 0.7
```

## Tool Use

```yaml
llm_tool_node:
  type: "llm"
  params:
    provider: "openai"
    api_key: "${secrets.OPENAI_API_KEY}"
    model: "gpt-4"
    messages:
      - role: "user"
        content: "What's the weather like in San Francisco and Tokyo?"
    tools:
      - type: "function"
        function:
          name: "get_weather"
          description: "Get the current weather in a given location"
          parameters:
            type: "object"
            properties:
              location:
                type: "string"
                description: "The city and state, e.g. San Francisco, CA"
              unit:
                type: "string"
                enum: ["celsius", "fahrenheit"]
            required: ["location"]
    temperature: 0.7
```

## Response Format

```yaml
llm_format_node:
  type: "llm"
  params:
    provider: "openai"
    api_key: "${secrets.OPENAI_API_KEY}"
    model: "gpt-3.5-turbo"
    messages:
      - role: "user"
        content: "Generate a JSON object with information about Paris."
    response_format:
      type: "json_object"
    temperature: 0.7
```

## Parameters

| Parameter | Type | Required | Description |
|-----------|------|----------|-------------|
| `provider` | string | Yes | LLM provider: "openai", "anthropic", or "generic" |
| `api_key` | string | Yes | API key for the provider |
| `model` | string | Yes | Model name (e.g., "gpt-3.5-turbo", "claude-3-haiku") |
| `messages` | array | No* | Array of message objects with "role" and "content" |
| `prompt` | string | No* | Simple prompt text (alternative to messages) |
| `template` | string | No* | Template string with variable placeholders |
| `variables` | object | No | Variables for template rendering |
| `templates` | array | No* | Array of template objects with "role" and "template" |
| `context` | object | No | Shared context for multiple templates |
| `temperature` | number | No | Sampling temperature (default: 0.7) |
| `max_tokens` | number | No | Maximum tokens to generate |
| `stop` | array | No | Array of stop sequences |
| `functions` | array | No | Function definitions for function calling |
| `tools` | array | No | Tool definitions for tool use |
| `parse_structured` | boolean | No | Parse response as structured YAML |
| `response_format` | object | No | Response format specification |
| `options` | object | No | Additional provider-specific options |

\* At least one of `messages`, `prompt`, `template`, or `templates` is required.

## Output

The LLM node returns a result object with the following fields:

```json
{
  "id": "response-id",
  "model": "gpt-3.5-turbo",
  "choices": [...],
  "usage": {
    "prompt_tokens": 50,
    "completion_tokens": 30,
    "total_tokens": 80
  },
  "content": "The capital of France is Paris.",
  "finish_reason": "stop",
  "raw_response": {...},
  "structured_output": {...}  // Only present if parse_structured is true
}
```

## Error Handling

The LLM node handles various error scenarios:

- API authentication errors
- Rate limiting and quota errors
- Invalid model or parameter errors
- Timeout errors
- Parsing errors for structured output

Errors are propagated through the flow execution and can be handled by error paths in the flow definition.

## Examples

### Question Answering

```yaml
qa_node:
  type: "llm"
  params:
    provider: "openai"
    api_key: "${secrets.OPENAI_API_KEY}"
    model: "gpt-3.5-turbo"
    messages:
      - role: "system"
        content: "You are a helpful assistant specialized in answering questions about geography."
      - role: "user"
        content: "What are the five largest cities in Japan by population?"
    temperature: 0.7
```

### Data Extraction

```yaml
extraction_node:
  type: "llm"
  params:
    provider: "openai"
    api_key: "${secrets.OPENAI_API_KEY}"
    model: "gpt-3.5-turbo"
    messages:
      - role: "system"
        content: "Extract structured information from the text and return it as YAML."
      - role: "user"
        content: "John Smith is 42 years old and lives in New York. He works as a software engineer at Tech Corp and has two children named Emma and Michael."
    parse_structured: true
    temperature: 0.2
```

### Content Generation

```yaml
generation_node:
  type: "llm"
  params:
    provider: "anthropic"
    api_key: "${secrets.ANTHROPIC_API_KEY}"
    model: "claude-3-haiku-20240307"
    messages:
      - role: "system"
        content: "You are a creative writer specialized in short stories."
      - role: "user"
        content: "Write a 100-word story about a robot discovering emotions."
    temperature: 0.9
    max_tokens: 300
```