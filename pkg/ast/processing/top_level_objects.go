package processing

import (
	"github.com/google/go-jsonnet"
	"github.com/google/go-jsonnet/ast"
	"github.com/grafana/jsonnet-language-server/pkg/nodestack"
	log "github.com/sirupsen/logrus"
)

var fileTopLevelObjectsCache = make(map[string][]*ast.DesugaredObject)

func FindTopLevelObjectsInFile(vm *jsonnet.VM, filename, importedFrom string) []*ast.DesugaredObject {
	cacheKey := importedFrom + ":" + filename
	if _, ok := fileTopLevelObjectsCache[cacheKey]; !ok {
		rootNode, _, _ := vm.ImportAST(importedFrom, filename)
		fileTopLevelObjectsCache[cacheKey] = FindTopLevelObjects(nodestack.NewNodeStack(rootNode), vm)
	}

	return fileTopLevelObjectsCache[cacheKey]
}

// Find all ast.DesugaredObject's from NodeStack
func FindTopLevelObjects(stack *nodestack.NodeStack, vm *jsonnet.VM) []*ast.DesugaredObject {
	var objects []*ast.DesugaredObject
	for !stack.IsEmpty() {
		curr := stack.Pop()
		switch curr := curr.(type) {
		case *ast.DesugaredObject:
			objects = append(objects, curr)
		case *ast.Binary:
			stack.Push(curr.Left)
			stack.Push(curr.Right)
		case *ast.Local:
			stack.Push(curr.Body)
		case *ast.Import:
			filename := curr.File.Value
			rootNode, _, _ := vm.ImportAST(string(curr.Loc().File.DiagnosticFileName), filename)
			stack.Push(rootNode)
		case *ast.Index:
			container := stack.Peek()
			if containerObj, containerIsObj := container.(*ast.DesugaredObject); containerIsObj {
				indexValue, indexIsString := curr.Index.(*ast.LiteralString)
				if !indexIsString {
					continue
				}
				objs := findObjectFieldsInObject(containerObj, indexValue.Value, false)
				if len(objs) > 0 {
					stack.Push(objs[0].Body)
				}
			}
		case *ast.Var:
			varReference, err := FindVarReference(curr, vm)
			if err != nil {
				log.WithError(err).Errorf("Error finding var reference, ignoring this node")
				continue
			}
			stack.Push(varReference)
		case *ast.Function:
			stack.Push(curr.Body)
		}
	}
	return objects
}
