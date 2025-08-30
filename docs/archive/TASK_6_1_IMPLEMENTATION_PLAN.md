# Task 6.1: Flow Runtime Implementation Plan

This document outlines the steps to implement the flow runtime service, which will orchestrate the execution of flows defined in YAML. The runtime will leverage the `flowlib` library to handle the underlying flow execution logic.

## 1. Enhance YAML Schema and Parsing

*   **Extend YAML Schema:** Update `pkg/loader/schema.go` to support all the features of `flowlib`, including:
    *   Batching (`batch`, `async_batch`, `parallel_batch`, `worker_pool_batch`)
    *   Retry logic (`max_retries`, `wait`)
    *   Concurrency (`max_parallel`)
    *   Branching (`next` with actions)
*   **Implement `Parse` function:** Complete the `Parse` function in `pkg/loader/yaml_loader.go` to build a `flowlib.Flow` or `flowlib.AsyncFlow` from the YAML definition. This will involve:
    *   Creating `flowlib` nodes based on the `type` specified in the YAML.
    *   Setting node parameters, including retry and batching configurations.
    *   Connecting nodes based on the `next` field.

## 2. Implement Flow Runtime Service

*   **Create `FlowRuntime` implementation:** In `pkg/runtime/flow_runtime.go`, implement the `FlowRuntime` interface.
    *   **`Execute`:** This method will:
        1.  Retrieve the flow definition from the registry.
        2.  Use the `YAMLLoader` to parse the YAML and create a `flowlib` graph.
        3.  Execute the flow using `flow.Run()` or `flow.RunAsync()`.
        4.  Store the execution status and return an execution ID.
    *   **`GetStatus`:** Retrieve and return the status of a flow execution.
    *   **`GetLogs`:** Retrieve and return the logs for a flow execution.
    *   **`SubscribeToLogs`:** Implement real-time log streaming.
    *   **`Cancel`:** Cancel a running flow execution.

## 3. Create Node Factories

*   **Implement `NodeFactory` interface:** Create a `NodeFactory` interface that defines a standard way to create `flowlib` nodes.
*   **Create concrete node factories:** Implement factories for each of the core node types defined in `flowlib`:
    *   `NodeWithRetry`
    *   `BatchNode`
    *   `AsyncBatchNode`
    *   `AsyncParallelBatchNode`
    *   `WorkerPoolBatchNode`

## 4. Write Comprehensive Tests

*   **Unit tests:** Write unit tests for the `YAMLLoader` to ensure it correctly parses all supported YAML configurations.
*   **Integration tests:** Write integration tests for the `FlowRuntime` service to verify that it can execute flows with various `flowlib` features, including:
    *   Simple linear flows
    *   Flows with branching
    *   Flows with retries
    *   Flows with different batching strategies
    *   Flows with concurrency

## 5. Refactor and Verify

*   **Refactor:** Refactor the code to ensure it is clean, efficient, and well-documented.
*   **Verify:** Run all existing tests in the project to ensure that the new functionality has not introduced any regressions.
