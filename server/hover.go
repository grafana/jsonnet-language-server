package server

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/google/go-jsonnet/ast"
	"github.com/jdbaldry/go-language-server-protocol/lsp/protocol"
)

func (s *server) Hover(ctx context.Context, params *protocol.HoverParams) (*protocol.Hover, error) {
	doc, err := s.cache.get(params.TextDocument.URI)
	if err != nil {
		err = fmt.Errorf("Definition: %s: %w", errorRetrievingDocument, err)
		fmt.Fprintln(os.Stderr, err)
		return nil, err
	}

	stack, err := findNodeByPosition(doc.ast, params.Position)
	if err != nil {
		return nil, err
	}

	_, node := stack.Pop()

	// // DEBUG
	// r := protocol.Range{
	// 	Start: protocol.Position{
	// 		Line:      uint32(node.Loc().Begin.Line) - 1,
	// 		Character: uint32(node.Loc().Begin.Column) - 1,
	// 	},
	// 	End: protocol.Position{
	// 		Line:      uint32(node.Loc().End.Line) - 1,
	// 		Character: uint32(node.Loc().End.Column) - 1,
	// 	},
	// }
	// return &protocol.Hover{Range: r,
	// 	Contents: protocol.MarkupContent{Kind: protocol.PlainText,
	// 		Value: fmt.Sprintf("%v: %+v\n\n%v: %+v", reflect.TypeOf(node), node, reflect.TypeOf(node2), node2)},
	// }, nil

	_, isIndex := node.(*ast.Index)
	_, isVar := node.(*ast.Var)
	lineIndex := uint32(node.Loc().Begin.Line) - 1
	startIndex := uint32(node.Loc().Begin.Column) - 1
	line := strings.Split(doc.item.Text, "\n")[lineIndex]
	if isIndex || isVar && strings.HasPrefix(line[startIndex:], "std") {
		functionNameIndex := startIndex + 4
		if functionNameIndex < uint32(len(line)) {
			functionName := strings.Split(line[functionNameIndex:], "(")[0]
			functionName = strings.TrimSpace(functionName)

			for _, function := range s.stdlib {
				if function.Name == functionName {
					return &protocol.Hover{
						Range: protocol.Range{
							Start: protocol.Position{Line: lineIndex, Character: startIndex},
							End:   protocol.Position{Line: lineIndex, Character: functionNameIndex + uint32(len(functionName))}},
						Contents: protocol.MarkupContent{
							Kind:  protocol.Markdown,
							Value: fmt.Sprintf("`std.%s(%s)`\n\n%s", function.Name, strings.Join(function.Params, ", "), function.MarkdownDescription),
						},
					}, nil
				}
			}
		}
	}

	return nil, nil
}
