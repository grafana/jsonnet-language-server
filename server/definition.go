package server

import (
	"context"
	"errors"
	"fmt"
	"path/filepath"
	"sort"
	"strings"

	"github.com/google/go-jsonnet"
	"github.com/google/go-jsonnet/ast"
	"github.com/jdbaldry/go-language-server-protocol/lsp/protocol"
	"github.com/jdbaldry/jsonnet-language-server/utils"
	log "github.com/sirupsen/logrus"
)

func (s *server) Definition(ctx context.Context, params *protocol.DefinitionParams) (protocol.Definition, error) {
	definitionLink, err := s.DefinitionLink(ctx, params)
	if err != nil {
		return nil, nil
	}
	definition := protocol.Definition{
		{
			URI:   definitionLink.TargetURI,
			Range: definitionLink.TargetRange,
		},
	}
	return definition, nil
}

func (s *server) DefinitionLink(ctx context.Context, params *protocol.DefinitionParams) (*protocol.DefinitionLink, error) {
	doc, err := s.cache.get(params.TextDocument.URI)
	if err != nil {
		return nil, utils.LogErrorf("Definition: %s: %w", errorRetrievingDocument, err)
	}

	if doc.ast == nil {
		return nil, utils.LogErrorf("Definition: error parsing the document")
	}

	vm, err := s.getVM(doc.item.URI.SpanURI().Filename())
	if err != nil {
		return nil, utils.LogErrorf("error creating the VM: %w", err)
	}
	definition, err := Definition(doc.ast, params, vm)
	if err != nil {
		log.Warn(err.Error())
		return nil, err
	}

	return definition, nil
}

type NodeStack struct {
	from  ast.Node
	stack []ast.Node
}

func NewNodeStack(from ast.Node) *NodeStack {
	return &NodeStack{
		from:  from,
		stack: []ast.Node{from},
	}
}

func (s *NodeStack) Push(n ast.Node) *NodeStack {
	s.stack = append(s.stack, n)
	return s
}

func (s *NodeStack) Pop() (*NodeStack, ast.Node) {
	l := len(s.stack)
	if l == 0 {
		return s, nil
	}
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

func Definition(node ast.Node, params *protocol.DefinitionParams, vm *jsonnet.VM) (*protocol.DefinitionLink, error) {
	responseDefLink, err := findDefinition(node, params, vm)
	if err != nil {
		return nil, err
	}
	return responseDefLink, nil
}

func findDefinition(root ast.Node, params *protocol.DefinitionParams, vm *jsonnet.VM) (*protocol.DefinitionLink, error) {
	position := params.Position
	searchStack, _ := findNodeByPosition(root, position)
	var deepestNode ast.Node
	searchStack, deepestNode = searchStack.Pop()
	var responseDefLink protocol.DefinitionLink
	switch deepestNode := deepestNode.(type) {
	case *ast.Var:
		log.Debugf("Found Var node %s", deepestNode.Id)
		matchingBind, err := findBindByIdViaStack(searchStack, deepestNode.Id)
		if err != nil {
			return nil, err
		}
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
					Line:      uint32(foundLocRange.Begin.Line - 1),
					Character: uint32(foundLocRange.Begin.Column - 1 + len(matchingBind.Variable)),
				},
			},
		}
	case *ast.SuperIndex, *ast.Index:
		indexSearchStack := NewNodeStack(deepestNode)
		indexList := buildIndexList(indexSearchStack)
		tempSearchStack := *searchStack
		matchingField, err := findObjectFieldFromIndexList(&tempSearchStack, indexList, vm)
		if err != nil {
			return nil, err
		}
		foundLocRange := &matchingField.LocRange
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
					Line:      uint32(foundLocRange.Begin.Line - 1),
					Character: uint32(foundLocRange.Begin.Column - 1 + len(matchingField.Name.(*ast.LiteralString).Value)),
				},
			},
		}
	case *ast.Import:
		filename := deepestNode.File.Value
		importedFile, _ := vm.ResolveImport(string(params.TextDocument.URI), filename)
		responseDefLink = protocol.DefinitionLink{
			TargetURI: protocol.DocumentURI(importedFile),
		}
	default:
		log.Debugf("cannot find definition for node type %T", deepestNode)
		return nil, fmt.Errorf("cannot find definition")

	}

	link := string(responseDefLink.TargetURI)
	if !strings.HasPrefix(link, "file://") {
		targetFile, err := filepath.Abs(link)
		if err != nil {
			return nil, err
		}
		responseDefLink.TargetURI = protocol.URIFromPath(targetFile)
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
		case *ast.Self:
			indexList = append(indexList, "self")
		case *ast.Var:
			indexList = append(indexList, string(curr.Id))
		case *ast.Import:
			indexList = append(indexList, curr.File.Value)
		}
	}
	return indexList
}

