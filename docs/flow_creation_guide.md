# Flow Creation Guide

This guide provides detailed instructions and examples for creating flows in FlowRunner.

## Table of Contents

1. [Introduction](#introduction)
2. [Flow Structure](#flow-structure)
3. [Simple Flow Examples](#simple-flow-examples)
4. [Advanced Flow Examples](#advanced-flow-examples)
5. [Best Practices](#best-practices)

## Introduction

FlowRunner uses YAML to define workflows. A flow consists of a series of connected nodes, each performing a specific task. Nodes can be connected in various ways to create complex workflows.

## Flow Structure

A flow definition consists of two main sections:

1. **Metadata**: Contains information about the flow
2. **Nodes**: Defines the nodes in the flow and their connections

### Metadata Section

```yaml
metadata:
  name: "My Flow"
  description: "A description of what this flow does"
  version: "1.0.0"
```

### Nodes Section

```yaml
nodes:
  node_name:
    type: "node_type"
    params:
      # Parameters specific to the node type
    next:
      default: "next_node"
      error: "error_node"
    batch:
      # Batch processing configuration
    retry:
      # Retry configuration
    hooks:
      # JavaScript hooks
```

## Simple Flow Examples

### Hello World Flow

This simple flow makes an HTTP request and logs the response.

```yaml
metadata:
  name: "Hello World Flow"
  description: "A simple flow that makes an HTTP request"
  version: "1.0.0"

nodes:
  start:
    type: "http.request"
    params:
      url: "https://httpbin.org/get"
      method: "GET"
    next:
      default: "log"
      error: "error"
  
  log:
    type: "transform"
    params:
      script: |
        console.log("Response:", input);
        return input;
    next:
      default: "end"
  
  error:
    type: "transform"
    params:
      script: |
        console.log("Error:", input);
        return { error: input };
    next:
      default: "end"
  
  end:
    type: "transform"
    params:
      script: |
        console.log("Flow completed");
        return { status: "completed" };
```

### Data Transformation Flow

This flow fetches data from an API, transforms it, and sends it to another API.

```yaml
metadata:
  name: "Data Transformation Flow"
  description: "Fetches data, transforms it, and sends it to another API"
  version: "1.0.0"

nodes:
  fetch_data:
    type: "http.request"
    params:
      url: "https://api.example.com/data"
      method: "GET"
      headers:
        Authorization: "Bearer ${secrets.API_KEY}"
    next:
      default: "transform_data"
      error: "handle_error"
  
  transform_data:
    type: "transform"
    params:
      script: |
        // Extract the items from the response
        const items = input.body.items || [];
        
        // Transform each item
        const transformed = items.map(item => ({
          id: item.id,
          name: item.name.toUpperCase(),
          created: new Date(item.created_at).toISOString(),
          status: item.active ? "ACTIVE" : "INACTIVE"
        }));
        
        return { items: transformed };
    next:
      default: "send_data"
  
  send_data:
    type: "http.request"
    params:
      url: "https://api.example.com/submit"
      method: "POST"
      headers:
        Content-Type: "application/json"
        Authorization: "Bearer ${secrets.API_KEY}"
      body: "${input.items}"
    next:
      default: "success"
      error: "handle_error"
  
  success:
    type: "transform"
    params:
      script: |
        return { status: "success", message: "Data processed successfully" };
  
  handle_error:
    type: "transform"
    params:
      script: |
        console.log("Error:", input);
        return { status: "error", message: "An error occurred", error: input };
```

### Conditional Flow

This flow demonstrates conditional branching based on input data.

```yaml
metadata:
  name: "Conditional Flow"
  description: "Demonstrates conditional branching"
  version: "1.0.0"

nodes:
  start:
    type: "condition"
    params:
      conditions:
        - condition: "input.status == 'success'"
          action: "success"
        - condition: "input.status == 'error'"
          action: "error"
        - condition: "input.status == 'pending'"
          action: "pending"
      default_action: "unknown"
    next:
      success: "handle_success"
      error: "handle_error"
      pending: "handle_pending"
      unknown: "handle_unknown"
  
  handle_success:
    type: "transform"
    params:
      script: |
        console.log("Success case");
        return { result: "success" };
    next:
      default: "end"
  
  handle_error:
    type: "transform"
    params:
      script: |
        console.log("Error case");
        return { result: "error" };
    next:
      default: "end"
  
  handle_pending:
    type: "transform"
    params:
      script: |
        console.log("Pending case");
        return { result: "pending" };
    next:
      default: "end"
  
  handle_unknown:
    type: "transform"
    params:
      script: |
        console.log("Unknown case");
        return { result: "unknown" };
    next:
      default: "end"
  
  end:
    type: "transform"
    params:
      script: |
        console.log("Flow completed with result:", input.result);
        return input;
```

## Advanced Flow Examples

### LLM-Powered Content Generation

This flow uses an LLM to generate content based on a prompt.

```yaml
metadata:
  name: "LLM Content Generator"
  description: "Generates content using an LLM"
  version: "1.0.0"

nodes:
  get_topic:
    type: "http.request"
    params:
      url: "https://api.example.com/topics/random"
      method: "GET"
    next:
      default: "generate_content"
      error: "handle_error"
  
  generate_content:
    type: "llm"
    params:
      provider: "openai"
      api_key: "${secrets.OPENAI_API_KEY}"
      model: "gpt-3.5-turbo"
      template: "Write a 300-word blog post about {{.topic}}. The tone should be informative and engaging."
      variables:
        topic: "${input.body.topic}"
      temperature: 0.7
      max_tokens: 500
    next:
      default: "format_content"
      error: "handle_error"
  
  format_content:
    type: "transform"
    params:
      script: |
        // Extract the content from the LLM response
        const content = input.content;
        
        // Format the content
        const formatted = {
          title: `Blog Post: ${input.input.body.topic}`,
          content: content,
          word_count: content.split(/\s+/).length,
          created_at: new Date().toISOString()
        };
        
        return formatted;
    next:
      default: "store_content"
  
  store_content:
    type: "http.request"
    params:
      url: "https://api.example.com/content"
      method: "POST"
      headers:
        Content-Type: "application/json"
        Authorization: "Bearer ${secrets.API_KEY}"
      body: "${input}"
    next:
      default: "success"
      error: "handle_error"
  
  success:
    type: "transform"
    params:
      script: |
        return { status: "success", message: "Content generated and stored successfully" };
  
  handle_error:
    type: "transform"
    params:
      script: |
        console.log("Error:", input);
        return { status: "error", message: "An error occurred", error: input };
```

### Email Processing Workflow

This flow monitors an email inbox, processes incoming emails, and sends responses.

```yaml
metadata:
  name: "Email Processor"
  description: "Processes incoming emails and sends responses"
  version: "1.0.0"

nodes:
  check_emails:
    type: "email.receive"
    params:
      imap_host: "imap.gmail.com"
      imap_port: 993
      username: "${secrets.EMAIL_USERNAME}"
      password: "${secrets.EMAIL_PASSWORD}"
      folder: "INBOX"
      limit: 10
      unseen: true
      with_body: true
      mark_as_read: true
    next:
      default: "process_emails"
      error: "handle_error"
  
  process_emails:
    type: "transform"
    params:
      script: |
        // Check if we have any emails
        if (!Array.isArray(input) || input.length === 0) {
          console.log("No new emails");
          return { status: "no_emails" };
        }
        
        // Process each email
        const processed = input.map(email => ({
          id: email.messageId,
          from: email.from,
          subject: email.subject,
          body: email.body,
          category: categorizeEmail(email.subject, email.body)
        }));
        
        // Helper function to categorize emails
        function categorizeEmail(subject, body) {
          const subjectLower = subject.toLowerCase();
          const bodyLower = body.toLowerCase();
          
          if (subjectLower.includes("support") || bodyLower.includes("help")) {
            return "support";
          } else if (subjectLower.includes("order") || bodyLower.includes("purchase")) {
            return "order";
          } else {
            return "general";
          }
        }
        
        return { emails: processed };
    next:
      default: "categorize_emails"
  
  categorize_emails:
    type: "condition"
    params:
      conditions:
        - condition: "input.status == 'no_emails'"
          action: "no_emails"
      default_action: "has_emails"
    next:
      no_emails: "end"
      has_emails: "send_responses"
  
  send_responses:
    type: "transform"
    params:
      script: |
        // Prepare responses for each email
        const responses = input.emails.map(email => {
          let responseTemplate;
          
          switch (email.category) {
            case "support":
              responseTemplate = "Thank you for contacting support. Your request has been received and we will get back to you shortly. Reference: {{id}}";
              break;
            case "order":
              responseTemplate = "Thank you for your order inquiry. Our team will review your request and respond within 24 hours. Reference: {{id}}";
              break;
            default:
              responseTemplate = "Thank you for your message. We have received it and will respond soon. Reference: {{id}}";
          }
          
          return {
            to: email.from,
            subject: `Re: ${email.subject}`,
            body: responseTemplate.replace("{{id}}", email.id)
          };
        });
        
        return { responses };
    next:
      default: "send_emails"
  
  send_emails:
    type: "transform"
    params:
      script: |
        // This is a placeholder for sending multiple emails
        // In a real implementation, you would use a batch node or loop
        console.log(`Preparing to send ${input.responses.length} email responses`);
        return input;
    next:
      default: "send_email_batch"
  
  send_email_batch:
    type: "email.send"
    params:
      smtp_host: "smtp.gmail.com"
      smtp_port: 587
      username: "${secrets.EMAIL_USERNAME}"
      password: "${secrets.EMAIL_PASSWORD}"
      from: "${secrets.EMAIL_USERNAME}"
      to: "${input.responses[0].to}"
      subject: "${input.responses[0].subject}"
      body: "${input.responses[0].body}"
    next:
      default: "success"
      error: "handle_error"
  
  success:
    type: "transform"
    params:
      script: |
        return { status: "success", message: "Emails processed and responses sent" };
    next:
      default: "end"
  
  handle_error:
    type: "transform"
    params:
      script: |
        console.log("Error:", input);
        return { status: "error", message: "An error occurred", error: input };
    next:
      default: "end"
  
  end:
    type: "transform"
    params:
      script: |
        console.log("Flow completed");
        return { status: "completed" };
```

### Data Pipeline with Database Integration

This flow demonstrates a data pipeline that extracts data from an API, transforms it, and loads it into a database.

```yaml
metadata:
  name: "ETL Pipeline"
  description: "Extracts data from an API, transforms it, and loads it into a database"
  version: "1.0.0"

nodes:
  extract:
    type: "http.request"
    params:
      url: "https://api.example.com/data"
      method: "GET"
      headers:
        Authorization: "Bearer ${secrets.API_KEY}"
    next:
      default: "transform"
      error: "handle_error"
  
  transform:
    type: "transform"
    params:
      script: |
        // Extract the items from the response
        const items = input.body.items || [];
        
        // Transform each item
        const transformed = items.map(item => ({
          id: item.id,
          name: item.name,
          email: item.email,
          created_at: new Date(item.created_at).toISOString(),
          status: item.active ? "active" : "inactive"
        }));
        
        return { items: transformed };
    next:
      default: "load"
  
  load:
    type: "postgres"
    params:
      operation: "batch_insert"
      table: "users"
      items: "${input.items}"
      columns: ["id", "name", "email", "created_at", "status"]
      on_conflict: "id"
      on_conflict_action: "update"
    next:
      default: "success"
      error: "handle_error"
  
  success:
    type: "transform"
    params:
      script: |
        return { 
          status: "success", 
          message: "Data pipeline executed successfully",
          items_processed: input.items.length
        };
  
  handle_error:
    type: "transform"
    params:
      script: |
        console.log("Error:", input);
        return { status: "error", message: "An error occurred", error: input };
```

## Best Practices

### Flow Design

1. **Start with a clear goal**: Define what the flow should accomplish before starting to write YAML.
2. **Keep flows focused**: Each flow should have a single responsibility.
3. **Use descriptive node names**: Names like `fetch_user_data` are better than `http1`.
4. **Add comments**: Use YAML comments (`# Comment`) to explain complex logic.
5. **Handle errors**: Always include error handling for each node.

### Node Organization

1. **Group related nodes**: Keep related functionality together.
2. **Use a consistent naming convention**: For example, use verb-noun format (`fetch_data`, `transform_data`, `store_result`).
3. **Limit flow complexity**: If a flow becomes too complex, consider splitting it into multiple flows.

### JavaScript Best Practices

1. **Keep scripts simple**: Complex logic should be moved to external services.
2. **Use proper error handling**: Wrap code in try-catch blocks for better error reporting.
3. **Validate inputs**: Check that inputs match expected formats before processing.
4. **Limit script size**: Large scripts are harder to maintain and debug.

### Security Considerations

1. **Use secrets for sensitive data**: Never hardcode API keys, passwords, or other sensitive information.
2. **Validate external inputs**: Always validate and sanitize data from external sources.
3. **Limit permissions**: Use the principle of least privilege for database operations and API calls.
4. **Audit flows**: Regularly review flows for security issues.

### Performance Optimization

1. **Use batch processing**: For operations on multiple items.
2. **Implement caching**: For frequently accessed data.
3. **Limit payload sizes**: Large payloads can slow down execution.
4. **Use appropriate timeouts**: Set reasonable timeouts for external services.

### Testing and Debugging

1. **Start with simple flows**: Test basic functionality before adding complexity.
2. **Use logging nodes**: Add transform nodes with `console.log()` statements to debug issues.
3. **Test with sample data**: Create sample inputs for testing.
4. **Monitor executions**: Use the WebSocket API to monitor flow execution in real-time.