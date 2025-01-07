package processing

import (
	"fmt"
	"reflect"
	"strings"

	"github.com/google/go-jsonnet/ast"
	"github.com/grafana/jsonnet-language-server/pkg/nodestack"
	log "github.com/sirupsen/logrus"
)

func (p *Processor) FindRangesFromIndexList(stack *nodestack.NodeStack, indexList []string, partialMatchFields bool) ([]ObjectRange, error) {
	var foundDesugaredObjects []*ast.DesugaredObject
	// First element will be super, self, or var name
	start, indexList := indexList[0], indexList[1:]
	switch {
	case start == "super":
		// Find the LHS desugared object of a binary node
		lhsObject, err := p.findLHSDesugaredObject(stack)
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
		foundDesugaredObjects = filterSelfScope(p.FindTopLevelObjects(tmpStack))
	case start == "std":
		return nil, fmt.Errorf("cannot get definition of std lib")
	case start == "$":
		foundDesugaredObjects = p.FindTopLevelObjects(nodestack.NewNodeStack(stack.From))
	case strings.Contains(start, "."):
		foundDesugaredObjects = p.FindTopLevelObjectsInFile(start, "")

	default:
		if strings.Count(start, "(") == 1 && strings.Count(start, ")") == 1 {
			// If the index is a function call, we need to find the function definition
			// We can ignore the arguments. We'll only consider static attributes from the function's body
			start = strings.Split(start, "(")[0]
		}
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
			foundDesugaredObjects = p.FindTopLevelObjects(tmpStack)
		case *ast.Import:
			filename := bodyNode.File.Value
			foundDesugaredObjects = p.FindTopLevelObjectsInFile(filename, "")
		case *ast.Index, *ast.Apply:
			tempStack := nodestack.NewNodeStack(bodyNode)
			indexList = append(tempStack.BuildIndexList(), indexList...)
			return p.FindRangesFromIndexList(stack, indexList, partialMatchFields)
		case *ast.Function:
			// If the function's body is an object, it means we can look for indexes within the function
			foundDesugaredObjects = append(foundDesugaredObjects, p.findChildDesugaredObjects(bodyNode.Body)...)
		default:
			return nil, fmt.Errorf("unexpected node type when finding bind for '%s': %s", start, reflect.TypeOf(bind.Body))
		}
	}

	return p.extractObjectRangesFromDesugaredObjs(foundDesugaredObjects, indexList, partialMatchFields)
}

func (p *Processor) extractObjectRangesFromDesugaredObjs(desugaredObjs []*ast.DesugaredObject, indexList []string, partialMatchFields bool) ([]ObjectRange, error) {
	var ranges []ObjectRange
	for len(indexList) > 0 {
		index := indexList[0]
		indexList = indexList[1:]
		partialMatchCurrentField := partialMatchFields && len(indexList) == 0 // Only partial match on the last index. Others are considered complete
		foundFields := findObjectFieldsInObjects(desugaredObjs, index, partialMatchCurrentField)
		desugaredObjs = nil
		if len(foundFields) == 0 {
			return nil, fmt.Errorf("field %s was not found in ast.DesugaredObject", index)
		}
		if len(indexList) == 0 {
			for _, found := range foundFields {
				ranges = append(ranges, p.FieldToRange(*found))

				// If the field is not PlusSuper (field+: value), we stop there. Other previous values are not relevant
				// If partialMatchCurrentField is true, we can continue to look for other fields
				if !found.PlusSuper && !partialMatchCurrentField {
					break
				}
			}
			return ranges, nil
		}

		fieldNodes := p.unpackFieldNodes(foundFields)
		i := 0
		for i < len(fieldNodes) {
			fieldNode := fieldNodes[i]
			switch fieldNode := fieldNode.(type) {
			default:
				desugaredObjs = append(desugaredObjs, p.findChildDesugaredObjects(fieldNode)...)
			case *ast.Apply:
				// Add the target of the Apply to the list of field nodes to look for
				// The target is a function and will be found by FindVarReference on the next loop
				fieldNodes = append(fieldNodes, fieldNode.Target)
			case *ast.Var:
				varReference, err := p.FindVarReference(fieldNode)
				if err != nil {
					return nil, err
				}
				// If the reference is an object, add it directly to the list of objects to look in
				// Otherwise, add it back to the list for further processing
				if varReferenceObjs := p.findChildDesugaredObjects(varReference); len(varReferenceObjs) > 0 {
					desugaredObjs = append(desugaredObjs, varReferenceObjs...)
				} else {
					fieldNodes = append(fieldNodes, varReference)
				}

			case *ast.Index:
				// if we're trying to find the a definition which is an index,
				// we need to find it from itself, meaning that we need to create a stack
				// from the index's target and search from there
				rootNode, _, _ := p.vm.ImportAST("", fieldNode.LocRange.FileName)
				stack, _ := FindNodeByPosition(rootNode, fieldNode.Target.Loc().Begin)
				if stack != nil {
					additionalIndexList := append(nodestack.NewNodeStack(fieldNode).BuildIndexList(), indexList...)
					result, _ := p.FindRangesFromIndexList(stack, additionalIndexList, partialMatchFields)
					if len(result) > 0 {
						return result, nil
					}
				}

				fieldNodes = append(fieldNodes, fieldNode.Target)
			case *ast.Function:
				fieldNodes = append(fieldNodes, fieldNode.Body)
			case *ast.Import:
				filename := fieldNode.File.Value
				newObjs := p.FindTopLevelObjectsInFile(filename, string(fieldNode.Loc().File.DiagnosticFileName))
				desugaredObjs = append(desugaredObjs, newObjs...)
			case *ast.Binary:
				fieldNodes = append(fieldNodes, flattenBinary(fieldNode)...)
			case *ast.Self:
				desugaredObjs = append(desugaredObjs, p.findSelfObject(fieldNode))
			}
			i++
		}
	}
	return ranges, nil
}

