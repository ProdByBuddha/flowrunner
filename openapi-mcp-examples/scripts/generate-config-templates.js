#!/usr/bin/env node

/**
 * Configuration Template Generator
 * 
 * Generates configuration templates for common use cases.
 * Creates validation scripts for configuration templates.
 * Provides migration guides from existing configurations.
 * 
 * Usage:
 *   node examples/scripts/generate-config-templates.js [options]
 * 
 * Options:
 *   --output-dir <dir>    Output directory for templates (default: examples/config-templates)
 *   --format <format>     Template format: json, yaml (default: json)
 *   --validate            Generate validation scripts
 *   --migration           Generate migration guides
 *   --help                Show this help message
 */

import fs from 'fs/promises';
import path from 'path';
import { fileURLToPath } from 'url';

const __filename = fileURLToPath(import.meta.url);
const __dirname = path.dirname(__filename);
const projectRoot = path.resolve(__dirname, '../..');

// Service configurations for templates
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

class ConfigTemplateGenerator {
  constructor(options = {}) {
    this.options = {
      outputDir: options.outputDir || 'examples/config-templates',
      format: options.format || 'json',
      validate: options.validate || false,
      migration: options.migration || false,
      ...options
    };
    
    this.stats = {
      templatesGenerated: 0,
      validatorsGenerated: 0,
      migrationsGenerated: 0,
      errors: []
    };
  }

  async generateTemplates() {
    console.log('🔧 Generating configuration templates...');
    
    // Ensure output directory exists
    const outputPath = path.resolve(this.options.outputDir);
    await fs.mkdir(outputPath, { recursive: true });
    
    // Generate common use case templates
    await this.generateCommonTemplates();
    
    // Generate validation scripts if requested
    if (this.options.validate) {
      await this.generateValidationScripts();
    }
    
    // Generate migration guides if requested
    if (this.options.migration) {
      await this.generateMigrationGuides();
    }
    
    this.printSummary();
  }

  async generateCommonTemplates() {
    console.log('� Creatring common use case templates...');
    
    // Single service template
    await this.generateSingleServiceTemplate();
    
    // Multi-service template
    await this.generateMultiServiceTemplate();
    
    // Development template
    await this.generateDevelopmentTemplate();
    
    // Production template
    await this.generateProductionTemplate();
    
    // Service-specific templates
    for (const [serviceName, config] of Object.entries(SERVICE_CONFIGS)) {
      await this.generateServiceTemplate(serviceName, config);
    }
  }

  async generateSingleServiceTemplate() {
    const template = {
      "$schema": "../schemas/services-config-schema.json",
      "description": "Single service configuration template - ideal for simple integrations",
      "services": [
        {
          "name": "my-service",
          "type": "openapi",
          "specUrl": "https://api.example.com/openapi.json",
          "baseUrl": "https://api.example.com",
          "auth": {
            "kind": "bearer",
            "env": "MY_SERVICE_API_TOKEN"
          },
          "filters": {}
        }
      ]
    };
    
    await this.writeTemplate('single-service', template);
    await this.generateTemplateReadme('single-service', {
      title: 'Single Service Configuration',
      description: 'Template for integrating a single OpenAPI service',
      envVars: ['MY_SERVICE_API_TOKEN'],
      usage: 'node examples/mcp-multi-host.js --config examples/config-templates/single-service.json'
    });
  }

  async generateMultiServiceTemplate() {
    const template = {
      "$schema": "../schemas/services-config-schema.json",
      "description": "Multi-service configuration template - supports multiple APIs",
      "services": [
        {
          "name": "n8n",
          "type": "openapi",
          "specFile": "../specs/n8n-api.json",
          "baseUrl": "${N8N_API_URL}",
          "auth": {
            "kind": "header",
            "name": "X-N8N-API-KEY",
            "env": "N8N_API_KEY"
          },
          "filters": {}
        },
        {
          "name": "hostinger",
          "type": "openapi",
          "specFile": "../specs/hostinger-api.json",
          "baseUrl": "https://developers.hostinger.com",
          "auth": {
            "kind": "bearer",
            "env": "HOSTINGER_API_TOKEN"
          },
          "filters": {}
        }
      ]
    };
    
    await this.writeTemplate('multi-service', template);
    await this.generateTemplateReadme('multi-service', {
      title: 'Multi-Service Configuration',
      description: 'Template for integrating multiple OpenAPI services',
      envVars: ['N8N_API_KEY', 'N8N_API_URL', 'HOSTINGER_API_TOKEN'],
      usage: 'node examples/mcp-multi-host.js --config examples/config-templates/multi-service.json'
    });
  }

