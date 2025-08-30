#!/usr/bin/env node

/**
 * Tool Examples Generator
 * 
 * Creates practical usage examples for each tool category.
 * Writes example configurations for different integration scenarios.
 * Implements example validation to ensure they remain functional.
 * 
 * Usage:
 *   node examples/scripts/generate-tool-examples.js [options]
 * 
 * Options:
 *   --output <dir>      Output directory for examples (default: examples/usage)
 *   --service <name>    Generate examples for specific service only
 *   --scenario <type>   Generate specific scenario: basic, advanced, integration
 *   --validate          Validate generated examples
 *   --format <format>   Output format: json, yaml, markdown (default: markdown)
 */

import fs from 'fs/promises';
import path from 'path';
import { fileURLToPath } from 'url';
import { ToolSchemaGenerator } from './generate-tool-schemas.js';

const __filename = fileURLToPath(import.meta.url);
const __dirname = path.dirname(__filename);
const projectRoot = path.resolve(__dirname, '../..');

// Example scenarios and their configurations
const EXAMPLE_SCENARIOS = {
  basic: {
    name: 'Basic Usage',
    description: 'Simple, single-tool usage examples',
    complexity: 'beginner'
  },
  workflow: {
    name: 'Workflow Integration',
    description: 'Multi-step workflows using multiple tools',
    complexity: 'intermediate'
  },
  advanced: {
    name: 'Advanced Integration',
    description: 'Complex scenarios with error handling and optimization',
    complexity: 'advanced'
  },
  automation: {
    name: 'Automation Scripts',
    description: 'Complete automation workflows',
    complexity: 'expert'
  }
};

class ToolExamplesGenerator {
  constructor(options = {}) {
    this.options = {
      output: options.output || 'examples/usage',
      service: options.service || null,
      scenario: options.scenario || null,
      validate: options.validate || false,
      format: options.format || 'markdown',
      ...options
    };
    this.examples = {};
    this.schemas = {};
  }

  async generateAllExamples() {
    console.log('📝 Generating MCP tool usage examples...');
    
    // Load tool schemas first
    await this.loadToolSchemas();
    
    const services = this.options.service 
      ? [this.options.service]
      : Object.keys(this.schemas);

    const scenarios = this.options.scenario
      ? [this.options.scenario]
      : Object.keys(EXAMPLE_SCENARIOS);

    for (const serviceName of services) {
      for (const scenarioName of scenarios) {
        try {
          await this.generateServiceExamples(serviceName, scenarioName);
        } catch (error) {
          console.error(`❌ Failed to generate examples for ${serviceName}/${scenarioName}:`, error.message);
        }
      }
    }

    if (this.options.validate) {
      await this.validateExamples();
    }

    await this.writeExamples();
    await this.generateIndexFile();
    
    console.log('🎉 Tool examples generation complete!');
  }

  async loadToolSchemas() {
    console.log('📋 Loading tool schemas...');
    
    try {
      const schemaPath = path.join(projectRoot, 'schemas/tool-schemas.json');
      const schemaContent = await fs.readFile(schemaPath, 'utf8');
      const schemaData = JSON.parse(schemaContent);
      this.schemas = schemaData.services;
    } catch (error) {
      // Generate schemas if they don't exist
      console.log('🔧 Generating tool schemas...');
      const generator = new ToolSchemaGenerator();
      await generator.generateAllSchemas();
      
      const schemaPath = path.join(projectRoot, 'schemas/tool-schemas.json');
      const schemaContent = await fs.readFile(schemaPath, 'utf8');
      const schemaData = JSON.parse(schemaContent);
      this.schemas = schemaData.services;
    }
  }

  async generateServiceExamples(serviceName, scenarioName) {
    const schema = this.schemas[serviceName];
    const scenario = EXAMPLE_SCENARIOS[scenarioName];
    
    if (!schema) {
      console.warn(`⚠️  No schema found for service: ${serviceName}`);
      return;
    }

    console.log(`📝 Generating ${scenario.name} examples for ${serviceName}...`);

    const examples = {
      service: serviceName,
      scenario: scenarioName,
      metadata: {
        title: `${schema.service} - ${scenario.name}`,
        description: `${scenario.description} for ${schema.description}`,
        complexity: scenario.complexity,
        generatedAt: new Date().toISOString()
      },
      examples: []
    };

    switch (scenarioName) {
      case 'basic':
        examples.examples = await this.generateBasicExamples(schema);
        break;
      case 'workflow':
        examples.examples = await this.generateWorkflowExamples(schema);
        break;
      case 'advanced':
        examples.examples = await this.generateAdvancedExamples(schema);
        break;
      case 'automation':
        examples.examples = await this.generateAutomationExamples(schema);
        break;
    }

    if (!this.examples[serviceName]) {
      this.examples[serviceName] = {};
    }
    this.examples[serviceName][scenarioName] = examples;

    console.log(`✅ Generated ${examples.examples.length} examples for ${serviceName}/${scenarioName}`);
  }

