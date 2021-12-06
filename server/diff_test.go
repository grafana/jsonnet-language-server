package server

import (
	"testing"

	"github.com/jdbaldry/go-language-server-protocol/lsp/protocol"
	"github.com/stretchr/testify/assert"
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
