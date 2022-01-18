package processing

import (
	"fmt"
	"strings"

	"github.com/google/go-jsonnet"
	"github.com/google/go-jsonnet/ast"
	"github.com/grafana/jsonnet-language-server/pkg/nodestack"
	log "github.com/sirupsen/logrus"
)

type objectRange struct {
	Filename       string
	SelectionRange ast.LocationRange
	FullRange      ast.LocationRange
}

func fieldToRange(field *ast.DesugaredObjectField) objectRange {
	selectionRange := ast.LocationRange{
		Begin: ast.Location{
			Line:   field.LocRange.Begin.Line,
			Column: field.LocRange.Begin.Column,
		},
		End: ast.Location{
			Line:   field.LocRange.Begin.Line,
			Column: field.LocRange.Begin.Column + len(field.Name.(*ast.LiteralString).Value),
		},
	}
	return objectRange{
		Filename:       field.LocRange.FileName,
		SelectionRange: selectionRange,
		FullRange:      field.LocRange,
	}
}

func FindRangesFromIndexList(stack *nodestack.NodeStack, indexList []string, vm *jsonnet.VM) ([]objectRange, error) {
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
		tmpStack := nodestack.NewNodeStack(stack.From)
		tmpStack.Stack = make([]ast.Node, len(stack.Stack))
		copy(tmpStack.Stack, stack.Stack)
		foundDesugaredObjects = findTopLevelObjects(tmpStack, vm)
	} else if start == "std" {
		return nil, fmt.Errorf("cannot get definition of std lib")
	} else if strings.Contains(start, ".") {
		rootNode, _, _ := vm.ImportAST("", start)
		foundDesugaredObjects = findTopLevelObjects(nodestack.NewNodeStack(rootNode), vm)
	} else if start == "$" {
		sameFileOnly = true
		foundDesugaredObjects = findTopLevelObjects(nodestack.NewNodeStack(stack.From), vm)
	} else {
		// Get ast.DesugaredObject at variable definition by getting bind then setting ast.DesugaredObject
		bind := FindBindByIdViaStack(stack, ast.Identifier(start))
		if bind == nil {
			param := FindParameterByIdViaStack(stack, ast.Identifier(start))
			if param != nil {
				return []objectRange{
					{
						Filename:       param.LocRange.FileName,
						SelectionRange: param.LocRange,
						FullRange:      param.LocRange,
					},
				}, nil
			}
			return nil, fmt.Errorf("could not find bind for %s", start)
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
			return FindRangesFromIndexList(stack, indexList, vm)
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
			return []objectRange{fieldToRange(foundField)}, nil
		}
		switch fieldNode := foundField.Body.(type) {
		case *ast.Var:
			bind := FindBindByIdViaStack(stack, fieldNode.Id)
			if bind == nil {
				return nil, fmt.Errorf("could not find bind for %s", fieldNode.Id)
			}
			foundDesugaredObjects = append(foundDesugaredObjects, bind.Body.(*ast.DesugaredObject))
		case *ast.DesugaredObject:
			stack = stack.Push(fieldNode)
			foundDesugaredObjects = append(foundDesugaredObjects, findDesugaredObjectFromStack(stack))
		case *ast.Index:
			tempStack := nodestack.NewNodeStack(fieldNode)
			additionalIndexList := tempStack.BuildIndexList()
			additionalIndexList = append(additionalIndexList, indexList...)
			result, err := FindRangesFromIndexList(stack, additionalIndexList, vm)
			if sameFileOnly && len(result) > 0 && result[0].Filename != stack.From.Loc().FileName {
				continue
			}
			return result, err
		case *ast.Import:
			filename := fieldNode.File.Value
			rootNode, _, _ := vm.ImportAST(string(fieldNode.Loc().File.DiagnosticFileName), filename)
			foundDesugaredObjects = findTopLevelObjects(nodestack.NewNodeStack(rootNode), vm)
		}
	}

	return []objectRange{fieldToRange(foundField)}, nil
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
				bind := FindBindByIdViaStack(stack, lhsNode.Id)
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
