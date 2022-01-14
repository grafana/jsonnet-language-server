package position

import (
	"github.com/google/go-jsonnet/ast"
	"github.com/jdbaldry/go-language-server-protocol/lsp/protocol"
)

func NewProtocolRange(startLine, startCharacter, endLine, endCharacter int) protocol.Range {
	return protocol.Range{
		Start: protocol.Position{
			Character: uint32(startCharacter),
			Line:      uint32(startLine),
		},
		End: protocol.Position{
			Character: uint32(endCharacter),
			Line:      uint32(endLine),
		},
	}
}

// RangeASTToProtocol translates a ast.LocationRange to a protocol.Range.
// The former is one indexed and the latter is zero indexed.
func RangeASTToProtocol(lr ast.LocationRange) protocol.Range {
	return protocol.Range{
		Start: protocol.Position{
			Line:      uint32(lr.Begin.Line - 1),
			Character: uint32(lr.Begin.Column - 1),
		},
		End: protocol.Position{
			Line:      uint32(lr.End.Line - 1),
			Character: uint32(lr.End.Column - 1),
		},
	}
}

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

// RangeGreaterThan returns true if the first range is greater than the second.
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
