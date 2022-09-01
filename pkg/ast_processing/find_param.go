package processing

import (
	"github.com/google/go-jsonnet/ast"
	"github.com/grafana/jsonnet-language-server/pkg/nodestack"
)

func FindParameterByIdViaStack(stack *nodestack.NodeStack, id ast.Identifier) *ast.Parameter {
	for _, node := range stack.Stack {
		if f, ok := node.(*ast.Function); ok {
			for _, param := range f.Parameters {
				if param.Name == id {
					return &param
				}
			}
		}
	}
	return nil
}
