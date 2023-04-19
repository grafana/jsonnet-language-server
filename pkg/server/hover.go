package server

import (
	"context"
	"fmt"
	"strings"

	"github.com/google/go-jsonnet/ast"
	"github.com/grafana/jsonnet-language-server/pkg/ast/processing"
	position "github.com/grafana/jsonnet-language-server/pkg/position_conversion"
	"github.com/grafana/jsonnet-language-server/pkg/utils"
	"github.com/jdbaldry/go-language-server-protocol/lsp/protocol"
	log "github.com/sirupsen/logrus"
)

func (s *Server) Hover(_ context.Context, params *protocol.HoverParams) (*protocol.Hover, error) {
	doc, err := s.cache.get(params.TextDocument.URI)
	if err != nil {
		return nil, utils.LogErrorf("Hover: %s: %w", errorRetrievingDocument, err)
	}

	if doc.err != nil {
		// Hover triggers often. Throwing an error on each request is noisy
		log.Errorf("Hover: %s", errorParsingDocument)
		return nil, nil
	}

	stack, err := processing.FindNodeByPosition(doc.ast, position.ProtocolToAST(params.Position))
	if err != nil {
		return nil, err
	}

	if stack.IsEmpty() {
		log.Debug("Hover: empty stack")
		return nil, nil
	}

	node := stack.Pop()

	// // DEBUG
	// var node2 ast.Node
	// if !stack.IsEmpty() {
	// 	_, node2 = stack.Pop()
	// }
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
	if (isIndex || isVar) && strings.HasPrefix(line[startIndex:], "std") {
		functionNameIndex := startIndex + 4
		if functionNameIndex < uint32(len(line)) {
			functionName := utils.FirstWord(line[functionNameIndex:])
			functionName = strings.TrimSpace(functionName)

			for _, function := range s.stdlib {
				if function.Name == functionName {
					return &protocol.Hover{
						Range: protocol.Range{
							Start: protocol.Position{Line: lineIndex, Character: startIndex},
							End:   protocol.Position{Line: lineIndex, Character: functionNameIndex + uint32(len(functionName))}},
						Contents: protocol.MarkupContent{
							Kind:  protocol.Markdown,
							Value: fmt.Sprintf("`%s`\n\n%s", function.Signature(), function.MarkdownDescription),
						},
					}, nil
				}
			}
		}
	}

	return nil, nil
}