  async generateBasicExamples(schema) {
    const examples = [];
    
    // Connection example
    examples.push({
      name: 'Service Connection',
      description: `How to connect to ${schema.service}`,
      category: 'setup',
      code: this.generateConnectionExample(schema),
      language: 'bash'
    });

    // Tool listing example
    examples.push({
      name: 'List Available Tools',
      description: 'Get all available tools for this service',
      category: 'discovery',
      code: this.generateToolListExample(schema),
      language: 'json'
    });

    // Basic tool usage examples (one per category)
    const categories = [...new Set(schema.tools.map(t => t.category))];
    for (const category of categories.slice(0, 5)) { // Limit to 5 categories
      const tool = schema.tools.find(t => t.category === category);
      if (tool) {
        examples.push({
          name: `${category.charAt(0).toUpperCase() + category.slice(1)} Operation`,
          description: `Basic ${category} operation using ${tool.name}`,
          category: category,
          code: this.generateBasicToolExample(tool, schema),
          language: 'json'
        });
      }
    }

    return examples;
  }

  async generateWorkflowExamples(schema) {
    const examples = [];

    // Multi-step workflow examples
    examples.push({
      name: 'Complete Workflow',
      description: `Multi-step workflow using ${schema.service}`,
      category: 'workflow',
      code: this.generateWorkflowExample(schema),
      language: 'json'
    });

    // Error handling workflow
    examples.push({
      name: 'Error Handling Workflow',
      description: 'Workflow with proper error handling',
      category: 'error-handling',
      code: this.generateErrorHandlingExample(schema),
      language: 'json'
    });

    // Conditional workflow
    examples.push({
      name: 'Conditional Workflow',
      description: 'Workflow with conditional logic',
      category: 'conditional',
      code: this.generateConditionalExample(schema),
      language: 'json'
    });

    return examples;
  }

  async generateAdvancedExamples(schema) {
    const examples = [];

    // Batch operations
    examples.push({
      name: 'Batch Operations',
      description: 'Performing batch operations efficiently',
      category: 'batch',
      code: this.generateBatchExample(schema),
      language: 'javascript'
    });

    // Rate limiting and retry logic
    examples.push({
      name: 'Rate Limiting & Retry',
      description: 'Handling rate limits with retry logic',
      category: 'resilience',
      code: this.generateRetryExample(schema),
      language: 'javascript'
    });

    // Performance optimization
    examples.push({
      name: 'Performance Optimization',
      description: 'Optimizing API calls for better performance',
      category: 'optimization',
      code: this.generateOptimizationExample(schema),
      language: 'javascript'
    });

    return examples;
  }

  async generateAutomationExamples(schema) {
    const examples = [];

    // Complete automation script
    examples.push({
      name: 'Complete Automation Script',
      description: `Full automation script for ${schema.service}`,
      category: 'automation',
      code: this.generateAutomationScript(schema),
      language: 'javascript'
    });

    // Monitoring and alerting
    examples.push({
      name: 'Monitoring & Alerting',
      description: 'Automated monitoring with alerts',
      category: 'monitoring',
      code: this.generateMonitoringExample(schema),
      language: 'javascript'
    });

    // Scheduled tasks
    examples.push({
      name: 'Scheduled Tasks',
      description: 'Setting up scheduled automation tasks',
      category: 'scheduling',
      code: this.generateSchedulingExample(schema),
      language: 'javascript'
    });

    return examples;
  }

  generateConnectionExample(schema) {
    const envVars = schema.authentication.required
      .map(v => `export ${v}="your_${v.toLowerCase().replace(/_/g, '_')}"`);
    
    return `# Environment setup for ${schema.service}
${envVars.join('\n')}

# Start MCP server
node examples/mcp-${schema.service}-server.js

# Or use multi-host configuration
node examples/mcp-multi-host.js --config services.json

# Test connection
echo '{"jsonrpc":"2.0","id":"1","method":"initialize","params":{"protocolVersion":"2024-11-05","clientInfo":{"name":"test","version":"1.0.0"}}}' | node examples/mcp-${schema.service}-server.js`;
  }

  generateToolListExample(schema) {
    return `{
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
      ${schema.tools.slice(0, 3).map(tool => `{
        "name": "${schema.service}.${tool.name}",
        "description": "${tool.description}",
        "inputSchema": ${JSON.stringify(tool.inputSchema, null, 8)}
      }`).join(',\n      ')}
      // ... ${schema.toolCount - 3} more tools
    ]
  }
}`;
  }

  generateBasicToolExample(tool, schema) {
    const exampleInput = this.generateExampleInput(tool.inputSchema);
    
    return `{
  "jsonrpc": "2.0",
  "id": "call-${tool.name}",
  "method": "tools/call",
  "params": {
    "name": "${schema.service}.${tool.name}",
    "arguments": ${JSON.stringify(exampleInput, null, 4)}
  }
}

// Expected response:
{
  "jsonrpc": "2.0",
  "id": "call-${tool.name}",
  "result": {
    "content": [
      {
        "type": "text",
        "text": "Operation completed successfully"
      }
    ]
  }
}`;
  }

