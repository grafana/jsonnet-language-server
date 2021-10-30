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

package main

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"

	"github.com/google/go-jsonnet"
	"github.com/google/go-jsonnet/formatter"
	"github.com/jdbaldry/go-language-server-protocol/jsonrpc2"
	"github.com/jdbaldry/go-language-server-protocol/lsp/protocol"
)

const (
	symbolTagDefinition     protocol.SymbolTag = 100
	errorRetrievingDocument                    = "unable to retrieve document from the cache"
)

var (
	// errRegexp matches the various Jsonnet location formats in errors.
	// file:line msg
	// file:line:col-endCol msg
	// file:(line:endLine)-(col:endCol) msg
	// Has 10 matching groups.
	errRegexp = regexp.MustCompile(`/.*:(?:(\d+)|(?:(\d+):(\d+)-(\d+))|(?:\((\d+):(\d+)\)-\((\d+):(\d+))\))\s(.*)`)
)

// newServer returns a new language server.
func newServer(client protocol.ClientCloser) (*server, error) {
	vm := jsonnet.MakeVM()
	importer := &jsonnet.FileImporter{JPaths: filepath.SplitList(os.Getenv("JSONNET_PATH"))}
	vm.Importer(importer)
	return &server{
		cache:  newCache(),
		client: client,
		vm:     vm,
	}, nil
}

// server is the Jsonnet language server.
type server struct {
	cache  *cache
	client protocol.ClientCloser
	vm     *jsonnet.VM
}

func (s *server) CodeAction(context.Context, *protocol.CodeActionParams) ([]protocol.CodeAction, error) {
	return nil, notImplemented("CodeAction")
}

func (s *server) CodeLens(ctx context.Context, params *protocol.CodeLensParams) ([]protocol.CodeLens, error) {
	return []protocol.CodeLens{}, nil
}

func (s *server) CodeLensRefresh(context.Context) error {
	return notImplemented("CodeLensRefresh")
}

func (s *server) ColorPresentation(context.Context, *protocol.ColorPresentationParams) ([]protocol.ColorPresentation, error) {
	return nil, notImplemented("ColorPresentation")
}

// Completion is not implemented.
// TODO(#6): Understand why the server capabilities includes completion.
func (s *server) Completion(context.Context, *protocol.CompletionParams) (*protocol.CompletionList, error) {
	return nil, nil
}

func (s *server) Declaration(context.Context, *protocol.DeclarationParams) (protocol.Declaration, error) {
	return nil, notImplemented("Declaration")
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
					foundAt, err := s.vm.ResolveImport(doc.item.URI.SpanURI().Filename(), ds.Name)
					if err != nil {
						err = fmt.Errorf("Definition: unable to resolve import: %w", err)
						fmt.Fprintln(os.Stderr, err)
						return nil, err
					}
					return protocol.Definition{{URI: "file:///" + protocol.DocumentURI(foundAt)}}, nil
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

func (s *server) Diagnostic(context.Context, *string) (*string, error) {
	return nil, notImplemented("Diagnostic")
}

func (s *server) DiagnosticRefresh(context.Context) error {
	return notImplemented("DiagnosticRefresh")
}

func (s *server) DiagnosticWorkspace(context.Context, *protocol.WorkspaceDiagnosticParams) (*protocol.WorkspaceDiagnosticReport, error) {
	return nil, notImplemented("DiagnosticWorkspace")
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
		// TODO(#10): Work out better way to invalidate the VM cache.
		s.vm.Importer(&jsonnet.FileImporter{})
		// TODO(#11): Evaluate whether the raw AST is better for analysis than the desugared AST.
		doc.val, doc.err = s.vm.EvaluateAnonymousSnippet(doc.item.URI.SpanURI().Filename(), doc.item.Text)
		return s.cache.put(doc)
	}
	return nil
}

func (s *server) DidChangeConfiguration(context.Context, *protocol.DidChangeConfigurationParams) error {
	return notImplemented("DidChangeConfiguration")
}

func (s *server) DidChangeWatchedFiles(context.Context, *protocol.DidChangeWatchedFilesParams) error {
	return notImplemented("DidChangeWatchedFiles")
}

func (s *server) DidChangeWorkspaceFolders(context.Context, *protocol.DidChangeWorkspaceFoldersParams) error {
	return notImplemented("DidChangeWorkspaceFolders")
}

func (s *server) DidClose(context.Context, *protocol.DidCloseTextDocumentParams) error {
	return notImplemented("DidClose")
}

func (s *server) DidCreateFiles(context.Context, *protocol.CreateFilesParams) error {
	return notImplemented("DidCreateFiles")
}

func (s *server) DidDeleteFiles(context.Context, *protocol.DeleteFilesParams) error {
	return notImplemented("DidDeleteFiles")
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
		// TODO(#12): Work out better way to invalidate the VM cache.
		doc.val, doc.err = s.vm.EvaluateAnonymousSnippet(params.TextDocument.URI.SpanURI().Filename(), params.TextDocument.Text)
	}
	return s.cache.put(doc)
}

