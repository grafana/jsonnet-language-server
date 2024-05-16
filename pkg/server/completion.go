package server

import (
	"context"
	"reflect"
	"sort"
	"strings"

	"github.com/google/go-jsonnet"
	"github.com/google/go-jsonnet/ast"
	"github.com/grafana/jsonnet-language-server/pkg/ast/processing"
	"github.com/grafana/jsonnet-language-server/pkg/nodestack"
	position "github.com/grafana/jsonnet-language-server/pkg/position_conversion"
	"github.com/grafana/jsonnet-language-server/pkg/utils"
	"github.com/jdbaldry/go-language-server-protocol/lsp/protocol"
	log "github.com/sirupsen/logrus"
)

func (s *Server) Completion(_ context.Context, params *protocol.CompletionParams) (*protocol.CompletionList, error) {
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

	vm := s.getVM(doc.item.URI.SpanURI().Filename())

	items := s.completionFromStack(line, searchStack, vm, params.Position)
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

func (s *Server) completionFromStack(line string, stack *nodestack.NodeStack, vm *jsonnet.VM, position protocol.Position) []protocol.CompletionItem {
	lineWords := strings.Split(line, " ")
	lastWord := lineWords[len(lineWords)-1]
	lastWord = strings.TrimRight(lastWord, ",;") // Ignore trailing commas and semicolons, they can present when someone is modifying an existing line

	indexes := strings.Split(lastWord, ".")

	if len(indexes) == 1 {
		var items []protocol.CompletionItem
		// firstIndex is a variable (local) completion
		for !stack.IsEmpty() {
			if curr, ok := stack.Pop().(*ast.Local); ok {
				for _, bind := range curr.Binds {
					label := string(bind.Variable)

					if !strings.HasPrefix(label, indexes[0]) {
						continue
					}

					items = append(items, createCompletionItem(label, "", protocol.VariableCompletion, bind.Body, position))
				}
			}
		}
		return items
	}

	ranges, err := processing.FindRangesFromIndexList(stack, indexes, vm, true)
	if err != nil {
		log.Errorf("Completion: error finding ranges: %v", err)
		return nil
	}

	completionPrefix := strings.Join(indexes[:len(indexes)-1], ".")
	return s.createCompletionItemsFromRanges(ranges, completionPrefix, line, position)
}

func (s *Server) completionStdLib(line string) []protocol.CompletionItem {
	var items []protocol.CompletionItem

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

func (s *Server) createCompletionItemsFromRanges(ranges []processing.ObjectRange, completionPrefix, currentLine string, position protocol.Position) []protocol.CompletionItem {
	var items []protocol.CompletionItem
	labels := make(map[string]bool)

	for _, field := range ranges {
		label := field.FieldName

		if field.Node == nil {
			continue
		}

		if labels[label] {
			continue
		}

		if !s.configuration.ShowDocstringInCompletion && strings.HasPrefix(label, "#") {
			continue
		}

		// Ignore the current field
		if strings.Contains(currentLine, label+":") && completionPrefix == "self" {
			continue
		}

		items = append(items, createCompletionItem(label, completionPrefix, protocol.FieldCompletion, field.Node, position))
		labels[label] = true
	}

	sort.Slice(items, func(i, j int) bool {
		return items[i].Label < items[j].Label
	})

	return items
}

func createCompletionItem(label, prefix string, kind protocol.CompletionItemKind, body ast.Node, position protocol.Position) protocol.CompletionItem {
	mustNotQuoteLabel := IsValidIdentifier(label)

	insertText := label
	detail := label
	if prefix != "" {
		detail = prefix + "." + insertText
	}
	if !mustNotQuoteLabel {
		insertText = "['" + label + "']"
		detail = prefix + insertText
	}

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

	item := protocol.CompletionItem{
		Label:  label,
		Detail: detail,
		Kind:   kind,
		LabelDetails: protocol.CompletionItemLabelDetails{
			Description: typeToString(body),
		},
		InsertText: insertText,
	}

	// Remove leading `.` character when quoting label
	if !mustNotQuoteLabel {
		item.TextEdit = &protocol.TextEdit{
			Range: protocol.Range{
				Start: protocol.Position{
					Line:      position.Line,
					Character: position.Character - 1,
				},
				End: protocol.Position{
					Line:      position.Line,
					Character: position.Character,
				},
			},
			NewText: insertText,
		}
	}

	return item
}

// Start - Copied from go-jsonnet/internal/parser/lexer.go

func isUpper(r rune) bool {
	return r >= 'A' && r <= 'Z'
}
func isLower(r rune) bool {
	return r >= 'a' && r <= 'z'
}
func isNumber(r rune) bool {
	return r >= '0' && r <= '9'
}
func isIdentifierFirst(r rune) bool {
	return isUpper(r) || isLower(r) || r == '_'
}
func isIdentifier(r rune) bool {
	return isIdentifierFirst(r) || isNumber(r)
}
func IsValidIdentifier(str string) bool {
	if len(str) == 0 {
		return false
	}
	for i, r := range str {
		if i == 0 {
			if !isIdentifierFirst(r) {
				return false
			}
		} else {
			if !isIdentifier(r) {
				return false
			}
		}
	}
	// Ignore tokens for now, we should ask upstream to make the formatter a public package
	// so we can use go-jsonnet/internal/formatter/pretty_field_names.go directly.
	// return getTokenKindFromID(str) == tokenIdentifier
	return true
}

// End - Copied from go-jsonnet/internal/parser/lexer.go

func typeToString(t ast.Node) string {
	switch t.(type) {
	case *ast.Array:
		return "array"
	case *ast.LiteralBoolean:
		return "boolean"
	case *ast.Function:
		return "function"
	case *ast.LiteralNull:
		return "null"
	case *ast.LiteralNumber:
		return "number"
	case *ast.Object, *ast.DesugaredObject:
		return "object"
	case *ast.LiteralString:
		return "string"
	case *ast.Import, *ast.ImportStr:
		return "import"
	case *ast.Index:
		return "object field"
	}
	return reflect.TypeOf(t).String()
}