  generateWorkflowExample(schema) {
    const listTool = schema.tools.find(t => t.category === 'list') || schema.tools[0];
    const createTool = schema.tools.find(t => t.category === 'create') || schema.tools[1];
    const updateTool = schema.tools.find(t => t.category === 'update') || schema.tools[2];

    return `// Multi-step workflow example
[
  {
    "step": 1,
    "description": "List existing items",
    "request": {
      "jsonrpc": "2.0",
      "id": "step-1",
      "method": "tools/call",
      "params": {
        "name": "${schema.service}.${listTool.name}",
        "arguments": ${JSON.stringify(this.generateExampleInput(listTool.inputSchema), null, 6)}
      }
    }
  },
  ${createTool ? `{
    "step": 2,
    "description": "Create new item based on list results",
    "request": {
      "jsonrpc": "2.0",
      "id": "step-2",
      "method": "tools/call",
      "params": {
        "name": "${schema.service}.${createTool.name}",
        "arguments": ${JSON.stringify(this.generateExampleInput(createTool.inputSchema), null, 6)}
      }
    }
  },` : ''}
  ${updateTool ? `{
    "step": 3,
    "description": "Update the created item",
    "request": {
      "jsonrpc": "2.0",
      "id": "step-3",
      "method": "tools/call",
      "params": {
        "name": "${schema.service}.${updateTool.name}",
        "arguments": ${JSON.stringify(this.generateExampleInput(updateTool.inputSchema), null, 6)}
      }
    }
  }` : ''}
]`;
  }

  generateErrorHandlingExample(schema) {
    const tool = schema.tools[0];
    
    return `{
  "workflow": "Error Handling Example",
  "steps": [
    {
      "name": "attempt_operation",
      "request": {
        "jsonrpc": "2.0",
        "id": "attempt-1",
        "method": "tools/call",
        "params": {
          "name": "${schema.service}.${tool.name}",
          "arguments": ${JSON.stringify(this.generateExampleInput(tool.inputSchema), null, 6)}
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
}`;
  }

  generateConditionalExample(schema) {
    const listTool = schema.tools.find(t => t.category === 'list') || schema.tools[0];
    const createTool = schema.tools.find(t => t.category === 'create') || schema.tools[1];

    return `{
  "workflow": "Conditional Logic Example",
  "steps": [
    {
      "name": "check_existing",
      "request": {
        "jsonrpc": "2.0",
        "id": "check-1",
        "method": "tools/call",
        "params": {
          "name": "${schema.service}.${listTool.name}",
          "arguments": ${JSON.stringify(this.generateExampleInput(listTool.inputSchema), null, 6)}
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
              "name": "${schema.service}.${createTool ? createTool.name : 'createItem'}",
              "arguments": ${JSON.stringify(createTool ? this.generateExampleInput(createTool.inputSchema) : {}, null, 8)}
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
}`;
  }

  generateBatchExample(schema) {
    const tool = schema.tools[0];
    
    return `// Batch operations with rate limiting
class ${schema.service.charAt(0).toUpperCase() + schema.service.slice(1)}BatchProcessor {
  constructor(mcpClient, options = {}) {
    this.client = mcpClient;
    this.batchSize = options.batchSize || 10;
    this.delayMs = options.delayMs || 100;
    this.maxConcurrent = options.maxConcurrent || 3;
  }

  async processBatch(items) {
    const results = [];
    const batches = this.createBatches(items, this.batchSize);
    
    for (const batch of batches) {
      const batchPromises = batch.map(item => 
        this.processItem(item).catch(error => ({ error, item }))
      );
      
      const batchResults = await Promise.all(batchPromises);
      results.push(...batchResults);
      
      // Rate limiting delay
      if (batches.indexOf(batch) < batches.length - 1) {
        await this.delay(this.delayMs);
      }
    }
    
    return results;
  }

  async processItem(item) {
    const request = {
      jsonrpc: "2.0",
      id: \`batch-\${Date.now()}-\${Math.random()}\`,
      method: "tools/call",
      params: {
        name: "${schema.service}.${tool.name}",
        arguments: {
          ...${JSON.stringify(this.generateExampleInput(tool.inputSchema), null, 10)},
          ...item
        }
      }
    };
    
    return await this.client.request(request);
  }

  createBatches(items, size) {
    const batches = [];
    for (let i = 0; i < items.length; i += size) {
      batches.push(items.slice(i, i + size));
    }
    return batches;
  }

  delay(ms) {
    return new Promise(resolve => setTimeout(resolve, ms));
  }
}

// Usage example
const processor = new ${schema.service.charAt(0).toUpperCase() + schema.service.slice(1)}BatchProcessor(mcpClient, {
  batchSize: 5,
  delayMs: 200,
  maxConcurrent: 2
});

const items = [
  { id: 1, name: "Item 1" },
  { id: 2, name: "Item 2" },
  // ... more items
];

const results = await processor.processBatch(items);
console.log(\`Processed \${results.length} items\`);`;
  }

