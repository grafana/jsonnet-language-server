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
			name:        "no error",
			fileContent: `{}`,
		},
		{
			name: "invalid function call",
			fileContent: `function(notPassed) {
	this: 'is wrong',
}()
`,
			expected: []protocol.Diagnostic{
				{
					Range: protocol.Range{
						Start: protocol.Position{Line: 0, Character: 20},
						End:   protocol.Position{Line: 2, Character: 3},
					},
					Severity: protocol.SeverityWarning,
					Source:   "lint",
					Message:  "Called value must be a function, but it is assumed to be an object",
				},
			},
		},
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

func TestGetEvalDiags(t *testing.T) {
	testCases := []struct {
		name        string
		fileContent string
		expected    []protocol.Diagnostic
	}{
		{
			name:        "no error",
			fileContent: `{}`,
		},
		{
			name:        "syntax error 1",
			fileContent: `{ s }`,
			expected: []protocol.Diagnostic{
				{
					Range: protocol.Range{
						Start: protocol.Position{Line: 0, Character: 4},
						End:   protocol.Position{Line: 0, Character: 5},
					},
					Severity: protocol.SeverityError,
					Source:   "jsonnet evaluation",
					Message:  `Expected token OPERATOR but got "}"`,
				},
			},
		},
		{
			name:        "syntax error 2",
			fileContent: `{ s: }`,
			expected: []protocol.Diagnostic{
				{
					Range: protocol.Range{
						Start: protocol.Position{Line: 0, Character: 5},
						End:   protocol.Position{Line: 0, Character: 6},
					},
					Severity: protocol.SeverityError,
					Source:   "jsonnet evaluation",
					Message:  `Unexpected: "}" while parsing terminal`,
				},
			},
		},
		{
			name:        "syntax error 3",
			fileContent: `{`,
			expected: []protocol.Diagnostic{
				{
					Range: protocol.Range{
						Start: protocol.Position{Line: 0, Character: 1},
						End:   protocol.Position{Line: 0, Character: 1},
					},
					Severity: protocol.SeverityError,
					Source:   "jsonnet evaluation",
					Message:  `Unexpected: end of file while parsing field definition`,
				},
			},
		},
		{
			name: "syntax error 4",
			fileContent: `{ 
    s: |||
|||
}`,
			expected: []protocol.Diagnostic{
				{
					Range: protocol.Range{
						Start: protocol.Position{Line: 1, Character: 7},
						End:   protocol.Position{Line: 1, Character: 7},
					},
					Severity: protocol.SeverityError,
					Source:   "jsonnet evaluation",
					Message:  `Text block's first line must start with whitespace`,
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

			diags := s.getEvalDiags(doc)
			assert.Equal(t, tc.expected, diags)
		})
	}
}
