package processing

import (
	"github.com/google/go-jsonnet/ast"
	"github.com/grafana/jsonnet-language-server/pkg/nodestack"
)

func FindBindByIDViaStack(stack *nodestack.NodeStack, id ast.Identifier) *ast.LocalBind {
	bind, _ := findBindWithDepth(stack, id)
	return bind
}

// findBindWithDepth returns the closest enclosing bind for id, and the index
// of the stack node that declares it. The index is -1 for the From node.
func findBindWithDepth(stack *nodestack.NodeStack, id ast.Identifier) (*ast.LocalBind, int) {
	for i := len(stack.Stack) - 1; i >= -1; i-- {
		var node ast.Node
		if i == -1 {
			node = stack.From
		} else {
			node = stack.Stack[i]
		}
		switch curr := node.(type) {
		case *ast.Local:
			for _, bind := range curr.Binds {
				if bind.Variable == id {
					return &bind, i
				}
			}
		case *ast.DesugaredObject:
			for _, bind := range curr.Locals {
				if bind.Variable == id {
					return &bind, i
				}
			}
		}
	}
	return nil, -2
}
