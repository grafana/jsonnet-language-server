package server

import (
	"context"
	"fmt"
	"testing"

	"github.com/jdbaldry/go-language-server-protocol/lsp/protocol"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGetTextEdits(t *testing.T) {
	testCases := []struct {
		name          string
		before, after string
		expected      []protocol.TextEdit
	}{
		{
			name:   "delete whole file",
			before: "one\ntwo\nthree",
			after:  "",
			expected: []protocol.TextEdit{
				{
					Range: protocol.Range{
						Start: protocol.Position{Line: 0, Character: 0},
						End:   protocol.Position{Line: 3, Character: 0},
					},
					NewText: "",
				},
			},
		},
		{
			name:   "add one char (replaces the whole line)",
			before: "one\ntwo\nthree",
			after:  "one\ntwoo\nthree",
			expected: []protocol.TextEdit{
				{
					Range: protocol.Range{
						Start: protocol.Position{Line: 1, Character: 0},
						End:   protocol.Position{Line: 2, Character: 0},
					},
					NewText: "",
				},
				{
					Range: protocol.Range{
						Start: protocol.Position{Line: 2, Character: 0},
						End:   protocol.Position{Line: 2, Character: 0},
					},
					NewText: "twoo\n",
				},
			},
		},
		{
			name:   "delete a line",
			before: "one\ntwo\nthree",
			after:  "one\nthree",
			expected: []protocol.TextEdit{
				{
					Range: protocol.Range{
						Start: protocol.Position{Line: 1, Character: 0},
						End:   protocol.Position{Line: 2, Character: 0},
					},
					NewText: "",
				},
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			got := getTextEdits(tc.before, tc.after)
			assert.Equal(t, tc.expected, got)
		})
	}
}

func TestFormatting(t *testing.T) {
	type kase struct {
		name        string
		settings    interface{}
		fileContent string

		expected []protocol.TextEdit
	}
	testCases := []kase{
		{
			name:     "default settings",
			settings: nil,
			fileContent: "{foo:		'bar'}",
			expected: []protocol.TextEdit{
				{Range: makeRange(t, "0:0-1:0"), NewText: ""},
				{Range: makeRange(t, "1:0-1:0"), NewText: "{ foo: 'bar' }\n"},
			},
		},
		{
			name: "new lines with indentation",
			settings: map[string]interface{}{
				"formatting": map[string]interface{}{"Indent": 4},
			},
			fileContent: `
{
	foo: 'bar',
}`,
			expected: []protocol.TextEdit{
				{Range: makeRange(t, "0:0-1:0"), NewText: ""},
				{Range: makeRange(t, "2:0-3:0"), NewText: ""},
				{Range: makeRange(t, "3:0-4:0"), NewText: ""},
				{Range: makeRange(t, "4:0-4:0"), NewText: "    foo: 'bar',\n"},
				{Range: makeRange(t, "4:0-4:0"), NewText: "}\n"},
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			s, fileURI := testServerWithFile(t, nil, tc.fileContent)

			if tc.settings != nil {
				err := s.DidChangeConfiguration(
					context.TODO(),
					&protocol.DidChangeConfigurationParams{
						Settings: tc.settings,
					},
				)
				require.NoError(t, err, "expected settings to not return an error")
			}

			edits, err := s.Formatting(context.TODO(), &protocol.DocumentFormattingParams{
				TextDocument: protocol.TextDocumentIdentifier{
					URI: fileURI,
				},
			})
			require.NoError(t, err, "expected Formatting to not return an error")
			assert.Equal(t, tc.expected, edits)
		})
	}
}

// makeRange parses rangeStr of the form
// <start-line>:<start-col>-<end-line>:<end-col> into a valid protocol.Range
func makeRange(t *testing.T, rangeStr string) protocol.Range {
	ret := protocol.Range{
		Start: protocol.Position{Line: 0, Character: 0},
		End:   protocol.Position{Line: 0, Character: 0},
	}
	n, err := fmt.Sscanf(
		rangeStr,
		"%d:%d-%d:%d",
		&ret.Start.Line,
		&ret.Start.Character,
		&ret.End.Line,
		&ret.End.Character,
	)
	require.NoError(t, err)
	require.Equal(t, 4, n)
	return ret
}
