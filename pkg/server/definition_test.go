package server

import (
	_ "embed"
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/google/go-jsonnet"
	"github.com/jdbaldry/go-language-server-protocol/lsp/protocol"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func getVM() (vm *jsonnet.VM) {
	vm = jsonnet.MakeVM()
	vm.Importer(&jsonnet.FileImporter{JPaths: []string{"testdata"}})
	return
}

func TestDefinition(t *testing.T) {
	testCases := []struct {
		name        string
		filename    string
		position    protocol.Position
		targetRange protocol.Range

		// Defaults to filename
		targetFilename string

		// Default to targetRange
		targetSelectionRange protocol.Range
	}{
		{
			name:     "test goto definition for var myvar",
			filename: "./testdata/test_goto_definition.jsonnet",
			position: protocol.Position{Line: 5, Character: 19},
			targetRange: protocol.Range{
				Start: protocol.Position{Line: 0, Character: 6},
				End:   protocol.Position{Line: 0, Character: 15},
			},
			targetSelectionRange: protocol.Range{
				Start: protocol.Position{Line: 0, Character: 6},
				End:   protocol.Position{Line: 0, Character: 11},
			},
		},
		{
			name:     "test goto definition on function helper",
			filename: "./testdata/test_goto_definition.jsonnet",
			position: protocol.Position{Line: 7, Character: 8},
			targetRange: protocol.Range{
				Start: protocol.Position{Line: 1, Character: 6},
				End:   protocol.Position{Line: 1, Character: 23},
			},
			targetSelectionRange: protocol.Range{
				Start: protocol.Position{Line: 1, Character: 6},
				End:   protocol.Position{Line: 1, Character: 12},
			},
		},
		{
			name:     "test goto inner definition",
			filename: "./testdata/test_goto_definition_multi_locals.jsonnet",
			position: protocol.Position{Line: 6, Character: 11},
			targetRange: protocol.Range{
				Start: protocol.Position{Line: 4, Character: 10},
				End:   protocol.Position{Line: 4, Character: 28},
			},
			targetSelectionRange: protocol.Range{
				Start: protocol.Position{Line: 4, Character: 10},
				End:   protocol.Position{Line: 4, Character: 18},
			},
		},
		{
			name:     "test goto super index",
			filename: "./testdata/test_combined_object.jsonnet",
			position: protocol.Position{Line: 5, Character: 13},
			targetRange: protocol.Range{
				Start: protocol.Position{Line: 1, Character: 4},
				End:   protocol.Position{Line: 3, Character: 5},
			},
			targetSelectionRange: protocol.Range{
				Start: protocol.Position{Line: 1, Character: 4},
				End:   protocol.Position{Line: 1, Character: 5},
			},
		},
		{
			name:     "test goto super nested",
			filename: "./testdata/test_combined_object.jsonnet",
			position: protocol.Position{Line: 5, Character: 15},
			targetRange: protocol.Range{
				Start: protocol.Position{Line: 2, Character: 8},
				End:   protocol.Position{Line: 2, Character: 22},
			},
			targetSelectionRange: protocol.Range{
				Start: protocol.Position{Line: 2, Character: 8},
				End:   protocol.Position{Line: 2, Character: 9},
			},
		},
		{
			name:     "test goto self object field function",
			filename: "./testdata/test_basic_lib.libsonnet",
			position: protocol.Position{Line: 4, Character: 19},
			targetRange: protocol.Range{
				Start: protocol.Position{Line: 1, Character: 4},
				End:   protocol.Position{Line: 3, Character: 20},
			},
			targetSelectionRange: protocol.Range{
				Start: protocol.Position{Line: 1, Character: 4},
				End:   protocol.Position{Line: 1, Character: 9},
			},
		},
		{
			name:     "test goto super object field local defined obj 'foo'",
			filename: "./testdata/oo-contrived.jsonnet",
			position: protocol.Position{Line: 12, Character: 17},
			targetRange: protocol.Range{
				Start: protocol.Position{Line: 1, Character: 2},
				End:   protocol.Position{Line: 1, Character: 8},
			},
			targetSelectionRange: protocol.Range{
				Start: protocol.Position{Line: 1, Character: 2},
				End:   protocol.Position{Line: 1, Character: 5},
			},
		},
		{
			name:     "test goto super object field local defined obj 'g'",
			filename: "./testdata/oo-contrived.jsonnet",
			position: protocol.Position{Line: 13, Character: 17},
			targetRange: protocol.Range{
				Start: protocol.Position{Line: 2, Character: 2},
				End:   protocol.Position{Line: 2, Character: 19},
			},
			targetSelectionRange: protocol.Range{
				Start: protocol.Position{Line: 2, Character: 2},
				End:   protocol.Position{Line: 2, Character: 3},
			},
		},
		{
			name:     "test goto local var from other local var",
			filename: "./testdata/oo-contrived.jsonnet",
			position: protocol.Position{Line: 6, Character: 9},
			targetRange: protocol.Range{
				Start: protocol.Position{Line: 0, Character: 6},
				End:   protocol.Position{Line: 3, Character: 1},
			},
			targetSelectionRange: protocol.Range{
				Start: protocol.Position{Line: 0, Character: 6},
				End:   protocol.Position{Line: 0, Character: 10},
			},
		},
		{
			name:     "test goto local obj field from 'self.attr' from other obj",
			filename: "./testdata/goto-indexes.jsonnet",
			position: protocol.Position{Line: 9, Character: 18},
			targetRange: protocol.Range{
				Start: protocol.Position{Line: 2, Character: 8},
				End:   protocol.Position{Line: 2, Character: 23},
			},
			targetSelectionRange: protocol.Range{
				Start: protocol.Position{Line: 2, Character: 8},
				End:   protocol.Position{Line: 2, Character: 11},
			},
		},
		{
			name:     "test goto local object 'obj' via obj index 'obj.foo'",
			filename: "./testdata/goto-indexes.jsonnet",
			position: protocol.Position{Line: 8, Character: 15},
			targetRange: protocol.Range{
				Start: protocol.Position{Line: 1, Character: 4},
				End:   protocol.Position{Line: 3, Character: 5},
			},
			targetSelectionRange: protocol.Range{
				Start: protocol.Position{Line: 1, Character: 4},
				End:   protocol.Position{Line: 1, Character: 7},
			},
		},
		{
			name:           "test goto imported file",
			filename:       "./testdata/goto-imported-file.jsonnet",
			position:       protocol.Position{Line: 0, Character: 22},
			targetFilename: "testdata/goto-basic-object.jsonnet",
			targetRange: protocol.Range{
				Start: protocol.Position{Line: 0, Character: 0},
				End:   protocol.Position{Line: 0, Character: 0},
			},
			targetSelectionRange: protocol.Range{
				Start: protocol.Position{Line: 0, Character: 0},
				End:   protocol.Position{Line: 0, Character: 0},
			},
		},
		{
			name:           "test goto imported file at lhs index",
			filename:       "./testdata/goto-imported-file.jsonnet",
			position:       protocol.Position{Line: 3, Character: 18},
			targetFilename: "testdata/goto-basic-object.jsonnet",
			targetRange: protocol.Range{
				Start: protocol.Position{Line: 3, Character: 4},
				End:   protocol.Position{Line: 3, Character: 14},
			},
			targetSelectionRange: protocol.Range{
				Start: protocol.Position{Line: 3, Character: 4},
				End:   protocol.Position{Line: 3, Character: 7},
			},
		},
		{
			name:           "test goto imported file at rhs index",
			filename:       "./testdata/goto-imported-file.jsonnet",
			position:       protocol.Position{Line: 4, Character: 18},
			targetFilename: "testdata/goto-basic-object.jsonnet",
			targetRange: protocol.Range{
				Start: protocol.Position{Line: 5, Character: 4},
				End:   protocol.Position{Line: 5, Character: 14},
			},
			targetSelectionRange: protocol.Range{
				Start: protocol.Position{Line: 5, Character: 4},
				End:   protocol.Position{Line: 5, Character: 7},
			},
		},
		{
			name:           "goto import index",
			filename:       "testdata/goto-import-attribute.jsonnet",
			position:       protocol.Position{Line: 0, Character: 48},
			targetFilename: "testdata/goto-basic-object.jsonnet",
			targetRange: protocol.Range{
				Start: protocol.Position{Line: 5, Character: 4},
				End:   protocol.Position{Line: 5, Character: 14},
			},
			targetSelectionRange: protocol.Range{
				Start: protocol.Position{Line: 5, Character: 4},
				End:   protocol.Position{Line: 5, Character: 7},
			},
		},
		{
			name:           "goto attribute of nested import",
			filename:       "testdata/goto-nested-imported-file.jsonnet",
			position:       protocol.Position{Line: 2, Character: 15},
			targetFilename: "testdata/goto-basic-object.jsonnet",
			targetRange: protocol.Range{
				Start: protocol.Position{Line: 3, Character: 4},
				End:   protocol.Position{Line: 3, Character: 14},
			},
			targetSelectionRange: protocol.Range{
				Start: protocol.Position{Line: 3, Character: 4},
				End:   protocol.Position{Line: 3, Character: 7},
			},
		},
		{
			name:     "goto dollar attribute",
			filename: "testdata/goto-dollar-simple.jsonnet",
			position: protocol.Position{Line: 7, Character: 17},
			targetRange: protocol.Range{
				Start: protocol.Position{Line: 1, Character: 2},
				End:   protocol.Position{Line: 3, Character: 3},
			},
			targetSelectionRange: protocol.Range{
				Start: protocol.Position{Line: 1, Character: 2},
				End:   protocol.Position{Line: 1, Character: 11},
			},
		},
		{
			name:     "goto dollar sub attribute",
			filename: "testdata/goto-dollar-simple.jsonnet",
			position: protocol.Position{Line: 8, Character: 28},
			targetRange: protocol.Range{
				Start: protocol.Position{Line: 2, Character: 4},
				End:   protocol.Position{Line: 2, Character: 15},
			},
			targetSelectionRange: protocol.Range{
				Start: protocol.Position{Line: 2, Character: 4},
				End:   protocol.Position{Line: 2, Character: 7},
			},
		},
		{
			name:     "goto dollar doesn't follow to imports",
			filename: "testdata/goto-dollar-no-follow.jsonnet",
			position: protocol.Position{Line: 7, Character: 13},
			targetRange: protocol.Range{
				Start: protocol.Position{Line: 3, Character: 2},
				End:   protocol.Position{Line: 3, Character: 30},
			},
			targetSelectionRange: protocol.Range{
				Start: protocol.Position{Line: 3, Character: 2},
				End:   protocol.Position{Line: 3, Character: 6},
			},
		},
		{
			name:           "goto attribute of nested import no object intermediary",
			filename:       "testdata/goto-nested-import-file-no-inter-obj.jsonnet",
			position:       protocol.Position{Line: 2, Character: 15},
			targetFilename: "testdata/goto-basic-object.jsonnet",
			targetRange: protocol.Range{
				Start: protocol.Position{Line: 3, Character: 4},
				End:   protocol.Position{Line: 3, Character: 14},
			},
			targetSelectionRange: protocol.Range{
				Start: protocol.Position{Line: 3, Character: 4},
				End:   protocol.Position{Line: 3, Character: 7},
			},
		},
		{
			name:           "goto self in import in binary",
			filename:       "testdata/goto-self-within-binary.jsonnet",
			position:       protocol.Position{Line: 4, Character: 15},
			targetFilename: "testdata/goto-basic-object.jsonnet",
			targetRange: protocol.Range{
				Start: protocol.Position{Line: 3, Character: 4},
				End:   protocol.Position{Line: 3, Character: 14},
			},
			targetSelectionRange: protocol.Range{
				Start: protocol.Position{Line: 3, Character: 4},
				End:   protocol.Position{Line: 3, Character: 7},
			},
		},
		{
			name:     "goto self attribute from local",
			filename: "testdata/goto-self-in-local.jsonnet",
			position: protocol.Position{Line: 3, Character: 23},
			targetRange: protocol.Range{
				Start: protocol.Position{Line: 2, Character: 2},
				End:   protocol.Position{Line: 2, Character: 21},
			},
			targetSelectionRange: protocol.Range{
				Start: protocol.Position{Line: 2, Character: 2},
				End:   protocol.Position{Line: 2, Character: 12},
			},
		},
		{
			name:     "goto function parameter from inside function",
			filename: "testdata/goto-functions.libsonnet",
			position: protocol.Position{Line: 7, Character: 10},
			targetRange: protocol.Range{
				Start: protocol.Position{Line: 6, Character: 10},
				End:   protocol.Position{Line: 6, Character: 14},
			},
		},
		{
			name:     "goto local func param",
			filename: "testdata/goto-local-function.libsonnet",
			position: protocol.Position{Line: 2, Character: 25},
			targetRange: protocol.Range{
				Start: protocol.Position{Line: 0, Character: 11},
				End:   protocol.Position{Line: 0, Character: 12},
			},
		},
		{
			name:     "goto self complex scope 1",
			filename: "testdata/goto-self-complex-scoping.jsonnet",
			position: protocol.Position{Line: 10, Character: 15},
			targetRange: protocol.Range{
				Start: protocol.Position{Line: 6, Character: 2},
				End:   protocol.Position{Line: 8, Character: 3},
			},
			targetSelectionRange: protocol.Range{
				Start: protocol.Position{Line: 6, Character: 2},
				End:   protocol.Position{Line: 6, Character: 6},
			},
		},
		{
			name:     "goto self complex scope 2",
			filename: "testdata/goto-self-complex-scoping.jsonnet",
			position: protocol.Position{Line: 11, Character: 19},
			targetRange: protocol.Range{
				Start: protocol.Position{Line: 7, Character: 4},
				End:   protocol.Position{Line: 7, Character: 18},
			},
			targetSelectionRange: protocol.Range{
				Start: protocol.Position{Line: 7, Character: 4},
				End:   protocol.Position{Line: 7, Character: 9},
			},
		},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			var content, err = os.ReadFile(tc.filename)
			require.NoError(t, err)
			ast, err := jsonnet.SnippetToAST(tc.filename, string(content))
			require.NoError(t, err)

			params := &protocol.DefinitionParams{
				TextDocumentPositionParams: protocol.TextDocumentPositionParams{
					TextDocument: protocol.TextDocumentIdentifier{
						URI: protocol.URIFromPath(tc.filename),
					},
					Position: tc.position,
				},
			}

			got, err := Definition(ast, params, getVM())
			assert.NoError(t, err)

			// Defaults
			if tc.targetSelectionRange.End.Character == 0 {
				tc.targetSelectionRange = tc.targetRange
			}
			if tc.targetFilename == "" {
				tc.targetFilename = tc.filename
			}

			// Check results
			expected := &protocol.LocationLink{
				TargetURI:            absUri(t, tc.targetFilename),
				TargetRange:          tc.targetRange,
				TargetSelectionRange: tc.targetSelectionRange,
			}

			assert.Equal(t, expected, got)
		})
	}
}

