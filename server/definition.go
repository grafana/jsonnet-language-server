package server

import (
	"errors"
	"sort"

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

func (s *NodeStack) reorderDesugaredObjects() *NodeStack {
	sort.SliceStable(s.stack, func(i, j int) bool {
		_, iIsDesugared := s.stack[i].(*ast.DesugaredObject)
		_, jIsDesugared := s.stack[j].(*ast.DesugaredObject)
		if !iIsDesugared && !jIsDesugared {
			return false
		}

		iLoc, jLoc := s.stack[i].Loc(), s.stack[j].Loc()
		if iLoc.Begin.Line < jLoc.Begin.Line && iLoc.End.Line > jLoc.End.Line {
			return true
		}

		return false
	})
	return s
}

func Definition(node ast.Node, params protocol.DefinitionParams) (protocol.DefinitionLink, error) {
	responseDefLink, _ := findDefinition(node, params.Position)
	return *responseDefLink, nil
}

func findDefinition(root ast.Node, position protocol.Position) (*protocol.DefinitionLink, error) {
	searchStack, _ := findNodeByPosition(root, position)
	var deepestNode ast.Node
	searchStack, deepestNode = searchStack.Pop()
	var responseDefLink protocol.DefinitionLink
	switch deepestNode := deepestNode.(type) {
	case *ast.Var:
		var matchingBind *ast.LocalBind
		matchingBind, _ = findBindByIdViaStack(searchStack, deepestNode.Id)
		foundLocRange := &matchingBind.LocRange
		if foundLocRange.Begin.Line == 0 {
			foundLocRange = matchingBind.Body.Loc()
		}
		responseDefLink = protocol.DefinitionLink{
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
					Character: uint32(foundLocRange.Begin.Column + len(matchingBind.Variable)),
				},
			},
		}
	case *ast.SuperIndex:
		rootSearchStack := NodeStack{
			stack: []ast.Node{searchStack.stack[len(searchStack.stack)-1]},
		}
		indexList := buildIndexList(&rootSearchStack)
		matchingField, _ := findObjectFieldFromIndexList(&rootSearchStack, indexList)
		print(matchingField)
		//foundLocRange := &matchingField.LocRange
		//if foundLocRange.Begin.Line == 0 {
		//	foundLocRange = matchingField.Body.Loc()
		//}
		//responseDefLink = protocol.DefinitionLink{
		//	TargetURI: protocol.DocumentURI(foundLocRange.FileName),
		//	TargetRange: protocol.Range{
		//		Start: protocol.Position{
		//			Line:      uint32(foundLocRange.Begin.Line - 1),
		//			Character: uint32(foundLocRange.Begin.Column - 1),
		//		},
		//		End: protocol.Position{
		//			Line:      uint32(foundLocRange.End.Line - 1),
		//			Character: uint32(foundLocRange.End.Column - 1),
		//		},
		//	},
		//	TargetSelectionRange: protocol.Range{
		//		Start: protocol.Position{
		//			Line:      uint32(foundLocRange.Begin.Line - 1),
		//			Character: uint32(foundLocRange.Begin.Column - 1),
		//		},
		//		End: protocol.Position{
		//			Line:      uint32(foundLocRange.End.Line - 1),
		//			Character: uint32(foundLocRange.End.Column - 1),
		//		},
		//	},
		//}
		//case *ast.Index:
		//	matchingField, _ := findObjectFieldFromIndexList(searchStack, deepestNode.Index)
		//	foundLocRange := &matchingField.LocRange
		//	if foundLocRange.Begin.Line == 0 {
		//		foundLocRange = matchingField.Body.Loc()
		//	}
		//	responseDefLink = protocol.DefinitionLink{
		//		TargetURI: protocol.DocumentURI(foundLocRange.FileName),
		//		TargetRange: protocol.Range{
		//			Start: protocol.Position{
		//				Line:      uint32(foundLocRange.Begin.Line - 1),
		//				Character: uint32(foundLocRange.Begin.Column - 1),
		//			},
		//			End: protocol.Position{
		//				Line:      uint32(foundLocRange.End.Line - 1),
		//				Character: uint32(foundLocRange.End.Column - 1),
		//			},
		//		},
		//		TargetSelectionRange: protocol.Range{
		//			Start: protocol.Position{
		//				Line:      uint32(foundLocRange.Begin.Line - 1),
		//				Character: uint32(foundLocRange.Begin.Column - 1),
		//			},
		//			End: protocol.Position{
		//				Line:      uint32(foundLocRange.End.Line - 1),
		//				Character: uint32(foundLocRange.End.Column - 1),
		//			},
		//		},
		//	}
	}
	return &responseDefLink, nil
}

