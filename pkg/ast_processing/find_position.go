package processing

import (
	"errors"

	"github.com/google/go-jsonnet/ast"
	"github.com/grafana/jsonnet-language-server/pkg/nodestack"
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
		curr = stack.Pop()
		// This is needed because SuperIndex only spans "key: super" and not the ".foo" after. This only occurs
		// when super only has 1 additional index. "super.foo.bar" will not have this issue
		if curr, isType := curr.(*ast.SuperIndex); isType {
			curr.Loc().End.Column = curr.Loc().End.Column + len(curr.Index.(*ast.LiteralString).Value) + 1
		}
		inRange := InRange(location, *curr.Loc())
		if inRange {
			searchStack.Push(curr)
		} else if curr.Loc().End.IsSet() {
			continue
		}
		switch curr := curr.(type) {
		case *ast.Local:
			for _, bind := range curr.Binds {
				stack.Push(bind.Body)
			}
			if curr.Body != nil {
				stack.Push(curr.Body)
			}
		case *ast.DesugaredObject:
			for _, field := range curr.Fields {
				body := field.Body
				// Functions do not have a LocRange, so we use the one from the field's body
				if funcBody, isFunc := body.(*ast.Function); isFunc {
					funcBody.LocRange = field.LocRange
					stack.Push(funcBody)
				} else {
					stack.Push(body)
				}
			}
			for _, local := range curr.Locals {
				stack.Push(local.Body)
			}
		case *ast.Binary:
			stack.Push(curr.Left)
			stack.Push(curr.Right)
		case *ast.Array:
			for _, element := range curr.Elements {
				stack.Push(element.Expr)
			}
		case *ast.Apply:
			for _, posArg := range curr.Arguments.Positional {
				stack.Push(posArg.Expr)
			}
			for _, namedArg := range curr.Arguments.Named {
				stack.Push(namedArg.Arg)
			}
			stack.Push(curr.Target)
		case *ast.Conditional:
			stack.Push(curr.Cond)
			stack.Push(curr.BranchTrue)
			stack.Push(curr.BranchFalse)
		case *ast.Error:
			stack.Push(curr.Expr)
		case *ast.Function:
			for _, param := range curr.Parameters {
				if param.DefaultArg != nil {
					stack.Push(param.DefaultArg)
				}
			}
			stack.Push(curr.Body)
		case *ast.Index:
			stack.Push(curr.Target)
			stack.Push(curr.Index)
		case *ast.InSuper:
			stack.Push(curr.Index)
		case *ast.SuperIndex:
			stack.Push(curr.Index)
		case *ast.Unary:
			stack.Push(curr.Expr)
		}
	}
	return searchStack.ReorderDesugaredObjects(), nil
}
