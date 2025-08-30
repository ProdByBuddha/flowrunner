package main

import (
    "github.com/tcmartin/flowlib"
    "github.com/tcmartin/flowrunner/pkg/plugins"
    "github.com/tcmartin/flowrunner/pkg/registry"
    "github.com/tcmartin/flowrunner/pkg/runtime"
)

// runtimeFactoryAdapter adapts a runtime.NodeFactory to the plugins.NodeFactory interface
type runtimeFactoryAdapter struct{ fn runtime.NodeFactory }

func (a runtimeFactoryAdapter) CreateNode(nodeDef plugins.NodeDefinition) (flowlib.Node, error) {
    return a.fn(nodeDef.Params)
}

// runtimeRegistryAdapter adapts the registry for the runtime to fetch flows
type runtimeRegistryAdapter struct{ registry registry.FlowRegistry }

func (a runtimeRegistryAdapter) GetFlow(accountID, flowID string) (*runtime.Flow, error) {
    yamlStr, err := a.registry.Get(accountID, flowID)
    if err != nil {
        return nil, err
    }
    return &runtime.Flow{ID: flowID, YAML: yamlStr}, nil
}

