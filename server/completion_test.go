package server

import (
	"context"
	"fmt"
	"os"
	"testing"

	"github.com/jdbaldry/go-language-server-protocol/jsonrpc2"
	"github.com/jdbaldry/go-language-server-protocol/lsp/protocol"
	"github.com/jdbaldry/jsonnet-language-server/stdlib"
	"github.com/jdbaldry/jsonnet-language-server/utils"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
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
		line        string
		expected    protocol.CompletionList
		expectedErr error
	}{
		{
			name: "std: no suggestion 1",
			line: "no_std1: d",
		},
		{
			name: "std: no suggestion 2",
			line: "no_std2: s",
		},
		{
			name: "std: no suggestion 3",
			line: "no_std3: d.",
		},
		{
			name: "std: no suggestion 4",
			line: "no_std4: s.",
		},
		{
			name: "std: all functions",
			line: "all_std_funcs: std.",
			expected: protocol.CompletionList{
				Items:        []protocol.CompletionItem{otherMinItem, maxItem, minItem},
				IsIncomplete: false,
			},
		},
		{
			name: "std: starting with aaa",
			line: "std_funcs_starting_with: std.aaa",
			expected: protocol.CompletionList{
				Items:        []protocol.CompletionItem{otherMinItem},
				IsIncomplete: false,
			},
		},
		{
			name: "std: partial match",
			line: "partial_match: std.ther",
			expected: protocol.CompletionList{
				Items:        []protocol.CompletionItem{otherMinItem},
				IsIncomplete: false,
			},
		},
		{
			name: "std: case insensitive",
			line: "case_insensitive: std.MAX",
			expected: protocol.CompletionList{
				Items:        []protocol.CompletionItem{maxItem},
				IsIncomplete: false,
			},
		},
		{
			name: "std: submatch + startswith",
			line: "submatch_and_startwith: std.Min",
			expected: protocol.CompletionList{
				Items:        []protocol.CompletionItem{minItem, otherMinItem},
				IsIncomplete: false,
			},
		},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			document := fmt.Sprintf("{ %s }", tc.line)

			if tc.expected.Items == nil {
				tc.expected = protocol.CompletionList{
					IsIncomplete: false,
					Items:        []protocol.CompletionItem{},
				}
			}

			server, fileURI := serverWithFile(t, document)

			result, err := server.Completion(context.TODO(), &protocol.CompletionParams{
				TextDocumentPositionParams: protocol.TextDocumentPositionParams{
					TextDocument: protocol.TextDocumentIdentifier{URI: fileURI},
					Position:     protocol.Position{Line: 0, Character: uint32(len(tc.line) + 2)},
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
	server = NewServer(client).WithStaticVM([]string{})
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
