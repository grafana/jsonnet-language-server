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
	"fmt"
	"os"
	"regexp"
	"strconv"
	"strings"

	"github.com/google/go-jsonnet"
	tankaJsonnet "github.com/grafana/tanka/pkg/jsonnet"
	"github.com/grafana/tanka/pkg/jsonnet/jpath"
	"github.com/jdbaldry/go-language-server-protocol/lsp/protocol"
	"github.com/jdbaldry/jsonnet-language-server/stdlib"
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

// Definition returns the location of the definition of the symbol at point.
// Looking up the symbol at point depends on only looking up nodes that have no children and are therefore terminal.
// Potentially it could backtrack outwards to guess what was probably meant.
// In the case of an index "obj.field", lookup would work when the point is on either "obj" or "field" but not the "."
// as it is not a terminal node and is instead part of the "index" symbol which has the children "obj" and "field".
// This works well in all cases where the Jsonnet parser has put correct location information. Unfortunately,
// the literal string node "field" of "obj.field" does not have correct location information.
// TODO(#8): Understand why the parser has not attached correct location range for indexes.
func (s *server) Definition(ctx context.Context, params *protocol.DefinitionParams) (protocol.Definition, error) {
	doc, err := s.cache.get(params.TextDocument.URI)
	if err != nil {
		err = fmt.Errorf("Definition: %s: %w", errorRetrievingDocument, err)
		fmt.Fprintln(os.Stderr, err)
		return nil, err
	}

	var aux func([]protocol.DocumentSymbol, protocol.DocumentSymbol) (protocol.Definition, error)
	aux = func(stack []protocol.DocumentSymbol, ds protocol.DocumentSymbol) (protocol.Definition, error) {
		// If the symbol has no children, it must be a terminal symbol.
		if len(ds.Children) == 0 {
			// Check if the point is contained within the symbol.
			if params.Position.Line == ds.Range.Start.Line &&
				params.Position.Character >= ds.Range.Start.Character &&
				params.Position.Character <= ds.Range.End.Character {

				want := ds
				// The point is on a super keyword.
				// super can only be used in the right hand side object of a binary `+` operation.
				// The definition the "super" is referring to would be the left hand side.
				// Simplified stack:
				// + lhs obj ... field x index super
				if want.Name == "super" {
					prev := stack[len(stack)-1]
					for len(stack) != 0 {
						ds := stack[len(stack)-1]
						stack = stack[:len(stack)-1]
						if ds.Kind == protocol.Operator {
							return protocol.Definition{{
								URI:   doc.item.URI,
								Range: prev.SelectionRange,
							}}, nil
						}
						prev = ds
					}
				}

				// The point is on a self keyword.
				// self can only be used in an object so we can jump to the start of that object.
				// I'm not sure that is very useful though.
				if want.Name == "self" {
					for len(stack) != 0 {
						ds := stack[len(stack)-1]
						stack = stack[:len(stack)-1]
						if ds.Kind == protocol.Object {
							return protocol.Definition{{
								URI:   doc.item.URI,
								Range: ds.SelectionRange,
							}}, nil
						}
					}
				}

				// The point is on a file symbol which must be an import.
				if want.Kind == protocol.File {
					vm, err := s.getVM(doc.item.URI.SpanURI().Filename())
					if err != nil {
						return nil, err
					}
					foundAt, err := vm.ResolveImport(doc.item.URI.SpanURI().Filename(), ds.Name)
					if err != nil {
						err = fmt.Errorf("Definition: unable to resolve import: %w", err)
						fmt.Fprintln(os.Stderr, err)
						return nil, err
					}
					return protocol.Definition{{URI: "file://" + protocol.DocumentURI(foundAt)}}, nil
				}

				// The point is on a variable, the definition of which is the first definition
				// with the same name that we find going back through the stack.
				if want.Kind == protocol.Variable && !isDefinition(want) {
					for len(stack) != 0 {
						ds := stack[len(stack)-1]
						stack = stack[:len(stack)-1]
						if ds.Kind == protocol.Variable && ds.Name == want.Name && isDefinition(ds) {
							return protocol.Definition{{
								URI:   doc.item.URI,
								Range: ds.SelectionRange,
							}}, nil
						}
					}
				}
			}
		}
		stack = append(stack, ds.Children...)
		for i := len(ds.Children); i != 0; i-- {
			if def, err := aux(stack, ds.Children[i-1]); def != nil || err != nil {
				return def, err
			}
			stack = stack[:len(stack)-1]
		}
		return nil, nil
	}

	return aux([]protocol.DocumentSymbol{doc.symbols}, doc.symbols)
}