func findObjectFieldFromIndexList(stack *NodeStack, indexList []string, vm *jsonnet.VM) (*ast.DesugaredObjectField, error) {
	var foundField *ast.DesugaredObjectField
	var foundDesugaredObjects []*ast.DesugaredObject
	// First element will be super, self, or var name
	start, indexList := indexList[0], indexList[1:]
	sameFileOnly := false
	if start == "super" {
		// Find the LHS desugared object of a binary node
		lhsObject, err := findLhsDesugaredObject(stack)
		if err != nil {
			return nil, err
		}
		foundDesugaredObjects = append(foundDesugaredObjects, lhsObject)
	} else if start == "self" {
		tmpStack := NewNodeStack(stack.from)
		tmpStack.stack = make([]ast.Node, len(stack.stack))
		copy(tmpStack.stack, stack.stack)
		foundDesugaredObjects = findTopLevelObjects(tmpStack, vm)
	} else if start == "std" {
		return nil, fmt.Errorf("cannot get definition of std lib")
	} else if strings.Contains(start, ".") {
		rootNode, _, _ := vm.ImportAST("", start)
		foundDesugaredObjects = findTopLevelObjects(NewNodeStack(rootNode), vm)
	} else if start == "$" {
		sameFileOnly = true
		foundDesugaredObjects = findTopLevelObjects(NewNodeStack(stack.from), vm)
	} else {
		// Get ast.DesugaredObject at variable definition by getting bind then setting ast.DesugaredObject
		bind, err := findBindByIdViaStack(stack, ast.Identifier(start))
		if err != nil {
			return nil, err
		}
		switch bodyNode := bind.Body.(type) {
		case *ast.DesugaredObject:
			foundDesugaredObjects = append(foundDesugaredObjects, bodyNode)
		case *ast.Self:
			tmpStack := NewNodeStack(stack.from)
			foundDesugaredObjects = findTopLevelObjects(tmpStack, vm)
		case *ast.Import:
			filename := bodyNode.File.Value
			rootNode, _, _ := vm.ImportAST("", filename)
			foundDesugaredObjects = findTopLevelObjects(NewNodeStack(rootNode), vm)
		case *ast.Index:
			tempStack := NewNodeStack(bodyNode)
			indexList = append(buildIndexList(tempStack), indexList...)
			return findObjectFieldFromIndexList(stack, indexList, vm)
		default:
			return nil, fmt.Errorf("unexpected node type when finding bind for '%s'", start)
		}
	}
	for len(indexList) > 0 {
		index := indexList[0]
		indexList = indexList[1:]
		foundField = findObjectFieldInObjects(foundDesugaredObjects, index)
		foundDesugaredObjects = foundDesugaredObjects[:0]
		if foundField == nil {
			return nil, fmt.Errorf("field %s was not found in ast.DesugaredObject", index)
		}
		if len(indexList) == 0 {
			return foundField, nil
		}
		switch fieldNode := foundField.Body.(type) {
		case *ast.Var:
			bind, _ := findBindByIdViaStack(stack, fieldNode.Id)
			foundDesugaredObjects = append(foundDesugaredObjects, bind.Body.(*ast.DesugaredObject))
		case *ast.DesugaredObject:
			stack = stack.Push(fieldNode)
			foundDesugaredObjects = append(foundDesugaredObjects, findDesugaredObjectFromStack(stack))
		case *ast.Index:
			tempStack := NewNodeStack(fieldNode)
			additionalIndexList := buildIndexList(tempStack)
			additionalIndexList = append(additionalIndexList, indexList...)
			result, err := findObjectFieldFromIndexList(stack, additionalIndexList, vm)
			if sameFileOnly && result.LocRange.FileName != stack.from.Loc().FileName {
				continue
			}
			return result, err
		case *ast.Import:
			filename := fieldNode.File.Value
			rootNode, _, _ := vm.ImportAST(string(fieldNode.Loc().File.DiagnosticFileName), filename)
			foundDesugaredObjects = findTopLevelObjects(NewNodeStack(rootNode), vm)
		}
	}
	return foundField, nil
}

