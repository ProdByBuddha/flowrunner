#!/usr/bin/env node

/**
 * Configuration Schema Generator
 * 
 * Extracts all environment variables from the codebase.
 * Generates structured configuration schema with descriptions and examples.
 * Creates validation functions for configuration completeness.
 * 
 * Usage:
 *   node examples/scripts/generate-config-schemas.js [options]
 * 
 * Options:
 *   --output <file>     Output file for configuration schema (default: schemas/config-schema.json)
 *   --format <format>   Output format: json, yaml, markdown, env (default: json)
 *   --validate          Run validation tests on generated schemas
 *   --scan-dirs <dirs>  Comma-separated directories to scan (default: lib,examples,tests)
 *   --include-examples  Include example configurations
 */

import fs from 'fs/promises';
import path from 'path';
import { fileURLToPath } from 'url';

const __filename = fileURLToPath(import.meta.url);
const __dirname = path.dirname(__filename);
const projectRoot = path.resolve(__dirname, '../..');

// Known environment variable patterns and their metadata
const ENV_VAR_PATTERNS = {
  // Authentication
  'API_KEY': { category: 'authentication', type: 'string', sensitive: true, description: 'API key for service authentication' },
  'ACCESS_TOKEN': { category: 'authentication', type: 'string', sensitive: true, description: 'Access token for API authentication' },
  'BEARER_TOKEN': { category: 'authentication', type: 'string', sensitive: true, description: 'Bearer token for authorization' },
  'CLIENT_ID': { category: 'authentication', type: 'string', sensitive: false, description: 'OAuth client identifier' },
  'CLIENT_SECRET': { category: 'authentication', type: 'string', sensitive: true, description: 'OAuth client secret' },
  'USERNAME': { category: 'authentication', type: 'string', sensitive: false, description: 'Username for basic authentication' },
  'PASSWORD': { category: 'authentication', type: 'string', sensitive: true, description: 'Password for basic authentication' },
  
  // URLs and endpoints
  'URL': { category: 'endpoints', type: 'url', sensitive: false, description: 'Service endpoint URL' },
  'BASE_URL': { category: 'endpoints', type: 'url', sensitive: false, description: 'Base URL for API requests' },
  'HOST': { category: 'endpoints', type: 'string', sensitive: false, description: 'Service hostname' },
  'PORT': { category: 'endpoints', type: 'integer', sensitive: false, description: 'Service port number' },
  
  // File paths
  'FILE': { category: 'files', type: 'path', sensitive: false, description: 'File path' },
  'DIR': { category: 'files', type: 'path', sensitive: false, description: 'Directory path' },
  'PATH': { category: 'files', type: 'path', sensitive: false, description: 'File or directory path' },
  
  // Configuration
  'TIMEOUT': { category: 'configuration', type: 'integer', sensitive: false, description: 'Timeout in milliseconds' },
  'LIMIT': { category: 'configuration', type: 'integer', sensitive: false, description: 'Rate limit or item limit' },
  'DEBUG': { category: 'configuration', type: 'boolean', sensitive: false, description: 'Enable debug mode' },
  'VERBOSE': { category: 'configuration', type: 'boolean', sensitive: false, description: 'Enable verbose logging' },
  'LOG_LEVEL': { category: 'configuration', type: 'string', sensitive: false, description: 'Logging level' },
  
  // Security
  'ALLOWED': { category: 'security', type: 'array', sensitive: false, description: 'Allowed items list' },
  'BLOCKED': { category: 'security', type: 'array', sensitive: false, description: 'Blocked items list' },
  'WHITELIST': { category: 'security', type: 'array', sensitive: false, description: 'Whitelist of allowed items' },
  'BLACKLIST': { category: 'security', type: 'array', sensitive: false, description: 'Blacklist of blocked items' }
};

