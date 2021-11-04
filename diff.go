package main

import (
	"github.com/hexops/gotextdiff/myers"
	"github.com/hexops/gotextdiff/span"
	"github.com/jdbaldry/go-language-server-protocol/lsp/protocol"
)

func getTextEdits(before, after string) []protocol.TextEdit {
	edits := myers.ComputeEdits(span.URI("any"), before, after)

	var result []protocol.TextEdit
	for _, edit := range edits {
		result = append(result, protocol.TextEdit{
			Range: protocol.Range{
				Start: protocol.Position{Line: uint32(edit.Span.Start().Line()) - 1, Character: uint32(edit.Span.Start().Column()) - 1},
				End:   protocol.Position{Line: uint32(edit.Span.End().Line()) - 1, Character: uint32(edit.Span.End().Column()) - 1},
			},
			NewText: edit.NewText,
		})
	}

	return result
}
