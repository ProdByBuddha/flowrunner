# FlowRunner CLI Guide

This guide provides detailed instructions for using the FlowRunner command-line interface (CLI).

## Table of Contents

1. [Installation](#installation)
2. [Configuration](#configuration)
3. [Authentication](#authentication)
4. [Flow Management](#flow-management)
5. [Flow Execution](#flow-execution)
6. [Account Management](#account-management)
7. [Secrets Management](#secrets-management)
8. [Monitoring and Logging](#monitoring-and-logging)
9. [Advanced Usage](#advanced-usage)

## Installation

### Building from Source

```bash
# Clone the repository
git clone https://github.com/tcmartin/flowrunner.git
cd flowrunner

# Build the CLI
go build -o flowrunner-cli cmd/flowrunner-cli/main.go
```

### Adding to PATH

For convenience, add the FlowRunner CLI to your PATH:

```bash
# Linux/macOS
cp flowrunner-cli /usr/local/bin/flowrunner

# Or add the build directory to your PATH
export PATH=$PATH:/path/to/flowrunner
```

## Configuration

The FlowRunner CLI can be configured using:

1. Command-line flags
2. Environment variables
3. Configuration file

### Command-line Flags

```bash
# Set the server URL
flowrunner --server http://localhost:8080

# Set the authentication token
flowrunner --token your-auth-token

# Set the output format
flowrunner --output json
```

### Environment Variables

```bash
# Set the server URL
export FLOWRUNNER_SERVER=http://localhost:8080

# Set the authentication token
export FLOWRUNNER_TOKEN=your-auth-token

# Set the output format
export FLOWRUNNER_OUTPUT=json
```

### Configuration File

Create a configuration file at `~/.flowrunner/config.yaml`:

```yaml
server: http://localhost:8080
token: your-auth-token
output: json
```

## Authentication

### Logging In

```bash
# Log in with username and password
flowrunner login --username user@example.com --password your-password

# Log in with username (password will be prompted)
flowrunner login --username user@example.com
```

### Using API Keys

```bash
# Set API key
flowrunner config set api-key your-api-key
```

### Checking Authentication Status

```bash
# Check if you're authenticated
flowrunner auth status
```

## Flow Management

### Creating Flows

```bash
# Create a flow from a YAML file
flowrunner flow create --file flow.yaml

# Create a flow with a specific name
flowrunner flow create --name "My Flow" --file flow.yaml

# Create a flow with a specific ID
flowrunner flow create --id my-flow-id --file flow.yaml
```

### Listing Flows

```bash
# List all flows
flowrunner flow list

# List flows with a specific tag
flowrunner flow list --tag production

# List flows in JSON format
flowrunner flow list --output json
```

### Getting Flow Details

```bash
# Get flow details
flowrunner flow get flow-id

# Get flow definition
flowrunner flow get flow-id --definition
```

### Updating Flows

```bash
# Update a flow from a YAML file
flowrunner flow update flow-id --file flow.yaml

# Update flow metadata
flowrunner flow update flow-id --name "New Name" --description "New description"
```

### Deleting Flows

```bash
# Delete a flow
flowrunner flow delete flow-id

# Force delete a flow (no confirmation)
flowrunner flow delete flow-id --force
```

### Exporting Flows

```bash
# Export a flow to a YAML file
flowrunner flow export flow-id --output flow.yaml

# Export a flow to stdout
flowrunner flow export flow-id
```

## Flow Execution

### Running Flows

```bash
# Run a flow
flowrunner flow run flow-id

# Run a flow with input from a JSON file
flowrunner flow run flow-id --input input.json

# Run a flow with input from stdin
cat input.json | flowrunner flow run flow-id

# Run a flow with input from command line
flowrunner flow run flow-id --input-json '{"key": "value"}'
```

### Listing Executions

```bash
# List all executions
flowrunner execution list

# List executions for a specific flow
flowrunner execution list --flow flow-id

# List recent executions
flowrunner execution list --limit 10
```

### Getting Execution Details

```bash
# Get execution details
flowrunner execution get execution-id

# Get execution result
flowrunner execution get execution-id --result
```

### Cancelling Executions

```bash
# Cancel an execution
flowrunner execution cancel execution-id
```

## Account Management

### Creating Accounts

```bash
# Create a new account
flowrunner account create --name "My Account"

# Create a new account with specific settings
flowrunner account create --name "My Account" --email user@example.com --role admin
```

### Listing Accounts

```bash
# List all accounts
flowrunner account list

# List accounts with a specific role
flowrunner account list --role admin
```

### Getting Account Details

```bash
# Get account details
flowrunner account get account-id
```

### Updating Accounts

```bash
# Update account details
flowrunner account update account-id --name "New Name" --email new-email@example.com
```

### Deleting Accounts

```bash
# Delete an account
flowrunner account delete account-id

# Force delete an account (no confirmation)
flowrunner account delete account-id --force
```

## Secrets Management

### Creating Secrets

```bash
# Create a secret
flowrunner secret create --key API_KEY --value your-api-key

# Create a secret for a specific account
flowrunner secret create --account account-id --key API_KEY --value your-api-key

# Create a secret from a file
flowrunner secret create --key CERTIFICATE --file certificate.pem
```

### Listing Secrets

```bash
# List all secrets
flowrunner secret list

# List secrets for a specific account
flowrunner secret list --account account-id
```

### Getting Secret Details

```bash
# Get secret details (without value)
flowrunner secret get API_KEY

# Get secret value
flowrunner secret get API_KEY --show-value
```

### Updating Secrets

```bash
# Update a secret
flowrunner secret update API_KEY --value new-api-key

# Update a secret from a file
flowrunner secret update CERTIFICATE --file new-certificate.pem
```

### Deleting Secrets

```bash
# Delete a secret
flowrunner secret delete API_KEY

# Force delete a secret (no confirmation)
flowrunner secret delete API_KEY --force
```

## Monitoring and Logging

### Viewing Logs

```bash
# View logs for an execution
flowrunner logs execution-id

# View logs for an execution with timestamps
flowrunner logs execution-id --timestamps

# View logs for an execution with a specific level
flowrunner logs execution-id --level error

# Follow logs in real-time
flowrunner logs execution-id --follow
```

### Monitoring Executions

```bash
# Monitor an execution in real-time
flowrunner execution monitor execution-id

# Monitor an execution with a specific refresh interval
flowrunner execution monitor execution-id --interval 5s
```

### Getting Metrics

```bash
# Get execution metrics
flowrunner metrics executions

# Get flow metrics
flowrunner metrics flows

# Get metrics for a specific time range
flowrunner metrics executions --from 2023-01-01 --to 2023-01-31
```

## Advanced Usage

### Batch Operations

```bash
# Run multiple flows
flowrunner flow run-batch --file flows.json

# Delete multiple flows
flowrunner flow delete-batch --file flows.json
```

### Scheduling Flows

```bash
# Schedule a flow to run at a specific time
flowrunner flow schedule flow-id --time "2023-01-01T12:00:00Z"

# Schedule a flow to run with a cron expression
flowrunner flow schedule flow-id --cron "0 * * * *"

# List scheduled flows
flowrunner schedule list

# Delete a schedule
flowrunner schedule delete schedule-id
```

### Importing and Exporting

```bash
# Export all flows
flowrunner flow export-all --directory flows/

# Import flows from a directory
flowrunner flow import --directory flows/
```

### Plugins

```bash
# List available plugins
flowrunner plugin list

# Install a plugin
flowrunner plugin install plugin-name

# Update a plugin
flowrunner plugin update plugin-name

# Remove a plugin
flowrunner plugin remove plugin-name
```

### Server Management

```bash
# Start the FlowRunner server
flowrunner server start

# Stop the FlowRunner server
flowrunner server stop

# Restart the FlowRunner server
flowrunner server restart

# Check server status
flowrunner server status
```

### Configuration Management

```bash
# View current configuration
flowrunner config view

# Set a configuration value
flowrunner config set key value

# Get a configuration value
flowrunner config get key

# Reset configuration to defaults
flowrunner config reset
```

### Debugging

```bash
# Enable debug mode
flowrunner --debug

# Run a flow in debug mode
flowrunner flow run flow-id --debug

# Validate a flow definition
flowrunner flow validate --file flow.yaml
```