package processing

import (
	"github.com/google/go-jsonnet/ast"
	"github.com/grafana/jsonnet-language-server/pkg/nodestack"
)

func FindBindByIDViaStack(stack *nodestack.NodeStack, id ast.Identifier) *ast.LocalBind {
	nodes := append(stack.Stack, stack.From)
	for _, node := range nodes {
		switch curr := node.(type) {
		case *ast.Local:
			for _, bind := range curr.Binds {
				if bind.Variable == id {
					return &bind
				}
			}
		case *ast.DesugaredObject:
			for _, bind := range curr.Locals {
				if bind.Variable == id {
					return &bind
				}
			}
		}
	}
	return nil
}
