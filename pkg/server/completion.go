package server

import (
	"context"
	"strings"

	"github.com/google/go-jsonnet"
	"github.com/google/go-jsonnet/ast"
	"github.com/grafana/jsonnet-language-server/pkg/utils"
	"github.com/jdbaldry/go-language-server-protocol/lsp/protocol"
)

func (s *server) Completion(ctx context.Context, params *protocol.CompletionParams) (*protocol.CompletionList, error) {
	doc, err := s.cache.get(params.TextDocument.URI)
	if err != nil {
		return nil, utils.LogErrorf("Completion: %s: %w", errorRetrievingDocument, err)
	}

	items := []protocol.CompletionItem{}

	lines := strings.Split(doc.item.Text, "\n")
	line := lines[params.Position.Line]
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

	selfIndex := strings.LastIndex(line, "self.")
	if selfIndex != -1 {
		// userInput := line[selfIndex+5:]
		lines[params.Position.Line] = ""

		docAST, err := jsonnet.SnippetToAST(doc.item.URI.SpanURI().Filename(), strings.Join(lines, "\n"))
		if err != nil {
			return nil, utils.LogErrorf("Completion, error parsing document: %v", err)
		}

		selfPos := params.Position
		selfPos.Character = 0

		stack, err := findNodeByPosition(docAST, selfPos)
		if err != nil {
			return nil, err
		}

		vm, err := s.getVM(doc.item.URI.SpanURI().Filename())
		if err != nil {
			return nil, err
		}

		fields, err := findObjectFields(stack, "self", vm)
		if err != nil {
			return nil, err
		}

		for fieldName, f := range fields {
			body, isString := f.Body.(*ast.LiteralString)
			fieldDoc := ""
			if isString {
				fieldDoc = "Value: " + body.Value
			}

			items = append(items, protocol.CompletionItem{
				Label:         fieldName,
				Kind:          protocol.FieldCompletion,
				Detail:        "self." + fieldName,
				Documentation: fieldDoc,
			})
		}
	}

	return &protocol.CompletionList{IsIncomplete: false, Items: items}, nil
}
