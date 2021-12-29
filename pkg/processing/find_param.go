package processing

import (
	"github.com/google/go-jsonnet/ast"
	"github.com/grafana/jsonnet-language-server/pkg/nodestack"
)

func FindParameterByIdViaStack(stack *nodestack.NodeStack, id ast.Identifier) *ast.Parameter {
	stack = stack.Clone()
	for !stack.IsEmpty() {
		_, curr := stack.Pop()
		switch curr := curr.(type) {
		case *ast.Function:
			for _, param := range curr.Parameters {
				if param.Name == id {
					return &param
				}
			}
		}

	}
	return nil
}
