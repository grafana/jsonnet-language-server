package server

import (
	"context"
	"path/filepath"

	"github.com/google/go-jsonnet/ast"
	"github.com/grafana/jsonnet-language-server/pkg/ast/processing"
	position "github.com/grafana/jsonnet-language-server/pkg/position_conversion"
	"github.com/grafana/jsonnet-language-server/pkg/utils"
	"github.com/jdbaldry/go-language-server-protocol/lsp/protocol"
	log "github.com/sirupsen/logrus"

	tankaJsonnet "github.com/grafana/tanka/pkg/jsonnet"
	"github.com/grafana/tanka/pkg/jsonnet/jpath"
)

func (s *Server) References(ctx context.Context, params *protocol.ReferenceParams) ([]protocol.Location, error) {
	doc, err := s.cache.Get(params.TextDocument.URI)
	if err != nil {
		return nil, utils.LogErrorf("References: %s: %w", errorRetrievingDocument, err)
	}

	// Only find references if the line we're trying to find references for hasn't changed since last successful AST parse
	if doc.AST == nil {
		return nil, utils.LogErrorf("References: document was never successfully parsed, can't find references")
	}
	if doc.LinesChangedSinceAST[int(params.Position.Line)] {
		return nil, utils.LogErrorf("References: document line %d was changed since last successful parse, can't find references", params.Position.Line)
	}

	vm := s.getVM(doc.Item.URI.SpanURI().Filename())
	processor := processing.NewProcessor(s.cache, vm)

	searchStack, _ := processing.FindNodeByPosition(doc.AST, position.ProtocolToAST(params.Position))

	// Only match locals and obj fields, as we're trying to find usages of these
	possibleFiles := []string{}
	idOfSymbol := ""
	for !searchStack.IsEmpty() {
		deepestNode := searchStack.Pop()
		switch deepestNode := deepestNode.(type) {
		case *ast.Local:
			idOfSymbol = string(deepestNode.Binds[0].Variable)
			possibleFiles = []string{doc.Item.URI.SpanURI().Filename()} // Local variables are always used in the current file
		case *ast.DesugaredObject:
			// Find the field on the position
			for _, field := range deepestNode.Fields {
				if position.RangeASTToProtocol(field.LocRange).Start.Line == params.Position.Line {
					fieldName, ok := field.Name.(*ast.LiteralString)
					if !ok {
						return nil, utils.LogErrorf("References: field name is not a string")
					}
					idOfSymbol = string(fieldName.Value)
					root, err := jpath.FindRoot(doc.Item.URI.SpanURI().Filename())
					if err != nil {
						log.Errorf("References: Error resolving Tanka root, using current directory: %v", err)
						root = filepath.Dir(doc.Item.URI.SpanURI().Filename())
					}
					possibleFiles, err = tankaJsonnet.FindTransitiveImportersForFile(root, []string{doc.Item.URI.SpanURI().Filename()})
					if err != nil {
						log.Errorf("References: Error finding transitive importers. Using current file only: %v", err)
						possibleFiles = []string{doc.Item.URI.SpanURI().Filename()}
					}
					break
				}
			}
		}
		if idOfSymbol != "" {
			break
		}
	}

	// Find all usages of the symbol
	objectRanges, err := processor.FindUsages(possibleFiles, idOfSymbol)
	if err != nil {
		return nil, err
	}

	// Convert ObjectRanges to protocol.Locations
	var locations []protocol.Location
	for _, r := range objectRanges {
		locations = append(locations, protocol.Location{
			URI:   protocol.URIFromPath(r.Filename),
			Range: position.RangeASTToProtocol(r.SelectionRange),
		})
	}

	return locations, nil
}
