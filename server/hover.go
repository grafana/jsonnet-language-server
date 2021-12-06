package server

import (
	"context"
	"fmt"
	"os"

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

	// DEBUG
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
	// return &protocol.Hover{Range: r, Contents: protocol.MarkupContent{Kind: protocol.PlainText, Value: fmt.Sprintf("%v: %+v", reflect.TypeOf(node), node)}}, nil

	if vars := node.FreeVariables(); len(vars) > 0 && vars[0] == "std" {
		lineIndex := uint32(node.Loc().Begin.Line) - 1
		startIndex := uint32(node.Loc().Begin.Column) - 1
		// line := strings.Split(doc.item.Text, "\n")[lineIndex]

		r := protocol.Range{
			Start: protocol.Position{
				Line:      lineIndex,
				Character: startIndex,
			},
			End: protocol.Position{
				Line:      uint32(node.Loc().End.Line) - 1,
				Character: uint32(node.Loc().End.Column) - 1,
			},
		}
		return &protocol.Hover{Range: r, Contents: protocol.MarkupContent{Kind: protocol.PlainText, Value: fmt.Sprintf("test")}}, nil
	}

	return nil, nil
}