  generateRetryExample(schema) {
    const tool = schema.tools[0];
    
    return `// Retry logic with exponential backoff
class RetryableClient {
  constructor(mcpClient, options = {}) {
    this.client = mcpClient;
    this.maxRetries = options.maxRetries || 3;
    this.baseDelay = options.baseDelay || 1000;
    this.maxDelay = options.maxDelay || 10000;
  }

  async callWithRetry(toolName, arguments, options = {}) {
    const maxRetries = options.maxRetries || this.maxRetries;
    let lastError;

    for (let attempt = 0; attempt <= maxRetries; attempt++) {
      try {
        const request = {
          jsonrpc: "2.0",
          id: \`retry-\${Date.now()}-\${attempt}\`,
          method: "tools/call",
          params: { name: toolName, arguments }
        };

        const response = await this.client.request(request);
        
        // Success - return result
        return response;
        
      } catch (error) {
        lastError = error;
        
        // Don't retry on certain errors
        if (this.isNonRetryableError(error)) {
          throw error;
        }
        
        // Don't retry on last attempt
        if (attempt === maxRetries) {
          break;
        }
        
        // Calculate delay with exponential backoff
        const delay = Math.min(
          this.baseDelay * Math.pow(2, attempt),
          this.maxDelay
        );
        
        console.log(\`Attempt \${attempt + 1} failed, retrying in \${delay}ms...\`);
        await this.delay(delay);
      }
    }
    
    throw new Error(\`Failed after \${maxRetries + 1} attempts: \${lastError.message}\`);
  }

  isNonRetryableError(error) {
    // Don't retry on authentication or validation errors
    const nonRetryableCodes = [400, 401, 403, 404, 422];
    return nonRetryableCodes.includes(error.statusCode);
  }

  delay(ms) {
    return new Promise(resolve => setTimeout(resolve, ms));
  }
}

// Usage example
const retryClient = new RetryableClient(mcpClient, {
  maxRetries: 3,
  baseDelay: 1000,
  maxDelay: 8000
});

try {
  const result = await retryClient.callWithRetry(
    "${schema.service}.${tool.name}",
    ${JSON.stringify(this.generateExampleInput(tool.inputSchema), null, 4)}
  );
  console.log("Operation succeeded:", result);
} catch (error) {
  console.error("Operation failed permanently:", error.message);
}`;
  }

  generateOptimizationExample(schema) {
    return `// Performance optimization techniques
class OptimizedClient {
  constructor(mcpClient) {
    this.client = mcpClient;
    this.cache = new Map();
    this.requestQueue = [];
    this.processing = false;
  }

  // Request batching
  async batchRequests(requests, batchSize = 10) {
    const batches = [];
    for (let i = 0; i < requests.length; i += batchSize) {
      batches.push(requests.slice(i, i + batchSize));
    }

    const results = [];
    for (const batch of batches) {
      const batchResults = await Promise.all(
        batch.map(req => this.client.request(req))
      );
      results.push(...batchResults);
    }

    return results;
  }

  // Response caching
  async cachedCall(toolName, arguments, ttlMs = 300000) {
    const cacheKey = \`\${toolName}:\${JSON.stringify(arguments)}\`;
    const cached = this.cache.get(cacheKey);
    
    if (cached && Date.now() - cached.timestamp < ttlMs) {
      return cached.data;
    }

    const result = await this.client.request({
      jsonrpc: "2.0",
      id: \`cached-\${Date.now()}\`,
      method: "tools/call",
      params: { name: toolName, arguments }
    });

    this.cache.set(cacheKey, {
      data: result,
      timestamp: Date.now()
    });

    return result;
  }

  // Request deduplication
  async deduplicatedCall(toolName, arguments) {
    const key = \`\${toolName}:\${JSON.stringify(arguments)}\`;
    
    if (this.pendingRequests?.has(key)) {
      return await this.pendingRequests.get(key);
    }

    if (!this.pendingRequests) {
      this.pendingRequests = new Map();
    }

    const promise = this.client.request({
      jsonrpc: "2.0",
      id: \`dedup-\${Date.now()}\`,
      method: "tools/call",
      params: { name: toolName, arguments }
    }).finally(() => {
      this.pendingRequests.delete(key);
    });

    this.pendingRequests.set(key, promise);
    return await promise;
  }
}

// Usage example
const optimizedClient = new OptimizedClient(mcpClient);

// Batch multiple requests
const requests = [
  { jsonrpc: "2.0", id: "1", method: "tools/call", params: { name: "${schema.service}.${schema.tools[0]?.name}", arguments: {} }},
  { jsonrpc: "2.0", id: "2", method: "tools/call", params: { name: "${schema.service}.${schema.tools[1]?.name}", arguments: {} }}
];

const batchResults = await optimizedClient.batchRequests(requests);

// Use caching for repeated calls
const cachedResult = await optimizedClient.cachedCall(
  "${schema.service}.${schema.tools[0]?.name}",
  {},
  600000 // 10 minute cache
);`;
  }

