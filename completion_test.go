package main

import (
	"context"
	"os"
	"testing"

	"github.com/jdbaldry/go-language-server-protocol/jsonrpc2"
	"github.com/jdbaldry/go-language-server-protocol/lsp/protocol"
	"github.com/jdbaldry/jsonnet-language-server/stdlib"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	testCompletionDocument = `{
	no_std1: d
	no_std2: s
	no_std3: d.
	no_std4: s.
	all_std_funcs: std.
	std_funcs_starting_with: std.m
}`
)

var (
	testStdLib = []stdlib.Function{
		{
			Name:                "length",
			Params:              []string{"x"},
			MarkdownDescription: "blabla",
		},
		{
			Name:                "min",
			Params:              []string{"a", "b"},
			MarkdownDescription: "min gets the min",
		},
		{
			Name:                "max",
			Params:              []string{"x"},
			MarkdownDescription: "max gets the max",
		},
	}
)

func TestCompletion(t *testing.T) {
	var testCases = []struct {
		name        string
		document    string
		position    protocol.Position
		expected    protocol.CompletionList
		expectedErr error
	}{
		{
			name:     "std: no suggestion 1",
			position: protocol.Position{Line: 1, Character: 14},
		},
		{
			name:     "std: no suggestion 2",
			position: protocol.Position{Line: 2, Character: 14},
		},
		{
			name:     "std: no suggestion 3",
			position: protocol.Position{Line: 3, Character: 15},
		},
		{
			name:     "std: no suggestion 4",
			position: protocol.Position{Line: 4, Character: 15},
		},
		{
			name:     "std: all functions",
			position: protocol.Position{Line: 5, Character: 23},
		},
		{
			name:     "std: starting with m",
			position: protocol.Position{Line: 6, Character: 34},
		},
	}
	for _, tc := range testCases {
		if tc.document == "" {
			tc.document = testCompletionDocument
		}

		t.Run(tc.name, func(t *testing.T) {
			server, fileURI := serverWithFile(t, tc.document)

			result, err := server.Completion(context.TODO(), &protocol.CompletionParams{
				TextDocumentPositionParams: protocol.TextDocumentPositionParams{
					TextDocument: protocol.TextDocumentIdentifier{URI: fileURI},
					Position:     tc.position,
				},
			})
			if tc.expectedErr != nil {
				assert.EqualError(t, err, tc.expectedErr.Error())
			} else {
				assert.NoError(t, err)
			}
			assert.Equal(t, tc.expected.Items, result.Items)
		})
	}
}

func serverWithFile(t *testing.T, fileContent string) (server *server, fileURI protocol.DocumentURI) {
	t.Helper()

	stream := jsonrpc2.NewHeaderStream(stdio{})
	conn := jsonrpc2.NewConn(stream)
	client := protocol.ClientDispatcher(conn)
	server = newServer(client, nil)
	require.NoError(t, server.Init())

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
