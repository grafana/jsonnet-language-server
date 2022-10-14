package server

import (
	"context"
	"fmt"
	"os"
	"strings"
	"testing"

	"github.com/grafana/jsonnet-language-server/pkg/stdlib"
	"github.com/jdbaldry/go-language-server-protocol/lsp/protocol"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
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
		InsertText:    "aaaotherMin(a)",
		Documentation: "blabla",
	}
	minItem = protocol.CompletionItem{
		Label:         "min",
		Kind:          protocol.FunctionCompletion,
		Detail:        "std.min(a, b)",
		InsertText:    "min(a, b)",
		Documentation: "min gets the min",
	}
	maxItem = protocol.CompletionItem{
		Label:         "max",
		Kind:          protocol.FunctionCompletion,
		Detail:        "std.max(a, b)",
		InsertText:    "max(a, b)",
		Documentation: "max gets the max",
	}
)

func TestCompletionStdLib(t *testing.T) {
	var testCases = []struct {
		name        string
		line        string
		expected    *protocol.CompletionList
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
			expected: &protocol.CompletionList{
				Items:        []protocol.CompletionItem{otherMinItem, maxItem, minItem},
				IsIncomplete: false,
			},
		},
		{
			name: "std: starting with aaa",
			line: "std_funcs_starting_with: std.aaa",
			expected: &protocol.CompletionList{
				Items:        []protocol.CompletionItem{otherMinItem},
				IsIncomplete: false,
			},
		},
		{
			name: "std: partial match",
			line: "partial_match: std.ther",
			expected: &protocol.CompletionList{
				Items:        []protocol.CompletionItem{otherMinItem},
				IsIncomplete: false,
			},
		},
		{
			name: "std: case insensitive",
			line: "case_insensitive: std.MAX",
			expected: &protocol.CompletionList{
				Items:        []protocol.CompletionItem{maxItem},
				IsIncomplete: false,
			},
		},
		{
			name: "std: submatch + startswith",
			line: "submatch_and_startwith: std.Min",
			expected: &protocol.CompletionList{
				Items:        []protocol.CompletionItem{minItem, otherMinItem},
				IsIncomplete: false,
			},
		},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			document := fmt.Sprintf("{ %s }", tc.line)

			server, fileURI := testServerWithFile(t, completionTestStdlib, document)

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
			assert.Equal(t, tc.expected, result)
		})
	}
}

