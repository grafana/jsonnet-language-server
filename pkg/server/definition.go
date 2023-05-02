package server

import (
	"context"
	"fmt"
	"path/filepath"
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

func (s *Server) Definition(_ context.Context, params *protocol.DefinitionParams) (protocol.Definition, error) {
	responseDefLinks, err := s.definitionLink(params)
	if err != nil {
		// Returning an error too often can lead to the client killing the language server
		// Logging the errors is sufficient
		log.WithError(err).Error("Definition: error finding definition")
		return nil, nil
	}

	// TODO: Support LocationLink instead of Location (this needs to be changed in the upstream protocol lib)
	// When that's done, we can get rid of the intermediary `definitionLink` function which is used for testing
	var response protocol.Definition
	for _, item := range responseDefLinks {
		response = append(response, protocol.Location{
			URI:   item.TargetURI,
			Range: item.TargetRange,
		})
	}

	return response, nil
}

func (s *Server) definitionLink(params *protocol.DefinitionParams) ([]protocol.DefinitionLink, error) {
	doc, err := s.cache.get(params.TextDocument.URI)
	if err != nil {
		return nil, utils.LogErrorf("Definition: %s: %w", errorRetrievingDocument, err)
	}

	// Only find definitions, if the the line we're trying to find a definition for hasn't changed since last successful AST parse
	if doc.ast == nil {
		return nil, utils.LogErrorf("Definition: document was never successfully parsed, can't find definitions")
	}
	if doc.linesChangedSinceAST[int(params.Position.Line)] {
		return nil, utils.LogErrorf("Definition: document line %d was changed since last successful parse, can't find definitions", params.Position.Line)
	}

	vm := s.getVM(doc.item.URI.SpanURI().Filename())
	responseDefLinks, err := findDefinition(doc.ast, params, vm)
	if err != nil {
		return nil, err
	}

	return responseDefLinks, nil
}

func findDefinition(root ast.Node, params *protocol.DefinitionParams, vm *jsonnet.VM) ([]protocol.DefinitionLink, error) {
	var response []protocol.DefinitionLink

	searchStack, _ := processing.FindNodeByPosition(root, position.ProtocolToAST(params.Position))
	deepestNode := searchStack.Pop()
	switch deepestNode := deepestNode.(type) {
	case *ast.Var:
		log.Debugf("Found Var node %s", deepestNode.Id)

		var objectRange processing.ObjectRange

		if bind := processing.FindBindByIDViaStack(searchStack, deepestNode.Id); bind != nil {
			objectRange = processing.LocalBindToRange(*bind)
		} else if param := processing.FindParameterByIDViaStack(searchStack, deepestNode.Id, false); param != nil {
			objectRange = processing.ObjectRange{
				Filename:       param.LocRange.FileName,
				FullRange:      param.LocRange,
				SelectionRange: param.LocRange,
			}
		} else {
			return nil, fmt.Errorf("no matching bind found for %s", deepestNode.Id)
		}

		response = append(response, protocol.DefinitionLink{
			TargetURI:            protocol.DocumentURI(objectRange.Filename),
			TargetRange:          position.RangeASTToProtocol(objectRange.FullRange),
			TargetSelectionRange: position.RangeASTToProtocol(objectRange.SelectionRange),
		})
	case *ast.SuperIndex, *ast.Index:
		indexSearchStack := nodestack.NewNodeStack(deepestNode)
		indexList := indexSearchStack.BuildIndexList()
		tempSearchStack := *searchStack
		objectRanges, err := processing.FindRangesFromIndexList(&tempSearchStack, indexList, vm, false)
		if err != nil {
			return nil, err
		}
		for _, o := range objectRanges {
			response = append(response, protocol.DefinitionLink{
				TargetURI:            protocol.DocumentURI(o.Filename),
				TargetRange:          position.RangeASTToProtocol(o.FullRange),
				TargetSelectionRange: position.RangeASTToProtocol(o.SelectionRange),
			})
		}
	case *ast.Import:
		filename := deepestNode.File.Value
		importedFile, _ := vm.ResolveImport(string(params.TextDocument.URI), filename)
		response = append(response, protocol.DefinitionLink{
			TargetURI: protocol.DocumentURI(importedFile),
		})
	default:
		log.Debugf("cannot find definition for node type %T", deepestNode)
		return nil, fmt.Errorf("cannot find definition")
	}

	for i, item := range response {
		link := string(item.TargetURI)
		if !strings.HasPrefix(link, "file://") {
			targetFile, err := filepath.Abs(link)
			if err != nil {
				return nil, err
			}
			response[i].TargetURI = protocol.URIFromPath(targetFile)
		}
	}

	return response, nil
}
