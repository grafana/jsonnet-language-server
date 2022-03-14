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
		tmpStack := stack.Clone()

		// Special case. If the index was part of a binary node (ex: self.foo + {...}),
		//   then the second element's content should not be considered to find the index's reference
		if _, ok := tmpStack.Peek().(*ast.Binary); ok {
			tmpStack.Pop()
		}

		foundDesugaredObjects = filterSelfScope(findTopLevelObjects(tmpStack, vm))
	} else if start == "std" {
		return nil, fmt.Errorf("cannot get definition of std lib")
	} else if strings.Contains(start, ".") {
		foundDesugaredObjects = findTopLevelObjectsInFile(vm, start, "")
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
			foundDesugaredObjects = findTopLevelObjectsInFile(vm, filename, "")
		case *ast.Index:
			tempStack := nodestack.NewNodeStack(bodyNode)
			indexList = append(tempStack.BuildIndexList(), indexList...)
			return FindRangesFromIndexList(stack, indexList, vm)
		default:
			return nil, fmt.Errorf("unexpected node type when finding bind for '%s'", start)
		}
	}
	var ranges []objectRange
	for len(indexList) > 0 {
		index := indexList[0]
		indexList = indexList[1:]
		foundFields := findObjectFieldsInObjects(foundDesugaredObjects, index)
		foundDesugaredObjects = nil
		if len(foundFields) == 0 {
			return nil, fmt.Errorf("field %s was not found in ast.DesugaredObject", index)
		}
		if len(indexList) == 0 {
			for _, found := range foundFields {
				ranges = append(ranges, fieldToRange(found))

				// If the field is not PlusSuper (field+: value), we stop there. Other previous values are not relevant
				if !found.PlusSuper {
					break
				}
			}
			return ranges, nil
		}

		fieldNodes, err := unpackFieldNodes(vm, foundFields)
		if err != nil {
			return nil, err
		}

		for _, fieldNode := range fieldNodes {
			switch fieldNode := fieldNode.(type) {
			case *ast.Var:
				varReference, err := findVarReference(fieldNode, vm)
				if err != nil {
					return nil, err
				}
				foundDesugaredObjects = append(foundDesugaredObjects, varReference.(*ast.DesugaredObject))
			case *ast.DesugaredObject:
				stack.Push(fieldNode)
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
				newObjs := findTopLevelObjectsInFile(vm, filename, string(fieldNode.Loc().File.DiagnosticFileName))
				foundDesugaredObjects = append(foundDesugaredObjects, newObjs...)
			}
		}
	}

	return ranges, nil
}

// unpackFieldNodes extracts nodes from fields
// - Binary nodes. A field could be either in the left or right side of the binary
// - Self nodes. We want the object self refers to, not the self node itself
func unpackFieldNodes(vm *jsonnet.VM, fields []*ast.DesugaredObjectField) ([]ast.Node, error) {
	var fieldNodes []ast.Node
	for _, foundField := range fields {
		switch fieldNode := foundField.Body.(type) {
		case *ast.Self:
			filename := fieldNode.LocRange.FileName
			rootNode, _, _ := vm.ImportAST("", filename)
			tmpStack, err := FindNodeByPosition(rootNode, fieldNode.LocRange.Begin)
			if err != nil {
				return nil, err
			}
			for !tmpStack.IsEmpty() {
				node := tmpStack.Pop()
				if _, ok := node.(*ast.DesugaredObject); ok {
					fieldNodes = append(fieldNodes, node)
				}
			}
		case *ast.Binary:
			fieldNodes = append(fieldNodes, fieldNode.Right)
			fieldNodes = append(fieldNodes, fieldNode.Left)
		default:
			fieldNodes = append(fieldNodes, fieldNode)
		}
	}

	return fieldNodes, nil
}

func findObjectFieldsInObjects(objectNodes []*ast.DesugaredObject, index string) []*ast.DesugaredObjectField {
	var matchingFields []*ast.DesugaredObjectField
	for _, object := range objectNodes {
		field := findObjectFieldInObject(object, index)
		if field != nil {
			matchingFields = append(matchingFields, field)
		}
	}
	return matchingFields
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
		curr := stack.Pop()
		switch curr := curr.(type) {
		case *ast.DesugaredObject:
			return curr
		}
	}
	return nil
}

// findVarReference finds the object that the variable is referencing
// To do so, we get the stack where the var is used and search that stack for the var's definition
func findVarReference(varNode *ast.Var, vm *jsonnet.VM) (ast.Node, error) {
	varFileNode, _, _ := vm.ImportAST("", varNode.LocRange.FileName)
	varStack, err := FindNodeByPosition(varFileNode, varNode.Loc().Begin)
	if err != nil {
		return nil, fmt.Errorf("got the following error when finding the bind for %s: %w", varNode.Id, err)
	}
	bind := FindBindByIdViaStack(varStack, varNode.Id)
	if bind == nil {
		return nil, fmt.Errorf("could not find bind for %s", varNode.Id)
	}
	return bind.Body, nil
}

func findLhsDesugaredObject(stack *nodestack.NodeStack) (*ast.DesugaredObject, error) {
	for !stack.IsEmpty() {
		curr := stack.Pop()
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
				stack.Push(bind.Body)
			}
			if curr.Body != nil {
				stack.Push(curr.Body)
			}
		}
	}
	return nil, fmt.Errorf("could not find a lhs object")
}
