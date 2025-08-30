#!/usr/bin/env node

/**
 * Tool Schema Generator
 * 
 * Extracts and formats MCP tool schemas from existing implementations.
 * Generates JSON schemas for all available tools (n8n, Hostinger, Docker, generic OpenAPI).
 * Creates validation functions to ensure schema accuracy.
 * 
 * Usage:
 *   node examples/scripts/generate-tool-schemas.js [options]
 * 
 * Options:
 *   --output <file>     Output file for generated schemas (default: schemas/tool-schemas.json)
 *   --format <format>   Output format: json, yaml, markdown (default: json)
 *   --validate          Run validation tests on generated schemas
 *   --service <name>    Generate schemas for specific service only
 *   --include-examples  Include usage examples in schemas
 */

import fs from 'fs/promises';
import path from 'path';
import { fileURLToPath } from 'url';
import { generateMcpTools } from '../../lib/openapi-generator/index.js';

const __filename = fileURLToPath(import.meta.url);
const __dirname = path.dirname(__filename);
const projectRoot = path.resolve(__dirname, '../..');

// Service configurations for schema generation
const SERVICE_CONFIGS = {
  n8n: {
    name: 'n8n',
    description: 'n8n workflow automation platform',
    specFile: 'specs/n8n-openapi.yaml',
    envVars: ['N8N_API_KEY', 'N8N_API_URL'],
    category: 'workflow-automation'
  },
  hostinger: {
    name: 'hostinger',
    description: 'Hostinger cloud hosting and domain management',
    specFile: 'specs/hostinger-openapi.yaml',
    envVars: ['HOSTINGER_API_TOKEN'],
    category: 'hosting'
  },
  docker: {
    name: 'docker',
    description: 'Docker container and image management',
    specFile: 'specs/docker-engine-api.yaml',
    envVars: ['DOCKER_HOST'],
    category: 'infrastructure'
  },
  adobe_pdf: {
    name: 'adobe_pdf',
    description: 'Adobe PDF Services for document processing',
    specFile: 'specs/adobe-pdf-services.yaml',
    envVars: ['ADOBE_ACCESS_TOKEN'],
    category: 'document-processing'
  }
};

class ToolSchemaGenerator {
  constructor(options = {}) {
    this.options = {
      output: options.output || 'schemas/tool-schemas.json',
      format: options.format || 'json',
      validate: options.validate || false,
      service: options.service || null,
      includeExamples: options.includeExamples || false,
      ...options
    };
    this.schemas = {};
    this.stats = {
      totalTools: 0,
      totalServices: 0,
      errors: [],
      warnings: []
    };
  }

  async generateAllSchemas() {
    console.log('🔧 Generating MCP tool schemas...');
    
    const services = this.options.service 
      ? [this.options.service]
      : Object.keys(SERVICE_CONFIGS);

    for (const serviceName of services) {
      try {
        await this.generateServiceSchema(serviceName);
      } catch (error) {
        this.stats.errors.push({
          service: serviceName,
          error: error.message
        });
        console.error(`❌ Failed to generate schema for ${serviceName}:`, error.message);
      }
    }

    if (this.options.validate) {
      await this.validateSchemas();
    }

    await this.writeOutput();
    this.printSummary();
  }

