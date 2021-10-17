package main

import (
	"context"
	"fmt"
	"regexp"
	"strconv"
	"strings"

	"github.com/google/go-jsonnet"
	"github.com/google/go-jsonnet/formatter"
	"github.com/jdbaldry/go-language-server-protocol/jsonrpc2"
	"github.com/jdbaldry/go-language-server-protocol/lsp/protocol"
)

var (
	// file:line msg
	// file:line:col-endCol msg
	// file:(line:endLine)-(col:endCol) msg
	jsonnetErr = regexp.MustCompile(`.*:(?:(\d+)|(?:(\d+):(\d+)-(\d+))|(?:\((\d+):(\d+)\)-\((\d+):(\d+))\)) (.*)`)
)

// newServer returns a new language server.
func newServer(client protocol.ClientCloser) (*server, error) {
	return &server{
		cache:  newCache(),
		client: client,
		vm:     jsonnet.MakeVM(),
	}, nil
}

type server struct {
	cache  *cache
	client protocol.ClientCloser
	vm     *jsonnet.VM
}

func (s *server) CodeAction(context.Context, *protocol.CodeActionParams) ([]protocol.CodeAction, error) {
	return nil, notImplemented("CodeAction")
}

func (s *server) CodeLens(context.Context, *protocol.CodeLensParams) ([]protocol.CodeLens, error) {
	return nil, notImplemented("CodeLens")
}

func (s *server) CodeLensRefresh(context.Context) error {
	return notImplemented("CodeLensRefresh")
}

func (s *server) ColorPresentation(context.Context, *protocol.ColorPresentationParams) ([]protocol.ColorPresentation, error) {
	return nil, notImplemented("ColorPresentation")
}

func (s *server) Completion(context.Context, *protocol.CompletionParams) (*protocol.CompletionList, error) {
	// TODO: This is not implemented but I was unable to disable the capability.
	return nil, nil
}

func (s *server) Declaration(context.Context, *protocol.DeclarationParams) (protocol.Declaration, error) {
	return nil, notImplemented("Declaration")
}

func (s *server) Definition(context.Context, *protocol.DefinitionParams) (protocol.Definition, error) {
	return nil, notImplemented("Definition")
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

func (s *server) DidChange(ctx context.Context, params *protocol.DidChangeTextDocumentParams) error {
	if err := s.cache.update(params.TextDocument, params.ContentChanges); err != nil {
		return fmt.Errorf("DidChange: %w", err)
	}
	go func() {
		var diags []protocol.Diagnostic
		// TODO: Not ignore error.
		doc, _ := s.cache.get(params.TextDocument.URI)
		_, err := s.vm.EvaluateAnonymousSnippet(params.TextDocument.URI.SpanURI().Filename(), doc.Text)
		if err != nil {
			var (
				// Initialize with 1 because we indiscriminately subtract one to map error ranges to LSP ranges.
				line, col, endLine, endCol = 1, 1, 1, 1
				msg                        = err.Error()
			)
			match := jsonnetErr.FindStringSubmatch(msg)
			if len(match) == 10 {
				msg = match[9]
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
			diags = append(diags, protocol.Diagnostic{
				Range: protocol.Range{
					Start: protocol.Position{Line: uint32(line - 1), Character: uint32(col - 1)},
					End:   protocol.Position{Line: uint32(endLine - 1), Character: uint32(endCol - 1)},
				},
				Severity: protocol.SeverityError,
				Source:   "jsonnet evaluation",
				Message:  msg,
			})
		}
		// TODO: Not ignore error.
		// TODO: Fix context.
		_ = s.client.PublishDiagnostics(context.TODO(), &protocol.PublishDiagnosticsParams{
			URI:         params.TextDocument.URI,
			Diagnostics: diags,
		})
	}()
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

func (s *server) DidOpen(ctx context.Context, params *protocol.DidOpenTextDocumentParams) error {
	return s.cache.put(params.TextDocument)
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

func (s *server) DocumentLink(context.Context, *protocol.DocumentLinkParams) ([]protocol.DocumentLink, error) {
	// TODO: This is not implemented but I was unable to disable the capability.
	return nil, nil
}

func (s *server) DocumentSymbol(context.Context, *protocol.DocumentSymbolParams) ([]interface{}, error) {
	return nil, notImplemented("DocumentSymbol")
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
		return nil, fmt.Errorf("unable to retrieve document from cache: %w", err)
	}
	// TODO: This should be user configurable.
	formatted, err := formatter.Format(params.TextDocument.URI.SpanURI().Filename(), doc.Text, formatter.DefaultOptions())
	if err != nil {
		return nil, fmt.Errorf("unable to format document: %w", err)
	}
	// TODO: Consider applying individual edits.
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
				End:   protocol.Position{Line: uint32(strings.Count(formatted+doc.Text, "\n")), Character: ^uint32(0)},
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
