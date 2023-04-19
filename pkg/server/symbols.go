package server

import (
	"context"
	"fmt"
	"reflect"
	"strings"

	"github.com/google/go-jsonnet/ast"
	"github.com/grafana/jsonnet-language-server/pkg/ast/processing"
	position "github.com/grafana/jsonnet-language-server/pkg/position_conversion"
	"github.com/grafana/jsonnet-language-server/pkg/utils"
	"github.com/jdbaldry/go-language-server-protocol/lsp/protocol"
	log "github.com/sirupsen/logrus"
)

func (s *Server) DocumentSymbol(_ context.Context, params *protocol.DocumentSymbolParams) ([]interface{}, error) {
	doc, err := s.cache.get(params.TextDocument.URI)
	if err != nil {
		return nil, utils.LogErrorf("DocumentSymbol: %s: %w", errorRetrievingDocument, err)
	}

	if doc.err != nil {
		// Returning an error too often can lead to the client killing the language server
		// Logging the errors is sufficient
		log.Errorf("DocumentSymbol: %s", errorParsingDocument)
		return nil, nil
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
			objectRange := processing.LocalBindToRange(bind)
			symbols = append(symbols, protocol.DocumentSymbol{
				Name:           string(bind.Variable),
				Kind:           protocol.Variable,
				Range:          position.RangeASTToProtocol(objectRange.FullRange),
				SelectionRange: position.RangeASTToProtocol(objectRange.SelectionRange),
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
			fieldRange := processing.FieldToRange(field)
			symbols = append(symbols, protocol.DocumentSymbol{
				Name:           processing.FieldNameToString(field.Name),
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
