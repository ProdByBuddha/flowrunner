# FlowRunner Project Overview

FlowRunner is a lightweight, YAML-driven orchestration service built on top of Flowlib that enables users to define, manage, and trigger workflows without writing Go code.

## Project Architecture

### Core Components
- **Flowlib**: The underlying workflow execution engine (local dependency)
- **HTTP Server**: RESTful API for flow management and execution
- **CLI Tool**: Command-line interface for flow operations
- **Storage Backends**: Multiple persistence options (in-memory, DynamoDB, PostgreSQL)
- **Node Registry**: Pluggable system for custom workflow nodes

### Key Technologies
- **Language**: Go 1.24.3
- **Web Framework**: Gorilla Mux for HTTP routing
- **Database**: PostgreSQL and DynamoDB support
- **Authentication**: JWT-based auth with encrypted secrets storage
- **Scripting**: JavaScript support via Otto engine
- **Configuration**: Environment variables and YAML files

## Project Structure

```
flowrunner/
├── cmd/                    # CLI and server entry points
├── pkg/                    # Core packages and libraries
├── flowlib/               # Local flowlib dependency
├── apps/                  # Application-specific code
├── docs/                  # User and developer documentation
├── demos/                 # Example flows and demonstrations
├── test_programs/         # Test utilities and programs
└── scripts/              # Build and deployment scripts
```

## Core Features

### Flow Management
- YAML-based workflow definitions with expression support
- RESTful API for CRUD operations on flows
- Version control and flow registry
- Multi-tenant account support with isolation

### Execution Engine
- Real-time flow execution with WebSocket monitoring
- Multiple storage backends for persistence
- Secure credential storage with encryption
- Expression evaluation in YAML definitions

### Node Types
- HTTP requests and webhooks
- Email (SMTP/IMAP) integration
- LLM integration (OpenAI, Anthropic)
- In-memory store operations
- AI agent nodes
- JavaScript scripting for transformations

## Development Standards

### Code Organization
- Follow Go project layout standards
- Separate concerns between packages
- Use dependency injection for testability
- Implement proper error handling and logging

### Testing
- Unit tests for all core functionality
- Integration tests for storage backends
- End-to-end tests for complete flows
- Performance tests for execution engine

### Documentation
- Comprehensive user guides and API documentation
- Code comments for complex logic
- Examples and demos for common use cases
- Architecture decision records for major changes