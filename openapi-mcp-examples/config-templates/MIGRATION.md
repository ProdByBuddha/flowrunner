# Configuration Migration Guide

This guide helps you migrate from existing configurations to the new template system.

## Migration Scenarios

### From services.example.json to Templates

If you're currently using `services.example.json`, you can migrate to specific templates:

1. **Single Service Migration**:
   - If you have only one service, use the `single-service.json` template
   - Copy your service configuration and update environment variables

2. **Multi-Service Migration**:
   - If you have multiple services, use the `multi-service.json` template
   - Migrate each service individually

3. **Service-Specific Migration**:
   - Use service-specific templates (e.g., `n8n-service.json`, `hostinger-service.json`)
   - These templates include optimized configurations for each service

### Migration Steps

#### Step 1: Identify Your Current Configuration

```bash
# Check your current configuration
cat services.example.json | jq '.services[].name'
```

#### Step 2: Choose the Right Template

- **1 service**: Use `single-service.json`
- **2-3 services**: Use `multi-service.json`
- **Production**: Use `production.json`
- **Development**: Use `development.json`
- **Specific service**: Use `{service}-service.json`

#### Step 3: Copy and Customize

```bash
# Copy the template
cp examples/config-templates/multi-service.json my-services.json

# Edit the configuration
nano my-services.json
```

#### Step 4: Update Environment Variables

```bash
# Check what environment variables are needed
node examples/config-templates/validate-templates.js

# Set up your environment
export N8N_API_KEY="your_key_here"
export N8N_API_URL="https://your-n8n.com/api/v1"
```

#### Step 5: Test the Configuration

```bash
# Test with the new configuration
node examples/mcp-multi-host.js --config my-services.json --once tools/list
```

## Common Migration Issues

### Issue: Missing Environment Variables

**Problem**: Template uses environment variables that aren't set.

**Solution**: 
```bash
# Check what variables are needed
grep -o '\$\{[^}]*\}' my-services.json
grep -o '"env": "[^"]*"' my-services.json

# Set the variables
export VARIABLE_NAME="value"
```

### Issue: Spec File Paths

**Problem**: Template references spec files that don't exist.

**Solution**:
```bash
# Check if spec files exist
ls -la specs/

# Update paths in your configuration
sed -i 's|../specs/|./specs/|g' my-services.json
```

### Issue: Authentication Configuration

**Problem**: Auth configuration doesn't match your API requirements.

**Solution**: Update the auth section based on your API:

```json
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
```

## Validation

Always validate your migrated configuration:

```bash
# Validate the configuration
node examples/config-templates/validate-templates.js

# Test the configuration
node examples/mcp-multi-host.js --config my-services.json --once tools/list
```

## Rollback

If migration fails, you can always rollback:

```bash
# Use your original configuration
node examples/mcp-multi-host.js --config services.example.json
```

## Getting Help

If you encounter issues during migration:

1. Check the template README files for specific guidance
2. Validate your configuration with the validation script
3. Test with a minimal configuration first
4. Check the logs for specific error messages