  async generateDevelopmentTemplate() {
    const template = {
      "$schema": "../schemas/services-config-schema.json",
      "description": "Development configuration template - includes debugging and local services",
      "services": [
        {
          "name": "local-api",
          "type": "openapi",
          "specFile": "./specs/local-api.json",
          "baseUrl": "http://localhost:3000",
          "auth": {},
          "filters": {}
        },
        {
          "name": "docker-local",
          "type": "openapi",
          "specUrl": "https://docs.docker.com/reference/api/engine/version/v1.51.yaml",
          "baseUrl": "http://localhost:2375",
          "auth": {},
          "filters": {
            "includeOpsRe": ["^(get|post).*container.*"]
          }
        }
      ]
    };
    
    await this.writeTemplate('development', template);
    await this.generateTemplateReadme('development', {
      title: 'Development Configuration',
      description: 'Template optimized for local development with debugging enabled',
      envVars: ['DEBUG_HTTP=1', 'DEBUG_DOCKER=1'],
      usage: 'DEBUG_HTTP=1 node examples/mcp-multi-host.js --config examples/config-templates/development.json'
    });
  }

  async generateProductionTemplate() {
    const template = {
      "$schema": "../schemas/services-config-schema.json",
      "description": "Production configuration template - optimized for performance and security",
      "services": [
        {
          "name": "n8n",
          "type": "openapi",
          "specFile": "../specs/n8n-api.json",
          "baseUrl": "${N8N_API_URL}",
          "auth": {
            "kind": "header",
            "name": "X-N8N-API-KEY",
            "env": "N8N_API_KEY"
          },
          "filters": {
            "excludeOpsRe": [".*debug.*", ".*test.*"],
            "excludeTagsRe": ["Deprecated", "Internal"]
          }
        },
        {
          "name": "hostinger",
          "type": "openapi",
          "specFile": "../specs/hostinger-api.json",
          "baseUrl": "https://developers.hostinger.com",
          "auth": {
            "kind": "bearer",
            "env": "HOSTINGER_API_TOKEN"
          },
          "filters": {
            "excludeOpsRe": [".*debug.*", ".*test.*"]
          }
        }
      ]
    };
    
    await this.writeTemplate('production', template);
    await this.generateTemplateReadme('production', {
      title: 'Production Configuration',
      description: 'Template optimized for production use with security and performance considerations',
      envVars: ['N8N_API_KEY', 'N8N_API_URL', 'HOSTINGER_API_TOKEN', 'MCP_RATE_LIMIT=100'],
      usage: 'node examples/mcp-multi-host.js --config examples/config-templates/production.json'
    });
  }

  async generateServiceTemplate(serviceName, config) {
    const upperServiceName = serviceName.toUpperCase();
    const template = {
      "$schema": "../schemas/services-config-schema.json",
      "description": `${config.name} service configuration template`,
      "services": [
        {
          "name": serviceName,
          "type": "openapi",
          "specFile": `../specs/${serviceName}-api.json`,
          "baseUrl": config.examples[`${upperServiceName}_API_URL`] || "https://api.example.com",
          "auth": this.generateAuthConfig(serviceName, config),
          "filters": {}
        }
      ]
    };
    
    await this.writeTemplate(`${serviceName}-service`, template);
    await this.generateTemplateReadme(`${serviceName}-service`, {
      title: `${config.name} Configuration`,
      description: `Template for ${config.description}`,
      envVars: [...config.requiredVars, ...config.optionalVars],
      usage: `node examples/mcp-multi-host.js --config examples/config-templates/${serviceName}-service.json`
    });
  }

  generateAuthConfig(serviceName, config) {
    const upperServiceName = serviceName.toUpperCase();
    
    if (config.requiredVars.includes(`${upperServiceName}_API_TOKEN`) || 
        config.requiredVars.includes(`${upperServiceName}_ACCESS_TOKEN`)) {
      return {
        "kind": "bearer",
        "env": `${upperServiceName}_API_TOKEN`
      };
    }
    
    if (config.requiredVars.includes(`${upperServiceName}_API_KEY`)) {
      return {
        "kind": "header",
        "name": "X-API-KEY",
        "env": `${upperServiceName}_API_KEY`
      };
    }
    
    return {};
  }

