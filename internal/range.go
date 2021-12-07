package internal

import (
	"github.com/google/go-jsonnet/ast"
	"github.com/jdbaldry/go-language-server-protocol/lsp/protocol"
)

// LocationRangeToProtocolRange translates a ast.LocationRange to a protocol.Range.
// The former is one indexed and the latter is zero indexed.
func LocationRangeToProtocolRange(lr ast.LocationRange) protocol.Range {
	return protocol.Range{
		Start: protocol.Position{Line: uint32(lr.Begin.Line - 1), Character: uint32(lr.Begin.Column - 1)},
		End:   protocol.Position{Line: uint32(lr.End.Line - 1), Character: uint32(lr.End.Column - 1)},
	}
}

func ProtocolRangeToLocationRange(pr protocol.Range) ast.LocationRange {
	return ast.LocationRange{
		Begin: ast.Location{Line: int(pr.Start.Line) + 1, Column: int(pr.Start.Character) + 1},
		End:   ast.Location{Line: int(pr.End.Line) + 1, Column: int(pr.End.Character) + 1},
	}
}
