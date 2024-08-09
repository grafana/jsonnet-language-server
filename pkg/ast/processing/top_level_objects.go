package processing

import (
	"github.com/google/go-jsonnet"
	"github.com/google/go-jsonnet/ast"
	"github.com/grafana/jsonnet-language-server/pkg/cache"
	"github.com/grafana/jsonnet-language-server/pkg/nodestack"
	log "github.com/sirupsen/logrus"
)

func FindTopLevelObjectsInFile(cache *cache.Cache, vm *jsonnet.VM, filename, importedFrom string) []*ast.DesugaredObject {
	v, ok := cache.GetTopLevelObject(filename, importedFrom)
	if !ok {
		rootNode, _, _ := vm.ImportAST(importedFrom, filename)
		v = FindTopLevelObjects(cache, nodestack.NewNodeStack(rootNode), vm)
		cache.PutTopLevelObject(filename, importedFrom, v)
	}
	return v
}

// Find all ast.DesugaredObject's from NodeStack
func FindTopLevelObjects(cache *cache.Cache, stack *nodestack.NodeStack, vm *jsonnet.VM) []*ast.DesugaredObject {
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
			indexValue, indexIsString := curr.Index.(*ast.LiteralString)
			if !indexIsString {
				continue
			}

			var container ast.Node
			// If our target is a var, the container for the index is the var ref
			if varTarget, targetIsVar := curr.Target.(*ast.Var); targetIsVar {
				ref, err := FindVarReference(varTarget, vm)
				if err != nil {
					log.WithError(err).Errorf("Error finding var reference, ignoring this node")
					continue
				}
				container = ref
			}

			// If we have not found a viable container, peek at the next object on the stack
			if container == nil {
				container = stack.Peek()
			}

			var possibleObjects []*ast.DesugaredObject
			if containerObj, containerIsObj := container.(*ast.DesugaredObject); containerIsObj {
				possibleObjects = []*ast.DesugaredObject{containerObj}
			} else if containerImport, containerIsImport := container.(*ast.Import); containerIsImport {
				possibleObjects = FindTopLevelObjectsInFile(cache, vm, containerImport.File.Value, string(containerImport.Loc().File.DiagnosticFileName))
			}

			for _, obj := range possibleObjects {
				for _, field := range findObjectFieldsInObject(obj, indexValue.Value, false) {
					stack.Push(field.Body)
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
