package server

import (
	"context"

	"github.com/google/go-jsonnet/formatter"
	"github.com/grafana/jsonnet-language-server/pkg/utils"
	"github.com/hexops/gotextdiff/myers"
	"github.com/hexops/gotextdiff/span"
	"github.com/jdbaldry/go-language-server-protocol/lsp/protocol"
	log "github.com/sirupsen/logrus"
)

func (s *Server) Formatting(ctx context.Context, params *protocol.DocumentFormattingParams) ([]protocol.TextEdit, error) {
	doc, err := s.cache.get(params.TextDocument.URI)
	if err != nil {
		return nil, utils.LogErrorf("Formatting: %s: %w", errorRetrievingDocument, err)
	}

	formatted, err := formatter.Format(params.TextDocument.URI.SpanURI().Filename(), doc.item.Text, s.configuration.FormattingOptions)
	if err != nil {
		log.Errorf("error formatting document: %v", err)
		return nil, nil
	}

	return getTextEdits(doc.item.Text, formatted), nil
}

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
