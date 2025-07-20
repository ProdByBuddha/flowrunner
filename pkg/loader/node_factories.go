package loader

import (
	"fmt"
	"time"

	"github.com/tcmartin/flowlib"
)

// BaseNodeFactory creates a basic flowlib.NodeWithRetry
type BaseNodeFactory struct{}

func (f *BaseNodeFactory) CreateNode(nodeDef NodeDefinition) (flowlib.Node, error) {
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

func (f *BatchNodeFactory) CreateNode(nodeDef NodeDefinition) (flowlib.Node, error) {
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

	return node, nil
}

// AsyncBatchNodeFactory creates a flowlib.AsyncBatchNode
type AsyncBatchNodeFactory struct{}

func (f *AsyncBatchNodeFactory) CreateNode(nodeDef NodeDefinition) (flowlib.Node, error) {
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

	return node, nil
}

// AsyncParallelBatchNodeFactory creates a flowlib.AsyncParallelBatchNode
type AsyncParallelBatchNodeFactory struct{}

func (f *AsyncParallelBatchNodeFactory) CreateNode(nodeDef NodeDefinition) (flowlib.Node, error) {
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

	return node, nil
}

// WorkerPoolBatchNodeFactory creates a flowlib.WorkerPoolBatchNode
type WorkerPoolBatchNodeFactory struct{}

func (f *WorkerPoolBatchNodeFactory) CreateNode(nodeDef NodeDefinition) (flowlib.Node, error) {
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

	return node, nil
}