  async writeTemplate(name, template) {
    const filename = `${name}.${this.options.format}`;
    const filepath = path.join(this.options.outputDir, filename);
    
    let content;
    if (this.options.format === 'yaml') {
      throw new Error('YAML output requires yaml package');
    } else {
      content = JSON.stringify(template, null, 2);
    }
    
    await fs.writeFile(filepath, content, 'utf8');
    this.stats.templatesGenerated++;
    console.log(`✅ Generated template: ${filename}`);
  }

  async generateTemplateReadme(templateName, metadata) {
    const readmePath = path.join(this.options.outputDir, `${templateName}-README.md`);
    
    let readme = `# ${metadata.title}\n\n`;
    readme += `${metadata.description}\n\n`;
    
    if (metadata.envVars && metadata.envVars.length > 0) {
      readme += `## Environment Variables\n\n`;
      readme += `\`\`\`bash\n`;
      for (const envVar of metadata.envVars) {
        if (envVar.includes('=')) {
          readme += `export ${envVar}\n`;
        } else {
          readme += `export ${envVar}="your_value_here"\n`;
        }
      }
      readme += `\`\`\`\n\n`;
    }
    
    readme += `## Usage\n\n`;
    readme += `\`\`\`bash\n`;
    readme += `${metadata.usage}\n`;
    readme += `\`\`\`\n`;
    
    await fs.writeFile(readmePath, readme, 'utf8');
  }

  async generateValidationScripts() {
    console.log('🔍 Generating validation scripts...');
    
    const validatorScript = `#!/usr/bin/env node

/**
 * Configuration Template Validator
 * 
 * Validates configuration templates and environment variables
 */

import fs from 'fs/promises';
import path from 'path';
import { fileURLToPath } from 'url';

const __filename = fileURLToPath(import.meta.url);
const __dirname = path.dirname(__filename);

const SERVICE_CONFIGS = ${JSON.stringify(SERVICE_CONFIGS, null, 2)};

class ConfigValidator {
  constructor() {
    this.errors = [];
    this.warnings = [];
  }

  async validateTemplate(templatePath) {
    console.log(\`🔍 Validating template: \${path.basename(templatePath)}\`);
    
    try {
      const content = await fs.readFile(templatePath, 'utf8');
      const config = JSON.parse(content);
      
      this.validateSchema(config, templatePath);
      this.validateServices(config.services || [], templatePath);
      this.validateEnvironmentVars(config, templatePath);
      
    } catch (error) {
      this.errors.push(\`\${templatePath}: Failed to parse JSON - \${error.message}\`);
    }
  }

  validateSchema(config, templatePath) {
    const requiredFields = ['services'];
    for (const field of requiredFields) {
      if (!config[field]) {
        this.errors.push(\`\${templatePath}: Missing required field '\${field}'\`);
      }
    }
    
    if (config.services && !Array.isArray(config.services)) {
      this.errors.push(\`\${templatePath}: 'services' must be an array\`);
    }
  }

  validateServices(services, templatePath) {
    for (let i = 0; i < services.length; i++) {
      const service = services[i];
      const servicePrefix = \`\${templatePath}:services[\${i}]\`;
      
      // Required fields
      const requiredFields = ['name', 'type'];
      for (const field of requiredFields) {
        if (!service[field]) {
          this.errors.push(\`\${servicePrefix}: Missing required field '\${field}'\`);
        }
      }
      
      // Must have either specUrl or specFile
      if (!service.specUrl && !service.specFile) {
        this.errors.push(\`\${servicePrefix}: Must have either 'specUrl' or 'specFile'\`);
      }
      
      // Validate auth configuration
      if (service.auth && typeof service.auth === 'object') {
        this.validateAuth(service.auth, \`\${servicePrefix}:auth\`);
      }
    }
  }

  validateAuth(auth, prefix) {
    if (auth.kind) {
      const validKinds = ['bearer', 'header', 'apiKey', 'basic'];
      if (!validKinds.includes(auth.kind)) {
        this.errors.push(\`\${prefix}: Invalid auth kind '\${auth.kind}'. Must be one of: \${validKinds.join(', ')}\`);
      }
      
      if (auth.kind === 'header' && !auth.name) {
        this.errors.push(\`\${prefix}: Header auth requires 'name' field\`);
      }
      
      if (auth.kind === 'apiKey' && (!auth.in || !auth.name)) {
        this.errors.push(\`\${prefix}: API key auth requires 'in' and 'name' fields\`);
      }
    }
  }

  validateEnvironmentVars(config, templatePath) {
    const configStr = JSON.stringify(config);
    const envVarPattern = /\\$\\{([A-Z_][A-Z0-9_]*)\\}/g;
    const envVars = new Set();
    
    let match;
    while ((match = envVarPattern.exec(configStr)) !== null) {
      envVars.add(match[1]);
    }
    
    // Check for auth env vars
    if (config.services) {
      for (const service of config.services) {
        if (service.auth && service.auth.env) {
          envVars.add(service.auth.env);
        }
      }
    }
    
    if (envVars.size === 0) {
      this.warnings.push(\`\${templatePath}: No environment variables found\`);
    }
  }

  async validateAll() {
    const templatesDir = path.resolve(__dirname);
    
    try {
      const files = await fs.readdir(templatesDir);
      const templateFiles = files.filter(f => f.endsWith('.json'));
      
      for (const file of templateFiles) {
        await this.validateTemplate(path.join(templatesDir, file));
      }
      
      this.printResults();
      
    } catch (error) {
      console.error('❌ Failed to validate templates:', error.message);
      process.exit(1);
    }
  }

  printResults() {
    console.log('\\n📊 Validation Results:');
    
    if (this.errors.length === 0 && this.warnings.length === 0) {
      console.log('✅ All templates are valid!');
    } else {
      if (this.errors.length > 0) {
        console.log(\`❌ Errors: \${this.errors.length}\`);
        for (const error of this.errors) {
          console.log(\`   - \${error}\`);
        }
      }
      
      if (this.warnings.length > 0) {
        console.log(\`⚠️  Warnings: \${this.warnings.length}\`);
        for (const warning of this.warnings) {
          console.log(\`   - \${warning}\`);
        }
      }
      
      if (this.errors.length > 0) {
        process.exit(1);
      }
    }
  }
}

// CLI handling
if (import.meta.url === \`file://\${process.argv[1]}\`) {
  const validator = new ConfigValidator();
  validator.validateAll().catch(console.error);
}

export { ConfigValidator };
`;

    const validatorPath = path.join(this.options.outputDir, 'validate-templates.js');
    await fs.writeFile(validatorPath, validatorScript, 'utf8');
    await fs.chmod(validatorPath, 0o755);
    
    this.stats.validatorsGenerated++;
    console.log('✅ Generated validation script: validate-templates.js');
  }

