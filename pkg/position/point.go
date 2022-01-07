package position

import (
	"github.com/google/go-jsonnet/ast"
	"github.com/jdbaldry/go-language-server-protocol/lsp/protocol"
)

func PositionProtocolToAST(point protocol.Position) ast.Location {
	return ast.Location{
		Line:   int(point.Line) + 1,
		Column: int(point.Character) + 1,
	}
}
