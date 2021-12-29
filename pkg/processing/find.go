package processing

import (
	"errors"
	"fmt"
	"strings"

	"github.com/google/go-jsonnet"
	"github.com/google/go-jsonnet/ast"
	"github.com/grafana/jsonnet-language-server/pkg/nodestack"
	"github.com/grafana/jsonnet-language-server/pkg/position"

	log "github.com/sirupsen/logrus"
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

func FindObjectFieldFromIndexList(stack *nodestack.NodeStack, indexList []string, vm *jsonnet.VM) (*ast.DesugaredObjectField, ast.LocationRange, error) {
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
		bind, locRange, err := FindBindByIdViaStack(stack, ast.Identifier(start))
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
			return FindObjectFieldFromIndexList(stack, indexList, vm)
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
			bind, _, _ := FindBindByIdViaStack(stack, fieldNode.Id)
			foundDesugaredObjects = append(foundDesugaredObjects, bind.Body.(*ast.DesugaredObject))
		case *ast.DesugaredObject:
			stack = stack.Push(fieldNode)
			foundDesugaredObjects = append(foundDesugaredObjects, findDesugaredObjectFromStack(stack))
		case *ast.Index:
			tempStack := nodestack.NewNodeStack(fieldNode)
			additionalIndexList := tempStack.BuildIndexList()
			additionalIndexList = append(additionalIndexList, indexList...)
			result, locRange, err := FindObjectFieldFromIndexList(stack, additionalIndexList, vm)
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

func FindBindByIdViaStack(stack *nodestack.NodeStack, id ast.Identifier) (*ast.LocalBind, ast.LocationRange, error) {
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
				bind, _, _ := FindBindByIdViaStack(stack, lhsNode.Id)
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
