package server

import (
	"context"
	"testing"

	"github.com/jdbaldry/go-language-server-protocol/lsp/protocol"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSymbols(t *testing.T) {
	for _, tc := range []struct {
		name          string
		filename      string
		expectSymbols []interface{}
	}{
		{
			name:     "One field",
			filename: "testdata/comment.jsonnet",
			expectSymbols: []interface{}{
				protocol.DocumentSymbol{
					Name:   "foo",
					Detail: "String",
					Kind:   protocol.Field,
					Range: protocol.Range{
						Start: protocol.Position{
							Line:      2,
							Character: 2,
						},
						End: protocol.Position{
							Line:      2,
							Character: 12,
						},
					},
					SelectionRange: protocol.Range{
						Start: protocol.Position{
							Line:      2,
							Character: 2,
						},
						End: protocol.Position{
							Line:      2,
							Character: 5,
						},
					},
				},
			},
		},
		{
			name:     "local var + two fields from plus root objects",
			filename: "testdata/basic-object.jsonnet",
			expectSymbols: []interface{}{
				protocol.DocumentSymbol{
					Name:   "somevar",
					Detail: "String",
					Kind:   protocol.Variable,
					Range: protocol.Range{
						Start: protocol.Position{
							Line:      0,
							Character: 6,
						},
						End: protocol.Position{
							Line:      0,
							Character: 23,
						},
					},
					SelectionRange: protocol.Range{
						Start: protocol.Position{
							Line:      0,
							Character: 6,
						},
						End: protocol.Position{
							Line:      0,
							Character: 13,
						},
					},
				},
				protocol.DocumentSymbol{
					Name:   "foo",
					Detail: "String",
					Kind:   protocol.Field,
					Range: protocol.Range{
						Start: protocol.Position{
							Line:      3,
							Character: 2,
						},
						End: protocol.Position{
							Line:      3,
							Character: 12,
						},
					},
					SelectionRange: protocol.Range{
						Start: protocol.Position{
							Line:      3,
							Character: 2,
						},
						End: protocol.Position{
							Line:      3,
							Character: 5,
						},
					},
				},
				protocol.DocumentSymbol{
					Name:   "bar",
					Detail: "String",
					Kind:   protocol.Field,
					Range: protocol.Range{
						Start: protocol.Position{
							Line:      6,
							Character: 2,
						},
						End: protocol.Position{
							Line:      6,
							Character: 12,
						},
					},
					SelectionRange: protocol.Range{
						Start: protocol.Position{
							Line:      6,
							Character: 2,
						},
						End: protocol.Position{
							Line:      6,
							Character: 5,
						},
					},
				},
			},
		},
		{
			name:     "Functions",
			filename: "testdata/functions.libsonnet",
			expectSymbols: []interface{}{
				protocol.DocumentSymbol{
					Name:   "myfunc",
					Detail: "Function(arg1, arg2)",
					Kind:   protocol.Variable,
					Range: protocol.Range{
						Start: protocol.Position{
							Line:      0,
							Character: 6,
						},
						End: protocol.Position{
							Line:      3,
							Character: 1,
						},
					},
					SelectionRange: protocol.Range{
						Start: protocol.Position{
							Line:      0,
							Character: 6,
						},
						End: protocol.Position{
							Line:      0,
							Character: 12,
						},
					},
				},

				protocol.DocumentSymbol{
					Name:   "objFunc",
					Detail: "Function(arg1, arg2, arg3)",
					Kind:   protocol.Field,
					Range: protocol.Range{
						Start: protocol.Position{
							Line:      6,
							Character: 2,
						},
						End: protocol.Position{
							Line:      11,
							Character: 3,
						},
					},
					SelectionRange: protocol.Range{
						Start: protocol.Position{
							Line:      6,
							Character: 2,
						},
						End: protocol.Position{
							Line:      6,
							Character: 9,
						},
					},
				},
			},
		},
		{
			name:     "Computed fields",
			filename: "testdata/computed-field-names.jsonnet",
			expectSymbols: []interface{}{
				protocol.DocumentSymbol{
					Name:   "obj",
					Detail: "Object",
					Kind:   protocol.Variable,
					Range: protocol.Range{
						Start: protocol.Position{
							Line:      0,
							Character: 6,
						},
						End: protocol.Position{
							Line:      0,
							Character: 54,
						},
					},
					SelectionRange: protocol.Range{
						Start: protocol.Position{
							Line:      0,
							Character: 6,
						},
						End: protocol.Position{
							Line:      0,
							Character: 9,
						},
					},
				},

				protocol.DocumentSymbol{
					Name:   "[obj.bar]",
					Detail: "String",
					Kind:   protocol.Field,
					Range: protocol.Range{
						Start: protocol.Position{
							Line:      3,
							Character: 2,
						},
						End: protocol.Position{
							Line:      3,
							Character: 21,
						},
					},
					SelectionRange: protocol.Range{
						Start: protocol.Position{
							Line:      3,
							Character: 2,
						},
						End: protocol.Position{
							Line:      3,
							Character: 11,
						},
					},
				},
				protocol.DocumentSymbol{
					Name:   "[obj.nested.bar]",
					Detail: "String",
					Kind:   protocol.Field,
					Range: protocol.Range{
						Start: protocol.Position{
							Line:      4,
							Character: 2,
						},
						End: protocol.Position{
							Line:      4,
							Character: 28,
						},
					},
					SelectionRange: protocol.Range{
						Start: protocol.Position{
							Line:      4,
							Character: 2,
						},
						End: protocol.Position{
							Line:      4,
							Character: 18,
						},
					},
				},
			},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			params := &protocol.DocumentSymbolParams{
				TextDocument: protocol.TextDocumentIdentifier{
					URI: protocol.URIFromPath(tc.filename),
				},
			}

			server := NewServer("any", "test version", nil, Configuration{
				JPaths: []string{"testdata"},
			})
			serverOpenTestFile(t, server, tc.filename)
			response, err := server.DocumentSymbol(context.Background(), params)
			require.NoError(t, err)

			assert.Equal(t, tc.expectSymbols, response)
		})
	}
}
