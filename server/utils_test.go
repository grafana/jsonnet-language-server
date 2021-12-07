package server

import (
	"context"
	"os"
	"testing"

	"github.com/jdbaldry/go-language-server-protocol/jsonrpc2"
	"github.com/jdbaldry/go-language-server-protocol/lsp/protocol"
	"github.com/jdbaldry/jsonnet-language-server/stdlib"
	"github.com/jdbaldry/jsonnet-language-server/utils"
	"github.com/stretchr/testify/require"
)

func testServer(t *testing.T, stdlib []stdlib.Function) (server *server) {
	t.Helper()

	stream := jsonrpc2.NewHeaderStream(utils.Stdio{})
	conn := jsonrpc2.NewConn(stream)
	client := protocol.ClientDispatcher(conn)
	server = NewServer("jsonnet-language-server", "dev", client).WithStaticVM([]string{})
	server.stdlib = stdlib
	_, err := server.Initialize(context.Background(), &protocol.ParamInitialize{})
	require.NoError(t, err)

	return server
}

func testServerWithFile(t *testing.T, stdlib []stdlib.Function, fileContent string) (server *server, fileURI protocol.DocumentURI) {
	t.Helper()

	server = testServer(t, stdlib)

	tmpFile, err := os.CreateTemp("", "")
	require.NoError(t, err)
	uri := protocol.URIFromPath(tmpFile.Name())

	_, err = tmpFile.WriteString(fileContent)
	require.NoError(t, err)

	err = server.DidOpen(context.Background(), &protocol.DidOpenTextDocumentParams{
		TextDocument: protocol.TextDocumentItem{
			URI:        uri,
			Text:       fileContent,
			Version:    1,
			LanguageID: "jsonnet",
		},
	})
	require.NoError(t, err)

	return server, uri
}
