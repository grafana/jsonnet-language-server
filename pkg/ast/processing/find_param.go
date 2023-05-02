package processing

import (
	"strings"

	"github.com/google/go-jsonnet/ast"
	"github.com/grafana/jsonnet-language-server/pkg/nodestack"
)

func FindParameterByIDViaStack(stack *nodestack.NodeStack, id ast.Identifier, partialMatchFields bool) *ast.Parameter {
	for _, node := range stack.Stack {
		if f, ok := node.(*ast.Function); ok {
			for _, param := range f.Parameters {
				if param.Name == id || (partialMatchFields && strings.HasPrefix(string(param.Name), string(id))) {
					return &param
				}
			}
		}
	}
	return nil
}