func findObjectFieldInObjects(objectNodes []*ast.DesugaredObject, index string) *ast.DesugaredObjectField {
	for _, object := range objectNodes {
		field := findObjectFieldInObject(object, index)
		if field != nil {
			return field
		}
	}
	return nil
}

func findObjectFieldInObject(objectNode *ast.DesugaredObject, index string) *ast.DesugaredObjectField {
	if objectNode == nil {
		return nil
	}
	for _, field := range objectNode.Fields {
		literalString, isString := field.Name.(*ast.LiteralString)
		if !isString {
			continue
		}
		log.Debugf("Checking index name %s against field name %s", index, literalString.Value)
		if index == literalString.Value {
			return &field
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

// Find all ast.DesugaredObject's from NodeStack
func findTopLevelObjects(stack *NodeStack, vm *jsonnet.VM) []*ast.DesugaredObject {
	var objects []*ast.DesugaredObject
	for !stack.IsEmpty() {
		_, curr := stack.Pop()
		switch curr := curr.(type) {
		case *ast.DesugaredObject:
			objects = append(objects, curr)
		case *ast.Binary:
			stack = stack.Push(curr.Left)
			stack = stack.Push(curr.Right)
		case *ast.Local:
			stack = stack.Push(curr.Body)
		case *ast.Import:
			filename := curr.File.Value
			rootNode, _, _ := vm.ImportAST(string(curr.Loc().File.DiagnosticFileName), filename)
			stack = stack.Push(rootNode)
		}
	}
	return objects
}

func findLhsDesugaredObject(stack *NodeStack) (*ast.DesugaredObject, error) {
	for !stack.IsEmpty() {
		_, curr := stack.Pop()
		switch curr := curr.(type) {
		case *ast.Binary:
			lhsNode := curr.Left
			switch lhsNode := lhsNode.(type) {
			case *ast.DesugaredObject:
				return lhsNode, nil
			case *ast.Var:
				bind, _ := findBindByIdViaStack(stack, lhsNode.Id)
				if bind != nil {
					return bind.Body.(*ast.DesugaredObject), nil
				}
			}
		case *ast.Local:
			for _, bind := range curr.Binds {
				stack = stack.Push(bind.Body)
			}
			if curr.Body != nil {
				stack = stack.Push(curr.Body)
			}
		}
	}
	return nil, fmt.Errorf("could not find a lhs object")
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
	return nil, fmt.Errorf("unable to find matching bind for %s", id)
}

func findNodeByPosition(node ast.Node, position protocol.Position) (*NodeStack, error) {
	if node == nil {
		return nil, errors.New("node is nil")
	}

	stack := NewNodeStack(node)
	// keeps the history of the navigation path to the requested Node.
	// used to backwards search Nodes from the found node to the root.
	searchStack := &NodeStack{from: stack.from}
	var curr ast.Node
	for !stack.IsEmpty() {
		stack, curr = stack.Pop()
		// This is needed because SuperIndex only spans "key: super" and not the ".foo" after. This only occurs
		// when super only has 1 additional index. "super.foo.bar" will not have this issue
		if curr, isType := curr.(*ast.SuperIndex); isType {
			curr.Loc().End.Column = curr.Loc().End.Column + len(curr.Index.(*ast.LiteralString).Value) + 1
		}
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