  async generateMigrationGuides() {
    console.log('📚 Generating migration guides...');
    
    const migrationGuide = `# Configuration Migration Guide

This guide helps you migrate from existing configurations to the new template system.

## Migration Scenarios

### From services.example.json to Templates

If you're currently using \`services.example.json\`, you can migrate to specific templates:

1. **Single Service Migration**:
   - If you have only one service, use the \`single-service.json\` template
   - Copy your service configuration and update environment variables

2. **Multi-Service Migration**:
   - If you have multiple services, use the \`multi-service.json\` template
   - Migrate each service individually

3. **Service-Specific Migration**:
   - Use service-specific templates (e.g., \`n8n-service.json\`, \`hostinger-service.json\`)
   - These templates include optimized configurations for each service

### Migration Steps

#### Step 1: Identify Your Current Configuration

\`\`\`bash
# Check your current configuration
cat services.example.json | jq '.services[].name'
\`\`\`

#### Step 2: Choose the Right Template

- **1 service**: Use \`single-service.json\`
- **2-3 services**: Use \`multi-service.json\`
- **Production**: Use \`production.json\`
- **Development**: Use \`development.json\`
- **Specific service**: Use \`{service}-service.json\`

#### Step 3: Copy and Customize

\`\`\`bash
# Copy the template
cp examples/config-templates/multi-service.json my-services.json

# Edit the configuration
nano my-services.json
\`\`\`

#### Step 4: Update Environment Variables

\`\`\`bash
# Check what environment variables are needed
node examples/config-templates/validate-templates.js

# Set up your environment
export N8N_API_KEY="your_key_here"
export N8N_API_URL="https://your-n8n.com/api/v1"
\`\`\`

#### Step 5: Test the Configuration

\`\`\`bash
# Test with the new configuration
node examples/mcp-multi-host.js --config my-services.json --once tools/list
\`\`\`

## Common Migration Issues

### Issue: Missing Environment Variables

**Problem**: Template uses environment variables that aren't set.

**Solution**: 
\`\`\`bash
# Check what variables are needed
grep -o '\\$\\{[^}]*\\}' my-services.json
grep -o '"env": "[^"]*"' my-services.json

# Set the variables
export VARIABLE_NAME="value"
\`\`\`

### Issue: Spec File Paths

**Problem**: Template references spec files that don't exist.

**Solution**:
\`\`\`bash
# Check if spec files exist
ls -la specs/

# Update paths in your configuration
sed -i 's|../specs/|./specs/|g' my-services.json
\`\`\`

### Issue: Authentication Configuration

**Problem**: Auth configuration doesn't match your API requirements.

**Solution**: Update the auth section based on your API:

\`\`\`json
// Bearer token
{
  "auth": {
    "kind": "bearer",
    "env": "YOUR_API_TOKEN"
  }
}

// API key in header
{
  "auth": {
    "kind": "header",
    "name": "X-API-KEY",
    "env": "YOUR_API_KEY"
  }
}

// API key in query
{
  "auth": {
    "kind": "apiKey",
    "in": "query",
    "name": "api_key",
    "env": "YOUR_API_KEY"
  }
}
\`\`\`

## Validation

Always validate your migrated configuration:

\`\`\`bash
# Validate the configuration
node examples/config-templates/validate-templates.js

# Test the configuration
node examples/mcp-multi-host.js --config my-services.json --once tools/list
\`\`\`

## Rollback

If migration fails, you can always rollback:

\`\`\`bash
# Use your original configuration
node examples/mcp-multi-host.js --config services.example.json
\`\`\`

## Getting Help

If you encounter issues during migration:

1. Check the template README files for specific guidance
2. Validate your configuration with the validation script
3. Test with a minimal configuration first
4. Check the logs for specific error messages
`;

    const migrationPath = path.join(this.options.outputDir, 'MIGRATION.md');
    await fs.writeFile(migrationPath, migrationGuide, 'utf8');
    
    this.stats.migrationsGenerated++;
    console.log('✅ Generated migration guide: MIGRATION.md');
  }

