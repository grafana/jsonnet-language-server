package server

import (
	"context"
	"errors"
	"fmt"
	"path/filepath"
	"strings"

	"github.com/google/go-jsonnet"
	"github.com/google/go-jsonnet/ast"
	"github.com/grafana/jsonnet-language-server/pkg/nodestack"
	"github.com/grafana/jsonnet-language-server/pkg/utils"
	"github.com/jdbaldry/go-language-server-protocol/lsp/protocol"
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
		matchingBind, foundLocRange, err := findBindByIdViaStack(searchStack, deepestNode.Id)
		if err != nil {
			return nil, err
		}
		if foundLocRange.Begin.Line == 0 {
			foundLocRange = *matchingBind.Body.Loc()
		}
		resultRange := ASTRangeToProtocolRange(foundLocRange)
		resultSelectionRange := resultRange
		if matchingBind != nil {
			resultSelectionRange = NewProtocolRange(
				foundLocRange.Begin.Line-1,
				foundLocRange.Begin.Column-1,
				foundLocRange.Begin.Line-1,
				foundLocRange.Begin.Column-1+len(matchingBind.Variable),
			)
		}

		responseDefLink = protocol.DefinitionLink{
			TargetURI:            protocol.DocumentURI(foundLocRange.FileName),
			TargetRange:          resultRange,
			TargetSelectionRange: resultSelectionRange,
		}
	case *ast.SuperIndex, *ast.Index:
		indexSearchStack := nodestack.NewNodeStack(deepestNode)
		indexList := indexSearchStack.BuildIndexList()
		tempSearchStack := *searchStack
		matchingField, locRange, err := findObjectFieldFromIndexList(&tempSearchStack, indexList, vm)
		if err != nil {
			return nil, err
		}
		foundLocRange := locRange
		resultRange := ASTRangeToProtocolRange(foundLocRange)
		resultSelectionRange := resultRange
		if matchingField != nil {
			resultSelectionRange = NewProtocolRange(
				foundLocRange.Begin.Line-1,
				foundLocRange.Begin.Column-1,
				foundLocRange.Begin.Line-1,
				foundLocRange.Begin.Column-1+len(matchingField.Name.(*ast.LiteralString).Value),
			)
		}
		responseDefLink = protocol.DefinitionLink{
			TargetURI:            protocol.DocumentURI(foundLocRange.FileName),
			TargetRange:          resultRange,
			TargetSelectionRange: resultSelectionRange,
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

func findObjectFieldFromIndexList(stack *nodestack.NodeStack, indexList []string, vm *jsonnet.VM) (*ast.DesugaredObjectField, ast.LocationRange, error) {
	var foundField *ast.DesugaredObjectField
	var foundDesugaredObjects []*ast.DesugaredObject
	// First element will be super, self, or var name
	start, indexList := indexList[0], indexList[1:]
	sameFileOnly := false
	if start == "super" {
		// Find the LHS desugared object of a binary node
		lhsObject, err := findLhsDesugaredObject(stack)
		if err != nil {
			return nil, ast.LocationRange{}, err
		}
		foundDesugaredObjects = append(foundDesugaredObjects, lhsObject)
	} else if start == "self" {
		tmpStack := nodestack.NewNodeStack(stack.From)
		tmpStack.Stack = make([]ast.Node, len(stack.Stack))
		copy(tmpStack.Stack, stack.Stack)
		foundDesugaredObjects = findTopLevelObjects(tmpStack, vm)
	} else if start == "std" {
		return nil, ast.LocationRange{}, fmt.Errorf("cannot get definition of std lib")
	} else if strings.Contains(start, ".") {
		rootNode, _, _ := vm.ImportAST("", start)
		foundDesugaredObjects = findTopLevelObjects(nodestack.NewNodeStack(rootNode), vm)
	} else if start == "$" {
		sameFileOnly = true
		foundDesugaredObjects = findTopLevelObjects(nodestack.NewNodeStack(stack.From), vm)
	} else {
		// Get ast.DesugaredObject at variable definition by getting bind then setting ast.DesugaredObject
		bind, locRange, err := findBindByIdViaStack(stack, ast.Identifier(start))
		if err != nil {
			return nil, ast.LocationRange{}, err
		}
		if bind == nil {
			return nil, locRange, nil
		}
		switch bodyNode := bind.Body.(type) {
		case *ast.DesugaredObject:
			foundDesugaredObjects = append(foundDesugaredObjects, bodyNode)
		case *ast.Self:
			tmpStack := nodestack.NewNodeStack(stack.From)
			foundDesugaredObjects = findTopLevelObjects(tmpStack, vm)
		case *ast.Import:
			filename := bodyNode.File.Value
			rootNode, _, _ := vm.ImportAST("", filename)
			foundDesugaredObjects = findTopLevelObjects(nodestack.NewNodeStack(rootNode), vm)
		case *ast.Index:
			tempStack := nodestack.NewNodeStack(bodyNode)
			indexList = append(tempStack.BuildIndexList(), indexList...)
			return findObjectFieldFromIndexList(stack, indexList, vm)
		default:
			return nil, ast.LocationRange{}, fmt.Errorf("unexpected node type when finding bind for '%s'", start)
		}
	}
	for len(indexList) > 0 {
		index := indexList[0]
		indexList = indexList[1:]
		foundField = findObjectFieldInObjects(foundDesugaredObjects, index)
		foundDesugaredObjects = foundDesugaredObjects[:0]
		if foundField == nil {
			return nil, ast.LocationRange{}, fmt.Errorf("field %s was not found in ast.DesugaredObject", index)
		}
		if len(indexList) == 0 {
			return foundField, foundField.LocRange, nil
		}
		switch fieldNode := foundField.Body.(type) {
		case *ast.Var:
			bind, _, _ := findBindByIdViaStack(stack, fieldNode.Id)
			foundDesugaredObjects = append(foundDesugaredObjects, bind.Body.(*ast.DesugaredObject))
		case *ast.DesugaredObject:
			stack = stack.Push(fieldNode)
			foundDesugaredObjects = append(foundDesugaredObjects, findDesugaredObjectFromStack(stack))
		case *ast.Index:
			tempStack := nodestack.NewNodeStack(fieldNode)
			additionalIndexList := tempStack.BuildIndexList()
			additionalIndexList = append(additionalIndexList, indexList...)
			result, locRange, err := findObjectFieldFromIndexList(stack, additionalIndexList, vm)
			if sameFileOnly && result.LocRange.FileName != stack.From.Loc().FileName {
				continue
			}
			return result, locRange, err
		case *ast.Import:
			filename := fieldNode.File.Value
			rootNode, _, _ := vm.ImportAST(string(fieldNode.Loc().File.DiagnosticFileName), filename)
			foundDesugaredObjects = findTopLevelObjects(nodestack.NewNodeStack(rootNode), vm)
		}
	}
	return foundField, foundField.LocRange, nil
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

func findDesugaredObjectFromStack(stack *nodestack.NodeStack) *ast.DesugaredObject {
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
func findTopLevelObjects(stack *nodestack.NodeStack, vm *jsonnet.VM) []*ast.DesugaredObject {
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

func findLhsDesugaredObject(stack *nodestack.NodeStack) (*ast.DesugaredObject, error) {
	for !stack.IsEmpty() {
		_, curr := stack.Pop()
		switch curr := curr.(type) {
		case *ast.Binary:
			lhsNode := curr.Left
			switch lhsNode := lhsNode.(type) {
			case *ast.DesugaredObject:
				return lhsNode, nil
			case *ast.Var:
				bind, _, _ := findBindByIdViaStack(stack, lhsNode.Id)
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

func findBindByIdViaStack(stack *nodestack.NodeStack, id ast.Identifier) (*ast.LocalBind, ast.LocationRange, error) {
	for !stack.IsEmpty() {
		_, curr := stack.Pop()
		switch curr := curr.(type) {
		case *ast.Local:
			for _, bind := range curr.Binds {
				if bind.Variable == id {
					return &bind, bind.LocRange, nil
				}
			}
		case *ast.DesugaredObject:
			for _, bind := range curr.Locals {
				if bind.Variable == id {
					return &bind, bind.LocRange, nil
				}
			}
		case *ast.Function:
			for _, param := range curr.Parameters {
				if param.Name == id {
					return nil, param.LocRange, nil
				}
			}
		}

	}
	return nil, ast.LocationRange{}, fmt.Errorf("unable to find matching bind for %s", id)
}

func findNodeByPosition(node ast.Node, position protocol.Position) (*nodestack.NodeStack, error) {
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
