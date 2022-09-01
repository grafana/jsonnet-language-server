package server

import (
	"context"
	"fmt"

	"github.com/jdbaldry/go-language-server-protocol/jsonrpc2"
	"github.com/jdbaldry/go-language-server-protocol/lsp/protocol"
)

func (s *server) Initialized(context.Context, *protocol.InitializedParams) error {
	return nil
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

func (s *server) Declaration(context.Context, *protocol.DeclarationParams) (protocol.Declaration, error) {
	return nil, notImplemented("Declaration")
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

func (s *server) Exit(context.Context) error {
	return notImplemented("Exit")
}

func (s *server) FoldingRange(context.Context, *protocol.FoldingRangeParams) ([]protocol.FoldingRange, error) {
	return nil, notImplemented("FoldingRange")
}

func (s *server) Implementation(context.Context, *protocol.ImplementationParams) (protocol.Definition, error) {
	return nil, notImplemented("Implementation")
}

func (s *server) IncomingCalls(context.Context, *protocol.CallHierarchyIncomingCallsParams) ([]protocol.CallHierarchyIncomingCall, error) {
	return nil, notImplemented("IncomingCalls")
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
	return nil
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

func (s *server) Diagnostic(context.Context, *string) (*string, error) {
	return nil, notImplemented("Diagnostic")
}

func (s *server) DiagnosticRefresh(context.Context) error {
	return notImplemented("DiagnosticRefresh")
}

func (s *server) DiagnosticWorkspace(context.Context, *protocol.WorkspaceDiagnosticParams) (*protocol.WorkspaceDiagnosticReport, error) {
	return nil, notImplemented("DiagnosticWorkspace")
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

// DocumentLink is not implemented.
// TODO(#13): Understand why the server capabilities includes documentlink.
func (s *server) DocumentLink(context.Context, *protocol.DocumentLinkParams) ([]protocol.DocumentLink, error) {
	return nil, nil
}

func notImplemented(method string) error {
	return fmt.Errorf("%w: %q not yet implemented", jsonrpc2.ErrMethodNotFound, method)
}
