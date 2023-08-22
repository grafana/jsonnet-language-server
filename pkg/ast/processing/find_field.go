package processing

import (
	"fmt"
	"reflect"
	"strings"

	"github.com/google/go-jsonnet"
	"github.com/google/go-jsonnet/ast"
	"github.com/grafana/jsonnet-language-server/pkg/nodestack"
	log "github.com/sirupsen/logrus"
)

func FindRangesFromIndexList(stack *nodestack.NodeStack, indexList []string, vm *jsonnet.VM, partialMatchFields bool) ([]ObjectRange, error) {
	var foundDesugaredObjects []*ast.DesugaredObject
	// First element will be super, self, or var name
	start, indexList := indexList[0], indexList[1:]
	sameFileOnly := false
	switch {
	case start == "super":
		// Find the LHS desugared object of a binary node
		lhsObject, err := findLHSDesugaredObject(stack)
		if err != nil {
			return nil, err
		}
		foundDesugaredObjects = append(foundDesugaredObjects, lhsObject)
	case start == "self":
		tmpStack := stack.Clone()

		// Special case. If the index was part of a binary node (ex: self.foo + {...}),
		//   then the second element's content should not be considered to find the index's reference
		if _, ok := tmpStack.Peek().(*ast.Binary); ok {
			tmpStack.Pop()
		}
		foundDesugaredObjects = filterSelfScope(FindTopLevelObjects(tmpStack, vm))
	case start == "std":
		return nil, fmt.Errorf("cannot get definition of std lib")
	case start == "$":
		sameFileOnly = true
		foundDesugaredObjects = FindTopLevelObjects(nodestack.NewNodeStack(stack.From), vm)
	case strings.Contains(start, "."):
		foundDesugaredObjects = FindTopLevelObjectsInFile(vm, start, "")

	default:
		// Get ast.DesugaredObject at variable definition by getting bind then setting ast.DesugaredObject
		bind := FindBindByIDViaStack(stack, ast.Identifier(start))
		if bind == nil {
			param := FindParameterByIDViaStack(stack, ast.Identifier(start), partialMatchFields)
			if param != nil {
				return []ObjectRange{
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
			foundDesugaredObjects = FindTopLevelObjects(tmpStack, vm)
		case *ast.Import:
			filename := bodyNode.File.Value
			foundDesugaredObjects = FindTopLevelObjectsInFile(vm, filename, "")
		case *ast.Index, *ast.Apply:
			tempStack := nodestack.NewNodeStack(bodyNode)
			indexList = append(tempStack.BuildIndexList(), indexList...)
			return FindRangesFromIndexList(stack, indexList, vm, partialMatchFields)
		case *ast.Function:
			// If the function's body is an object, it means we can look for indexes within the function
			if funcBody := findChildDesugaredObject(bodyNode.Body); funcBody != nil {
				foundDesugaredObjects = append(foundDesugaredObjects, funcBody)
			}
		default:
			return nil, fmt.Errorf("unexpected node type when finding bind for '%s': %s", start, reflect.TypeOf(bind.Body))
		}
	}

	return extractObjectRangesFromDesugaredObjs(stack, vm, foundDesugaredObjects, sameFileOnly, indexList, partialMatchFields)
}

func extractObjectRangesFromDesugaredObjs(stack *nodestack.NodeStack, vm *jsonnet.VM, desugaredObjs []*ast.DesugaredObject, sameFileOnly bool, indexList []string, partialMatchFields bool) ([]ObjectRange, error) {
	var ranges []ObjectRange
	for len(indexList) > 0 {
		index := indexList[0]
		indexList = indexList[1:]
		partialMatchFields := partialMatchFields && len(indexList) == 0 // Only partial match on the last index. Others are considered complete
		foundFields := findObjectFieldsInObjects(desugaredObjs, index, partialMatchFields)
		desugaredObjs = nil
		if len(foundFields) == 0 {
			return nil, fmt.Errorf("field %s was not found in ast.DesugaredObject", index)
		}
		if len(indexList) == 0 {
			for _, found := range foundFields {
				ranges = append(ranges, FieldToRange(*found))

				// If the field is not PlusSuper (field+: value), we stop there. Other previous values are not relevant
				// If partialMatchFields is true, we can continue to look for other fields
				if !found.PlusSuper && !partialMatchFields {
					break
				}
			}
			return ranges, nil
		}

		fieldNodes, err := unpackFieldNodes(vm, foundFields)
		if err != nil {
			return nil, err
		}

		i := 0
		for i < len(fieldNodes) {
			fieldNode := fieldNodes[i]
			switch fieldNode := fieldNode.(type) {
			case *ast.Apply:
				// Add the target of the Apply to the list of field nodes to look for
				// The target is a function and will be found by FindVarReference on the next loop
				fieldNodes = append(fieldNodes, fieldNode.Target)
			case *ast.Var:
				varReference, err := FindVarReference(fieldNode, vm)
				if err != nil {
					return nil, err
				}
				// If the reference is an object, add it directly to the list of objects to look in
				// Otherwise, add it back to the list for further processing
				if varReferenceObj := findChildDesugaredObject(varReference); varReferenceObj != nil {
					desugaredObjs = append(desugaredObjs, varReferenceObj)
				} else {
					fieldNodes = append(fieldNodes, varReference)
				}
			case *ast.DesugaredObject:
				desugaredObjs = append(desugaredObjs, fieldNode)
			case *ast.Index:
				additionalIndexList := append(nodestack.NewNodeStack(fieldNode).BuildIndexList(), indexList...)
				result, err := FindRangesFromIndexList(stack, additionalIndexList, vm, partialMatchFields)
				if len(result) > 0 {
					if !sameFileOnly || result[0].Filename == stack.From.Loc().FileName {
						return result, err
					}
				}

				fieldNodes = append(fieldNodes, fieldNode.Target)
			case *ast.Function:
				desugaredObjs = append(desugaredObjs, findChildDesugaredObject(fieldNode.Body))
			case *ast.Import:
				filename := fieldNode.File.Value
				newObjs := FindTopLevelObjectsInFile(vm, filename, string(fieldNode.Loc().File.DiagnosticFileName))
				desugaredObjs = append(desugaredObjs, newObjs...)
			}
			i++
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

func findObjectFieldsInObjects(objectNodes []*ast.DesugaredObject, index string, partialMatchFields bool) []*ast.DesugaredObjectField {
	var matchingFields []*ast.DesugaredObjectField
	for _, object := range objectNodes {
		fields := findObjectFieldsInObject(object, index, partialMatchFields)
		matchingFields = append(matchingFields, fields...)
	}
	return matchingFields
}

func findObjectFieldsInObject(objectNode *ast.DesugaredObject, index string, partialMatchFields bool) []*ast.DesugaredObjectField {
	if objectNode == nil {
		return nil
	}

	var matchingFields []*ast.DesugaredObjectField
	for _, field := range objectNode.Fields {
		field := field
		literalString, isString := field.Name.(*ast.LiteralString)
		if !isString {
			continue
		}
		log.Debugf("Checking index name %s against field name %s", index, literalString.Value)
		if index == literalString.Value || (partialMatchFields && strings.HasPrefix(literalString.Value, index)) {
			matchingFields = append(matchingFields, &field)
			if !partialMatchFields {
				break
			}
		}
	}
	return matchingFields
}

func findChildDesugaredObject(node ast.Node) *ast.DesugaredObject {
	switch node := node.(type) {
	case *ast.DesugaredObject:
		return node
	case *ast.Binary:
		if res := findChildDesugaredObject(node.Left); res != nil {
			return res
		}
		if res := findChildDesugaredObject(node.Right); res != nil {
			return res
		}
	}
	return nil
}

// FindVarReference finds the object that the variable is referencing
// To do so, we get the stack where the var is used and search that stack for the var's definition
func FindVarReference(varNode *ast.Var, vm *jsonnet.VM) (ast.Node, error) {
	varFileNode, _, _ := vm.ImportAST("", varNode.LocRange.FileName)
	varStack, err := FindNodeByPosition(varFileNode, varNode.Loc().Begin)
	if err != nil {
		return nil, fmt.Errorf("got the following error when finding the bind for %s: %w", varNode.Id, err)
	}
	bind := FindBindByIDViaStack(varStack, varNode.Id)
	if bind == nil {
		return nil, fmt.Errorf("could not find bind for %s", varNode.Id)
	}
	return bind.Body, nil
}

func findLHSDesugaredObject(stack *nodestack.NodeStack) (*ast.DesugaredObject, error) {
	for !stack.IsEmpty() {
		curr := stack.Pop()
		switch curr := curr.(type) {
		case *ast.Binary:
			lhsNode := curr.Left
			switch lhsNode := lhsNode.(type) {
			case *ast.DesugaredObject:
				return lhsNode, nil
			case *ast.Var:
				bind := FindBindByIDViaStack(stack, lhsNode.Id)
				if bind != nil {
					if bindBody := findChildDesugaredObject(bind.Body); bindBody != nil {
						return bindBody, nil
					}
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
