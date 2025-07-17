# Requirements Document

## Introduction

Flowrunner is a lightweight, YAML-driven orchestration service built on top of Flowlib that enables users to define, manage, and trigger workflows without writing Go code. The system provides a RESTful HTTP API for flow management, execution capabilities, multi-tenant account support with secrets management, and extensibility through plugins and inline scripting.

## Requirements

### Requirement 1

**User Story:** As a workflow integrator, I want to create and manage flows using YAML definitions, so that I can define complex workflows without writing Go code.

#### Acceptance Criteria

1. WHEN a user submits a valid YAML flow definition THEN the system SHALL parse it into a Flowlib graph structure
2. WHEN a user submits an invalid YAML flow definition THEN the system SHALL return a 400 error with validation details
3. WHEN a user creates a flow THEN the system SHALL assign it a unique identifier and store it
4. WHEN a user requests a flow by ID THEN the system SHALL return the YAML definition
5. WHEN a user updates a flow THEN the system SHALL validate and replace the existing definition
6. WHEN a user deletes a flow THEN the system SHALL remove it from storage and prevent future executions

### Requirement 2

**User Story:** As a system administrator, I want to trigger flow executions via HTTP API, so that I can integrate workflows into existing systems and processes.

#### Acceptance Criteria

1. WHEN a user sends a POST request to /api/v1/flows/{id}/run THEN the system SHALL start flow execution
2. WHEN a flow execution starts THEN the system SHALL return an execution ID and status
3. WHEN a flow execution includes shared context data THEN the system SHALL make it available to all nodes
4. WHEN a flow execution completes THEN the system SHALL update the execution status
5. WHEN a flow execution fails THEN the system SHALL capture error details and update status accordingly
6. WHEN a 10-node flow executes THEN it SHALL complete within 5 seconds under normal conditions

### Requirement 3

**User Story:** As a multi-tenant service user, I want account-based isolation for flows and secrets, so that my workflows and credentials remain private and secure.

#### Acceptance Criteria

1. WHEN a user authenticates with account credentials THEN the system SHALL provide access only to that account's flows
2. WHEN a user stores secrets THEN the system SHALL isolate them per account
3. WHEN a user accesses flows THEN the system SHALL only return flows belonging to their account
4. WHEN a user manages secrets THEN the system SHALL provide CRUD operations via /api/v1/accounts/{acct}/secrets
5. IF a user lacks proper authentication THEN the system SHALL deny access with appropriate HTTP status codes

### Requirement 4

**User Story:** As a developer, I want to extend flowrunner with custom nodes through plugins, so that I can add domain-specific functionality without modifying the core system.

#### Acceptance Criteria

1. WHEN a plugin is loaded THEN the system SHALL register it without requiring server restart
2. WHEN a plugin implements the flowlib.Node interface THEN the system SHALL make it available for use in flows
3. WHEN a flow references a plugin node THEN the system SHALL execute it as part of the workflow
4. WHEN plugin loading fails THEN the system SHALL log the error and continue operation
5. WHEN multiple plugins are loaded THEN the system SHALL maintain a registry of all available node types

### Requirement 5

**User Story:** As a workflow designer, I want to include small JavaScript snippets in my flows, so that I can perform simple data transformations and logic without creating full plugins.

#### Acceptance Criteria

1. WHEN a flow contains JavaScript snippets THEN the system SHALL execute them in a sandboxed environment
2. WHEN JavaScript execution occurs THEN it SHALL complete within 50ms per snippet
3. WHEN JavaScript snippets access flow context THEN they SHALL have read/write access to node data
4. WHEN JavaScript execution fails THEN the system SHALL capture the error and fail the node gracefully
5. WHEN JavaScript snippets are used as prep or exec hooks THEN they SHALL execute at the appropriate workflow stages

### Requirement 6

**User Story:** As a system integrator, I want to receive HTTP callbacks when flows complete or nodes execute, so that I can integrate workflow events with external systems.

#### Acceptance Criteria

1. WHEN a flow completes THEN the system SHALL send HTTP callbacks to configured webhook URLs
2. WHEN a node completes THEN the system SHALL optionally send per-node event callbacks
3. WHEN webhook delivery occurs THEN it SHALL complete within 200ms of the triggering event
4. WHEN webhook delivery fails THEN the system SHALL implement retry logic with exponential backoff
5. WHEN webhook payloads are sent THEN they SHALL include relevant execution context and results

### Requirement 7

**User Story:** As a system administrator, I want comprehensive logging and audit trails for flow executions, so that I can troubleshoot issues and maintain compliance.

#### Acceptance Criteria

1. WHEN a flow executes THEN the system SHALL create structured logs for the entire execution
2. WHEN nodes execute THEN the system SHALL log individual node execution details
3. WHEN errors occur THEN the system SHALL capture detailed error information in logs
4. WHEN audit trails are requested THEN the system SHALL provide execution history and outcomes
5. WHEN logs are generated THEN they SHALL include timestamps, execution IDs, and relevant context data

### Requirement 8

**User Story:** As an API consumer, I want RESTful endpoints for flow management, so that I can integrate flowrunner into existing toolchains and automation systems.

#### Acceptance Criteria

1. WHEN accessing /api/v1/flows THEN the system SHALL provide CRUD operations for flow management
2. WHEN authentication is required THEN the system SHALL support HTTP Basic or Bearer Token authentication
3. WHEN API requests are made THEN the system SHALL return appropriate HTTP status codes and JSON responses
4. WHEN listing flows THEN the system SHALL return paginated results with flow metadata
5. WHEN API errors occur THEN the system SHALL return structured error responses with helpful messages