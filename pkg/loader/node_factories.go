package loader

import (
	"context"
	"fmt"
	"time"

	"github.com/tcmartin/flowlib"
	"github.com/tcmartin/flowrunner/pkg/plugins"
)

// BaseNodeFactory creates a basic flowlib.NodeWithRetry
type BaseNodeFactory struct{}

func (f *BaseNodeFactory) CreateNode(nodeDef plugins.NodeDefinition) (flowlib.Node, error) {
	maxRetries := 0
	if nodeDef.Retry.MaxRetries > 0 {
		maxRetries = nodeDef.Retry.MaxRetries
	}

	wait := 0 * time.Second
	if nodeDef.Retry.Wait != "" {
		var err error
		wait, err = time.ParseDuration(nodeDef.Retry.Wait)
		if err != nil {
			return nil, fmt.Errorf("invalid wait duration: %w", err)
		}
	}

	node := flowlib.NewNode(maxRetries, wait)
	node.SetParams(nodeDef.Params)

	return node, nil
}

// BatchNodeFactory creates a flowlib.BatchNode
type BatchNodeFactory struct{}

func (f *BatchNodeFactory) CreateNode(nodeDef plugins.NodeDefinition) (flowlib.Node, error) {
	maxRetries := 0
	if nodeDef.Retry.MaxRetries > 0 {
		maxRetries = nodeDef.Retry.MaxRetries
	}

	wait := 0 * time.Second
	if nodeDef.Retry.Wait != "" {
		var err error
		wait, err = time.ParseDuration(nodeDef.Retry.Wait)
		if err != nil {
			return nil, fmt.Errorf("invalid wait duration: %w", err)
		}
	}

	node := flowlib.NewBatchNode(maxRetries, wait)
	node.SetParams(nodeDef.Params)

	// Set a default execFn for BatchNode that returns an empty slice
	// In a real scenario, this would likely come from nodeDef.Params or a shared context
	node.SetExecFn(func(any) (any, error) { return []any{}, nil })
	node.SetPrepFn(func(any) (any, error) { return []any{}, nil })

	return node, nil
}

// AsyncBatchNodeFactory creates a flowlib.AsyncBatchNode
type AsyncBatchNodeFactory struct{}

func (f *AsyncBatchNodeFactory) CreateNode(nodeDef plugins.NodeDefinition) (flowlib.Node, error) {
	maxRetries := 0
	if nodeDef.Retry.MaxRetries > 0 {
		maxRetries = nodeDef.Retry.MaxRetries
	}

	wait := 0 * time.Second
	if nodeDef.Retry.Wait != "" {
		var err error
		wait, err = time.ParseDuration(nodeDef.Retry.Wait)
		if err != nil {
			return nil, fmt.Errorf("invalid wait duration: %w", err)
		}
	}

	node := flowlib.NewAsyncBatchNode(maxRetries, wait)
	node.SetParams(nodeDef.Params)

	// Set a default execAsyncFn for AsyncBatchNode that returns an empty slice
	node.SetExecAsyncFn(func(ctx context.Context, input any) (any, error) { return []any{}, nil })
	node.SetPrepFn(func(any) (any, error) { return []any{}, nil })

	return node, nil
}

// AsyncParallelBatchNodeFactory creates a flowlib.AsyncParallelBatchNode
type AsyncParallelBatchNodeFactory struct{}

func (f *AsyncParallelBatchNodeFactory) CreateNode(nodeDef plugins.NodeDefinition) (flowlib.Node, error) {
	maxRetries := 0
	if nodeDef.Retry.MaxRetries > 0 {
		maxRetries = nodeDef.Retry.MaxRetries
	}

	wait := 0 * time.Second
	if nodeDef.Retry.Wait != "" {
		var err error
		wait, err = time.ParseDuration(nodeDef.Retry.Wait)
		if err != nil {
			return nil, fmt.Errorf("invalid wait duration: %w", err)
		}
	}

	node := flowlib.NewAsyncParallelBatchNode(maxRetries, wait)
	node.SetParams(nodeDef.Params)

	// Set a default execAsyncFn for AsyncParallelBatchNode that returns an empty slice
	node.SetExecAsyncFn(func(ctx context.Context, input any) (any, error) { return []any{}, nil })
	node.SetPrepFn(func(any) (any, error) { return []any{}, nil })

	return node, nil
}

// WorkerPoolBatchNodeFactory creates a flowlib.WorkerPoolBatchNode
type WorkerPoolBatchNodeFactory struct{}

func (f *WorkerPoolBatchNodeFactory) CreateNode(nodeDef plugins.NodeDefinition) (flowlib.Node, error) {
	maxRetries := 0
	if nodeDef.Retry.MaxRetries > 0 {
		maxRetries = nodeDef.Retry.MaxRetries
	}

	wait := 0 * time.Second
	if nodeDef.Retry.Wait != "" {
		var err error
		wait, err = time.ParseDuration(nodeDef.Retry.Wait)
		if err != nil {
			return nil, fmt.Errorf("invalid wait duration: %w", err)
		}
	}

	maxParallel := 0
	if nodeDef.Batch.MaxParallel > 0 {
		maxParallel = nodeDef.Batch.MaxParallel
	}

	node := flowlib.NewWorkerPoolBatchNode(maxRetries, wait, maxParallel)
	node.SetParams(nodeDef.Params)

	// Set a default execAsyncFn for WorkerPoolBatchNode that returns an empty slice
	node.SetExecAsyncFn(func(ctx context.Context, input any) (any, error) { return []any{}, nil })
	node.SetPrepFn(func(any) (any, error) { return []any{}, nil })

	return node, nil
}