package server

import (
	"context"
	"os"
	"testing"

	"github.com/jdbaldry/go-language-server-protocol/lsp/protocol"
	"github.com/jdbaldry/jsonnet-language-server/stdlib"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var (
	hoverTestStdlib = []stdlib.Function{
		{
			Name:                "thisFile",
			Params:              []string{},
			MarkdownDescription: "Note that this is a field. It contains the current Jsonnet filename as a string.",
		},
		{
			Name:                "objectFields",
			Params:              []string{"o"},
			MarkdownDescription: "Returns an array of strings, each element being a field from the given object. Does not include\nhidden fields.",
		},
		{
			Name:                "map",
			Params:              []string{"any"},
			MarkdownDescription: "desc",
		},
		{
			Name:                "manifestJson",
			Params:              []string{"any"},
			MarkdownDescription: "desc",
		},
	}
	expectedThisFileHover = &protocol.Hover{
		Contents: protocol.MarkupContent{Kind: protocol.Markdown, Value: "`std.thisFile`\n\nNote that this is a field. It contains the current Jsonnet filename as a string."},
		Range: protocol.Range{
			Start: protocol.Position{Line: 1, Character: 12},
			End:   protocol.Position{Line: 1, Character: 24},
		},
	}
	expectedObjectFieldsHover = &protocol.Hover{
		Contents: protocol.MarkupContent{Kind: protocol.Markdown, Value: "`std.objectFields(o)`\n\nReturns an array of strings, each element being a field from the given object. Does not include\nhidden fields."},
		Range: protocol.Range{
			Start: protocol.Position{Line: 2, Character: 10},
			End:   protocol.Position{Line: 2, Character: 26},
		},
	}
	expectedMapHover = &protocol.Hover{
		Contents: protocol.MarkupContent{Kind: protocol.Markdown, Value: "`std.map(any)`\n\ndesc"},
		Range: protocol.Range{
			Start: protocol.Position{Line: 5, Character: 17},
			End:   protocol.Position{Line: 5, Character: 24},
		},
	}
	expectedManifestJson = &protocol.Hover{
		Contents: protocol.MarkupContent{Kind: protocol.Markdown, Value: "`std.manifestJson(any)`\n\ndesc"},
		Range: protocol.Range{
			Start: protocol.Position{Line: 7, Character: 71},
			End:   protocol.Position{Line: 7, Character: 87},
		},
	}
	expectedListComprehensionFor = &protocol.Hover{
		Contents: protocol.MarkupContent{Kind: protocol.Markdown, Value: "`std.objectFields(o)`\n\nReturns an array of strings, each element being a field from the given object. Does not include\nhidden fields."},
		Range: protocol.Range{
			Start: protocol.Position{Line: 14, Character: 7},
			End:   protocol.Position{Line: 14, Character: 23},
		},
	}
	expectedListComprehensionIf = &protocol.Hover{
		Contents: protocol.MarkupContent{Kind: protocol.Markdown, Value: "`std.map(any)`\n\ndesc"},
		Range: protocol.Range{
			Start: protocol.Position{Line: 15, Character: 7},
			End:   protocol.Position{Line: 15, Character: 14},
		},
	}
	expectedMapComprehensionFor = &protocol.Hover{
		Contents: protocol.MarkupContent{Kind: protocol.Markdown, Value: "`std.objectFields(o)`\n\nReturns an array of strings, each element being a field from the given object. Does not include\nhidden fields."},
		Range: protocol.Range{
			Start: protocol.Position{Line: 4, Character: 7},
			End:   protocol.Position{Line: 4, Character: 23},
		},
	}
	expectedMapComprehensionIf = &protocol.Hover{
		Contents: protocol.MarkupContent{Kind: protocol.Markdown, Value: "`std.map(any)`\n\ndesc"},
		Range: protocol.Range{
			Start: protocol.Position{Line: 5, Character: 7},
			End:   protocol.Position{Line: 5, Character: 14},
		},
	}
)

func TestHover(t *testing.T) {
	var testCases = []struct {
		name        string
		document    string
		position    protocol.Position
		expected    *protocol.Hover
		expectedErr error
	}{
		{
			name:     "std.thisFile over std",
			document: "./testdata/hover-std.jsonnet",
			position: protocol.Position{Line: 1, Character: 14},
			expected: expectedThisFileHover,
		},
		{
			name:     "std.thisFile over std",
			document: "./testdata/hover-std.jsonnet",
			position: protocol.Position{Line: 1, Character: 19},
			expected: expectedThisFileHover,
		},
		{
			name:     "std.objectFields over std",
			document: "./testdata/hover-std.jsonnet",
			position: protocol.Position{Line: 2, Character: 12},
			expected: expectedObjectFieldsHover,
		},
		{
			name:     "std.objectFields over func name",
			document: "./testdata/hover-std.jsonnet",
			position: protocol.Position{Line: 2, Character: 22},
			expected: expectedObjectFieldsHover,
		},
		{
			name:     "std.map over std",
			document: "./testdata/hover-std.jsonnet",
			position: protocol.Position{Line: 5, Character: 19},
			expected: expectedMapHover,
		},
		{
			name:     "std.map over func name",
			document: "./testdata/hover-std.jsonnet",
			position: protocol.Position{Line: 5, Character: 23},
			expected: expectedMapHover,
		},
		{
			name:     "std.manifestJson over std",
			document: "./testdata/hover-std.jsonnet",
			position: protocol.Position{Line: 7, Character: 73},
			expected: expectedManifestJson,
		},
		{
			name:     "std.manifestJson over func name",
			document: "./testdata/hover-std.jsonnet",
			position: protocol.Position{Line: 7, Character: 82},
			expected: expectedManifestJson,
		},
		{
			name:     "list comprehension for",
			document: "./testdata/hover-std.jsonnet",
			position: protocol.Position{Line: 14, Character: 21},
			expected: expectedListComprehensionFor,
		},
		{
			name:     "list comprehension if",
			document: "./testdata/hover-std.jsonnet",
			position: protocol.Position{Line: 15, Character: 12},
			expected: expectedListComprehensionIf,
		},
		{
			name:     "map comprehension for",
			document: "./testdata/map-comprehension.jsonnet",
			position: protocol.Position{Line: 4, Character: 21},
			expected: expectedMapComprehensionFor,
		},
		{
			name:     "map comprehension if",
			document: "./testdata/map-comprehension.jsonnet",
			position: protocol.Position{Line: 5, Character: 12},
			expected: expectedMapComprehensionIf,
		},
		{
			// We don't want to crash the server if we get an error
			name:     "hover parsing error",
			document: "./testdata/hover-error.jsonnet",
			position: protocol.Position{Line: 0, Character: 0},
			expected: nil,
		},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			server := testServer(t, hoverTestStdlib)
			uri := protocol.URIFromPath(tc.document)
			content, err := os.ReadFile(tc.document)
			require.NoError(t, err)
			err = server.DidOpen(context.Background(), &protocol.DidOpenTextDocumentParams{
				TextDocument: protocol.TextDocumentItem{
					URI:        uri,
					Text:       string(content),
					Version:    1,
					LanguageID: "jsonnet",
				},
			})
			require.NoError(t, err)

			result, err := server.Hover(context.TODO(), &protocol.HoverParams{
				TextDocumentPositionParams: protocol.TextDocumentPositionParams{
					TextDocument: protocol.TextDocumentIdentifier{URI: uri},
					Position:     tc.position,
				},
			})
			if tc.expectedErr != nil {
				assert.EqualError(t, err, tc.expectedErr.Error())
			} else {
				assert.NoError(t, err)
			}
			assert.Equal(t, tc.expected, result)
		})
	}
}