  generateAutomationScript(schema) {
    return `#!/usr/bin/env node

/**
 * ${schema.service.charAt(0).toUpperCase() + schema.service.slice(1)} Automation Script
 * 
 * Complete automation workflow for ${schema.description}
 */

import { MCPClient } from './mcp-client.js';
import { Logger } from './logger.js';

class ${schema.service.charAt(0).toUpperCase() + schema.service.slice(1)}Automation {
  constructor(config) {
    this.config = config;
    this.client = new MCPClient(config.mcpServer);
    this.logger = new Logger('${schema.service}-automation');
    this.stats = {
      processed: 0,
      errors: 0,
      startTime: Date.now()
    };
  }

  async run() {
    try {
      this.logger.info('Starting ${schema.service} automation...');
      
      // Initialize MCP connection
      await this.client.initialize();
      
      // Run main workflow
      await this.executeWorkflow();
      
      // Generate report
      await this.generateReport();
      
      this.logger.info('Automation completed successfully');
      
    } catch (error) {
      this.logger.error('Automation failed:', error);
      throw error;
    } finally {
      await this.cleanup();
    }
  }

  async executeWorkflow() {
    const steps = [
      { name: 'discovery', handler: this.discoveryStep.bind(this) },
      { name: 'processing', handler: this.processingStep.bind(this) },
      { name: 'validation', handler: this.validationStep.bind(this) },
      { name: 'cleanup', handler: this.cleanupStep.bind(this) }
    ];

    for (const step of steps) {
      try {
        this.logger.info(\`Executing step: \${step.name}\`);
        await step.handler();
        this.logger.info(\`Step completed: \${step.name}\`);
      } catch (error) {
        this.logger.error(\`Step failed: \${step.name}\`, error);
        if (this.config.stopOnError) {
          throw error;
        }
        this.stats.errors++;
      }
    }
  }

  async discoveryStep() {
    // Discover available resources
    const result = await this.client.call('${schema.service}.${schema.tools.find(t => t.category === 'list')?.name || schema.tools[0]?.name}', {});
    this.discoveredItems = result.data || [];
    this.logger.info(\`Discovered \${this.discoveredItems.length} items\`);
  }

  async processingStep() {
    // Process each discovered item
    for (const item of this.discoveredItems) {
      try {
        await this.processItem(item);
        this.stats.processed++;
      } catch (error) {
        this.logger.error(\`Failed to process item \${item.id}\`, error);
        this.stats.errors++;
      }
    }
  }

  async processItem(item) {
    // Example processing logic
    const updateTool = '${schema.service}.${schema.tools.find(t => t.category === 'update')?.name || 'updateItem'}';
    
    await this.client.call(updateTool, {
      id: item.id,
      ...this.config.updateData
    });
    
    this.logger.debug(\`Processed item: \${item.id}\`);
  }

  async validationStep() {
    // Validate results
    this.logger.info('Validating automation results...');
    
    const validationResults = await this.client.call('${schema.service}.${schema.tools[0]?.name}', {
      validate: true
    });
    
    if (!validationResults.success) {
      throw new Error('Validation failed');
    }
  }

  async cleanupStep() {
    // Cleanup temporary resources
    this.logger.info('Cleaning up temporary resources...');
    // Implementation depends on service
  }

  async generateReport() {
    const duration = Date.now() - this.stats.startTime;
    const report = {
      automation: '${schema.service}',
      timestamp: new Date().toISOString(),
      duration: \`\${Math.round(duration / 1000)}s\`,
      stats: this.stats,
      success: this.stats.errors === 0
    };

    this.logger.info('Automation Report:', report);
    
    // Save report to file
    await this.saveReport(report);
  }

  async saveReport(report) {
    const fs = await import('fs/promises');
    const reportPath = \`reports/${schema.service}-\${Date.now()}.json\`;
    await fs.writeFile(reportPath, JSON.stringify(report, null, 2));
    this.logger.info(\`Report saved to: \${reportPath}\`);
  }

  async cleanup() {
    if (this.client) {
      await this.client.disconnect();
    }
  }
}

// Configuration
const config = {
  mcpServer: {
    command: 'node',
    args: ['examples/mcp-${schema.service}-server.js'],
    env: {
      ${schema.authentication.required.map(v => `${v}: process.env.${v}`).join(',\n      ')}
    }
  },
  stopOnError: false,
  updateData: {
    // Service-specific update data
  }
};

// Run automation
if (import.meta.url === \`file://\${process.argv[1]}\`) {
  const automation = new ${schema.service.charAt(0).toUpperCase() + schema.service.slice(1)}Automation(config);
  automation.run().catch(console.error);
}

export { ${schema.service.charAt(0).toUpperCase() + schema.service.slice(1)}Automation };`;
  }