func TestDefinitionFail(t *testing.T) {
	testCases := []struct {
		name     string
		params   protocol.DefinitionParams
		expected error
	}{
		{
			name: "goto local keyword fails",
			params: protocol.DefinitionParams{
				TextDocumentPositionParams: protocol.TextDocumentPositionParams{
					TextDocument: protocol.TextDocumentIdentifier{
						URI: "testdata/goto-basic-object.jsonnet",
					},
					Position: protocol.Position{
						Line:      0,
						Character: 3,
					},
				},
			},
			expected: fmt.Errorf("cannot find definition"),
		},

		{
			name: "goto index of std fails",
			params: protocol.DefinitionParams{
				TextDocumentPositionParams: protocol.TextDocumentPositionParams{
					TextDocument: protocol.TextDocumentIdentifier{
						URI: "testdata/goto-std.jsonnet",
					},
					Position: protocol.Position{
						Line:      1,
						Character: 20,
					},
				},
			},
			expected: fmt.Errorf("cannot get definition of std lib"),
		},
		{
			name: "goto comment fails",
			params: protocol.DefinitionParams{
				TextDocumentPositionParams: protocol.TextDocumentPositionParams{
					TextDocument: protocol.TextDocumentIdentifier{
						URI: "testdata/goto-comment.jsonnet",
					},
					Position: protocol.Position{
						Line:      0,
						Character: 1,
					},
				},
			},
			expected: fmt.Errorf("cannot find definition"),
		},

		{
			name: "goto range index fails",
			params: protocol.DefinitionParams{
				TextDocumentPositionParams: protocol.TextDocumentPositionParams{
					TextDocument: protocol.TextDocumentIdentifier{
						URI: "testdata/goto-local-function.libsonnet",
					},
					Position: protocol.Position{
						Line:      15,
						Character: 57,
					},
				},
			},
			expected: fmt.Errorf("unexpected node type when finding bind for 'ports'"),
		},
		{
			name: "goto super fails as no LHS object exists",
			params: protocol.DefinitionParams{
				TextDocumentPositionParams: protocol.TextDocumentPositionParams{
					TextDocument: protocol.TextDocumentIdentifier{
						URI: "testdata/goto-local-function.libsonnet",
					},
					Position: protocol.Position{
						Line:      33,
						Character: 23,
					},
				},
			},
			expected: fmt.Errorf("could not find a lhs object"),
		},
		{
			name: "goto self fails when out of scope",
			params: protocol.DefinitionParams{
				TextDocumentPositionParams: protocol.TextDocumentPositionParams{
					TextDocument: protocol.TextDocumentIdentifier{
						URI: "testdata/goto-self-complex-scoping.jsonnet",
					},
					Position: protocol.Position{
						Line:      3,
						Character: 18,
					},
				},
			},
			expected: fmt.Errorf("field test was not found in ast.DesugaredObject"),
		},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			filename := string(tc.params.TextDocument.URI)
			var content, err = os.ReadFile(filename)
			require.NoError(t, err)
			ast, err := jsonnet.SnippetToAST(filename, string(content))
			require.NoError(t, err)
			got, err := Definition(ast, &tc.params, getVM())
			require.Error(t, err)
			assert.Equal(t, tc.expected.Error(), err.Error())
			assert.Nil(t, got)
		})
	}
}

func absUri(t *testing.T, path string) protocol.DocumentURI {
	t.Helper()

	abs, err := filepath.Abs(path)
	require.NoError(t, err)
	return protocol.URIFromPath(abs)
}
