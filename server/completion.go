package server

import (
	"context"
	"fmt"
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

	items := []protocol.CompletionItem{}

	line := strings.Split(doc.item.Text, "\n")[params.Position.Line]
	stdIndex := strings.LastIndex(line[:params.Position.Character], "std.")
	if stdIndex != -1 {
		userInput := strings.Split(line[stdIndex+4:], "(")[0]
		funcStartWith := []protocol.CompletionItem{}
		funcContains := []protocol.CompletionItem{}
		for _, f := range s.stdlib {
			if f.Name == userInput {
				break
			}
			lowerFuncName := strings.ToLower(f.Name)
			findName := strings.ToLower(userInput)
			item := protocol.CompletionItem{
				Label:         f.Name,
				Kind:          protocol.FunctionCompletion,
				Detail:        fmt.Sprintf("%s(%s)", f.Name, strings.Join(f.Params, ", ")),
				Documentation: f.MarkdownDescription,
			}

			if len(findName) > 0 && strings.HasPrefix(lowerFuncName, findName) {
				funcStartWith = append(funcStartWith, item)
				continue
			}

			if strings.Contains(lowerFuncName, findName) {
				funcContains = append(funcContains, item)
			}
		}

		items = append(items, funcStartWith...)
		items = append(items, funcContains...)
	}

	return &protocol.CompletionList{IsIncomplete: false, Items: items}, nil
}