func buildIndexList(stack *NodeStack) []string {
	var indexList []string
	for !stack.IsEmpty() {
		_, curr := stack.Pop()
		switch curr := curr.(type) {
		case *ast.SuperIndex:
			stack = stack.Push(curr.Index)
			indexList = append(indexList, "super")
		case *ast.Index:
			stack = stack.Push(curr.Index)
			stack = stack.Push(curr.Target)
		case *ast.LiteralString:
			indexList = append(indexList, curr.Value)
		}
	}
	return indexList
}

func findObjectFieldFromIndexList(stack *NodeStack, indexList []string) (*ast.Node, error) {
	var foundNode *ast.DesugaredObject
	for _, index := range indexList {
		if index == "super" {
			// Find the LHS desugared object of a binary node
			foundNode = findLhsDesugaredObject(stack)
		} else if index == "self" {
			// In our search stack, we want to find the closest ast.DesugaredObject as this will be our
			// self object
			foundNode = findDesugaredObjectFromStack(stack)
		} else {
			field := findFieldInObject(foundNode, index)
			print(field)
		}
	}
	return nil, nil
}

func findFieldInObject(objectNode *ast.DesugaredObject, index string) *ast.Node {
	for _, field := range objectNode.Fields {
		literalString := field.Name.(*ast.LiteralString)
		if index == literalString.Value {
			return &field.Body
		}
	}
	return nil
}

func findDesugaredObjectFromStack(stack *NodeStack) *ast.DesugaredObject {
	for !stack.IsEmpty() {
		_, curr := stack.Pop()
		switch curr := curr.(type) {
		case *ast.DesugaredObject:
			return curr
		}
	}
	return nil
}

func findLhsDesugaredObject(stack *NodeStack) *ast.DesugaredObject {
	for !stack.IsEmpty() {
		_, curr := stack.Pop()
		switch curr := curr.(type) {
		case *ast.Binary:
			if lhs, isType := curr.Left.(*ast.DesugaredObject); isType {
				return lhs
			}
		}
	}
	return nil
}

func findObjectFieldFromIndex(stack *NodeStack, index string) (*ast.DesugaredObjectField, *NodeStack, error) {
	for !stack.IsEmpty() {
		_, curr := stack.Pop()
		switch curr := curr.(type) {
		case *ast.Binary:
			stack = stack.Push(curr.Left)
			stack = stack.Push(curr.Right)
		case *ast.DesugaredObject:
			for _, field := range curr.Fields {
				switch name := field.Name.(type) {
				case *ast.LiteralString:
					if name.Value == index {
						return &field, stack, nil
					}
				}
			}

		}
	}
	return nil, nil, nil
}

func findBindByIdViaStack(stack *NodeStack, id ast.Identifier) (*ast.LocalBind, error) {
	for !stack.IsEmpty() {
		_, curr := stack.Pop()
		switch curr := curr.(type) {
		case *ast.Local:
			for _, bind := range curr.Binds {
				if bind.Variable == id {
					return &bind, nil
				}
			}
		case *ast.DesugaredObject:
			for _, bind := range curr.Locals {
				if bind.Variable == id {
					return &bind, nil
				}
			}
		}
	}
	return nil, nil
}

func findNodeByPosition(node ast.Node, position protocol.Position) (*NodeStack, error) {
	if node == nil {
		return nil, errors.New("node is nil")
	}

	stack := &NodeStack{}
	stack.Push(node)
	// keeps the history of the navigation path to the requested Node.
	// used to backwards search Nodes from the found node to the root.
	searchStack := &NodeStack{}
	for !stack.IsEmpty() {
		stack, curr := stack.Pop()
		inRange := inRange(position, *curr.Loc())
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
				stack = stack.Push(field.Body)
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
	return searchStack.reorderDesugaredObjects(), nil
}

func inRange(point protocol.Position, theRange ast.LocationRange) bool {
	if int(point.Line) == theRange.Begin.Line-1 && int(point.Character) < theRange.Begin.Column-1 {
		return false
	}

	if int(point.Line) == theRange.End.Line-1 && int(point.Character) >= theRange.End.Column-1 {
		return false
	}

	if int(point.Line) != theRange.Begin.Line-1 || int(point.Line) != theRange.End.Line-1 {
		return theRange.Begin.Line-1 <= int(point.Line) && int(point.Line) <= theRange.End.Line-1
	}

	return true
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