func flattenBinary(node ast.Node) []ast.Node {
	binary, nodeIsBinary := node.(*ast.Binary)
	if !nodeIsBinary {
		return []ast.Node{node}
	}
	return append(flattenBinary(binary.Right), flattenBinary(binary.Left)...)
}

// unpackFieldNodes extracts nodes from fields
// - Binary nodes. A field could be either in the left or right side of the binary
// - Self nodes. We want the object self refers to, not the self node itself
func (p *Processor) unpackFieldNodes(fields []*ast.DesugaredObjectField) []ast.Node {
	var fieldNodes []ast.Node
	for _, foundField := range fields {
		switch fieldNode := foundField.Body.(type) {
		case *ast.Self:
			fieldNodes = append(fieldNodes, p.findSelfObject(fieldNode))
		case *ast.Binary:
			fieldNodes = append(fieldNodes, flattenBinary(fieldNode)...)
		default:
			fieldNodes = append(fieldNodes, fieldNode)
		}
	}

	return fieldNodes
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

func (p *Processor) findChildDesugaredObjects(node ast.Node) []*ast.DesugaredObject {
	switch node := node.(type) {
	case *ast.DesugaredObject:
		return []*ast.DesugaredObject{node}
	case *ast.Binary:
		var res []*ast.DesugaredObject
		res = append(res, p.findChildDesugaredObjects(node.Left)...)
		res = append(res, p.findChildDesugaredObjects(node.Right)...)
		return res
	}
	return nil
}

// FindVarReference finds the object that the variable is referencing
// To do so, we get the stack where the var is used and search that stack for the var's definition
func (p *Processor) FindVarReference(varNode *ast.Var) (ast.Node, error) {
	varFileNode, _, _ := p.vm.ImportAST("", varNode.LocRange.FileName)
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

func (p *Processor) findLHSDesugaredObject(stack *nodestack.NodeStack) (*ast.DesugaredObject, error) {
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
					if binds := p.findChildDesugaredObjects(bind.Body); len(binds) > 0 {
						return binds[0], nil
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

func (p *Processor) findSelfObject(self *ast.Self) *ast.DesugaredObject {
	filename := self.LocRange.FileName
	rootNode, _, _ := p.vm.ImportAST("", filename)
	tmpStack, err := FindNodeByPosition(rootNode, self.LocRange.Begin)
	if err != nil {
		return nil
	}
	for !tmpStack.IsEmpty() {
		node := tmpStack.Pop()
		if castNode, ok := node.(*ast.DesugaredObject); ok {
			return castNode
		}
	}
	return nil
}