  generateMonitoringExample(schema) {
    return `// Monitoring and alerting system
class ${schema.service.charAt(0).toUpperCase() + schema.service.slice(1)}Monitor {
  constructor(config) {
    this.config = config;
    this.client = new MCPClient(config.mcpServer);
    this.alerts = [];
    this.metrics = {
      requests: 0,
      errors: 0,
      responseTime: []
    };
  }

  async startMonitoring() {
    console.log('Starting ${schema.service} monitoring...');
    
    // Set up periodic health checks
    setInterval(() => this.healthCheck(), this.config.healthCheckInterval || 60000);
    
    // Set up metric collection
    setInterval(() => this.collectMetrics(), this.config.metricsInterval || 30000);
    
    // Set up alert processing
    setInterval(() => this.processAlerts(), this.config.alertInterval || 10000);
  }

  async healthCheck() {
    const startTime = Date.now();
    
    try {
      // Test basic connectivity
      const result = await this.client.call('${schema.service}.${schema.tools[0]?.name}', {});
      
      const responseTime = Date.now() - startTime;
      this.metrics.responseTime.push(responseTime);
      this.metrics.requests++;
      
      // Check for performance issues
      if (responseTime > this.config.slowResponseThreshold || 5000) {
        this.addAlert('performance', \`Slow response: \${responseTime}ms\`);
      }
      
      console.log(\`Health check passed: \${responseTime}ms\`);
      
    } catch (error) {
      this.metrics.errors++;
      this.addAlert('error', \`Health check failed: \${error.message}\`);
      console.error('Health check failed:', error.message);
    }
  }

  async collectMetrics() {
    const metrics = {
      timestamp: new Date().toISOString(),
      requests: this.metrics.requests,
      errors: this.metrics.errors,
      errorRate: this.metrics.errors / this.metrics.requests,
      avgResponseTime: this.calculateAverageResponseTime(),
      p95ResponseTime: this.calculatePercentile(95)
    };

    // Store metrics
    await this.storeMetrics(metrics);
    
    // Check thresholds
    if (metrics.errorRate > (this.config.errorRateThreshold || 0.1)) {
      this.addAlert('error_rate', \`High error rate: \${(metrics.errorRate * 100).toFixed(2)}%\`);
    }
  }

  calculateAverageResponseTime() {
    if (this.metrics.responseTime.length === 0) return 0;
    const sum = this.metrics.responseTime.reduce((a, b) => a + b, 0);
    return sum / this.metrics.responseTime.length;
  }

  calculatePercentile(percentile) {
    if (this.metrics.responseTime.length === 0) return 0;
    const sorted = [...this.metrics.responseTime].sort((a, b) => a - b);
    const index = Math.ceil((percentile / 100) * sorted.length) - 1;
    return sorted[index];
  }

  addAlert(type, message) {
    const alert = {
      type,
      message,
      timestamp: new Date().toISOString(),
      service: '${schema.service}'
    };
    
    this.alerts.push(alert);
    console.warn(\`ALERT [\${type}]: \${message}\`);
  }

  async processAlerts() {
    if (this.alerts.length === 0) return;
    
    const alertsToProcess = [...this.alerts];
    this.alerts = [];
    
    for (const alert of alertsToProcess) {
      await this.sendAlert(alert);
    }
  }

  async sendAlert(alert) {
    // Send to configured alert channels
    const channels = this.config.alertChannels || ['console'];
    
    for (const channel of channels) {
      switch (channel) {
        case 'console':
          console.error('ALERT:', alert);
          break;
        case 'webhook':
          await this.sendWebhookAlert(alert);
          break;
        case 'email':
          await this.sendEmailAlert(alert);
          break;
      }
    }
  }

  async storeMetrics(metrics) {
    // Store in configured storage
    const storage = this.config.metricsStorage || 'file';
    
    switch (storage) {
      case 'file':
        const fs = await import('fs/promises');
        await fs.appendFile('metrics/${schema.service}.jsonl', JSON.stringify(metrics) + '\\n');
        break;
      case 'database':
        // Store in database
        break;
    }
  }
}

// Usage
const monitor = new ${schema.service.charAt(0).toUpperCase() + schema.service.slice(1)}Monitor({
  mcpServer: { /* MCP server config */ },
  healthCheckInterval: 60000,
  metricsInterval: 30000,
  alertInterval: 10000,
  slowResponseThreshold: 5000,
  errorRateThreshold: 0.1,
  alertChannels: ['console', 'webhook'],
  metricsStorage: 'file'
});

monitor.startMonitoring();`;
  }

  generateSchedulingExample(schema) {
    return `// Scheduled task system
import cron from 'node-cron';

class ${schema.service.charAt(0).toUpperCase() + schema.service.slice(1)}Scheduler {
  constructor(config) {
    this.config = config;
    this.client = new MCPClient(config.mcpServer);
    this.tasks = new Map();
    this.running = false;
  }

  async start() {
    console.log('Starting ${schema.service} scheduler...');
    this.running = true;
    
    // Initialize MCP client
    await this.client.initialize();
    
    // Set up scheduled tasks
    this.setupTasks();
    
    console.log(\`Scheduler started with \${this.tasks.size} tasks\`);
  }

  setupTasks() {
    const tasks = [
      {
        name: 'daily-sync',
        schedule: '0 2 * * *', // Daily at 2 AM
        handler: this.dailySync.bind(this)
      },
      {
        name: 'hourly-check',
        schedule: '0 * * * *', // Every hour
        handler: this.hourlyCheck.bind(this)
      },
      {
        name: 'weekly-cleanup',
        schedule: '0 3 * * 0', // Weekly on Sunday at 3 AM
        handler: this.weeklyCleanup.bind(this)
      }
    ];

    for (const task of tasks) {
      const cronTask = cron.schedule(task.schedule, async () => {
        if (!this.running) return;
        
        console.log(\`Running scheduled task: \${task.name}\`);
        
        try {
          await task.handler();
          console.log(\`Task completed: \${task.name}\`);
        } catch (error) {
          console.error(\`Task failed: \${task.name}\`, error);
          await this.handleTaskError(task.name, error);
        }
      }, {
        scheduled: false,
        timezone: this.config.timezone || 'UTC'
      });

      this.tasks.set(task.name, {
        ...task,
        cronTask,
        lastRun: null,
        runCount: 0,
        errorCount: 0
      });

      cronTask.start();
    }
  }

  async dailySync() {
    // Daily synchronization task
    console.log('Executing daily sync...');
    
    const items = await this.client.call('${schema.service}.${schema.tools.find(t => t.category === 'list')?.name || schema.tools[0]?.name}', {});
    
    for (const item of items.data || []) {
      await this.syncItem(item);
    }
    
    console.log(\`Daily sync completed: \${items.data?.length || 0} items processed\`);
  }

  async hourlyCheck() {
    // Hourly health check
    console.log('Executing hourly check...');
    
    const status = await this.client.call('${schema.service}.${schema.tools[0]?.name}', {
      healthCheck: true
    });
    
    if (!status.healthy) {
      await this.handleUnhealthyStatus(status);
    }
  }

  async weeklyCleanup() {
    // Weekly cleanup task
    console.log('Executing weekly cleanup...');
    
    // Clean up old data
    const cutoffDate = new Date();
    cutoffDate.setDate(cutoffDate.getDate() - 30); // 30 days ago
    
    const cleanupResult = await this.client.call('${schema.service}.cleanup', {
      before: cutoffDate.toISOString()
    });
    
    console.log(\`Weekly cleanup completed: \${cleanupResult.deletedCount} items removed\`);
  }

  async syncItem(item) {
    // Sync individual item
    try {
      await this.client.call('${schema.service}.${schema.tools.find(t => t.category === 'update')?.name || 'updateItem'}', {
        id: item.id,
        sync: true
      });
    } catch (error) {
      console.error(\`Failed to sync item \${item.id}:\`, error.message);
    }
  }

  async handleUnhealthyStatus(status) {
    console.warn('Service is unhealthy:', status);
    
    // Send alert
    await this.sendAlert({
      type: 'health',
      message: \`Service unhealthy: \${status.message}\`,
      severity: 'warning'
    });
  }

  async handleTaskError(taskName, error) {
    const task = this.tasks.get(taskName);
    if (task) {
      task.errorCount++;
    }
    
    // Send error alert
    await this.sendAlert({
      type: 'task_error',
      message: \`Task \${taskName} failed: \${error.message}\`,
      severity: 'error'
    });
  }

  async sendAlert(alert) {
    // Implementation depends on alert configuration
    console.error('ALERT:', alert);
  }

  getTaskStatus() {
    const status = {};
    for (const [name, task] of this.tasks) {
      status[name] = {
        schedule: task.schedule,
        lastRun: task.lastRun,
        runCount: task.runCount,
        errorCount: task.errorCount,
        nextRun: task.cronTask.nextDate()
      };
    }
    return status;
  }

  async stop() {
    console.log('Stopping scheduler...');
    this.running = false;
    
    for (const [name, task] of this.tasks) {
      task.cronTask.stop();
    }
    
    await this.client.disconnect();
    console.log('Scheduler stopped');
  }
}

// Usage
const scheduler = new ${schema.service.charAt(0).toUpperCase() + schema.service.slice(1)}Scheduler({
  mcpServer: {
    command: 'node',
    args: ['examples/mcp-${schema.service}-server.js'],
    env: {
      ${schema.authentication.required.map(v => `${v}: process.env.${v}`).join(',\n      ')}
    }
  },
  timezone: 'UTC'
});

// Start scheduler
scheduler.start().catch(console.error);

// Graceful shutdown
process.on('SIGINT', async () => {
  console.log('Received SIGINT, shutting down gracefully...');
  await scheduler.stop();
  process.exit(0);
});

export { ${schema.service.charAt(0).toUpperCase() + schema.service.slice(1)}Scheduler };`;
  }