  async generateServiceSchema(serviceName) {
    const config = SERVICE_CONFIGS[serviceName];
    if (!config) {
      throw new Error(`Unknown service: ${serviceName}`);
    }

    console.log(`📋 Processing ${config.name}...`);

    const specPath = path.join(projectRoot, config.specFile);
    let specExists = false;
    
    try {
      await fs.access(specPath);
      specExists = true;
    } catch {
      // Spec file doesn't exist, try to load from generated tools
      console.log(`⚠️  Spec file not found for ${serviceName}, trying generated tools...`);
    }

    let tools = [];
    
    if (specExists) {
      // Generate tools from OpenAPI spec
      try {
        const specContent = await fs.readFile(specPath, 'utf8');
        const spec = specContent.trim().startsWith('{') 
          ? JSON.parse(specContent)
          : await this.parseYaml(specContent);
        
        const generatedTools = await generateMcpTools(spec, {
          baseUrl: this.getBaseUrl(serviceName),
          filters: this.getFilters(serviceName)
        });
        
        tools = generatedTools.map(tool => this.normalizeToolSchema(tool, serviceName));
      } catch (error) {
        console.warn(`⚠️  Failed to generate from spec for ${serviceName}: ${error.message}`);
      }
    }

    // Try to load from existing generated tools as fallback
    if (tools.length === 0) {
      const generatedPath = path.join(projectRoot, 'examples/generated', `${serviceName}-openapi-tools.json`);
      try {
        const generatedContent = await fs.readFile(generatedPath, 'utf8');
        const generated = JSON.parse(generatedContent);
        tools = generated.tools.map(tool => this.normalizeToolSchema(tool, serviceName));
      } catch (error) {
        console.warn(`⚠️  No generated tools found for ${serviceName}: ${error.message}`);
      }
    }

    if (tools.length === 0) {
      this.stats.warnings.push({
        service: serviceName,
        warning: 'No tools found'
      });
      return;
    }

    // Create service schema
    const serviceSchema = {
      service: config.name,
      description: config.description,
      category: config.category,
      authentication: {
        required: config.envVars,
        type: this.getAuthType(serviceName)
      },
      tools: tools,
      toolCount: tools.length,
      generatedAt: new Date().toISOString(),
      version: await this.getPackageVersion()
    };

    if (this.options.includeExamples) {
      serviceSchema.examples = await this.generateExamples(serviceName, tools);
    }

    this.schemas[serviceName] = serviceSchema;
    this.stats.totalTools += tools.length;
    this.stats.totalServices++;

    console.log(`✅ Generated ${tools.length} tool schemas for ${config.name}`);
  }

  normalizeToolSchema(tool, serviceName) {
    return {
      name: tool.name,
      description: tool.description || `${tool.name} operation`,
      inputSchema: this.normalizeInputSchema(tool.inputSchema),
      outputSchema: this.generateOutputSchema(tool),
      method: tool.method || 'POST',
      pathTemplate: tool.pathTemplate || '/',
      category: this.categorizeOperation(tool.name, tool.description),
      tags: this.extractTags(tool),
      security: this.extractSecurity(tool, serviceName),
      examples: this.options.includeExamples ? this.generateToolExamples(tool) : undefined
    };
  }

  normalizeInputSchema(schema) {
    if (!schema || typeof schema !== 'object') {
      return { type: 'object', properties: {}, required: [] };
    }

    const normalized = {
      type: schema.type || 'object',
      properties: {},
      required: schema.required || []
    };

    if (schema.properties) {
      for (const [propName, propSchema] of Object.entries(schema.properties)) {
        normalized.properties[propName] = this.normalizePropertySchema(propSchema);
      }
    }

    return normalized;
  }

  normalizePropertySchema(schema) {
    if (!schema || typeof schema !== 'object') {
      return { type: 'string' };
    }

    const normalized = {
      type: schema.type || 'string'
    };

    // Copy relevant properties
    const copyProps = [
      'description', 'example', 'enum', 'format', 'pattern',
      'minLength', 'maxLength', 'minimum', 'maximum',
      'minItems', 'maxItems', 'uniqueItems', 'default'
    ];

    for (const prop of copyProps) {
      if (schema[prop] !== undefined) {
        normalized[prop] = schema[prop];
      }
    }

    // Handle nested objects and arrays
    if (schema.properties) {
      normalized.properties = {};
      for (const [propName, propSchema] of Object.entries(schema.properties)) {
        normalized.properties[propName] = this.normalizePropertySchema(propSchema);
      }
      if (schema.required) {
        normalized.required = schema.required;
      }
    }

    if (schema.items) {
      normalized.items = this.normalizePropertySchema(schema.items);
    }

    return normalized;
  }

  generateOutputSchema(tool) {
    // Generate basic output schema based on operation type
    const baseSchema = {
      type: 'object',
      properties: {
        success: { type: 'boolean', description: 'Operation success status' },
        data: { type: 'object', description: 'Response data' },
        message: { type: 'string', description: 'Response message' }
      }
    };

    // Customize based on operation type
    if (tool.method === 'GET') {
      if (tool.name.includes('List') || tool.name.includes('getAll')) {
        baseSchema.properties.data = {
          type: 'array',
          items: { type: 'object' },
          description: 'List of items'
        };
      }
    } else if (tool.method === 'POST' && tool.name.includes('create')) {
      baseSchema.properties.data.properties = {
        id: { type: 'string', description: 'Created resource ID' }
      };
    }

    return baseSchema;
  }

