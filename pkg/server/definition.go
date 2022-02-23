package server

import (
	"context"
	"fmt"
	"path/filepath"
	"strings"

	"github.com/google/go-jsonnet"
	"github.com/google/go-jsonnet/ast"
	"github.com/grafana/jsonnet-language-server/pkg/nodestack"
	"github.com/grafana/jsonnet-language-server/pkg/position"
	"github.com/grafana/jsonnet-language-server/pkg/processing"
	"github.com/grafana/jsonnet-language-server/pkg/utils"
	"github.com/jdbaldry/go-language-server-protocol/lsp/protocol"
	log "github.com/sirupsen/logrus"
)

func (s *server) Definition(ctx context.Context, params *protocol.DefinitionParams) (protocol.Definition, error) {
	responseDefLinks, err := s.definitionLink(ctx, params)
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

func (s *server) definitionLink(ctx context.Context, params *protocol.DefinitionParams) ([]protocol.DefinitionLink, error) {
	doc, err := s.cache.get(params.TextDocument.URI)
	if err != nil {
		return nil, utils.LogErrorf("Definition: %s: %w", errorRetrievingDocument, err)
	}

	if doc.ast == nil {
		return nil, utils.LogErrorf("Definition: error parsing the document")
	}

	vm, err := s.getVM(doc.item.URI.SpanURI().Filename())
	if err != nil {
		return nil, utils.LogErrorf("error creating the VM: %w", err)
	}
	responseDefLinks, err := findDefinition(doc.ast, params, vm)
	if err != nil {
		return nil, err
	}

	return responseDefLinks, nil
}

func findDefinition(root ast.Node, params *protocol.DefinitionParams, vm *jsonnet.VM) ([]protocol.DefinitionLink, error) {
	var response []protocol.DefinitionLink

	searchStack, _ := processing.FindNodeByPosition(root, position.PositionProtocolToAST(params.Position))
	var deepestNode ast.Node
	searchStack, deepestNode = searchStack.Pop()
	switch deepestNode := deepestNode.(type) {
	case *ast.Var:
		log.Debugf("Found Var node %s", deepestNode.Id)

		var (
			filename                          string
			resultRange, resultSelectionRange protocol.Range
		)

		if bind := processing.FindBindByIdViaStack(searchStack, deepestNode.Id); bind != nil {
			locRange := bind.LocRange
			if !locRange.Begin.IsSet() {
				locRange = *bind.Body.Loc()
			}
			filename = locRange.FileName
			resultRange = position.RangeASTToProtocol(locRange)
			resultSelectionRange = position.NewProtocolRange(
				locRange.Begin.Line-1,
				locRange.Begin.Column-1,
				locRange.Begin.Line-1,
				locRange.Begin.Column-1+len(bind.Variable),
			)
		} else if param := processing.FindParameterByIdViaStack(searchStack, deepestNode.Id); param != nil {
			filename = param.LocRange.FileName
			resultRange = position.RangeASTToProtocol(param.LocRange)
			resultSelectionRange = position.RangeASTToProtocol(param.LocRange)
		} else {
			return nil, fmt.Errorf("no matching bind found for %s", deepestNode.Id)
		}

		response = append(response, protocol.DefinitionLink{
			TargetURI:            protocol.DocumentURI(filename),
			TargetRange:          resultRange,
			TargetSelectionRange: resultSelectionRange,
		})
	case *ast.SuperIndex, *ast.Index:
		indexSearchStack := nodestack.NewNodeStack(deepestNode)
		indexList := indexSearchStack.BuildIndexList()
		tempSearchStack := *searchStack
		objectRanges, err := processing.FindRangesFromIndexList(&tempSearchStack, indexList, vm)
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
