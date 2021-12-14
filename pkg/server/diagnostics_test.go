package server

import (
	"testing"

	"github.com/jdbaldry/go-language-server-protocol/lsp/protocol"
	"github.com/stretchr/testify/assert"
)

func TestGetLintDiags(t *testing.T) {
	testCases := []struct {
		name        string
		fileContent string
		expected    []protocol.Diagnostic
	}{
		{
			name: "unused variable",
			fileContent: `
local unused = 'test';
{}
`,
			expected: []protocol.Diagnostic{
				{
					Range: protocol.Range{
						Start: protocol.Position{Line: 1, Character: 6},
						End:   protocol.Position{Line: 1, Character: 21},
					},
					Severity: protocol.SeverityWarning,
					Source:   "lint",
					Message:  "Unused variable: unused",
				},
			},
		},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			s, fileURI := testServerWithFile(t, nil, tc.fileContent)
			doc, err := s.cache.get(fileURI)
			if err != nil {
				t.Fatalf("%s: %v", errorRetrievingDocument, err)
			}

			diags := s.getLintDiags(doc)
			assert.Equal(t, tc.expected, diags)
		})
	}
}