  categorizeOperation(name, description) {
    const categories = {
      'list': ['list', 'get', 'fetch', 'retrieve'],
      'create': ['create', 'add', 'new', 'post'],
      'update': ['update', 'modify', 'edit', 'patch', 'put'],
      'delete': ['delete', 'remove', 'destroy'],
      'manage': ['manage', 'control', 'admin'],
      'monitor': ['monitor', 'status', 'health', 'check']
    };

    const text = `${name} ${description}`.toLowerCase();
    
    for (const [category, keywords] of Object.entries(categories)) {
      if (keywords.some(keyword => text.includes(keyword))) {
        return category;
      }
    }

    return 'other';
  }

  extractTags(tool) {
    const tags = [];
    
    // Extract from operation name
    if (tool.name.includes('_')) {
      const parts = tool.name.split('_');
      if (parts.length > 1) {
        tags.push(parts[0]); // First part is usually the resource/category
      }
    }

    // Extract from path
    if (tool.pathTemplate) {
      const pathParts = tool.pathTemplate.split('/').filter(p => p && !p.startsWith('{'));
      tags.push(...pathParts.slice(0, 2)); // First two path segments
    }

    return [...new Set(tags)]; // Remove duplicates
  }

  extractSecurity(tool, serviceName) {
    const config = SERVICE_CONFIGS[serviceName];
    if (!config) return {};

    const security = {
      required: true,
      schemes: []
    };

    // Analyze input schema for auth parameters
    if (tool.inputSchema && tool.inputSchema.properties) {
      const props = tool.inputSchema.properties;
      
      if (props.bearerToken || props.Authorization) {
        security.schemes.push('bearer');
      }
      if (props['X-API-KEY'] || props['X-N8N-API-KEY']) {
        security.schemes.push('apikey');
      }
      if (props.username && props.password) {
        security.schemes.push('basic');
      }
    }

    // Default based on service
    if (security.schemes.length === 0) {
      switch (serviceName) {
        case 'n8n':
          security.schemes.push('apikey');
          break;
        case 'hostinger':
          security.schemes.push('bearer');
          break;
        case 'adobe_pdf':
          security.schemes.push('bearer');
          break;
        default:
          security.schemes.push('bearer');
      }
    }

    return security;
  }

  generateToolExamples(tool) {
    const examples = [];

    // Generate basic example
    const basicExample = {
      name: 'Basic usage',
      description: `Basic example of using ${tool.name}`,
      input: this.generateExampleInput(tool.inputSchema),
      expectedOutput: this.generateExampleOutput(tool)
    };

    examples.push(basicExample);

    // Generate error example
    const errorExample = {
      name: 'Error handling',
      description: 'Example of error response',
      input: { invalid: 'input' },
      expectedOutput: {
        success: false,
        error: 'Invalid input parameters',
        code: 400
      }
    };

    examples.push(errorExample);

    return examples;
  }