  printSummary() {
    console.log('\n📊 Configuration Template Generation Summary:');
    console.log(`✅ Templates generated: ${this.stats.templatesGenerated}`);
    
    if (this.options.validate) {
      console.log(`🔍 Validators generated: ${this.stats.validatorsGenerated}`);
    }
    
    if (this.options.migration) {
      console.log(`📚 Migration guides generated: ${this.stats.migrationsGenerated}`);
    }
    
    if (this.stats.errors.length > 0) {
      console.log(`❌ Errors: ${this.stats.errors.length}`);
      for (const error of this.stats.errors.slice(0, 5)) {
        console.log(`   - ${error}`);
      }
      if (this.stats.errors.length > 5) {
        console.log(`   ... and ${this.stats.errors.length - 5} more errors`);
      }
    }
    
    console.log('\n🎉 Configuration template generation complete!');
    console.log(`📁 Templates available in: ${this.options.outputDir}`);
  }
}

// CLI handling
async function main() {
  const args = process.argv.slice(2);
  const options = {};
  
  for (let i = 0; i < args.length; i++) {
    const arg = args[i];
    switch (arg) {
      case '--output-dir':
        options.outputDir = args[++i];
        break;
      case '--format':
        options.format = args[++i];
        break;
      case '--validate':
        options.validate = true;
        break;
      case '--migration':
        options.migration = true;
        break;
      case '--help':
        console.log(`
Configuration Template Generator

Usage: node examples/scripts/generate-config-templates.js [options]

Options:
  --output-dir <dir>    Output directory for templates (default: examples/config-templates)
  --format <format>     Template format: json, yaml (default: json)
  --validate            Generate validation scripts
  --migration           Generate migration guides
  --help                Show this help message

Examples:
  # Generate basic templates
  node examples/scripts/generate-config-templates.js

  # Generate with validation scripts
  node examples/scripts/generate-config-templates.js --validate

  # Generate with migration guides
  node examples/scripts/generate-config-templates.js --migration

  # Generate everything in custom directory
  node examples/scripts/generate-config-templates.js --output-dir ./my-templates --validate --migration
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
    const generator = new ConfigTemplateGenerator(options);
    await generator.generateTemplates();
  } catch (error) {
    console.error('❌ Generation failed:', error.message);
    process.exit(1);
  }
}

if (import.meta.url === `file://${process.argv[1]}`) {
  main().catch(console.error);
}

export { ConfigTemplateGenerator };