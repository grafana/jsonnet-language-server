package server

import (
	"context"
	"io"
	"os"
	"path/filepath"
	"testing"

	"github.com/google/go-jsonnet/formatter"
	"github.com/grafana/jsonnet-language-server/pkg/stdlib"
	"github.com/grafana/jsonnet-language-server/pkg/utils"
	"github.com/jdbaldry/go-language-server-protocol/jsonrpc2"
	"github.com/jdbaldry/go-language-server-protocol/lsp/protocol"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/require"
)

type fakeWriterCloser struct {
	io.Writer
}

func (fakeWriterCloser) Close() error {
	return nil
}

func init() {
	logrus.SetLevel(logrus.WarnLevel)
}

func absURI(t *testing.T, path string) protocol.DocumentURI {
	t.Helper()

	abs, err := filepath.Abs(path)
	require.NoError(t, err)
	return protocol.URIFromPath(abs)
}

func testServer(t *testing.T, stdlib []stdlib.Function) (server *server) {
	t.Helper()

	stream := jsonrpc2.NewHeaderStream(utils.NewStdio(nil, fakeWriterCloser{io.Discard}))
	conn := jsonrpc2.NewConn(stream)
	client := protocol.ClientDispatcher(conn)
	server = NewServer("jsonnet-language-server", "dev", client, Configuration{
		FormattingOptions: formatter.DefaultOptions(),
	})
	server.stdlib = stdlib
	_, err := server.Initialize(context.Background(), &protocol.ParamInitialize{})
	require.NoError(t, err)

	return server
}

func serverOpenTestFile(t require.TestingT, server *server, filename string) protocol.DocumentURI {
	fileContent, err := os.ReadFile(filename)
	require.NoError(t, err)

	uri := protocol.URIFromPath(filename)
	err = server.DidOpen(context.Background(), &protocol.DidOpenTextDocumentParams{
		TextDocument: protocol.TextDocumentItem{
			URI:        uri,
			Text:       string(fileContent),
			Version:    1,
			LanguageID: "jsonnet",
		},
	})
	require.NoError(t, err)

	return uri
}

func testServerWithFile(t *testing.T, stdlib []stdlib.Function, fileContent string) (server *server, fileURI protocol.DocumentURI) {
	t.Helper()

	server = testServer(t, stdlib)

	tmpFile, err := os.CreateTemp("", "")
	require.NoError(t, err)

	_, err = tmpFile.WriteString(fileContent)
	require.NoError(t, err)

	return server, serverOpenTestFile(t, server, tmpFile.Name())
}