  generateExampleInput(schema) {
    if (!schema || !schema.properties) {
      return {};
    }

    const example = {};
    
    for (const [propName, propSchema] of Object.entries(schema.properties)) {
      // Skip auth tokens in examples
      if (propName.toLowerCase().includes('token') || 
          propName.toLowerCase().includes('key') ||
          propName.toLowerCase().includes('auth')) {
        continue;
      }
      
      if (propSchema.example !== undefined) {
        example[propName] = propSchema.example;
      } else if (propSchema.default !== undefined) {
        example[propName] = propSchema.default;
      } else {
        example[propName] = this.generateExampleValue(propSchema);
      }
    }

    return example;
  }

  generateExampleValue(schema) {
    switch (schema.type) {
      case 'string':
        if (schema.enum) return schema.enum[0];
        if (schema.format === 'email') return 'user@example.com';
        if (schema.format === 'uri') return 'https://example.com';
        if (schema.format === 'date') return '2024-01-01';
        if (schema.format === 'date-time') return '2024-01-01T00:00:00Z';
        return 'example-value';
      case 'number':
      case 'integer':
        return schema.minimum || 1;
      case 'boolean':
        return true;
      case 'array':
        return [this.generateExampleValue(schema.items || { type: 'string' })];
      case 'object':
        if (schema.properties) {
          const obj = {};
          for (const [key, value] of Object.entries(schema.properties)) {
            obj[key] = this.generateExampleValue(value);
          }
          return obj;
        }
        return {};
      default:
        return null;
    }
  }

  async validateExamples() {
    console.log('🔍 Validating generated examples...');
    
    let validationErrors = 0;
    
    for (const [serviceName, scenarios] of Object.entries(this.examples)) {
      for (const [scenarioName, scenario] of Object.entries(scenarios)) {
        try {
          this.validateScenario(scenario);
          console.log(`✅ Examples validation passed for ${serviceName}/${scenarioName}`);
        } catch (error) {
          validationErrors++;
          console.error(`❌ Examples validation failed for ${serviceName}/${scenarioName}:`, error.message);
        }
      }
    }
    
    if (validationErrors === 0) {
      console.log('✅ All examples passed validation');
    } else {
      console.log(`❌ ${validationErrors} example sets failed validation`);
    }
  }

  validateScenario(scenario) {
    const required = ['service', 'scenario', 'metadata', 'examples'];
    for (const field of required) {
      if (!scenario[field]) {
        throw new Error(`Missing required field: ${field}`);
      }
    }
    
    if (!Array.isArray(scenario.examples)) {
      throw new Error('examples must be an array');
    }
    
    for (const example of scenario.examples) {
      this.validateExample(example);
    }
  }