// Service-specific configurations
const SERVICE_CONFIGS = {
  n8n: {
    name: 'n8n',
    description: 'n8n workflow automation platform',
    requiredVars: ['N8N_API_KEY', 'N8N_API_URL'],
    optionalVars: ['N8N_BEARER_TOKEN', 'N8N_BASIC_AUTH_USER', 'N8N_BASIC_AUTH_PASS'],
    examples: {
      N8N_API_KEY: 'n8n_api_1234567890abcdef',
      N8N_API_URL: 'https://your-n8n-instance.com/api/v1',
      N8N_BEARER_TOKEN: 'your_bearer_token_here'
    }
  },
  hostinger: {
    name: 'Hostinger',
    description: 'Hostinger cloud hosting and domain management',
    requiredVars: ['HOSTINGER_API_TOKEN'],
    optionalVars: ['HOSTINGER_PROFILE'],
    examples: {
      HOSTINGER_API_TOKEN: 'your_hostinger_api_token_here',
      HOSTINGER_PROFILE: 'curated'
    }
  },
  adobe_pdf: {
    name: 'Adobe PDF Services',
    description: 'Adobe PDF Services for document processing',
    requiredVars: ['ADOBE_ACCESS_TOKEN'],
    optionalVars: [],
    examples: {
      ADOBE_ACCESS_TOKEN: 'your_adobe_access_token_here'
    }
  },
  docker: {
    name: 'Docker',
    description: 'Docker container and image management',
    requiredVars: [],
    optionalVars: ['DOCKER_HOST', 'DOCKER_ALLOW_RUN', 'DOCKER_ALLOWED_IMAGES', 'DEBUG_DOCKER'],
    examples: {
      DOCKER_HOST: 'unix:///var/run/docker.sock',
      DOCKER_ALLOW_RUN: '1',
      DOCKER_ALLOWED_IMAGES: 'nginx,redis,postgres',
      DEBUG_DOCKER: '1'
    }
  },
  openapi: {
    name: 'Generic OpenAPI',
    description: 'Generic OpenAPI service integration',
    requiredVars: ['OPENAPI_SPEC_URL'],
    optionalVars: [
      'OPENAPI_SPEC_FILE', 'OPENAPI_BASE_URL', 'OPENAPI_API_KEY', 
      'OPENAPI_BEARER_TOKEN', 'OPENAPI_BASIC_USER', 'OPENAPI_BASIC_PASS',
      'OPENAPI_MCP_ALLOWED_METHODS', 'OPENAPI_MCP_RATE_LIMIT'
    ],
    examples: {
      OPENAPI_SPEC_URL: 'https://api.example.com/openapi.json',
      OPENAPI_BASE_URL: 'https://api.example.com',
      OPENAPI_API_KEY: 'your_api_key_here',
      OPENAPI_BEARER_TOKEN: 'your_bearer_token_here'
    }
  }
};

class ConfigSchemaGenerator {
  constructor(options = {}) {
    this.options = {
      output: options.output || 'schemas/config-schema.json',
      format: options.format || 'json',
      validate: options.validate || false,
      scanDirs: options.scanDirs ? options.scanDirs.split(',') : ['lib', 'examples', 'tests'],
      includeExamples: options.includeExamples || false,
      ...options
    };
    this.envVars = new Map();
    this.configSchema = {};
    this.stats = {
      filesScanned: 0,
      envVarsFound: 0,
      servicesConfigured: 0,
      errors: []
    };
  }

  async generateConfigSchema() {
    console.log('🔧 Generating configuration schema...');
    
    // Scan codebase for environment variables
    await this.scanCodebase();
    
    // Load existing service configurations
    await this.loadServiceConfigs();
    
    // Generate comprehensive schema
    await this.buildConfigSchema();
    
    if (this.options.validate) {
      await this.validateSchema();
    }
    
    await this.writeSchema();
    this.printSummary();
  }

  async scanCodebase() {
    console.log('📁 Scanning codebase for environment variables...');
    
    for (const dir of this.options.scanDirs) {
      const dirPath = path.join(projectRoot, dir);
      try {
        await this.scanDirectory(dirPath);
      } catch (error) {
        console.warn(`⚠️  Failed to scan directory ${dir}: ${error.message}`);
      }
    }
  }

