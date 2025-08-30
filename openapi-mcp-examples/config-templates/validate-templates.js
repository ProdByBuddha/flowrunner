#!/usr/bin/env node

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

const SERVICE_CONFIGS = {
  "n8n": {
    "name": "n8n",
    "description": "n8n workflow automation platform",
    "requiredVars": [
      "N8N_API_KEY",
      "N8N_API_URL"
    ],
    "optionalVars": [
      "N8N_BEARER_TOKEN",
      "N8N_BASIC_AUTH_USER",
      "N8N_BASIC_AUTH_PASS"
    ],
    "examples": {
      "N8N_API_KEY": "n8n_api_1234567890abcdef",
      "N8N_API_URL": "https://your-n8n-instance.com/api/v1",
      "N8N_BEARER_TOKEN": "your_bearer_token_here"
    }
  },
  "hostinger": {
    "name": "Hostinger",
    "description": "Hostinger cloud hosting and domain management",
    "requiredVars": [
      "HOSTINGER_API_TOKEN"
    ],
    "optionalVars": [
      "HOSTINGER_PROFILE"
    ],
    "examples": {
      "HOSTINGER_API_TOKEN": "your_hostinger_api_token_here",
      "HOSTINGER_PROFILE": "curated"
    }
  },
  "adobe_pdf": {
    "name": "Adobe PDF Services",
    "description": "Adobe PDF Services for document processing",
    "requiredVars": [
      "ADOBE_ACCESS_TOKEN"
    ],
    "optionalVars": [],
    "examples": {
      "ADOBE_ACCESS_TOKEN": "your_adobe_access_token_here"
    }
  },
  "docker": {
    "name": "Docker",
    "description": "Docker container and image management",
    "requiredVars": [],
    "optionalVars": [
      "DOCKER_HOST",
      "DOCKER_ALLOW_RUN",
      "DOCKER_ALLOWED_IMAGES",
      "DEBUG_DOCKER"
    ],
    "examples": {
      "DOCKER_HOST": "unix:///var/run/docker.sock",
      "DOCKER_ALLOW_RUN": "1",
      "DOCKER_ALLOWED_IMAGES": "nginx,redis,postgres",
      "DEBUG_DOCKER": "1"
    }
  },
  "openapi": {
    "name": "Generic OpenAPI",
    "description": "Generic OpenAPI service integration",
    "requiredVars": [
      "OPENAPI_SPEC_URL"
    ],
    "optionalVars": [
      "OPENAPI_SPEC_FILE",
      "OPENAPI_BASE_URL",
      "OPENAPI_API_KEY",
      "OPENAPI_BEARER_TOKEN",
      "OPENAPI_BASIC_USER",
      "OPENAPI_BASIC_PASS",
      "OPENAPI_MCP_ALLOWED_METHODS",
      "OPENAPI_MCP_RATE_LIMIT"
    ],
    "examples": {
      "OPENAPI_SPEC_URL": "https://api.example.com/openapi.json",
      "OPENAPI_BASE_URL": "https://api.example.com",
      "OPENAPI_API_KEY": "your_api_key_here",
      "OPENAPI_BEARER_TOKEN": "your_bearer_token_here"
    }
  }
};

class ConfigValidator {
  constructor() {
    this.errors = [];
    this.warnings = [];
  }

  async validateTemplate(templatePath) {
    console.log(`🔍 Validating template: ${path.basename(templatePath)}`);
    
    try {
      const content = await fs.readFile(templatePath, 'utf8');
      const config = JSON.parse(content);
      
      this.validateSchema(config, templatePath);
      this.validateServices(config.services || [], templatePath);
      this.validateEnvironmentVars(config, templatePath);
      
    } catch (error) {
      this.errors.push(`${templatePath}: Failed to parse JSON - ${error.message}`);
    }
  }

  validateSchema(config, templatePath) {
    const requiredFields = ['services'];
    for (const field of requiredFields) {
      if (!config[field]) {
        this.errors.push(`${templatePath}: Missing required field '${field}'`);
      }
    }
    
    if (config.services && !Array.isArray(config.services)) {
      this.errors.push(`${templatePath}: 'services' must be an array`);
    }
  }

  validateServices(services, templatePath) {
    for (let i = 0; i < services.length; i++) {
      const service = services[i];
      const servicePrefix = `${templatePath}:services[${i}]`;
      
      // Required fields
      const requiredFields = ['name', 'type'];
      for (const field of requiredFields) {
        if (!service[field]) {
          this.errors.push(`${servicePrefix}: Missing required field '${field}'`);
        }
      }
      
      // Must have either specUrl or specFile
      if (!service.specUrl && !service.specFile) {
        this.errors.push(`${servicePrefix}: Must have either 'specUrl' or 'specFile'`);
      }
      
      // Validate auth configuration
      if (service.auth && typeof service.auth === 'object') {
        this.validateAuth(service.auth, `${servicePrefix}:auth`);
      }
    }
  }

  validateAuth(auth, prefix) {
    if (auth.kind) {
      const validKinds = ['bearer', 'header', 'apiKey', 'basic'];
      if (!validKinds.includes(auth.kind)) {
        this.errors.push(`${prefix}: Invalid auth kind '${auth.kind}'. Must be one of: ${validKinds.join(', ')}`);
      }
      
      if (auth.kind === 'header' && !auth.name) {
        this.errors.push(`${prefix}: Header auth requires 'name' field`);
      }
      
      if (auth.kind === 'apiKey' && (!auth.in || !auth.name)) {
        this.errors.push(`${prefix}: API key auth requires 'in' and 'name' fields`);
      }
    }
  }

  validateEnvironmentVars(config, templatePath) {
    const configStr = JSON.stringify(config);
    const envVarPattern = /\$\{([A-Z_][A-Z0-9_]*)\}/g;
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
      this.warnings.push(`${templatePath}: No environment variables found`);
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
    console.log('\n📊 Validation Results:');
    
    if (this.errors.length === 0 && this.warnings.length === 0) {
      console.log('✅ All templates are valid!');
    } else {
      if (this.errors.length > 0) {
        console.log(`❌ Errors: ${this.errors.length}`);
        for (const error of this.errors) {
          console.log(`   - ${error}`);
        }
      }
      
      if (this.warnings.length > 0) {
        console.log(`⚠️  Warnings: ${this.warnings.length}`);
        for (const warning of this.warnings) {
          console.log(`   - ${warning}`);
        }
      }
      
      if (this.errors.length > 0) {
        process.exit(1);
      }
    }
  }
}

// CLI handling
if (import.meta.url === `file://${process.argv[1]}`) {
  const validator = new ConfigValidator();
  validator.validateAll().catch(console.error);
}

export { ConfigValidator };
