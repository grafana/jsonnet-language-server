package server

import (
	"context"
	"strings"

	"github.com/google/go-jsonnet/ast"
	"github.com/grafana/jsonnet-language-server/pkg/ast/processing"
	"github.com/grafana/jsonnet-language-server/pkg/nodestack"
	position "github.com/grafana/jsonnet-language-server/pkg/position_conversion"
	"github.com/grafana/jsonnet-language-server/pkg/utils"
	"github.com/jdbaldry/go-language-server-protocol/lsp/protocol"
	log "github.com/sirupsen/logrus"
)

func (s *Server) Completion(ctx context.Context, params *protocol.CompletionParams) (*protocol.CompletionList, error) {
	doc, err := s.cache.get(params.TextDocument.URI)
	if err != nil {
		return nil, utils.LogErrorf("Completion: %s: %w", errorRetrievingDocument, err)
	}

	line := getCompletionLine(doc.item.Text, params.Position)

	// Short-circuit if it's a stdlib completion
	if items := s.completionStdLib(line); len(items) > 0 {
		return &protocol.CompletionList{IsIncomplete: false, Items: items}, nil
	}

	// Otherwise, parse the AST and search for completions
	if doc.ast == nil {
		log.Errorf("Completion: document was never successfully parsed, can't autocomplete")
		return nil, nil
	}

	searchStack, err := processing.FindNodeByPosition(doc.ast, position.ProtocolToAST(params.Position))
	if err != nil {
		log.Errorf("Completion: error computing node: %v", err)
		return nil, nil
	}

	items := s.completionFromStack(line, searchStack)
	return &protocol.CompletionList{IsIncomplete: false, Items: items}, nil
}

func getCompletionLine(fileContent string, position protocol.Position) string {
	line := strings.Split(fileContent, "\n")[position.Line]
	charIndex := int(position.Character)
	if charIndex > len(line) {
		charIndex = len(line)
	}
	line = line[:charIndex]
	return line
}

func (s *Server) completionFromStack(line string, stack *nodestack.NodeStack) []protocol.CompletionItem {
	var items []protocol.CompletionItem

	lineWords := strings.Split(line, " ")
	lastWord := lineWords[len(lineWords)-1]

	indexes := strings.Split(lastWord, ".")
	firstIndex, indexes := indexes[0], indexes[1:]

	if firstIndex == "self" && len(indexes) > 0 {
		fieldPrefix := indexes[0]

		for !stack.IsEmpty() {
			curr := stack.Pop()

			switch curr := curr.(type) {
			case *ast.Binary:
				stack.Push(curr.Left)
				stack.Push(curr.Right)
			case *ast.DesugaredObject:
				for _, field := range curr.Fields {
					label := processing.FieldNameToString(field.Name)
					// Ignore fields that don't match the prefix
					if !strings.HasPrefix(label, fieldPrefix) {
						continue
					}

					// Ignore the current field
					if strings.Contains(line, label+":") {
						continue
					}

					items = append(items, createCompletionItem(label, "self."+label, protocol.FieldCompletion, field.Body))
				}
			}
		}
	} else if len(indexes) == 0 {
		// firstIndex is a variable (local) completion
		for !stack.IsEmpty() {
			if curr, ok := stack.Pop().(*ast.Local); ok {
				for _, bind := range curr.Binds {
					label := string(bind.Variable)

					if !strings.HasPrefix(label, firstIndex) {
						continue
					}

					items = append(items, createCompletionItem(label, label, protocol.VariableCompletion, bind.Body))
				}
			}
		}
	}

	return items
}

func (s *Server) completionStdLib(line string) []protocol.CompletionItem {
	items := []protocol.CompletionItem{}

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
				InsertText:    strings.ReplaceAll(f.Signature(), "std.", ""),
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

	return items
}

func createCompletionItem(label, detail string, kind protocol.CompletionItemKind, body ast.Node) protocol.CompletionItem {
	insertText := label
	if asFunc, ok := body.(*ast.Function); ok {
		kind = protocol.FunctionCompletion
		params := []string{}
		for _, param := range asFunc.Parameters {
			params = append(params, string(param.Name))
		}
		paramsString := "(" + strings.Join(params, ", ") + ")"
		detail += paramsString
		insertText += paramsString
	}

	return protocol.CompletionItem{
		Label:      label,
		Detail:     detail,
		Kind:       kind,
		InsertText: insertText,
	}
}