  async scanDirectory(dirPath) {
    try {
      const entries = await fs.readdir(dirPath, { withFileTypes: true });
      
      for (const entry of entries) {
        const fullPath = path.join(dirPath, entry.name);
        
        if (entry.isDirectory()) {
          // Skip node_modules and other irrelevant directories
          if (!['node_modules', '.git', '.vscode', 'logs'].includes(entry.name)) {
            await this.scanDirectory(fullPath);
          }
        } else if (entry.isFile() && this.shouldScanFile(entry.name)) {
          await this.scanFile(fullPath);
        }
      }
    } catch (error) {
      console.warn(`⚠️  Failed to read directory ${dirPath}: ${error.message}`);
    }
  }

  shouldScanFile(filename) {
    const extensions = ['.js', '.ts', '.json', '.yaml', '.yml', '.md'];
    return extensions.some(ext => filename.endsWith(ext));
  }

  async scanFile(filePath) {
    try {
      const content = await fs.readFile(filePath, 'utf8');
      this.extractEnvVars(content, filePath);
      this.stats.filesScanned++;
    } catch (error) {
      this.stats.errors.push({
        file: filePath,
        error: error.message
      });
    }
  }

  extractEnvVars(content, filePath) {
    // Patterns to match environment variable usage
    const patterns = [
      /process\.env\.([A-Z_][A-Z0-9_]*)/g,           // process.env.VAR_NAME
      /process\.env\[['"]([A-Z_][A-Z0-9_]*)['"]]/g,  // process.env['VAR_NAME']
      /\$\{([A-Z_][A-Z0-9_]*)\}/g,                   // ${VAR_NAME}
      /\$([A-Z_][A-Z0-9_]*)/g,                       // $VAR_NAME
      /env:\s*([A-Z_][A-Z0-9_]*)/g,                  // env: VAR_NAME (YAML)
      /['"]([A-Z_][A-Z0-9_]*)['"]:\s*process\.env/g  // "VAR_NAME": process.env
    ];

    for (const pattern of patterns) {
      let match;
      while ((match = pattern.exec(content)) !== null) {
        const varName = match[1];
        if (varName && varName.length > 1) {
          this.addEnvVar(varName, filePath, content, match.index);
        }
      }
    }
  }

  addEnvVar(varName, filePath, content, index) {
    if (!this.envVars.has(varName)) {
      this.envVars.set(varName, {
        name: varName,
        usages: [],
        category: this.categorizeEnvVar(varName),
        type: this.inferType(varName, content, index),
        sensitive: this.isSensitive(varName),
        description: this.generateDescription(varName),
        examples: this.generateExamples(varName)
      });
      this.stats.envVarsFound++;
    }

    const envVar = this.envVars.get(varName);
    envVar.usages.push({
      file: path.relative(projectRoot, filePath),
      context: this.extractContext(content, index)
    });
  }

  categorizeEnvVar(varName) {
    for (const [pattern, metadata] of Object.entries(ENV_VAR_PATTERNS)) {
      if (varName.includes(pattern)) {
        return metadata.category;
      }
    }
    
    // Service-specific categorization
    if (varName.startsWith('N8N_')) return 'n8n';
    if (varName.startsWith('HOSTINGER_')) return 'hostinger';
    if (varName.startsWith('ADOBE_')) return 'adobe';
    if (varName.startsWith('DOCKER_')) return 'docker';
    if (varName.startsWith('OPENAPI_')) return 'openapi';
    
    return 'general';
  }

  inferType(varName, content, index) {
    // Check for type hints in the surrounding context
    const context = this.extractContext(content, index, 200);
    
    if (context.includes('parseInt') || context.includes('Number(')) return 'integer';
    if (context.includes('parseFloat')) return 'number';
    if (context.includes('JSON.parse')) return 'json';
    if (context.includes('split(')) return 'array';
    if (varName.includes('URL') || varName.includes('ENDPOINT')) return 'url';
    if (varName.includes('PATH') || varName.includes('FILE') || varName.includes('DIR')) return 'path';
    if (varName.includes('PORT')) return 'integer';
    if (varName.includes('TIMEOUT') || varName.includes('LIMIT')) return 'integer';
    if (varName.includes('DEBUG') || varName.includes('ENABLE')) return 'boolean';
    
    return 'string';
  }

  isSensitive(varName) {
    const sensitivePatterns = [
      'KEY', 'TOKEN', 'SECRET', 'PASSWORD', 'PASS', 'AUTH', 'CREDENTIAL'
    ];
    return sensitivePatterns.some(pattern => varName.includes(pattern));
  }

  generateDescription(varName) {
    // Try to match against known patterns
    for (const [pattern, metadata] of Object.entries(ENV_VAR_PATTERNS)) {
      if (varName.includes(pattern)) {
        return metadata.description;
      }
    }
    
    // Generate description based on variable name
    const parts = varName.toLowerCase().split('_');
    const service = parts[0];
    const purpose = parts.slice(1).join(' ');
    
    if (SERVICE_CONFIGS[service]) {
      return `${purpose} for ${SERVICE_CONFIGS[service].description}`;
    }
    
    return `Configuration variable: ${purpose.replace(/_/g, ' ')}`;
  }

  generateExamples(varName) {
    // Check service-specific examples first
    for (const config of Object.values(SERVICE_CONFIGS)) {
      if (config.examples[varName]) {
        return [config.examples[varName]];
      }
    }
    
    // Generate generic examples based on type and name
    if (varName.includes('URL')) return ['https://api.example.com', 'http://localhost:3000'];
    if (varName.includes('PORT')) return [3000, 8080, 5432];
    if (varName.includes('TIMEOUT')) return [5000, 10000, 30000];
    if (varName.includes('LIMIT')) return [100, 1000, 10000];
    if (varName.includes('DEBUG')) return ['true', 'false', '1', '0'];
    if (varName.includes('KEY') || varName.includes('TOKEN')) return ['your_api_key_here', 'abc123def456'];
    if (varName.includes('HOST')) return ['localhost', 'api.example.com', '127.0.0.1'];
    if (varName.includes('PATH')) return ['/path/to/file', './config/file.json'];
    
    return ['example_value'];
  }

  extractContext(content, index, length = 100) {
    const start = Math.max(0, index - length);
    const end = Math.min(content.length, index + length);
    return content.slice(start, end);
  }

  async loadServiceConfigs() {
    console.log('📋 Loading service configurations...');
    
    // Load existing service configuration files
    const configFiles = [
      'services.default.json',
      'services.dynamic.json',
      'services.example.json'
    ];
    
    for (const configFile of configFiles) {
      try {
        const configPath = path.join(projectRoot, configFile);
        const configContent = await fs.readFile(configPath, 'utf8');
        const config = JSON.parse(configContent);
        this.processServiceConfig(config, configFile);
      } catch (error) {
        console.warn(`⚠️  Failed to load ${configFile}: ${error.message}`);
      }
    }
  }

  processServiceConfig(config, filename) {
    if (config.services && Array.isArray(config.services)) {
      for (const service of config.services) {
        this.extractServiceEnvVars(service, filename);
      }
    }
  }

  extractServiceEnvVars(service, filename) {
    // Extract environment variables from service configuration
    const serviceStr = JSON.stringify(service);
    this.extractEnvVars(serviceStr, filename);
    
    // Add service-specific metadata
    if (service.name && SERVICE_CONFIGS[service.name]) {
      const serviceConfig = SERVICE_CONFIGS[service.name];
      
      for (const varName of [...serviceConfig.requiredVars, ...serviceConfig.optionalVars]) {
        if (!this.envVars.has(varName)) {
          this.addEnvVar(varName, filename, serviceStr, 0);
        }
      }
    }
  }

  async buildConfigSchema() {
    console.log('🏗️  Building configuration schema...');
    
    this.configSchema = {
      $schema: 'http://json-schema.org/draft-07/schema#',
      title: 'OpenAPI MCP Server Configuration Schema',
      description: 'Configuration schema for the OpenAPI MCP Server with all supported environment variables',
      type: 'object',
      properties: {
        metadata: {
          type: 'object',
          properties: {
            generatedAt: { type: 'string', format: 'date-time' },
            generator: { type: 'string', const: 'openapi-mcp-server/config-schema-generator' },
            version: { type: 'string' },
            totalVariables: { type: 'integer' },
            categories: { type: 'array', items: { type: 'string' } }
          }
        },
        services: {
          type: 'object',
          description: 'Service-specific configuration schemas',
          properties: {}
        },
        environment: {
          type: 'object',
          description: 'Environment variable definitions',
          properties: {}
        },
        categories: {
          type: 'object',
          description: 'Environment variables grouped by category',
          properties: {}
        }
      }
    };

    // Add metadata
    this.configSchema.properties.metadata.properties = {
      ...this.configSchema.properties.metadata.properties,
      generatedAt: { type: 'string', default: new Date().toISOString() },
      totalVariables: { type: 'integer', default: this.envVars.size },
      categories: { 
        type: 'array', 
        default: [...new Set([...this.envVars.values()].map(v => v.category))]
      }
    };

    // Build service schemas
    await this.buildServiceSchemas();
    
    // Build environment variable schemas
    await this.buildEnvironmentSchemas();
    
    // Build category schemas
    await this.buildCategorySchemas();
  }

  async buildServiceSchemas() {
    for (const [serviceName, config] of Object.entries(SERVICE_CONFIGS)) {
      const serviceSchema = {
        type: 'object',
        title: config.name,
        description: config.description,
        properties: {
          required: {
            type: 'object',
            title: 'Required Configuration',
            properties: {},
            required: config.requiredVars
          },
          optional: {
            type: 'object',
            title: 'Optional Configuration',
            properties: {}
          }
        }
      };

      // Add required variables
      for (const varName of config.requiredVars) {
        const envVar = this.envVars.get(varName);
        if (envVar) {
          serviceSchema.properties.required.properties[varName] = this.buildVarSchema(envVar);
        }
      }

      // Add optional variables
      for (const varName of config.optionalVars) {
        const envVar = this.envVars.get(varName);
        if (envVar) {
          serviceSchema.properties.optional.properties[varName] = this.buildVarSchema(envVar);
        }
      }

      // Add examples if requested
      if (this.options.includeExamples) {
        serviceSchema.examples = [config.examples];
      }

      this.configSchema.properties.services.properties[serviceName] = serviceSchema;
      this.stats.servicesConfigured++;
    }
  }

  async buildEnvironmentSchemas() {
    for (const [varName, envVar] of this.envVars) {
      this.configSchema.properties.environment.properties[varName] = this.buildVarSchema(envVar);
    }
  }

  async buildCategorySchemas() {
    const categories = {};
    
    for (const [varName, envVar] of this.envVars) {
      if (!categories[envVar.category]) {
        categories[envVar.category] = {
          type: 'object',
          title: `${envVar.category.charAt(0).toUpperCase() + envVar.category.slice(1)} Variables`,
          description: `Environment variables related to ${envVar.category}`,
          properties: {}
        };
      }
      
      categories[envVar.category].properties[varName] = this.buildVarSchema(envVar);
    }
    
    this.configSchema.properties.categories.properties = categories;
  }

  buildVarSchema(envVar) {
    const schema = {
      type: envVar.type === 'integer' ? 'integer' : 
            envVar.type === 'number' ? 'number' :
            envVar.type === 'boolean' ? 'boolean' :
            envVar.type === 'array' ? 'array' : 'string',
      title: envVar.name,
      description: envVar.description
    };

    // Add format for specific types
    if (envVar.type === 'url') {
      schema.format = 'uri';
    } else if (envVar.type === 'path') {
      schema.pattern = '^[^\\0]+$'; // Basic path validation
    }

    // Add examples
    if (envVar.examples && envVar.examples.length > 0) {
      if (envVar.sensitive) {
        schema.examples = ['your_' + envVar.name.toLowerCase() + '_here'];
      } else {
        schema.examples = envVar.examples;
      }
    }

    // Add usage information
    schema.usage = {
      files: envVar.usages.map(u => u.file),
      occurrences: envVar.usages.length
    };

    // Mark sensitive variables
    if (envVar.sensitive) {
      schema.sensitive = true;
      schema.description += ' (sensitive - do not log or expose)';
    }

    return schema;
  }

  async validateSchema() {
    console.log('🔍 Validating configuration schema...');
    
    try {
      // Basic schema validation
      this.validateSchemaStructure();
      
      // Validate service configurations
      this.validateServiceConfigs();
      
      // Validate environment variables
      this.validateEnvironmentVars();
      
      console.log('✅ Configuration schema validation passed');
    } catch (error) {
      console.error('❌ Configuration schema validation failed:', error.message);
      this.stats.errors.push({
        type: 'validation',
        error: error.message
      });
    }
  }

  validateSchemaStructure() {
    const required = ['$schema', 'title', 'description', 'type', 'properties'];
    for (const field of required) {
      if (!this.configSchema[field]) {
        throw new Error(`Missing required schema field: ${field}`);
      }
    }
  }

  validateServiceConfigs() {
    for (const [serviceName, serviceSchema] of Object.entries(this.configSchema.properties.services.properties)) {
      if (!serviceSchema.title || !serviceSchema.description) {
        throw new Error(`Service ${serviceName} missing title or description`);
      }
    }
  }

  validateEnvironmentVars() {
    for (const [varName, varSchema] of Object.entries(this.configSchema.properties.environment.properties)) {
      if (!varSchema.description) {
        throw new Error(`Environment variable ${varName} missing description`);
      }
    }
  }

  async writeSchema() {
    const outputPath = path.resolve(this.options.output);
    const outputDir = path.dirname(outputPath);
    
    // Ensure output directory exists
    await fs.mkdir(outputDir, { recursive: true });
    
    let content;
    switch (this.options.format) {
      case 'yaml':
        try {
          const yaml = await import('yaml');
          content = yaml.stringify(this.configSchema);
        } catch {
          throw new Error('YAML output requires yaml package');
        }
        break;
      case 'markdown':
        content = this.generateMarkdownSchema();
        break;
      case 'env':
        content = this.generateEnvTemplate();
        break;
      case 'json':
      default:
        content = JSON.stringify(this.configSchema, null, 2);
        break;
    }
    
    await fs.writeFile(outputPath, content, 'utf8');
    console.log(`📄 Configuration schema written to ${outputPath}`);
  }

  generateMarkdownSchema() {
    let md = `# Configuration Schema\n\n`;
    md += `Generated at: ${new Date().toISOString()}\n\n`;
    md += `This document describes all configuration options for the OpenAPI MCP Server.\n\n`;
    
    md += `## Summary\n\n`;
    md += `- **Total Variables**: ${this.envVars.size}\n`;
    md += `- **Services**: ${Object.keys(SERVICE_CONFIGS).length}\n`;
    md += `- **Categories**: ${[...new Set([...this.envVars.values()].map(v => v.category))].length}\n\n`;
    
    // Service configurations
    md += `## Service Configurations\n\n`;
    for (const [serviceName, config] of Object.entries(SERVICE_CONFIGS)) {
      md += `### ${config.name}\n\n`;
      md += `${config.description}\n\n`;
      
      if (config.requiredVars.length > 0) {
        md += `**Required Variables:**\n`;
        for (const varName of config.requiredVars) {
          const envVar = this.envVars.get(varName);
          md += `- \`${varName}\`: ${envVar ? envVar.description : 'Configuration variable'}\n`;
        }
        md += `\n`;
      }
      
      if (config.optionalVars.length > 0) {
        md += `**Optional Variables:**\n`;
        for (const varName of config.optionalVars) {
          const envVar = this.envVars.get(varName);
          md += `- \`${varName}\`: ${envVar ? envVar.description : 'Configuration variable'}\n`;
        }
        md += `\n`;
      }
      
      if (this.options.includeExamples && config.examples) {
        md += `**Example Configuration:**\n\`\`\`bash\n`;
        for (const [varName, example] of Object.entries(config.examples)) {
          md += `${varName}="${example}"\n`;
        }
        md += `\`\`\`\n\n`;
      }
    }
    
    // Environment variables by category
    md += `## Environment Variables by Category\n\n`;
    const categories = {};
    for (const [varName, envVar] of this.envVars) {
      if (!categories[envVar.category]) {
        categories[envVar.category] = [];
      }
      categories[envVar.category].push({ varName, envVar });
    }
    
    for (const [category, vars] of Object.entries(categories)) {
      md += `### ${category.charAt(0).toUpperCase() + category.slice(1)}\n\n`;
      
      for (const { varName, envVar } of vars) {
        md += `#### \`${varName}\`\n\n`;
        md += `${envVar.description}\n\n`;
        md += `- **Type**: ${envVar.type}\n`;
        md += `- **Sensitive**: ${envVar.sensitive ? 'Yes' : 'No'}\n`;
        if (envVar.examples.length > 0) {
          md += `- **Examples**: ${envVar.examples.map(e => `\`${e}\``).join(', ')}\n`;
        }
        md += `- **Used in**: ${envVar.usages.length} file(s)\n\n`;
      }
    }
    
    return md;
  }

  generateEnvTemplate() {
    let env = `# OpenAPI MCP Server Configuration\n`;
    env += `# Generated at: ${new Date().toISOString()}\n\n`;
    
    // Group by service
    for (const [serviceName, config] of Object.entries(SERVICE_CONFIGS)) {
      env += `# ${config.name} - ${config.description}\n`;
      
      for (const varName of config.requiredVars) {
        const envVar = this.envVars.get(varName);
        env += `# ${envVar ? envVar.description : 'Required configuration'}\n`;
        if (envVar && envVar.sensitive) {
          env += `${varName}=your_${varName.toLowerCase()}_here\n`;
        } else if (envVar && envVar.examples.length > 0) {
          env += `${varName}=${envVar.examples[0]}\n`;
        } else {
          env += `${varName}=\n`;
        }
      }
      
      if (config.optionalVars.length > 0) {
        env += `\n# Optional configuration\n`;
        for (const varName of config.optionalVars) {
          const envVar = this.envVars.get(varName);
          env += `# ${envVar ? envVar.description : 'Optional configuration'}\n`;
          if (envVar && envVar.examples.length > 0) {
            env += `# ${varName}=${envVar.examples[0]}\n`;
          } else {
            env += `# ${varName}=\n`;
          }
        }
      }
      
      env += `\n`;
    }
    
    return env;
  }

  printSummary() {
    console.log('\n📊 Configuration Schema Generation Summary:');
    console.log(`✅ Files scanned: ${this.stats.filesScanned}`);
    console.log(`🔧 Environment variables found: ${this.stats.envVarsFound}`);
    console.log(`⚙️  Services configured: ${this.stats.servicesConfigured}`);
    
    if (this.stats.errors.length > 0) {
      console.log(`❌ Errors: ${this.stats.errors.length}`);
      for (const error of this.stats.errors.slice(0, 5)) {
        console.log(`   - ${error.file || error.type}: ${error.error}`);
      }
      if (this.stats.errors.length > 5) {
        console.log(`   ... and ${this.stats.errors.length - 5} more errors`);
      }
    }
    
    console.log('\n🎉 Configuration schema generation complete!');
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
      case '--scan-dirs':
        options.scanDirs = args[++i];
        break;
      case '--validate':
        options.validate = true;
        break;
      case '--include-examples':
        options.includeExamples = true;
        break;
      case '--help':
        console.log(`
Configuration Schema Generator

Usage: node examples/scripts/generate-config-schemas.js [options]

Options:
  --output <file>     Output file for configuration schema (default: schemas/config-schema.json)
  --format <format>   Output format: json, yaml, markdown, env (default: json)
  --validate          Run validation tests on generated schemas
  --scan-dirs <dirs>  Comma-separated directories to scan (default: lib,examples,tests)
  --include-examples  Include example configurations
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
    const generator = new ConfigSchemaGenerator(options);
    await generator.generateConfigSchema();
  } catch (error) {
    console.error('❌ Generation failed:', error.message);
    process.exit(1);
  }
}

if (import.meta.url === `file://${process.argv[1]}`) {
  main().catch(console.error);
}

export { ConfigSchemaGenerator };