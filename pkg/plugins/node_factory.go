package plugins

import (
	"github.com/tcmartin/flowlib"
)

// NodeFactory creates a flowlib.Node
type NodeFactory interface {
	CreateNode(nodeDef NodeDefinition) (flowlib.Node, error)
}
