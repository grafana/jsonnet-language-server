package processing

import "github.com/google/go-jsonnet/ast"

func InRange(point ast.Location, theRange ast.LocationRange) bool {
	if point.Line == theRange.Begin.Line && point.Column < theRange.Begin.Column {
		return false
	}

	if point.Line == theRange.End.Line && point.Column >= theRange.End.Column {
		return false
	}

	if point.Line != theRange.Begin.Line || point.Line != theRange.End.Line {
		return theRange.Begin.Line <= point.Line && point.Line <= theRange.End.Line
	}

	return true
}

// RangeGreaterOrEqual returns true if the first range is greater than the second.
func RangeGreaterOrEqual(a ast.LocationRange, b ast.LocationRange) bool {
	if a.Begin.Line > b.Begin.Line {
		return false
	}
	if a.End.Line < b.End.Line {
		return false
	}
	if a.Begin.Line == b.Begin.Line && a.Begin.Column > b.Begin.Column {
		return false
	}
	if a.End.Line == b.End.Line && a.End.Column < b.End.Column {
		return false
	}

	return true
}
