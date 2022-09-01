package ast_processing

import (
	"github.com/google/go-jsonnet/ast"
	"github.com/grafana/jsonnet-language-server/pkg/nodestack"
)

func FindBindByIdViaStack(stack *nodestack.NodeStack, id ast.Identifier) *ast.LocalBind {
	for _, node := range stack.Stack {
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
