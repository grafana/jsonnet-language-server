package server

import (
	"context"
	"fmt"
	"reflect"
	"strings"

	"github.com/google/go-jsonnet/ast"
	processing "github.com/grafana/jsonnet-language-server/pkg/ast_processing"
	position "github.com/grafana/jsonnet-language-server/pkg/position_conversion"
	"github.com/grafana/jsonnet-language-server/pkg/utils"
	"github.com/jdbaldry/go-language-server-protocol/lsp/protocol"
)

func (s *server) DocumentSymbol(ctx context.Context, params *protocol.DocumentSymbolParams) ([]interface{}, error) {
	doc, err := s.cache.get(params.TextDocument.URI)
	if err != nil {
		return nil, utils.LogErrorf("Definition: %s: %w", errorRetrievingDocument, err)
	}

	if doc.ast == nil {
		return nil, utils.LogErrorf("Definition: error parsing the document")
	}

	symbols := buildDocumentSymbols(doc.ast)

	result := make([]interface{}, len(symbols))
	for i, symbol := range symbols {
		result[i] = symbol
	}

	return result, nil
}

func buildDocumentSymbols(node ast.Node) []protocol.DocumentSymbol {
	var symbols []protocol.DocumentSymbol

	switch node := node.(type) {
	case *ast.Binary:
		symbols = append(symbols, buildDocumentSymbols(node.Left)...)
		symbols = append(symbols, buildDocumentSymbols(node.Right)...)
	case *ast.Local:
		for _, bind := range node.Binds {
			locRange := bind.LocRange
			if !locRange.IsSet() {
				locRange = *bind.Body.Loc()
			}
			resultRange := position.RangeASTToProtocol(locRange)
			resultSelectionRange := position.NewProtocolRange(
				locRange.Begin.Line-1,
				locRange.Begin.Column-1,
				locRange.Begin.Line-1,
				locRange.Begin.Column-1+len(bind.Variable),
			)

			symbols = append(symbols, protocol.DocumentSymbol{
				Name:           string(bind.Variable),
				Kind:           protocol.Variable,
				Range:          resultRange,
				SelectionRange: resultSelectionRange,
				Detail:         symbolDetails(bind.Body),
			})
		}
		symbols = append(symbols, buildDocumentSymbols(node.Body)...)
	case *ast.DesugaredObject:
		for _, field := range node.Fields {
			kind := protocol.Field
			if field.Hide == ast.ObjectFieldHidden {
				kind = protocol.Property
			}
			fieldRange := processing.FieldToRange(&field)
			symbols = append(symbols, protocol.DocumentSymbol{
				Name:           field.Name.(*ast.LiteralString).Value,
				Kind:           kind,
				Range:          position.RangeASTToProtocol(fieldRange.FullRange),
				SelectionRange: position.RangeASTToProtocol(fieldRange.SelectionRange),
				Detail:         symbolDetails(field.Body),
				Children:       buildDocumentSymbols(field.Body),
			})
		}
	}

	return symbols
}

func symbolDetails(node ast.Node) string {

	switch node := node.(type) {
	case *ast.Function:
		var args []string
		for _, param := range node.Parameters {
			args = append(args, string(param.Name))
		}
		return fmt.Sprintf("Function(%s)", strings.Join(args, ", "))
	case *ast.DesugaredObject:
		return "Object"
	case *ast.LiteralString:
		return "String"
	case *ast.LiteralNumber:
		return "Number"
	case *ast.LiteralBoolean:
		return "Boolean"
	case *ast.Import:
		return "Import " + node.File.Value
	case *ast.ImportStr:
		return "Import " + node.File.Value
	case *ast.Index:
		return ""
	}

	return strings.TrimPrefix(reflect.TypeOf(node).String(), "*ast.")
}
