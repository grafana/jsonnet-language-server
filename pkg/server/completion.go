package server

import (
	"context"
	"strings"

	"github.com/grafana/jsonnet-language-server/pkg/utils"
	"github.com/jdbaldry/go-language-server-protocol/lsp/protocol"
)

func (s *server) Completion(ctx context.Context, params *protocol.CompletionParams) (*protocol.CompletionList, error) {
	doc, err := s.cache.get(params.TextDocument.URI)
	if err != nil {
		return nil, utils.LogErrorf("Completion: %s: %w", errorRetrievingDocument, err)
	}

	items := []protocol.CompletionItem{}

	line := strings.Split(doc.item.Text, "\n")[params.Position.Line]
	charIndex := int(params.Position.Character)
	if charIndex > len(line) {
		charIndex = len(line)
	}
	line = line[:charIndex]
	stdIndex := strings.LastIndex(line, "std.")
	if stdIndex != -1 {
		userInput := line[stdIndex+4:]
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
				Detail:        f.Signature(),
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