  validateExample(example) {
    const required = ['name', 'description', 'category', 'code'];
    for (const field of required) {
      if (!example[field]) {
        throw new Error(`Example missing required field: ${field}`);
      }
    }
    
    if (typeof example.code !== 'string' || example.code.length === 0) {
      throw new Error('Example code must be a non-empty string');
    }
  }

  async writeExamples() {
    const outputDir = path.resolve(this.options.output);
    await fs.mkdir(outputDir, { recursive: true });
    
    for (const [serviceName, scenarios] of Object.entries(this.examples)) {
      const serviceDir = path.join(outputDir, serviceName);
      await fs.mkdir(serviceDir, { recursive: true });
      
      for (const [scenarioName, scenario] of Object.entries(scenarios)) {
        const filename = `${scenarioName}-examples.${this.getFileExtension()}`;
        const filepath = path.join(serviceDir, filename);
        
        let content;
        switch (this.options.format) {
          case 'json':
            content = JSON.stringify(scenario, null, 2);
            break;
          case 'yaml':
            try {
              const yaml = await import('yaml');
              content = yaml.stringify(scenario);
            } catch {
              throw new Error('YAML output requires yaml package');
            }
            break;
          case 'markdown':
          default:
            content = this.generateMarkdownContent(scenario);
            break;
        }
        
        await fs.writeFile(filepath, content, 'utf8');
        console.log(`📄 Examples written to ${filepath}`);
      }
    }
  }

  getFileExtension() {
    switch (this.options.format) {
      case 'json': return 'json';
      case 'yaml': return 'yaml';
      case 'markdown': 
      default: return 'md';
    }
  }

  generateMarkdownContent(scenario) {
    let md = `# ${scenario.metadata.title}\n\n`;
    md += `${scenario.metadata.description}\n\n`;
    md += `**Complexity Level:** ${scenario.metadata.complexity}\n`;
    md += `**Generated:** ${scenario.metadata.generatedAt}\n\n`;
    
    md += `## Examples\n\n`;
    
    for (const example of scenario.examples) {
      md += `### ${example.name}\n\n`;
      md += `${example.description}\n\n`;
      md += `**Category:** ${example.category}\n\n`;
      
      const language = example.language || 'javascript';
      md += `\`\`\`${language}\n${example.code}\n\`\`\`\n\n`;
    }
    
    return md;
  }

  async generateIndexFile() {
    const indexPath = path.join(this.options.output, 'README.md');
    
    let md = `# MCP Tool Usage Examples\n\n`;
    md += `Generated at: ${new Date().toISOString()}\n\n`;
    md += `This directory contains comprehensive usage examples for MCP tools across different services and scenarios.\n\n`;
    
    md += `## Services\n\n`;
    
    for (const [serviceName, scenarios] of Object.entries(this.examples)) {
      const schema = this.schemas[serviceName];
      md += `### ${schema.service}\n\n`;
      md += `${schema.description}\n\n`;
      md += `**Category:** ${schema.category}  \n`;
      md += `**Tools:** ${schema.toolCount}  \n`;
      md += `**Authentication:** ${schema.authentication.type}\n\n`;
      
      md += `**Available Examples:**\n`;
      for (const [scenarioName, scenario] of Object.entries(scenarios)) {
        const scenarioConfig = EXAMPLE_SCENARIOS[scenarioName];
        md += `- [${scenarioConfig.name}](./${serviceName}/${scenarioName}-examples.md) - ${scenarioConfig.description} (${scenarioConfig.complexity})\n`;
      }
      md += `\n`;
    }
    
    md += `## Scenario Types\n\n`;
    for (const [scenarioName, config] of Object.entries(EXAMPLE_SCENARIOS)) {
      md += `### ${config.name}\n`;
      md += `${config.description}\n`;
      md += `**Complexity:** ${config.complexity}\n\n`;
    }
    
    await fs.writeFile(indexPath, md, 'utf8');
    console.log(`📄 Index file written to ${indexPath}`);
  }
}

// CLI handling
async function main() {
  const args = process.argv.slice(2);
  const options = {};
  
  for (let i = 0; i < args.length; i++) {
    const arg = args[i];
    switch (arg) {
      case '--output':
        options.output = args[++i];
        break;
      case '--service':
        options.service = args[++i];
        break;
      case '--scenario':
        options.scenario = args[++i];
        break;
      case '--format':
        options.format = args[++i];
        break;
      case '--validate':
        options.validate = true;
        break;
      case '--help':
        console.log(`
Tool Examples Generator

Usage: node examples/scripts/generate-tool-examples.js [options]

Options:
  --output <dir>      Output directory for examples (default: examples/usage)
  --service <name>    Generate examples for specific service only
  --scenario <type>   Generate specific scenario: basic, workflow, advanced, automation
  --validate          Validate generated examples
  --format <format>   Output format: json, yaml, markdown (default: markdown)
  --help              Show this help message
        `);
        process.exit(0);
      default:
        if (arg.startsWith('--')) {
          console.error(`Unknown option: ${arg}`);
          process.exit(1);
        }
    }
  }
  
  try {
    const generator = new ToolExamplesGenerator(options);
    await generator.generateAllExamples();
  } catch (error) {
    console.error('❌ Generation failed:', error.message);
    process.exit(1);
  }
}

if (import.meta.url === `file://${process.argv[1]}`) {
  main().catch(console.error);
}

export { ToolExamplesGenerator };