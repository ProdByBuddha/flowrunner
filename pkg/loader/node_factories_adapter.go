package loader

import (
	"github.com/tcmartin/flowlib"
	"github.com/tcmartin/flowrunner/pkg/plugins"
)

// BaseNodeFactoryAdapter adapts a runtime.NodeFactory (func(params) node)
// to a plugins.NodeFactory expected by the YAML loader.
// It injects the nodeDef.Params into the constructor.

type BaseNodeFactoryAdapter struct {
	Factory func(params map[string]interface{}) (flowlib.Node, error)
}

func (a *BaseNodeFactoryAdapter) CreateNode(nodeDef plugins.NodeDefinition) (flowlib.Node, error) {
	return a.Factory(nodeDef.Params)
}
