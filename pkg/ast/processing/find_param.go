package processing

import (
	"strings"

	"github.com/google/go-jsonnet/ast"
	"github.com/grafana/jsonnet-language-server/pkg/nodestack"
)

func FindParameterByIDViaStack(stack *nodestack.NodeStack, id ast.Identifier, partialMatchFields bool) *ast.Parameter {
	param, _ := findParameterWithDepth(stack, id, partialMatchFields)
	return param
}

// findParameterWithDepth returns the closest enclosing function parameter for
// id, and the index of the stack node that declares it.
func findParameterWithDepth(stack *nodestack.NodeStack, id ast.Identifier, partialMatchFields bool) (*ast.Parameter, int) {
	for i := len(stack.Stack) - 1; i >= 0; i-- {
		if f, ok := stack.Stack[i].(*ast.Function); ok {
			for _, param := range f.Parameters {
				if param.Name == id || (partialMatchFields && strings.HasPrefix(string(param.Name), string(id))) {
					return &param, i
				}
			}
		}
	}
	return nil, -2
}

// FindDeclarationByIDViaStack reports whether id resolves to a local bind
// or a function parameter, whichever is declared closest to the usage.
// Exactly one of the returned values is non-nil, unless id is not declared.
func FindDeclarationByIDViaStack(stack *nodestack.NodeStack, id ast.Identifier) (*ast.LocalBind, *ast.Parameter) {
	bind, bindDepth := findBindWithDepth(stack, id)
	param, paramDepth := findParameterWithDepth(stack, id, false)
	if bind != nil && (param == nil || bindDepth >= paramDepth) {
		return bind, nil
	}
	return nil, param
}
