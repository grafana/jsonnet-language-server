package server

import (
	"context"
	"path/filepath"
	"testing"

	"github.com/jdbaldry/go-language-server-protocol/lsp/protocol"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type referenceResult struct {
	// Defaults to filename
	targetFilename string
	targetRange    protocol.Range
}

type referenceTestCase struct {
	name     string
	filename string
	position protocol.Position

	results []referenceResult
}

var referenceTestCases = []referenceTestCase{
	{
		name:     "local var",
		filename: "testdata/test_goto_definition.jsonnet",
		position: protocol.Position{Line: 0, Character: 9},
		results: []referenceResult{
			{
				targetRange: protocol.Range{
					Start: protocol.Position{Line: 4, Character: 5},
					End:   protocol.Position{Line: 4, Character: 10},
				},
			},
			{
				targetRange: protocol.Range{
					Start: protocol.Position{Line: 5, Character: 15},
					End:   protocol.Position{Line: 5, Character: 20},
				},
			},
			{
				targetRange: protocol.Range{
					Start: protocol.Position{Line: 7, Character: 12},
					End:   protocol.Position{Line: 7, Character: 17},
				},
			},
		},
	},
	{
		name:     "local function",
		filename: "testdata/test_goto_definition.jsonnet",
		position: protocol.Position{Line: 1, Character: 9},
		results: []referenceResult{
			{
				targetRange: protocol.Range{
					Start: protocol.Position{Line: 7, Character: 5},
					End:   protocol.Position{Line: 7, Character: 11},
				},
			},
		},
	},
	{
		name:     "function field",
		filename: "testdata/test_basic_lib.libsonnet",
		position: protocol.Position{Line: 1, Character: 5},
		results: []referenceResult{
			{
				targetRange: protocol.Range{
					Start: protocol.Position{Line: 4, Character: 11},
					End:   protocol.Position{Line: 4, Character: 21},
				},
			},
		},
	},
	{
		name:     "dollar field",
		filename: "testdata/dollar-simple.jsonnet",
		position: protocol.Position{Line: 1, Character: 8},
		results: []referenceResult{
			{
				targetRange: protocol.Range{
					Start: protocol.Position{Line: 3, Character: 8},
					End:   protocol.Position{Line: 3, Character: 26},
				},
				targetFilename: "testdata/dollar-no-follow.jsonnet",
			},
			{
				targetRange: protocol.Range{
					Start: protocol.Position{Line: 7, Character: 10},
					End:   protocol.Position{Line: 7, Character: 21},
				},
			},
			{
				targetRange: protocol.Range{
					Start: protocol.Position{Line: 8, Character: 14},
					End:   protocol.Position{Line: 8, Character: 25},
				},
			},
		},
	},
	{
		name:     "imported through locals",
		filename: "testdata/local-at-root.jsonnet",
		position: protocol.Position{Line: 8, Character: 11},
		results: []referenceResult{
			{
				targetRange: protocol.Range{
					Start: protocol.Position{Line: 2, Character: 0},
					End:   protocol.Position{Line: 2, Character: 12},
				},
				targetFilename: "testdata/local-at-root-4.jsonnet",
			},
		},
	},
}

func TestReferences(t *testing.T) {
	for _, tc := range referenceTestCases {
		t.Run(tc.name, func(t *testing.T) {
			params := &protocol.ReferenceParams{
				TextDocumentPositionParams: protocol.TextDocumentPositionParams{
					TextDocument: protocol.TextDocumentIdentifier{
						URI: protocol.URIFromPath(tc.filename),
					},
					Position: tc.position,
				},
			}

			server := NewServer("any", "test version", nil, Configuration{
				JPaths: []string{"testdata", filepath.Join(filepath.Dir(tc.filename), "vendor")},
			})
			serverOpenTestFile(t, server, tc.filename)
			response, err := server.References(context.Background(), params)
			require.NoError(t, err)

			var expected []protocol.Location
			for _, r := range tc.results {
				if r.targetFilename == "" {
					r.targetFilename = tc.filename
				}
				expected = append(expected, protocol.Location{
					URI:   protocol.URIFromPath(r.targetFilename),
					Range: r.targetRange,
				})
			}

			assert.Equal(t, expected, response)
		})
	}
}
