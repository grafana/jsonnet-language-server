// jsonnet-language-server: A Language Server Protocol server for Jsonnet.
// Copyright (C) 2021 Jack Baldry

// This program is free software: you can redistribute it and/or modify
// it under the terms of the GNU Affero General Public License as published
// by the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.

// This program is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
// GNU Affero General Public License for more details.

// You should have received a copy of the GNU Affero General Public License
// along with this program.  If not, see <https://www.gnu.org/licenses/>.

package server

import (
	"context"
	"regexp"

	"github.com/google/go-jsonnet"
	"github.com/google/go-jsonnet/ast"
	tankaJsonnet "github.com/grafana/tanka/pkg/jsonnet"
	"github.com/grafana/tanka/pkg/jsonnet/jpath"
	"github.com/jdbaldry/go-language-server-protocol/lsp/protocol"
	"github.com/jdbaldry/jsonnet-language-server/stdlib"
	"github.com/jdbaldry/jsonnet-language-server/utils"
	log "github.com/sirupsen/logrus"
)

const (
	errorRetrievingDocument = "unable to retrieve document from the cache"
)

var (
	// errRegexp matches the various Jsonnet location formats in errors.
	// file:line msg
	// file:line:col-endCol msg
	// file:(line:endLine)-(col:endCol) msg
	// Has 10 matching groups.
	errRegexp = regexp.MustCompile(`/.*:(?:(\d+)|(?:(\d+):(\d+)-(\d+))|(?:\((\d+):(\d+)\)-\((\d+):(\d+))\))\s(.*)`)
)

// New returns a new language server.
func NewServer(name, version string, client protocol.ClientCloser) *server {
	server := &server{
		name:    name,
		version: version,
		cache:   newCache(),
		client:  client,
	}

	return server
}

// server is the Jsonnet language server.
type server struct {
	name, version string

	stdlib []stdlib.Function
	cache  *cache
	client protocol.ClientCloser
	getVM  func(path string) (*jsonnet.VM, error)
}

func (s *server) WithStaticVM(jpaths []string) *server {
	log.Infof("Using the following jpaths: %v", jpaths)
	s.getVM = func(path string) (*jsonnet.VM, error) {
		vm := jsonnet.MakeVM()
		importer := &jsonnet.FileImporter{JPaths: jpaths}
		vm.Importer(importer)
		return vm, nil
	}
	return s
}

func (s *server) WithTankaVM(fallbackJPath []string) *server {
	log.Infof("Using tanka mode. Will fall back to the following jpaths: %v", fallbackJPath)
	s.getVM = func(path string) (*jsonnet.VM, error) {
		jpath, _, _, err := jpath.Resolve(path)
		if err != nil {
			log.Debugf("Unable to resolve jpath for %s: %s", path, err)
			jpath = fallbackJPath
		}
		opts := tankaJsonnet.Opts{
			ImportPaths: jpath,
		}
		return tankaJsonnet.MakeVM(opts), nil
	}
	return s
}

func (s *server) Definition(ctx context.Context, params *protocol.DefinitionParams) (protocol.Definition, error) {
	doc, err := s.cache.get(params.TextDocument.URI)
	if err != nil {
		return nil, utils.LogErrorf("Definition: %s: %w", errorRetrievingDocument, err)
	}

	if doc.ast == nil {
		return nil, utils.LogErrorf("Definition: error parsing the document")
	}

	stack, err := findNodeByPosition(doc.ast, params.Position)
	if err != nil {
		return nil, err
	}

	if stack.IsEmpty() {
		return nil, nil
	}

	var result protocol.Definition
	_, node := stack.Pop()
	switch node := node.(type) {
	case *ast.Import:
		vm, err := s.getVM(doc.item.URI.SpanURI().Filename())
		if err != nil {
			return nil, err
		}
		foundAt, err := vm.ResolveImport(doc.item.URI.SpanURI().Filename(), node.File.Value)
		if err != nil {
			return nil, utils.LogErrorf("Definition: unable to resolve import: %w", err)
		}
		result = protocol.Definition{
			{URI: "file://" + protocol.DocumentURI(foundAt)},
		}
	}

	return result, nil

}

func (s *server) DidChange(ctx context.Context, params *protocol.DidChangeTextDocumentParams) error {
	defer s.queueDiagnostics(params.TextDocument.URI)

	doc, err := s.cache.get(params.TextDocument.URI)
	if err != nil {
		return utils.LogErrorf("DidChange: %s: %w", errorRetrievingDocument, err)
	}

	if params.TextDocument.Version > doc.item.Version && len(params.ContentChanges) != 0 {
		doc.item.Text = params.ContentChanges[len(params.ContentChanges)-1].Text
		doc.ast, doc.err = jsonnet.SnippetToAST(doc.item.URI.SpanURI().Filename(), doc.item.Text)
		if doc.err != nil {
			return s.cache.put(doc)
		}
		doc.vm, err = s.getVM(doc.item.URI.SpanURI().Filename())
		if err != nil {
			return err
		}
		return s.cache.put(doc)
	}
	return nil
}

func (s *server) DidOpen(ctx context.Context, params *protocol.DidOpenTextDocumentParams) (err error) {
	defer s.queueDiagnostics(params.TextDocument.URI)

	doc := &document{item: params.TextDocument}
	if params.TextDocument.Text != "" {
		doc.ast, doc.err = jsonnet.SnippetToAST(params.TextDocument.URI.SpanURI().Filename(), params.TextDocument.Text)
		if doc.err != nil {
			return s.cache.put(doc)
		}
		doc.vm, err = s.getVM(params.TextDocument.URI.SpanURI().Filename())
		if err != nil {
			log.Infof("DidOpen: %v", err)
			return err
		}
	}
	return s.cache.put(doc)
}

func (s *server) Initialize(ctx context.Context, params *protocol.ParamInitialize) (*protocol.InitializeResult, error) {
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