func TestCompletion(t *testing.T) {
	var testCases = []struct {
		name                           string
		filename                       string
		replaceString, replaceByString string
		expected                       protocol.CompletionList
	}{
		{
			name:            "self function",
			filename:        "testdata/test_basic_lib.libsonnet",
			replaceString:   "self.greet('Zack')",
			replaceByString: "self.",
			expected: protocol.CompletionList{
				IsIncomplete: false,
				Items: []protocol.CompletionItem{{
					Label:      "greet",
					Kind:       protocol.FunctionCompletion,
					Detail:     "self.greet(name)",
					InsertText: "greet(name)",
				}},
			},
		},
		{
			name:            "self function with bad first letter letter",
			filename:        "testdata/test_basic_lib.libsonnet",
			replaceString:   "self.greet('Zack')",
			replaceByString: "self.h",
			expected: protocol.CompletionList{
				IsIncomplete: false,
				Items:        nil,
			},
		},
		{
			name:            "self function with first letter",
			filename:        "testdata/test_basic_lib.libsonnet",
			replaceString:   "self.greet('Zack')",
			replaceByString: "self.g",
			expected: protocol.CompletionList{
				IsIncomplete: false,
				Items: []protocol.CompletionItem{{
					Label:      "greet",
					Kind:       protocol.FunctionCompletion,
					Detail:     "self.greet(name)",
					InsertText: "greet(name)",
				}},
			},
		},
		{
			name:            "autocomplete through binary",
			filename:        "testdata/goto-basic-object.jsonnet",
			replaceString:   "bar: 'foo',",
			replaceByString: "bar: self.",
			expected: protocol.CompletionList{
				IsIncomplete: false,
				Items: []protocol.CompletionItem{{
					Label:      "foo",
					Kind:       protocol.FieldCompletion,
					Detail:     "self.foo",
					InsertText: "foo",
				}},
			},
		},
		{
			name:            "autocomplete locals",
			filename:        "testdata/goto-basic-object.jsonnet",
			replaceString:   "bar: 'foo',",
			replaceByString: "bar: ",
			expected: protocol.CompletionList{
				IsIncomplete: false,
				Items: []protocol.CompletionItem{{
					Label:      "somevar",
					Kind:       protocol.VariableCompletion,
					Detail:     "somevar",
					InsertText: "somevar",
				}},
			},
		},
		{
			name:            "autocomplete locals: good prefix",
			filename:        "testdata/goto-basic-object.jsonnet",
			replaceString:   "bar: 'foo',",
			replaceByString: "bar: some",
			expected: protocol.CompletionList{
				IsIncomplete: false,
				Items: []protocol.CompletionItem{{
					Label:      "somevar",
					Kind:       protocol.VariableCompletion,
					Detail:     "somevar",
					InsertText: "somevar",
				}},
			},
		},
		{
			name:            "autocomplete locals: bad prefix",
			filename:        "testdata/goto-basic-object.jsonnet",
			replaceString:   "bar: 'foo',",
			replaceByString: "bar: bad",
			expected: protocol.CompletionList{
				IsIncomplete: false,
				Items:        nil,
			},
		},
		{
			name:            "autocomplete through import",
			filename:        "testdata/goto-imported-file.jsonnet",
			replaceString:   "b: otherfile.bar,",
			replaceByString: "b: otherfile.",
			expected: protocol.CompletionList{
				IsIncomplete: false,
				Items: []protocol.CompletionItem{
					{
						Label:      "bar",
						Kind:       protocol.FieldCompletion,
						Detail:     "otherfile.bar",
						InsertText: "bar",
					},
					{
						Label:      "foo",
						Kind:       protocol.FieldCompletion,
						Detail:     "otherfile.foo",
						InsertText: "foo",
					},
				},
			},
		},
		{
			name:            "autocomplete through import with prefix",
			filename:        "testdata/goto-imported-file.jsonnet",
			replaceString:   "b: otherfile.bar,",
			replaceByString: "b: otherfile.b",
			expected: protocol.CompletionList{
				IsIncomplete: false,
				Items: []protocol.CompletionItem{
					{
						Label:      "bar",
						Kind:       protocol.FieldCompletion,
						Detail:     "otherfile.bar",
						InsertText: "bar",
					},
				},
			},
		},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			content, err := os.ReadFile(tc.filename)
			require.NoError(t, err)

			server, fileURI := testServerWithFile(t, completionTestStdlib, string(content))
			server.configuration.JPaths = []string{"testdata"}

			replacedContent := strings.ReplaceAll(string(content), tc.replaceString, tc.replaceByString)

			err = server.DidChange(context.Background(), &protocol.DidChangeTextDocumentParams{
				ContentChanges: []protocol.TextDocumentContentChangeEvent{{
					Text: replacedContent,
				}},
				TextDocument: protocol.VersionedTextDocumentIdentifier{
					TextDocumentIdentifier: protocol.TextDocumentIdentifier{URI: fileURI},
					Version:                2,
				},
			})
			require.NoError(t, err)

			cursorPosition := protocol.Position{}
			for _, line := range strings.Split(replacedContent, "\n") {
				if strings.Contains(line, tc.replaceByString) {
					cursorPosition.Character = uint32(strings.Index(line, tc.replaceByString) + len(tc.replaceByString))
					break
				}
				cursorPosition.Line++
			}
			if cursorPosition.Character == 0 {
				t.Fatal("Could not find cursor position for test. Replace probably didn't work")
			}

			result, err := server.Completion(context.TODO(), &protocol.CompletionParams{
				TextDocumentPositionParams: protocol.TextDocumentPositionParams{
					TextDocument: protocol.TextDocumentIdentifier{URI: fileURI},
					Position:     cursorPosition,
				},
			})
			require.NoError(t, err)
			assert.Equal(t, &tc.expected, result)
		})
	}
}
