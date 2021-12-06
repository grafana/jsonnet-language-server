package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"strings"

	"github.com/jdbaldry/go-language-server-protocol/lsp/protocol"
)

// Completion is not implemented.
// TODO(#6): Understand why the server capabilities includes completion.
func (s *server) Completion(ctx context.Context, params *protocol.CompletionParams) (*protocol.CompletionList, error) {
	doc, err := s.cache.get(params.TextDocument.URI)
	if err != nil {
		err = fmt.Errorf("Definition: %s: %w", errorRetrievingDocument, err)
		fmt.Fprintln(os.Stderr, err)
		return nil, err
	}
	log.Printf("%+v\nLine: %s", params, strings.Split(doc.item.Text, "\n")[params.Position.Line])

	return &protocol.CompletionList{IsIncomplete: false, Items: nil}, nil
}
