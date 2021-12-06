package server

import (
	"context"
	"os"
	"testing"

	"github.com/jdbaldry/go-language-server-protocol/jsonrpc2"
	"github.com/jdbaldry/go-language-server-protocol/lsp/protocol"
	"github.com/jdbaldry/jsonnet-language-server/stdlib"
	"github.com/jdbaldry/jsonnet-language-server/utils"
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
    std_funcs_starting_with: std.aaa
    partial_match: std.ther
    case_insensitive: std.MAX
    submatch_and_startwith: std.Min
}`
)

var (
	testStdLib = []stdlib.Function{
		// Starts with aaa to be the first match
		// A `min` subquery should matche this and `min`, but `min` should be first anyways
		{
			Name:                "aaaotherMin",
			Params:              []string{"a"},
			MarkdownDescription: "blabla",
		},
		{
			Name:                "max",
			Params:              []string{"a", "b"},
			MarkdownDescription: "max gets the max",
		},
		{
			Name:                "min",
			Params:              []string{"a", "b"},
			MarkdownDescription: "min gets the min",
		},
	}

	otherMinItem = protocol.CompletionItem{
		Label:         "aaaotherMin",
		Kind:          protocol.FunctionCompletion,
		Detail:        "aaaotherMin(a)",
		Documentation: "blabla",
	}
	minItem = protocol.CompletionItem{
		Label:         "min",
		Kind:          protocol.FunctionCompletion,
		Detail:        "min(a, b)",
		Documentation: "min gets the min",
	}
	maxItem = protocol.CompletionItem{
		Label:         "max",
		Kind:          protocol.FunctionCompletion,
		Detail:        "max(a, b)",
		Documentation: "max gets the max",
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
			expected: protocol.CompletionList{
				Items:        []protocol.CompletionItem{otherMinItem, maxItem, minItem},
				IsIncomplete: false,
			},
		},
		{
			name:     "std: starting with aaa",
			position: protocol.Position{Line: 6, Character: 34},
			expected: protocol.CompletionList{
				Items:        []protocol.CompletionItem{otherMinItem},
				IsIncomplete: false,
			},
		},
		{
			name:     "std: partial match",
			position: protocol.Position{Line: 7, Character: 26},
			expected: protocol.CompletionList{
				Items:        []protocol.CompletionItem{otherMinItem},
				IsIncomplete: false,
			},
		},
		{
			name:     "std: case insensitive",
			position: protocol.Position{Line: 8, Character: 29},
			expected: protocol.CompletionList{
				Items:        []protocol.CompletionItem{maxItem},
				IsIncomplete: false,
			},
		},
		{
			name:     "std: submatch + startswith",
			position: protocol.Position{Line: 9, Character: 35},
			expected: protocol.CompletionList{
				Items:        []protocol.CompletionItem{minItem, otherMinItem},
				IsIncomplete: false,
			},
		},
	}
	for _, tc := range testCases {
		if tc.document == "" {
			tc.document = testCompletionDocument
		}
		if tc.expected.Items == nil {
			tc.expected = protocol.CompletionList{
				IsIncomplete: false,
				Items:        []protocol.CompletionItem{},
			}
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
			assert.Equal(t, &tc.expected, result)
		})
	}
}

func serverWithFile(t *testing.T, fileContent string) (server *server, fileURI protocol.DocumentURI) {
	t.Helper()

	stream := jsonrpc2.NewHeaderStream(utils.Stdio{})
	conn := jsonrpc2.NewConn(stream)
	client := protocol.ClientDispatcher(conn)
	server = NewServer(client, nil)
	server.stdlib = testStdLib
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