func (s *server) publishDiagnostics(uri protocol.DocumentURI) {
	diags := []protocol.Diagnostic{}
	doc, err := s.cache.get(uri)
	if err != nil {
		fmt.Fprintf(os.Stderr, "publishDiagnostics: %s: %v\n", errorRetrievingDocument, err)
		return
	}

	diag := protocol.Diagnostic{Source: "jsonnet evaluation"}
	// Initialize with 1 because we indiscriminately subtract one to map error ranges to LSP ranges.
	line, col, endLine, endCol := 1, 1, 1, 1
	if doc.err != nil {
		lines := strings.Split(doc.err.Error(), "\n")
		if len(lines) == 0 {
			fmt.Fprintf(os.Stderr, "publishDiagnostics: expected at least two lines of Jsonnet evaluation error output, got: %v\n", lines)
			return
		}

		var match []string
		// TODO(#22): Runtime errors that come from imported files report an incorrect location
		runtimeErr := strings.HasPrefix(lines[0], "RUNTIME ERROR:")
		if runtimeErr {
			match = errRegexp.FindStringSubmatch(lines[1])
		} else {
			match = errRegexp.FindStringSubmatch(lines[0])
		}
		if len(match) == 10 {
			if match[1] != "" {
				line, _ = strconv.Atoi(match[1])
				endLine = line + 1
			}
			if match[2] != "" {
				line, _ = strconv.Atoi(match[2])
				col, _ = strconv.Atoi(match[3])
				endLine = line
				endCol, _ = strconv.Atoi(match[4])
			}
			if match[5] != "" {
				line, _ = strconv.Atoi(match[5])
				col, _ = strconv.Atoi(match[6])
				endLine, _ = strconv.Atoi(match[7])
				endCol, _ = strconv.Atoi(match[8])
			}
		}

		if runtimeErr {
			diag.Message = doc.err.Error()
			diag.Severity = protocol.SeverityWarning
		} else {
			diag.Message = match[9]
			diag.Severity = protocol.SeverityError
		}

		diag.Range = protocol.Range{
			Start: protocol.Position{Line: uint32(line - 1), Character: uint32(col - 1)},
			End:   protocol.Position{Line: uint32(endLine - 1), Character: uint32(endCol - 1)},
		}
		diags = append(diags, diag)
	}

	// TODO(#9): Replace empty context with appropriate context.
	err = s.client.PublishDiagnostics(context.TODO(), &protocol.PublishDiagnosticsParams{
		URI:         uri,
		Diagnostics: diags,
	})
	if err != nil {
		fmt.Fprintf(os.Stderr, "publishDiagnostics: unable to publish diagnostics: %v\n", err)
	}
}

func (s *server) DidChange(ctx context.Context, params *protocol.DidChangeTextDocumentParams) error {
	doc, err := s.cache.get(params.TextDocument.URI)
	if err != nil {
		err = fmt.Errorf("DidChange: %s: %w", errorRetrievingDocument, err)
		fmt.Fprintln(os.Stderr, err)
		return err
	}

	defer s.publishDiagnostics(params.TextDocument.URI)

	if params.TextDocument.Version > doc.item.Version && len(params.ContentChanges) != 0 {
		doc.item.Text = params.ContentChanges[len(params.ContentChanges)-1].Text
		doc.ast, doc.err = jsonnet.SnippetToAST(doc.item.URI.SpanURI().Filename(), doc.item.Text)
		if doc.err != nil {
			return s.cache.put(doc)
		}
		symbols := analyseSymbols(doc.ast)
		if len(symbols) != 1 {
			panic("There should only be a single root symbol for an AST")
		}
		doc.symbols = symbols[0]
		vm, err := s.getVM(doc.item.URI.SpanURI().Filename())
		if err != nil {
			return err
		}
		// TODO(#11): Evaluate whether the raw AST is better for analysis than the desugared AST.
		doc.val, doc.err = vm.EvaluateAnonymousSnippet(doc.item.URI.SpanURI().Filename(), doc.item.Text)
		return s.cache.put(doc)
	}
	return nil
}

func (s *server) DidOpen(ctx context.Context, params *protocol.DidOpenTextDocumentParams) (err error) {
	defer s.publishDiagnostics(params.TextDocument.URI)
	doc := document{item: params.TextDocument}
	if params.TextDocument.Text != "" {
		doc.ast, doc.err = jsonnet.SnippetToAST(params.TextDocument.URI.SpanURI().Filename(), params.TextDocument.Text)
		if doc.err != nil {
			return s.cache.put(doc)
		}
		symbols := analyseSymbols(doc.ast)
		if len(symbols) != 1 {
			panic("There should only be a single root symbol for an AST")
		}
		doc.symbols = symbols[0]
		vm, err := s.getVM(params.TextDocument.URI.SpanURI().Filename())
		if err != nil {
			log.Infof("DidOpen: %v", err)
			return err
		}
		doc.val, doc.err = vm.EvaluateAnonymousSnippet(params.TextDocument.URI.SpanURI().Filename(), params.TextDocument.Text)
	}
	return s.cache.put(doc)
}

func (s *server) DocumentSymbol(ctx context.Context, params *protocol.DocumentSymbolParams) ([]interface{}, error) {
	doc, err := s.cache.get(params.TextDocument.URI)
	if err != nil {
		err = fmt.Errorf("DocumentSymbol: %s: %w", errorRetrievingDocument, err)
		fmt.Fprintln(os.Stderr, err)
		return nil, err
	}

	return []interface{}{doc.symbols}, nil
}

func (s *server) Initialize(ctx context.Context, params *protocol.ParamInitialize) (*protocol.InitializeResult, error) {
	log.Infof("Initializing %s version %s", s.name, s.version)

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
			DocumentSymbolProvider:     true,
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
