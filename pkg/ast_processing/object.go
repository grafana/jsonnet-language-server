package ast_processing

import (
	"github.com/google/go-jsonnet/ast"
)

// filterSelfScope takes in an array of objects (blocks delimited by curly braces) and
// returns a new array of objects, where only objects in scope of the first one are kept.

// This is done by comparing the location ranges. If the range of the first object is
// contained within the range of another object, the latter object is removed because
// it is a parent of the first object.
func filterSelfScope(objs []*ast.DesugaredObject) (result []*ast.DesugaredObject) {
	if len(objs) == 0 {
		return objs
	}

	// Copy the array so we don't modify the original
	result = objs[:]

	topLevel := result[0]
	i := 1
	for i < len(result) {
		obj := result[i]
		// If the current object is contained within the top level object, remove it
		if RangeGreaterOrEqual(obj.LocRange, topLevel.LocRange) {
			result = append(result[:i], result[i+1:]...)
			continue
		}
		i++
	}
	return
}
