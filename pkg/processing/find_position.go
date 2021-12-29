package processing

import (
	"errors"

	"github.com/google/go-jsonnet/ast"
	"github.com/grafana/jsonnet-language-server/pkg/nodestack"
	"github.com/grafana/jsonnet-language-server/pkg/position"
)

func FindNodeByPosition(node ast.Node, location ast.Location) (*nodestack.NodeStack, error) {
	if node == nil {
		return nil, errors.New("node is nil")
	}

	stack := nodestack.NewNodeStack(node)
	// keeps the history of the navigation path to the requested Node.
	// used to backwards search Nodes from the found node to the root.
	searchStack := &nodestack.NodeStack{From: stack.From}
	var curr ast.Node
	for !stack.IsEmpty() {
		stack, curr = stack.Pop()
		// This is needed because SuperIndex only spans "key: super" and not the ".foo" after. This only occurs
		// when super only has 1 additional index. "super.foo.bar" will not have this issue
		if curr, isType := curr.(*ast.SuperIndex); isType {
			curr.Loc().End.Column = curr.Loc().End.Column + len(curr.Index.(*ast.LiteralString).Value) + 1
		}
		inRange := position.InRange(location, *curr.Loc())
		if inRange {
			searchStack = searchStack.Push(curr)
		} else if curr.Loc().End.IsSet() {
			continue
		}
		switch curr := curr.(type) {
		case *ast.Local:
			for _, bind := range curr.Binds {
				stack = stack.Push(bind.Body)
			}
			if curr.Body != nil {
				stack = stack.Push(curr.Body)
			}
		case *ast.DesugaredObject:
			for _, field := range curr.Fields {
				body := field.Body
				// Functions do not have a LocRange, so we use the one from the field's body
				if funcBody, isFunc := body.(*ast.Function); isFunc {
					funcBody.LocRange = field.LocRange
					stack = stack.Push(funcBody)
				} else {
					stack = stack.Push(body)
				}
			}
			for _, local := range curr.Locals {
				stack = stack.Push(local.Body)
			}
		case *ast.Binary:
			stack = stack.Push(curr.Left)
			stack = stack.Push(curr.Right)
		case *ast.Array:
			for _, element := range curr.Elements {
				stack = stack.Push(element.Expr)
			}
		case *ast.Apply:
			for _, posArg := range curr.Arguments.Positional {
				stack = stack.Push(posArg.Expr)
			}
			for _, namedArg := range curr.Arguments.Named {
				stack = stack.Push(namedArg.Arg)
			}
			stack = stack.Push(curr.Target)
		case *ast.Conditional:
			stack = stack.Push(curr.Cond)
			stack = stack.Push(curr.BranchTrue)
			stack = stack.Push(curr.BranchFalse)
		case *ast.Error:
			stack = stack.Push(curr.Expr)
		case *ast.Function:
			for _, param := range curr.Parameters {
				if param.DefaultArg != nil {
					stack = stack.Push(param.DefaultArg)
				}
			}
			stack = stack.Push(curr.Body)
		case *ast.Index:
			stack = stack.Push(curr.Target)
			stack = stack.Push(curr.Index)
		case *ast.InSuper:
			stack = stack.Push(curr.Index)
		case *ast.SuperIndex:
			stack = stack.Push(curr.Index)
		case *ast.Unary:
			stack = stack.Push(curr.Expr)
		}
	}
	return searchStack.ReorderDesugaredObjects(), nil
}