  generateExampleInput(schema) {
    if (!schema || !schema.properties) {
      return {};
    }

    const example = {};
    
    for (const [propName, propSchema] of Object.entries(schema.properties)) {
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
        return 'example';
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

  generateExampleOutput(tool) {
    return {
      success: true,
      data: tool.method === 'GET' && tool.name.includes('List') 
        ? [{ id: '1', name: 'Example item' }]
        : { id: '123', status: 'completed' },
      message: 'Operation completed successfully'
    };
  }

  async generateExamples(serviceName, tools) {
    const examples = [];

    // Service connection example
    examples.push({
      name: 'Service Connection',
      description: `How to connect to ${serviceName}`,
      code: this.generateConnectionExample(serviceName)
    });

    // Tool usage examples
    const sampleTools = tools.slice(0, 3); // First 3 tools
    for (const tool of sampleTools) {
      examples.push({
        name: `Using ${tool.name}`,
        description: tool.description,
        code: this.generateToolUsageExample(tool, serviceName)
      });
    }

    return examples;
  }

  generateConnectionExample(serviceName) {
    const config = SERVICE_CONFIGS[serviceName];
    const envVars = config.envVars.map(v => `${v}=your_${v.toLowerCase()}`).join('\n');
    
    return `# Environment setup
${envVars}

# Start MCP server
node examples/mcp-${serviceName}-server.js

# Or use multi-host
node examples/mcp-multi-host.js --config services.json`;
  }

  generateToolUsageExample(tool, serviceName) {
    const input = this.generateExampleInput(tool.inputSchema);
    
    return `# Call ${tool.name}
{
  "jsonrpc": "2.0",
  "id": "1",
  "method": "tools/call",
  "params": {
    "name": "${serviceName}.${tool.name}",
    "arguments": ${JSON.stringify(input, null, 2)}
  }
}`;
  }

  getBaseUrl(serviceName) {
    switch (serviceName) {
      case 'n8n':
        return process.env.N8N_API_URL || 'http://localhost:5678/api/v1';
      case 'hostinger':
        return 'https://api.hostinger.com';
      case 'adobe_pdf':
        return 'https://pdf-services.adobe.io';
      default:
        return 'https://api.example.com';
    }
  }

  getFilters(serviceName) {
    // Service-specific filtering rules
    switch (serviceName) {
      case 'hostinger':
        return {
          excludeTags: ['internal', 'deprecated']
        };
      case 'n8n':
        return {
          includeTags: ['workflows', 'credentials', 'executions']
        };
      default:
        return {};
    }
  }

  getAuthType(serviceName) {
    switch (serviceName) {
      case 'n8n':
        return 'apikey';
      case 'hostinger':
      case 'adobe_pdf':
        return 'bearer';
      default:
        return 'bearer';
    }
  }

  async parseYaml(content) {
    try {
      const yaml = await import('yaml');
      return yaml.parse(content);
    } catch {
      // Fallback to simple YAML parsing
      throw new Error('YAML parsing not available');
    }
  }

  async getPackageVersion() {
    try {
      const packagePath = path.join(projectRoot, 'package.json');
      const packageContent = await fs.readFile(packagePath, 'utf8');
      const packageJson = JSON.parse(packageContent);
      return packageJson.version;
    } catch {
      return '1.0.0';
    }
  }

  async validateSchemas() {
    console.log('🔍 Validating generated schemas...');
    
    let validationErrors = 0;
    
    for (const [serviceName, schema] of Object.entries(this.schemas)) {
      try {
        // Validate schema structure
        this.validateSchemaStructure(schema);
        
        // Validate tool schemas
        for (const tool of schema.tools) {
          this.validateToolSchema(tool, serviceName);
        }
        
        console.log(`✅ Schema validation passed for ${serviceName}`);
      } catch (error) {
        validationErrors++;
        this.stats.errors.push({
          service: serviceName,
          error: `Validation failed: ${error.message}`
        });
        console.error(`❌ Schema validation failed for ${serviceName}:`, error.message);
      }
    }
    
    if (validationErrors === 0) {
      console.log('✅ All schemas passed validation');
    } else {
      console.log(`❌ ${validationErrors} schemas failed validation`);
    }
  }

  validateSchemaStructure(schema) {
    const required = ['service', 'description', 'category', 'tools', 'toolCount'];
    for (const field of required) {
      if (!schema[field]) {
        throw new Error(`Missing required field: ${field}`);
      }
    }
    
    if (!Array.isArray(schema.tools)) {
      throw new Error('tools must be an array');
    }
    
    if (schema.tools.length !== schema.toolCount) {
      throw new Error('toolCount does not match actual tool count');
    }
  }

  validateToolSchema(tool, serviceName) {
    const required = ['name', 'description', 'inputSchema', 'method'];
    for (const field of required) {
      if (!tool[field]) {
        throw new Error(`Tool ${tool.name || 'unknown'} missing required field: ${field}`);
      }
    }
    
    if (!tool.inputSchema.type) {
      throw new Error(`Tool ${tool.name} inputSchema missing type`);
    }
    
    if (!['GET', 'POST', 'PUT', 'PATCH', 'DELETE'].includes(tool.method)) {
      throw new Error(`Tool ${tool.name} has invalid method: ${tool.method}`);
    }
  }

  async writeOutput() {
    const outputPath = path.resolve(this.options.output);
    const outputDir = path.dirname(outputPath);
    
    // Ensure output directory exists
    await fs.mkdir(outputDir, { recursive: true });
    
    const output = {
      metadata: {
        generatedAt: new Date().toISOString(),
        generator: 'openapi-mcp-server/tool-schema-generator',
        version: await this.getPackageVersion(),
        totalServices: this.stats.totalServices,
        totalTools: this.stats.totalTools
      },
      services: this.schemas,
      stats: this.stats
    };
    
    let content;
    switch (this.options.format) {
      case 'yaml':
        try {
          const yaml = await import('yaml');
          content = yaml.stringify(output);
        } catch {
          throw new Error('YAML output requires yaml package');
        }
        break;
      case 'markdown':
        content = this.generateMarkdownOutput(output);
        break;
      case 'json':
      default:
        content = JSON.stringify(output, null, 2);
        break;
    }
    
    await fs.writeFile(outputPath, content, 'utf8');
    console.log(`📄 Schemas written to ${outputPath}`);
  }

  generateMarkdownOutput(output) {
    let md = `# MCP Tool Schemas\n\n`;
    md += `Generated at: ${output.metadata.generatedAt}\n`;
    md += `Total Services: ${output.metadata.totalServices}\n`;
    md += `Total Tools: ${output.metadata.totalTools}\n\n`;
    
    for (const [serviceName, schema] of Object.entries(output.services)) {
      md += `## ${schema.service}\n\n`;
      md += `${schema.description}\n\n`;
      md += `- **Category**: ${schema.category}\n`;
      md += `- **Tools**: ${schema.toolCount}\n`;
      md += `- **Authentication**: ${schema.authentication.type}\n\n`;
      
      md += `### Tools\n\n`;
      for (const tool of schema.tools) {
        md += `#### ${tool.name}\n\n`;
        md += `${tool.description}\n\n`;
        md += `- **Method**: ${tool.method}\n`;
        md += `- **Category**: ${tool.category}\n`;
        if (tool.tags.length > 0) {
          md += `- **Tags**: ${tool.tags.join(', ')}\n`;
        }
        md += `\n`;
      }
      md += `\n`;
    }
    
    return md;
  }

  printSummary() {
    console.log('\n📊 Generation Summary:');
    console.log(`✅ Services processed: ${this.stats.totalServices}`);
    console.log(`🔧 Tools generated: ${this.stats.totalTools}`);
    
    if (this.stats.warnings.length > 0) {
      console.log(`⚠️  Warnings: ${this.stats.warnings.length}`);
      for (const warning of this.stats.warnings) {
        console.log(`   - ${warning.service}: ${warning.warning}`);
      }
    }
    
    if (this.stats.errors.length > 0) {
      console.log(`❌ Errors: ${this.stats.errors.length}`);
      for (const error of this.stats.errors) {
        console.log(`   - ${error.service}: ${error.error}`);
      }
    }
    
    console.log('\n🎉 Tool schema generation complete!');
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
      case '--format':
        options.format = args[++i];
        break;
      case '--service':
        options.service = args[++i];
        break;
      case '--validate':
        options.validate = true;
        break;
      case '--include-examples':
        options.includeExamples = true;
        break;
      case '--help':
        console.log(`
Tool Schema Generator

Usage: node examples/scripts/generate-tool-schemas.js [options]

Options:
  --output <file>     Output file for generated schemas (default: schemas/tool-schemas.json)
  --format <format>   Output format: json, yaml, markdown (default: json)
  --validate          Run validation tests on generated schemas
  --service <name>    Generate schemas for specific service only
  --include-examples  Include usage examples in schemas
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
    const generator = new ToolSchemaGenerator(options);
    await generator.generateAllSchemas();
  } catch (error) {
    console.error('❌ Generation failed:', error.message);
    process.exit(1);
  }
}

if (import.meta.url === `file://${process.argv[1]}`) {
  main().catch(console.error);
}

export { ToolSchemaGenerator };