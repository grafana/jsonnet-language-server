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
	var ranges []objectRange
	for len(indexList) > 0 {
		index := indexList[0]
		indexList = indexList[1:]
		foundFields := findObjectFieldInObjects(foundDesugaredObjects, index)
		foundDesugaredObjects = foundDesugaredObjects[:0]
		if len(foundFields) == 0 {
			return nil, fmt.Errorf("field %s was not found in ast.DesugaredObject", index)
		}
		if len(indexList) == 0 {
			for i, found := range foundFields {
				if i == 0 || foundFields[i-1].PlusSuper {
					ranges = append(ranges, fieldToRange(found))
				}
			}
			return ranges, nil
		}

		// TODO: Multiple levels
		switch fieldNode := foundFields[0].Body.(type) {
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

	return ranges, nil
}

func findObjectFieldInObjects(objectNodes []*ast.DesugaredObject, index string) []*ast.DesugaredObjectField {
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
		case *ast.Index:
			container := stack.Peek()
			if containerObj, containerIsObj := container.(*ast.DesugaredObject); containerIsObj {
				indexValue, indexIsString := curr.Index.(*ast.LiteralString)
				if !indexIsString {
					continue
				}
				obj := findObjectFieldInObject(containerObj, indexValue.Value)
				if obj != nil {
					stack.Push(obj.Body)
				}
			}
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
