package server

import (
	"context"
	"testing"

	"github.com/grafana/jsonnet-language-server/pkg/stdlib"
	"github.com/jdbaldry/go-language-server-protocol/lsp/protocol"
	"github.com/stretchr/testify/assert"
)

var (
	completionTestStdlib = []stdlib.Function{
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
		Detail:        "std.aaaotherMin(a)",
		Documentation: "blabla",
	}
	minItem = protocol.CompletionItem{
		Label:         "min",
		Kind:          protocol.FunctionCompletion,
		Detail:        "std.min(a, b)",
		Documentation: "min gets the min",
	}
	maxItem = protocol.CompletionItem{
		Label:         "max",
		Kind:          protocol.FunctionCompletion,
		Detail:        "std.max(a, b)",
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
		// {
		// 	name:     "std: no suggestion 1",
		// 	position: protocol.Position{Line: 0, Character: 12},
		// 	document: "{ no_std1: d }",
		// },
		// {
		// 	name:     "std: no suggestion 2",
		// 	position: protocol.Position{Line: 0, Character: 12},
		// 	document: "{ no_std2: s }",
		// },
		// {
		// 	name:     "std: no suggestion 3",
		// 	position: protocol.Position{Line: 0, Character: 13},
		// 	document: "{ no_std3: d. }",
		// },
		// {
		// 	name:     "std: no suggestion 4",
		// 	position: protocol.Position{Line: 0, Character: 13},
		// 	document: "{ no_std4: s. }",
		// },
		// {
		// 	name:     "std: all functions",
		// 	document: "{ all_std_funcs: std. }",
		// 	position: protocol.Position{Line: 0, Character: 21},
		// 	expected: protocol.CompletionList{
		// 		Items:        []protocol.CompletionItem{otherMinItem, maxItem, minItem},
		// 		IsIncomplete: false,
		// 	},
		// },
		// {
		// 	name:     "std: starting with aaa",
		// 	document: "{ std_funcs_starting_with: std.aaa }",
		// 	position: protocol.Position{Line: 0, Character: 34},
		// 	expected: protocol.CompletionList{
		// 		Items:        []protocol.CompletionItem{otherMinItem},
		// 		IsIncomplete: false,
		// 	},
		// },
		// {
		// 	name:     "std: partial match",
		// 	document: "{ partial_match: std.ther }",
		// 	position: protocol.Position{Line: 0, Character: 25},
		// 	expected: protocol.CompletionList{
		// 		Items:        []protocol.CompletionItem{otherMinItem},
		// 		IsIncomplete: false,
		// 	},
		// },
		// {
		// 	name:     "std: case insensitive",
		// 	document: "{ case_insensitive: std.MAX }",
		// 	position: protocol.Position{Line: 0, Character: 27},
		// 	expected: protocol.CompletionList{
		// 		Items:        []protocol.CompletionItem{maxItem},
		// 		IsIncomplete: false,
		// 	},
		// },
		// {
		// 	name:     "std: submatch + startswith",
		// 	document: "{ submatch_and_startwith: std.Min }",
		// 	position: protocol.Position{Line: 0, Character: 33},
		// 	expected: protocol.CompletionList{
		// 		Items:        []protocol.CompletionItem{minItem, otherMinItem},
		// 		IsIncomplete: false,
		// 	},
		// },
		{
			name: "self: other attribute",
			document: `{ 
	my_attribute: 'test',
	other_attribute: self.
}`,
			position: protocol.Position{Line: 2, Character: 26},
			expected: protocol.CompletionList{
				Items: []protocol.CompletionItem{
					{
						Label:         "my_attribute",
						Kind:          protocol.FieldCompletion,
						Detail:        "self.my_attribute",
						Documentation: "Value: test",
					},
				},
				IsIncomplete: false,
			},
		},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			if tc.expected.Items == nil {
				tc.expected = protocol.CompletionList{
					IsIncomplete: false,
					Items:        []protocol.CompletionItem{},
				}
			}

			server, fileURI := testServerWithFile(t, completionTestStdlib, tc.document)

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