func (s *server) DidRenameFiles(context.Context, *protocol.RenameFilesParams) error {
	return notImplemented("DidRenameFiles")
}

func (s *server) DidSave(context.Context, *protocol.DidSaveTextDocumentParams) error {
	return notImplemented("DidSave")
}

func (s *server) DocumentColor(context.Context, *protocol.DocumentColorParams) ([]protocol.ColorInformation, error) {
	return nil, notImplemented("DocumentColor")
}

func (s *server) DocumentHighlight(context.Context, *protocol.DocumentHighlightParams) ([]protocol.DocumentHighlight, error) {
	return nil, notImplemented("DocumentHighlight")
}

// DocumentLink is not implemented.
// TODO(#13): Understand why the server capabilities includes documentlink.
func (s *server) DocumentLink(context.Context, *protocol.DocumentLinkParams) ([]protocol.DocumentLink, error) {
	return nil, nil
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

func (s *server) ExecuteCommand(context.Context, *protocol.ExecuteCommandParams) (interface{}, error) {
	return nil, notImplemented("ExecuteCommand")
}

func (s *server) Exit(context.Context) error {
	return notImplemented("Exit")
}

func (s *server) FoldingRange(context.Context, *protocol.FoldingRangeParams) ([]protocol.FoldingRange, error) {
	return nil, notImplemented("FoldingRange")
}

func (s *server) Formatting(ctx context.Context, params *protocol.DocumentFormattingParams) ([]protocol.TextEdit, error) {
	doc, err := s.cache.get(params.TextDocument.URI)
	if err != nil {
		err = fmt.Errorf("Formatting: %s: %w", errorRetrievingDocument, err)
		fmt.Fprintln(os.Stderr, err)
		return nil, err
	}
	// TODO(#14): Formatting options should be user configurable.
	formatted, err := formatter.Format(params.TextDocument.URI.SpanURI().Filename(), doc.item.Text, formatter.DefaultOptions())
	if err != nil {
		err = fmt.Errorf("Formatting: unable to format document: %w", err)
		fmt.Fprintln(os.Stderr, err)
		return nil, err
	}
	// TODO(#15): Consider applying individual edits instead of replacing the whole file when formatting.
	return []protocol.TextEdit{
		{
			Range: protocol.Range{
				Start: protocol.Position{Line: 0, Character: 0},
				End:   protocol.Position{Line: 0, Character: 0},
			},
			NewText: formatted,
		},
		{
			Range: protocol.Range{
				Start: protocol.Position{Line: 0, Character: 0},
				End:   protocol.Position{Line: uint32(strings.Count(formatted+doc.item.Text, "\n")), Character: ^uint32(0)},
			},
			NewText: "",
		},
	}, nil
}

func (s *server) Hover(context.Context, *protocol.HoverParams) (*protocol.Hover, error) {
	return nil, notImplemented("Hover")
}

func (s *server) Implementation(context.Context, *protocol.ImplementationParams) (protocol.Definition, error) {
	return nil, notImplemented("Implementation")
}

func (s *server) IncomingCalls(context.Context, *protocol.CallHierarchyIncomingCallsParams) ([]protocol.CallHierarchyIncomingCall, error) {
	return nil, notImplemented("IncomingCalls")
}

func (s *server) Initialize(ctx context.Context, params *protocol.ParamInitialize) (*protocol.InitializeResult, error) {
	return &protocol.InitializeResult{
		Capabilities: protocol.ServerCapabilities{
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
			Name: "jsonnet-language-server",
		},
	}, nil
}

func (s *server) Initialized(context.Context, *protocol.InitializedParams) error {
	return nil
}

func (s *server) LinkedEditingRange(context.Context, *protocol.LinkedEditingRangeParams) (*protocol.LinkedEditingRanges, error) {
	return nil, notImplemented("LinkedEditingRange")
}

func (s *server) LogTrace(context.Context, *protocol.LogTraceParams) error {
	return notImplemented("LogTrace")
}

func (s *server) Moniker(context.Context, *protocol.MonikerParams) ([]protocol.Moniker, error) {
	return nil, notImplemented("Moniker")
}

func (s *server) NonstandardRequest(context.Context, string, interface{}) (interface{}, error) {
	return nil, notImplemented("NonstandardRequest")
}

func (s *server) OnTypeFormatting(context.Context, *protocol.DocumentOnTypeFormattingParams) ([]protocol.TextEdit, error) {
	return nil, notImplemented("OnTypeFormatting")
}

func (s *server) OutgoingCalls(context.Context, *protocol.CallHierarchyOutgoingCallsParams) ([]protocol.CallHierarchyOutgoingCall, error) {
	return nil, notImplemented("OutgoingCalls")
}

func (s *server) PrepareCallHierarchy(context.Context, *protocol.CallHierarchyPrepareParams) ([]protocol.CallHierarchyItem, error) {
	return nil, notImplemented("PrepareCallHierarchy")
}

func (s *server) PrepareRename(context.Context, *protocol.PrepareRenameParams) (*protocol.Range, error) {
	return nil, notImplemented("PrepareRange")
}

func (s *server) PrepareTypeHierarchy(context.Context, *protocol.TypeHierarchyPrepareParams) ([]protocol.TypeHierarchyItem, error) {
	return nil, notImplemented("PrepareTypeHierarchy")
}

func (s *server) RangeFormatting(context.Context, *protocol.DocumentRangeFormattingParams) ([]protocol.TextEdit, error) {
	return nil, notImplemented("RangeFormatting")
}

func (s *server) References(context.Context, *protocol.ReferenceParams) ([]protocol.Location, error) {
	return nil, notImplemented("References")
}

func (s *server) Rename(context.Context, *protocol.RenameParams) (*protocol.WorkspaceEdit, error) {
	return nil, notImplemented("Rename")
}

func (s *server) Resolve(context.Context, *protocol.CompletionItem) (*protocol.CompletionItem, error) {
	return nil, notImplemented("Resolve")
}

func (s *server) ResolveCodeAction(context.Context, *protocol.CodeAction) (*protocol.CodeAction, error) {
	return nil, notImplemented("ResolveCodeAction")
}

func (s *server) ResolveCodeLens(context.Context, *protocol.CodeLens) (*protocol.CodeLens, error) {
	return nil, notImplemented("ResolveCodeLens")
}

func (s *server) ResolveDocumentLink(context.Context, *protocol.DocumentLink) (*protocol.DocumentLink, error) {
	return nil, notImplemented("ResolveDocumentLink")
}

func (s *server) SelectionRange(context.Context, *protocol.SelectionRangeParams) ([]protocol.SelectionRange, error) {
	return nil, notImplemented("SelectionRange")
}

func (s *server) SemanticTokensFull(context.Context, *protocol.SemanticTokensParams) (*protocol.SemanticTokens, error) {
	return nil, notImplemented("SemanticTokensFull")
}

func (s *server) SemanticTokensFullDelta(context.Context, *protocol.SemanticTokensDeltaParams) (interface{}, error) {
	return nil, notImplemented("SemanticTokensFullDelta")
}

func (s *server) SemanticTokensRange(context.Context, *protocol.SemanticTokensRangeParams) (*protocol.SemanticTokens, error) {
	return nil, notImplemented("SemanticTokensRange")
}

func (s *server) SemanticTokensRefresh(context.Context) error {
	return notImplemented("SemanticTokensRefresh")
}

func (s *server) SetTrace(context.Context, *protocol.SetTraceParams) error {
	return notImplemented("SetTrace")
}

func (s *server) Shutdown(context.Context) error {
	return notImplemented("Shutdown")
}

func (s *server) SignatureHelp(context.Context, *protocol.SignatureHelpParams) (*protocol.SignatureHelp, error) {
	return nil, notImplemented("SignatureHelp")
}

func (s *server) Subtypes(context.Context, *protocol.TypeHierarchySubtypesParams) ([]protocol.TypeHierarchyItem, error) {
	return nil, notImplemented("Subtypes")
}

func (s *server) Supertypes(context.Context, *protocol.TypeHierarchySupertypesParams) ([]protocol.TypeHierarchyItem, error) {
	return nil, notImplemented("Supertypes")
}

func (s *server) Symbol(context.Context, *protocol.WorkspaceSymbolParams) ([]protocol.SymbolInformation, error) {
	return nil, notImplemented("Symbol")
}

func (s *server) TypeDefinition(context.Context, *protocol.TypeDefinitionParams) (protocol.Definition, error) {
	return nil, notImplemented("TypeDefinition")
}

func (s *server) WillCreateFiles(context.Context, *protocol.CreateFilesParams) (*protocol.WorkspaceEdit, error) {
	return nil, notImplemented("WillCreateFiles")
}

func (s *server) WillDeleteFiles(context.Context, *protocol.DeleteFilesParams) (*protocol.WorkspaceEdit, error) {
	return nil, notImplemented("WillDeleteFiles")
}

func (s *server) WillRenameFiles(context.Context, *protocol.RenameFilesParams) (*protocol.WorkspaceEdit, error) {
	return nil, notImplemented("WillRenameFiles")
}

func (s *server) WillSave(context.Context, *protocol.WillSaveTextDocumentParams) error {
	return notImplemented("WillSave")
}

func (s *server) WillSaveWaitUntil(context.Context, *protocol.WillSaveTextDocumentParams) ([]protocol.TextEdit, error) {
	return nil, notImplemented("WillSaveWaitUntil")
}

func (s *server) WorkDoneProgressCancel(context.Context, *protocol.WorkDoneProgressCancelParams) error {
	return notImplemented("WorkDoneProgressCancel")
}

func notImplemented(method string) error {
	return fmt.Errorf("%w: %q not yet implemented", jsonrpc2.ErrMethodNotFound, method)
}
