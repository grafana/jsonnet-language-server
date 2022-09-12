package server

import (
	"context"
	"path/filepath"
	"strings"

	"github.com/google/go-jsonnet"
	"github.com/google/go-jsonnet/ast"
	"github.com/grafana/jsonnet-language-server/pkg/stdlib"
	"github.com/grafana/jsonnet-language-server/pkg/utils"
	tankaJsonnet "github.com/grafana/tanka/pkg/jsonnet"
	"github.com/grafana/tanka/pkg/jsonnet/jpath"
	"github.com/jdbaldry/go-language-server-protocol/lsp/protocol"
	log "github.com/sirupsen/logrus"
)

const (
	errorRetrievingDocument = "unable to retrieve document from the cache"
	errorParsingDocument    = "error parsing the document"
)

// New returns a new language server.
func NewServer(name, version string, client protocol.ClientCloser, configuration Configuration) *Server {
	server := &Server{
		name:          name,
		version:       version,
		cache:         newCache(),
		client:        client,
		configuration: configuration,
	}

	return server
}

// server is the Jsonnet language server.
type Server struct {
	name, version string

	stdlib []stdlib.Function
	cache  *cache
	client protocol.ClientCloser

	configuration Configuration
}

func (s *Server) getVM(path string) *jsonnet.VM {
	var vm *jsonnet.VM
	if s.configuration.ResolvePathsWithTanka {
		jpath, _, _, err := jpath.Resolve(path, false)
		if err != nil {
			log.Debugf("Unable to resolve jpath for %s: %s", path, err)
			// nolint: gocritic
			jpath = append(s.configuration.JPaths, filepath.Dir(path))
		}
		opts := tankaJsonnet.Opts{
			ImportPaths: jpath,
		}
		vm = tankaJsonnet.MakeVM(opts)
	} else {
		// nolint: gocritic
		jpath := append(s.configuration.JPaths, filepath.Dir(path))
		vm = jsonnet.MakeVM()
		importer := &jsonnet.FileImporter{JPaths: jpath}
		vm.Importer(importer)
	}

	resetExtVars(vm, s.configuration.ExtVars)
	return vm
}

func (s *Server) DidChange(ctx context.Context, params *protocol.DidChangeTextDocumentParams) error {
	defer s.queueDiagnostics(params.TextDocument.URI)

	doc, err := s.cache.get(params.TextDocument.URI)
	if err != nil {
		return utils.LogErrorf("DidChange: %s: %w", errorRetrievingDocument, err)
	}

	if params.TextDocument.Version > doc.item.Version && len(params.ContentChanges) != 0 {
		oldText := doc.item.Text
		doc.item.Text = params.ContentChanges[len(params.ContentChanges)-1].Text

		var ast ast.Node
		ast, doc.err = jsonnet.SnippetToAST(doc.item.URI.SpanURI().Filename(), doc.item.Text)

		// If the AST parsed correctly, set it on the document
		// Otherwise, keep the old AST, and find all the lines that have changed since last AST
		if ast != nil {
			doc.ast = ast
			doc.linesChangedSinceAST = map[int]bool{}
		} else {
			splitOldText := strings.Split(oldText, "\n")
			splitNewText := strings.Split(doc.item.Text, "\n")
			for index, oldLine := range splitOldText {
				if index >= len(splitNewText) || oldLine != splitNewText[index] {
					doc.linesChangedSinceAST[index] = true
				}
			}
		}
	}
	return nil
}

func (s *Server) DidOpen(ctx context.Context, params *protocol.DidOpenTextDocumentParams) (err error) {
	defer s.queueDiagnostics(params.TextDocument.URI)

	doc := &document{item: params.TextDocument, linesChangedSinceAST: map[int]bool{}}
	if params.TextDocument.Text != "" {
		doc.ast, doc.err = jsonnet.SnippetToAST(params.TextDocument.URI.SpanURI().Filename(), params.TextDocument.Text)
	}
	return s.cache.put(doc)
}

func (s *Server) Initialize(ctx context.Context, params *protocol.ParamInitialize) (*protocol.InitializeResult, error) {
	log.Infof("Initializing %s version %s", s.name, s.version)

	s.diagnosticsLoop()

	var err error

	if s.stdlib == nil {
		log.Infoln("Reading stdlib")
		if s.stdlib, err = stdlib.Functions(); err != nil {
			return nil, err
		}
	}

	return &protocol.InitializeResult{
		Capabilities: protocol.ServerCapabilities{
			CompletionProvider:         protocol.CompletionOptions{TriggerCharacters: []string{"."}},
			HoverProvider:              true,
			DefinitionProvider:         true,
			DocumentFormattingProvider: true,
			DocumentSymbolProvider:     true,
			ExecuteCommandProvider:     protocol.ExecuteCommandOptions{Commands: []string{}},
			TextDocumentSync: &protocol.TextDocumentSyncOptions{
				Change:    protocol.Full,
				OpenClose: true,
				Save: protocol.SaveOptions{
					IncludeText: false,
				},
			},
		},
		ServerInfo: struct {
			Name    string `json:"name"`
			Version string `json:"version,omitempty"`
		}{
			Name:    s.name,
			Version: s.version,
		},
	}, nil
}
