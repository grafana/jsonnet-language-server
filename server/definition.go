package server

import (
	"github.com/google/go-jsonnet/ast"
	"github.com/jdbaldry/go-language-server-protocol/lsp/protocol"
)

type NodeStack struct {
	stack []ast.Node
}

func (s *NodeStack) Push(n ast.Node) *NodeStack {
	s.stack = append(s.stack, n)
	return s
}

func (s *NodeStack) Pop() (*NodeStack, ast.Node) {
	l := len(s.stack)
	n := s.stack[l-1]
	s.stack = s.stack[:l-1]
	return s, n
}

func (s *NodeStack) IsEmpty() bool {
	return len(s.stack) == 0
}

func Definition(node ast.Node, params protocol.DefinitionParams) (protocol.DefinitionLink, error) {
	foundDefinition, _ := findDefinition(node, params.Position)
	foundLocRange := foundDefinition.LocRange
	if foundLocRange.Begin.Line == 0 {
		foundLocRange = *foundDefinition.Body.Loc()
	}
	responseDefLink := protocol.DefinitionLink{
		TargetURI: protocol.DocumentURI(foundLocRange.FileName),
		TargetRange: protocol.Range{
			Start: protocol.Position{
				Line:      uint32(foundLocRange.Begin.Line - 1),
				Character: uint32(foundLocRange.Begin.Column - 1),
			},
			End: protocol.Position{
				Line:      uint32(foundLocRange.End.Line - 1),
				Character: uint32(foundLocRange.End.Column - 1),
			},
		},
		TargetSelectionRange: protocol.Range{
			Start: protocol.Position{
				Line:      uint32(foundLocRange.Begin.Line - 1),
				Character: uint32(foundLocRange.Begin.Column - 1),
			},
			End: protocol.Position{
				Line:      uint32(foundLocRange.End.Line - 1),
				Character: uint32(foundLocRange.Begin.Column + len(foundDefinition.Variable)),
			},
		},
	}
	return responseDefLink, nil
}

func findDefinition(node ast.Node, position protocol.Position) (ast.LocalBind, error) {
	queriedNode, searchStack, _ := findNodeByPosition(node, position)
	var matchingBind ast.LocalBind
	switch queriedNode := queriedNode.(type) {
	case *ast.Var:
		matchingBind, _ = findBindByIdViaStack(searchStack, queriedNode.Id)
	}
	return matchingBind, nil
}

func findBindByIdViaStack(stack *NodeStack, id ast.Identifier) (ast.LocalBind, error) {
	for !stack.IsEmpty() {
		_, curr := stack.Pop()
		switch curr := curr.(type) {
		case *ast.Local:
			for _, bind := range curr.Binds {
				if bind.Variable == id {
					return bind, nil
				}
			}
		case *ast.DesugaredObject:
			for _, bind := range curr.Locals {
				if bind.Variable == id {
					return bind, nil
				}
			}
		}
	}
	return ast.LocalBind{}, nil
}

func findBindById(node ast.Node, id ast.Identifier) (ast.LocalBind, error) {
	stack := NodeStack{}
	stack.Push(node)
	for !stack.IsEmpty() {
		stack, curr := stack.Pop()
		switch curr := curr.(type) {
		case *ast.Local:
			for _, bind := range curr.Binds {
				if bind.Variable == id {
					return bind, nil
				}
				stack = stack.Push(bind.Body)
			}
			if curr.Body != nil {
				stack = stack.Push(curr.Body)
			}
		case *ast.DesugaredObject:
			for _, bind := range curr.Locals {
				if bind.Variable == id {
					return bind, nil
				}
				stack = stack.Push(bind.Body)
			}
			for _, field := range curr.Fields {
				stack = stack.Push(field.Body)
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
	return ast.LocalBind{}, nil
}

func findNodeByPosition(node ast.Node, position protocol.Position) (ast.Node, *NodeStack, error) {
	stack := &NodeStack{}
	stack.Push(node)
	// keeps the history of the navigation path to the requested Node.
	// used to backwards search Nodes from the found node to the root.
	searchStack := &NodeStack{}
	for !stack.IsEmpty() {
		stack, curr := stack.Pop()
		if !inRange(position, *curr.Loc()) {
			continue
		} else {
			searchStack = searchStack.Push(curr)
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
				stack = stack.Push(field.Body)
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
		case *ast.Var:
			return curr, searchStack, nil
		}
	}
	return nil, nil, nil
}

func inRange(point protocol.Position, theRange ast.LocationRange) bool {
	if int(point.Line) == theRange.Begin.Line-1 && int(point.Line) == theRange.End.Line-1 {
		return theRange.Begin.Column <= int(point.Character) && int(point.Character) <= theRange.End.Column
	} else {
		return theRange.Begin.Line-1 <= int(point.Line) && int(point.Line) < theRange.End.Line-1
	}
}

// isDefinition returns true if a symbol is tagged as a definition.
func isDefinition(s protocol.DocumentSymbol) bool {
	for _, t := range s.Tags {
		if t == symbolTagDefinition {
			return true
		}
	}
	return false
}
