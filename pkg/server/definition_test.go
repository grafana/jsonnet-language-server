package server

import (
	_ "embed"
	"fmt"
	"path/filepath"
	"testing"

	"github.com/jdbaldry/go-language-server-protocol/lsp/protocol"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type definitionResult struct {
	// Defaults to filename
	targetFilename string
	targetRange    protocol.Range
	// Defaults to targetRange
	targetSelectionRange protocol.Range
}

type definitionTestCase struct {
	name     string
	filename string
	position protocol.Position

	results []definitionResult
}

var definitionTestCases = []definitionTestCase{
	{
		name:     "test goto definition for var myvar",
		filename: "./testdata/test_goto_definition.jsonnet",
		position: protocol.Position{Line: 5, Character: 19},
		results: []definitionResult{{
			targetRange: protocol.Range{
				Start: protocol.Position{Line: 0, Character: 6},
				End:   protocol.Position{Line: 0, Character: 15},
			},
			targetSelectionRange: protocol.Range{
				Start: protocol.Position{Line: 0, Character: 6},
				End:   protocol.Position{Line: 0, Character: 11},
			},
		}},
	},
	{
		name:     "test goto definition on function helper",
		filename: "./testdata/test_goto_definition.jsonnet",
		position: protocol.Position{Line: 7, Character: 8},
		results: []definitionResult{{
			targetRange: protocol.Range{
				Start: protocol.Position{Line: 1, Character: 6},
				End:   protocol.Position{Line: 1, Character: 23},
			},
			targetSelectionRange: protocol.Range{
				Start: protocol.Position{Line: 1, Character: 6},
				End:   protocol.Position{Line: 1, Character: 12},
			},
		}},
	},
	{
		name:     "test goto inner definition",
		filename: "./testdata/test_goto_definition_multi_locals.jsonnet",
		position: protocol.Position{Line: 6, Character: 9},
		results: []definitionResult{{
			targetRange: protocol.Range{
				Start: protocol.Position{Line: 4, Character: 8},
				End:   protocol.Position{Line: 4, Character: 26},
			},
			targetSelectionRange: protocol.Range{
				Start: protocol.Position{Line: 4, Character: 8},
				End:   protocol.Position{Line: 4, Character: 16},
			},
		}},
	},
	{
		name:     "test goto super index",
		filename: "./testdata/test_combined_object.jsonnet",
		position: protocol.Position{Line: 5, Character: 11},
		results: []definitionResult{{
			targetRange: protocol.Range{
				Start: protocol.Position{Line: 1, Character: 2},
				End:   protocol.Position{Line: 3, Character: 3},
			},
			targetSelectionRange: protocol.Range{
				Start: protocol.Position{Line: 1, Character: 2},
				End:   protocol.Position{Line: 1, Character: 3},
			},
		}},
	},
	{
		name:     "test goto super nested",
		filename: "./testdata/test_combined_object.jsonnet",
		position: protocol.Position{Line: 5, Character: 13},
		results: []definitionResult{{
			targetRange: protocol.Range{
				Start: protocol.Position{Line: 2, Character: 4},
				End:   protocol.Position{Line: 2, Character: 18},
			},
			targetSelectionRange: protocol.Range{
				Start: protocol.Position{Line: 2, Character: 4},
				End:   protocol.Position{Line: 2, Character: 5},
			},
		}},
	},
	{
		name:     "test goto self object field function",
		filename: "./testdata/test_basic_lib.libsonnet",
		position: protocol.Position{Line: 4, Character: 17},
		results: []definitionResult{{
			targetRange: protocol.Range{
				Start: protocol.Position{Line: 1, Character: 2},
				End:   protocol.Position{Line: 3, Character: 19},
			},
			targetSelectionRange: protocol.Range{
				Start: protocol.Position{Line: 1, Character: 2},
				End:   protocol.Position{Line: 1, Character: 7},
			},
		}},
	},
	{
		name:     "test goto super object field local defined obj 'foo'",
		filename: "./testdata/oo-contrived.jsonnet",
		position: protocol.Position{Line: 12, Character: 17},
		results: []definitionResult{{
			targetRange: protocol.Range{
				Start: protocol.Position{Line: 1, Character: 2},
				End:   protocol.Position{Line: 1, Character: 8},
			},
			targetSelectionRange: protocol.Range{
				Start: protocol.Position{Line: 1, Character: 2},
				End:   protocol.Position{Line: 1, Character: 5},
			},
		}},
	},
	{
		name:     "test goto super object field local defined obj 'g'",
		filename: "./testdata/oo-contrived.jsonnet",
		position: protocol.Position{Line: 13, Character: 17},
		results: []definitionResult{{
			targetRange: protocol.Range{
				Start: protocol.Position{Line: 2, Character: 2},
				End:   protocol.Position{Line: 2, Character: 19},
			},
			targetSelectionRange: protocol.Range{
				Start: protocol.Position{Line: 2, Character: 2},
				End:   protocol.Position{Line: 2, Character: 3},
			},
		}},
	},
	{
		name:     "test goto local var from other local var",
		filename: "./testdata/oo-contrived.jsonnet",
		position: protocol.Position{Line: 6, Character: 9},
		results: []definitionResult{{
			targetRange: protocol.Range{
				Start: protocol.Position{Line: 0, Character: 6},
				End:   protocol.Position{Line: 3, Character: 1},
			},
			targetSelectionRange: protocol.Range{
				Start: protocol.Position{Line: 0, Character: 6},
				End:   protocol.Position{Line: 0, Character: 10},
			},
		}},
	},
	{
		name:     "test goto local obj field from 'self.attr' from other obj",
		filename: "./testdata/goto-indexes.jsonnet",
		position: protocol.Position{Line: 9, Character: 16},
		results: []definitionResult{{
			targetRange: protocol.Range{
				Start: protocol.Position{Line: 2, Character: 4},
				End:   protocol.Position{Line: 2, Character: 19},
			},
			targetSelectionRange: protocol.Range{
				Start: protocol.Position{Line: 2, Character: 4},
				End:   protocol.Position{Line: 2, Character: 7},
			},
		}},
	},
	{
		name:     "test goto local object 'obj' via obj index 'obj.foo'",
		filename: "./testdata/goto-indexes.jsonnet",
		position: protocol.Position{Line: 8, Character: 13},
		results: []definitionResult{{
			targetRange: protocol.Range{
				Start: protocol.Position{Line: 1, Character: 2},
				End:   protocol.Position{Line: 3, Character: 3},
			},
			targetSelectionRange: protocol.Range{
				Start: protocol.Position{Line: 1, Character: 2},
				End:   protocol.Position{Line: 1, Character: 5},
			},
		}},
	},
	{
		name:     "test goto imported file",
		filename: "./testdata/goto-imported-file.jsonnet",
		position: protocol.Position{Line: 0, Character: 22},
		results: []definitionResult{{
			targetFilename: "testdata/goto-basic-object.jsonnet",
			targetRange: protocol.Range{
				Start: protocol.Position{Line: 0, Character: 0},
				End:   protocol.Position{Line: 0, Character: 0},
			},
			targetSelectionRange: protocol.Range{
				Start: protocol.Position{Line: 0, Character: 0},
				End:   protocol.Position{Line: 0, Character: 0},
			},
		}},
	},
	{
		name:     "test goto imported file at lhs index",
		filename: "./testdata/goto-imported-file.jsonnet",
		position: protocol.Position{Line: 3, Character: 16},
		results: []definitionResult{{
			targetFilename: "testdata/goto-basic-object.jsonnet",
			targetRange: protocol.Range{
				Start: protocol.Position{Line: 3, Character: 2},
				End:   protocol.Position{Line: 3, Character: 12},
			},
			targetSelectionRange: protocol.Range{
				Start: protocol.Position{Line: 3, Character: 2},
				End:   protocol.Position{Line: 3, Character: 5},
			},
		}},
	},
	{
		name:     "test goto imported file at rhs index",
		filename: "./testdata/goto-imported-file.jsonnet",
		position: protocol.Position{Line: 4, Character: 16},
		results: []definitionResult{{
			targetFilename: "testdata/goto-basic-object.jsonnet",
			targetRange: protocol.Range{
				Start: protocol.Position{Line: 5, Character: 2},
				End:   protocol.Position{Line: 5, Character: 12},
			},
			targetSelectionRange: protocol.Range{
				Start: protocol.Position{Line: 5, Character: 2},
				End:   protocol.Position{Line: 5, Character: 5},
			},
		}},
	},
	{
		name:     "goto import index",
		filename: "testdata/goto-import-attribute.jsonnet",
		position: protocol.Position{Line: 0, Character: 48},
		results: []definitionResult{{
			targetFilename: "testdata/goto-basic-object.jsonnet",
			targetRange: protocol.Range{
				Start: protocol.Position{Line: 5, Character: 2},
				End:   protocol.Position{Line: 5, Character: 12},
			},
			targetSelectionRange: protocol.Range{
				Start: protocol.Position{Line: 5, Character: 2},
				End:   protocol.Position{Line: 5, Character: 5},
			},
		}},
	},
	{
		name:     "goto attribute of nested import",
		filename: "testdata/goto-nested-imported-file.jsonnet",
		position: protocol.Position{Line: 2, Character: 13},
		results: []definitionResult{{
			targetFilename: "testdata/goto-basic-object.jsonnet",
			targetRange: protocol.Range{
				Start: protocol.Position{Line: 3, Character: 2},
				End:   protocol.Position{Line: 3, Character: 12},
			},
			targetSelectionRange: protocol.Range{
				Start: protocol.Position{Line: 3, Character: 2},
				End:   protocol.Position{Line: 3, Character: 5},
			},
		}},
	},
	{
		name:     "goto dollar attribute",
		filename: "testdata/goto-dollar-simple.jsonnet",
		position: protocol.Position{Line: 7, Character: 17},
		results: []definitionResult{{
			targetRange: protocol.Range{
				Start: protocol.Position{Line: 1, Character: 2},
				End:   protocol.Position{Line: 3, Character: 3},
			},
			targetSelectionRange: protocol.Range{
				Start: protocol.Position{Line: 1, Character: 2},
				End:   protocol.Position{Line: 1, Character: 11},
			},
		}},
	},
	{
		name:     "goto dollar sub attribute",
		filename: "testdata/goto-dollar-simple.jsonnet",
		position: protocol.Position{Line: 8, Character: 28},
		results: []definitionResult{{
			targetRange: protocol.Range{
				Start: protocol.Position{Line: 2, Character: 4},
				End:   protocol.Position{Line: 2, Character: 15},
			},
			targetSelectionRange: protocol.Range{
				Start: protocol.Position{Line: 2, Character: 4},
				End:   protocol.Position{Line: 2, Character: 7},
			},
		}},
	},
	{
		name:     "goto dollar doesn't follow to imports",
		filename: "testdata/goto-dollar-no-follow.jsonnet",
		position: protocol.Position{Line: 7, Character: 13},
		results: []definitionResult{{
			targetRange: protocol.Range{
				Start: protocol.Position{Line: 3, Character: 2},
				End:   protocol.Position{Line: 3, Character: 30},
			},
			targetSelectionRange: protocol.Range{
				Start: protocol.Position{Line: 3, Character: 2},
				End:   protocol.Position{Line: 3, Character: 6},
			},
		}},
	},
	{
		name:     "goto attribute of nested import no object intermediary",
		filename: "testdata/goto-nested-import-file-no-inter-obj.jsonnet",
		position: protocol.Position{Line: 2, Character: 13},
		results: []definitionResult{{
			targetFilename: "testdata/goto-basic-object.jsonnet",
			targetRange: protocol.Range{
				Start: protocol.Position{Line: 3, Character: 2},
				End:   protocol.Position{Line: 3, Character: 12},
			},
			targetSelectionRange: protocol.Range{
				Start: protocol.Position{Line: 3, Character: 2},
				End:   protocol.Position{Line: 3, Character: 5},
			},
		}},
	},
	{
		name:     "goto self in import in binary",
		filename: "testdata/goto-self-within-binary.jsonnet",
		position: protocol.Position{Line: 4, Character: 13},
		results: []definitionResult{{
			targetFilename: "testdata/goto-basic-object.jsonnet",
			targetRange: protocol.Range{
				Start: protocol.Position{Line: 3, Character: 2},
				End:   protocol.Position{Line: 3, Character: 12},
			},
			targetSelectionRange: protocol.Range{
				Start: protocol.Position{Line: 3, Character: 2},
				End:   protocol.Position{Line: 3, Character: 5},
			},
		}},
	},
	{
		name:     "goto self attribute from local",
		filename: "testdata/goto-self-in-local.jsonnet",
		position: protocol.Position{Line: 3, Character: 23},
		results: []definitionResult{{
			targetRange: protocol.Range{
				Start: protocol.Position{Line: 2, Character: 2},
				End:   protocol.Position{Line: 2, Character: 21},
			},
			targetSelectionRange: protocol.Range{
				Start: protocol.Position{Line: 2, Character: 2},
				End:   protocol.Position{Line: 2, Character: 12},
			},
		}},
	},
	{
		name:     "goto function parameter from inside function",
		filename: "testdata/goto-functions.libsonnet",
		position: protocol.Position{Line: 7, Character: 10},
		results: []definitionResult{{
			targetRange: protocol.Range{
				Start: protocol.Position{Line: 6, Character: 10},
				End:   protocol.Position{Line: 6, Character: 14},
			},
		}},
	},
	{
		name:     "goto local func param",
		filename: "testdata/goto-local-function.libsonnet",
		position: protocol.Position{Line: 2, Character: 25},
		results: []definitionResult{{
			targetRange: protocol.Range{
				Start: protocol.Position{Line: 0, Character: 11},
				End:   protocol.Position{Line: 0, Character: 12},
			},
		}},
	},
	{
		name:     "goto self complex scope 1",
		filename: "testdata/goto-self-complex-scoping.jsonnet",
		position: protocol.Position{Line: 10, Character: 15},
		results: []definitionResult{{
			targetRange: protocol.Range{
				Start: protocol.Position{Line: 6, Character: 2},
				End:   protocol.Position{Line: 8, Character: 3},
			},
			targetSelectionRange: protocol.Range{
				Start: protocol.Position{Line: 6, Character: 2},
				End:   protocol.Position{Line: 6, Character: 6},
			},
		}},
	},
	{
		name:     "goto self complex scope 2",
		filename: "testdata/goto-self-complex-scoping.jsonnet",
		position: protocol.Position{Line: 11, Character: 19},
		results: []definitionResult{{
			targetRange: protocol.Range{
				Start: protocol.Position{Line: 7, Character: 4},
				End:   protocol.Position{Line: 7, Character: 18},
			},
			targetSelectionRange: protocol.Range{
				Start: protocol.Position{Line: 7, Character: 4},
				End:   protocol.Position{Line: 7, Character: 9},
			},
		}},
	},
	{
		name:     "goto with overrides: clobber string",
		filename: "testdata/goto-overrides.jsonnet",
		position: protocol.Position{Line: 41, Character: 30},
		results: []definitionResult{{
			targetRange: protocol.Range{
				Start: protocol.Position{Line: 24, Character: 4},
				End:   protocol.Position{Line: 24, Character: 23},
			},
			targetSelectionRange: protocol.Range{
				Start: protocol.Position{Line: 24, Character: 4},
				End:   protocol.Position{Line: 24, Character: 10},
			},
		}},
	},
	{
		name:     "goto with overrides: clobber nested string",
		filename: "testdata/goto-overrides.jsonnet",
		position: protocol.Position{Line: 42, Character: 44},
		results: []definitionResult{{
			targetRange: protocol.Range{
				Start: protocol.Position{Line: 26, Character: 6},
				End:   protocol.Position{Line: 26, Character: 24},
			},
			targetSelectionRange: protocol.Range{
				Start: protocol.Position{Line: 26, Character: 6},
				End:   protocol.Position{Line: 26, Character: 11},
			},
		}},
	},
	{
		name:     "goto with overrides: clobber map",
		filename: "testdata/goto-overrides.jsonnet",
		position: protocol.Position{Line: 43, Character: 28},
		results: []definitionResult{{
			targetRange: protocol.Range{
				Start: protocol.Position{Line: 28, Character: 4},
				End:   protocol.Position{Line: 28, Character: 15},
			},
			targetSelectionRange: protocol.Range{
				Start: protocol.Position{Line: 28, Character: 4},
				End:   protocol.Position{Line: 28, Character: 11},
			},
		}},
	},
	{
		name:     "goto with overrides: map (multiple definitions)",
		filename: "testdata/goto-overrides.jsonnet",
		position: protocol.Position{Line: 32, Character: 22},
		results: []definitionResult{
			{
				targetRange: protocol.Range{
					Start: protocol.Position{Line: 23, Character: 2},
					End:   protocol.Position{Line: 29, Character: 3},
				},
				targetSelectionRange: protocol.Range{
					Start: protocol.Position{Line: 23, Character: 2},
					End:   protocol.Position{Line: 23, Character: 3},
				},
			},
			{
				targetRange: protocol.Range{
					Start: protocol.Position{Line: 14, Character: 2},
					End:   protocol.Position{Line: 19, Character: 3},
				},
				targetSelectionRange: protocol.Range{
					Start: protocol.Position{Line: 14, Character: 2},
					End:   protocol.Position{Line: 14, Character: 3},
				},
			},
			{
				targetRange: protocol.Range{
					Start: protocol.Position{Line: 2, Character: 2},
					End:   protocol.Position{Line: 10, Character: 3},
				},
				targetSelectionRange: protocol.Range{
					Start: protocol.Position{Line: 2, Character: 2},
					End:   protocol.Position{Line: 2, Character: 3},
				},
			},
			{
				targetFilename: "testdata/goto-overrides-base.jsonnet",
				targetRange: protocol.Range{
					Start: protocol.Position{Line: 19, Character: 2},
					End:   protocol.Position{Line: 19, Character: 94},
				},
				targetSelectionRange: protocol.Range{
					Start: protocol.Position{Line: 19, Character: 2},
					End:   protocol.Position{Line: 19, Character: 3},
				},
			},
			{
				targetFilename: "testdata/goto-overrides-base.jsonnet",
				targetRange: protocol.Range{
					Start: protocol.Position{Line: 16, Character: 2},
					End:   protocol.Position{Line: 16, Character: 24},
				},
				targetSelectionRange: protocol.Range{
					Start: protocol.Position{Line: 16, Character: 2},
					End:   protocol.Position{Line: 16, Character: 3},
				},
			},
			{
				targetFilename: "testdata/goto-overrides-base.jsonnet",
				targetRange: protocol.Range{
					Start: protocol.Position{Line: 1, Character: 2},
					End:   protocol.Position{Line: 7, Character: 3},
				},
				targetSelectionRange: protocol.Range{
					Start: protocol.Position{Line: 1, Character: 2},
					End:   protocol.Position{Line: 1, Character: 3},
				},
			},
		},
	},
	{
		name:     "goto with overrides: nested map (multiple definitions)",
		filename: "testdata/goto-overrides.jsonnet",
		position: protocol.Position{Line: 33, Character: 34},
		results: []definitionResult{
			{
				targetRange: protocol.Range{
					Start: protocol.Position{Line: 25, Character: 4},
					End:   protocol.Position{Line: 27, Character: 5},
				},
				targetSelectionRange: protocol.Range{
					Start: protocol.Position{Line: 25, Character: 4},
					End:   protocol.Position{Line: 25, Character: 11},
				},
			},
			{
				targetRange: protocol.Range{
					Start: protocol.Position{Line: 16, Character: 4},
					End:   protocol.Position{Line: 18, Character: 5},
				},
				targetSelectionRange: protocol.Range{
					Start: protocol.Position{Line: 16, Character: 4},
					End:   protocol.Position{Line: 16, Character: 11},
				},
			},
			{
				targetRange: protocol.Range{
					Start: protocol.Position{Line: 4, Character: 4},
					End:   protocol.Position{Line: 6, Character: 5},
				},
				targetSelectionRange: protocol.Range{
					Start: protocol.Position{Line: 4, Character: 4},
					End:   protocol.Position{Line: 4, Character: 11},
				},
			},
			{
				targetFilename: "testdata/goto-overrides-imported2.jsonnet",
				targetRange: protocol.Range{
					Start: protocol.Position{Line: 1, Character: 2},
					End:   protocol.Position{Line: 3, Character: 3},
				},
				targetSelectionRange: protocol.Range{
					Start: protocol.Position{Line: 1, Character: 2},
					End:   protocol.Position{Line: 1, Character: 9},
				},
			},
			{
				targetFilename: "testdata/goto-overrides-imported.jsonnet",
				targetRange: protocol.Range{
					Start: protocol.Position{Line: 1, Character: 2},
					End:   protocol.Position{Line: 3, Character: 3},
				},
				targetSelectionRange: protocol.Range{
					Start: protocol.Position{Line: 1, Character: 2},
					End:   protocol.Position{Line: 1, Character: 9},
				},
			},
			{
				targetFilename: "testdata/goto-overrides-base.jsonnet",
				targetRange: protocol.Range{
					Start: protocol.Position{Line: 12, Character: 4},
					End:   protocol.Position{Line: 14, Character: 5},
				},
				targetSelectionRange: protocol.Range{
					Start: protocol.Position{Line: 12, Character: 4},
					End:   protocol.Position{Line: 12, Character: 11},
				},
			},
			{
				targetFilename: "testdata/goto-overrides-base.jsonnet",
				targetRange: protocol.Range{
					Start: protocol.Position{Line: 3, Character: 4},
					End:   protocol.Position{Line: 5, Character: 5},
				},
				targetSelectionRange: protocol.Range{
					Start: protocol.Position{Line: 3, Character: 4},
					End:   protocol.Position{Line: 3, Character: 11},
				},
			},
		},
	},
	{
		name:     "goto with overrides: string carried from super",
		filename: "testdata/goto-overrides.jsonnet",
		position: protocol.Position{Line: 35, Character: 27},
		results: []definitionResult{{
			targetRange: protocol.Range{
				Start: protocol.Position{Line: 3, Character: 4},
				End:   protocol.Position{Line: 3, Character: 18},
			},
			targetSelectionRange: protocol.Range{
				Start: protocol.Position{Line: 3, Character: 4},
				End:   protocol.Position{Line: 3, Character: 9},
			},
		}},
	},
	{
		name:     "goto with overrides: nested string carried from super",
		filename: "testdata/goto-overrides.jsonnet",
		position: protocol.Position{Line: 36, Character: 44},
		results: []definitionResult{{
			targetRange: protocol.Range{
				Start: protocol.Position{Line: 17, Character: 6},
				End:   protocol.Position{Line: 17, Character: 22},
			},
			targetSelectionRange: protocol.Range{
				Start: protocol.Position{Line: 17, Character: 6},
				End:   protocol.Position{Line: 17, Character: 12},
			},
		}},
	},
	{
		name:     "goto with overrides: string carried from local",
		filename: "testdata/goto-overrides.jsonnet",
		position: protocol.Position{Line: 37, Character: 57},
		results: []definitionResult{{
			targetFilename: "testdata/goto-overrides-base.jsonnet",
			targetRange: protocol.Range{
				Start: protocol.Position{Line: 13, Character: 6},
				End:   protocol.Position{Line: 13, Character: 24},
			},
			targetSelectionRange: protocol.Range{
				Start: protocol.Position{Line: 13, Character: 6},
				End:   protocol.Position{Line: 13, Character: 16},
			},
		}},
	},
	{
		name:     "goto with overrides: string carried from import",
		filename: "testdata/goto-overrides.jsonnet",
		position: protocol.Position{Line: 38, Character: 57},
		results: []definitionResult{{
			targetFilename: "testdata/goto-overrides-imported.jsonnet",
			targetRange: protocol.Range{
				Start: protocol.Position{Line: 2, Character: 4},
				End:   protocol.Position{Line: 2, Character: 23},
			},
			targetSelectionRange: protocol.Range{
				Start: protocol.Position{Line: 2, Character: 4},
				End:   protocol.Position{Line: 2, Character: 15},
			},
		}},
	},
	{
		name:     "goto with overrides: string carried from second import",
		filename: "testdata/goto-overrides.jsonnet",
		position: protocol.Position{Line: 39, Character: 67},
		results: []definitionResult{{
			targetFilename: "testdata/goto-overrides-imported2.jsonnet",
			targetRange: protocol.Range{
				Start: protocol.Position{Line: 2, Character: 4},
				End:   protocol.Position{Line: 2, Character: 30},
			},
			targetSelectionRange: protocol.Range{
				Start: protocol.Position{Line: 2, Character: 4},
				End:   protocol.Position{Line: 2, Character: 22},
			},
		}},
	},
	{
		name:     "goto deeply nested imported attribute",
		filename: "testdata/goto-import-nested-main.jsonnet",
		position: protocol.Position{Line: 6, Character: 14},
		results: []definitionResult{{
			targetFilename: "testdata/goto-import-nested-obj.libsonnet",
			targetRange: protocol.Range{
				Start: protocol.Position{Line: 2, Character: 2},
				End:   protocol.Position{Line: 2, Character: 26},
			},
			targetSelectionRange: protocol.Range{
				Start: protocol.Position{Line: 2, Character: 2},
				End:   protocol.Position{Line: 2, Character: 15},
			},
		}},
	},
	{
		name:     "goto deeply nested imported attribute through self",
		filename: "testdata/goto-import-nested-main.jsonnet",
		position: protocol.Position{Line: 7, Character: 27},
		results: []definitionResult{{
			targetFilename: "testdata/goto-import-nested-obj.libsonnet",
			targetRange: protocol.Range{
				Start: protocol.Position{Line: 2, Character: 2},
				End:   protocol.Position{Line: 2, Character: 26},
			},
			targetSelectionRange: protocol.Range{
				Start: protocol.Position{Line: 2, Character: 2},
				End:   protocol.Position{Line: 2, Character: 15},
			},
		}},
	},
	{
		name:     "goto computed field name object field",
		filename: "testdata/goto-computed-field-names.jsonnet",
		position: protocol.Position{Line: 3, Character: 9},
		results: []definitionResult{{
			targetFilename: "testdata/goto-computed-field-names.jsonnet",
			targetRange: protocol.Range{
				Start: protocol.Position{Line: 0, Character: 14},
				End:   protocol.Position{Line: 0, Character: 26},
			},
			targetSelectionRange: protocol.Range{
				Start: protocol.Position{Line: 0, Character: 14},
				End:   protocol.Position{Line: 0, Character: 17},
			},
		}},
	},
	{
		name:     "goto field through function",
		filename: "testdata/goto-functions-advanced.libsonnet",
		position: protocol.Position{Line: 15, Character: 48},
		results: []definitionResult{{
			targetRange: protocol.Range{
				Start: protocol.Position{Line: 1, Character: 2},
				End:   protocol.Position{Line: 1, Character: 12},
			},
			targetSelectionRange: protocol.Range{
				Start: protocol.Position{Line: 1, Character: 2},
				End:   protocol.Position{Line: 1, Character: 6},
			},
		}},
	},
	{
		name:     "goto field through function-created object",
		filename: "testdata/goto-functions-advanced.libsonnet",
		position: protocol.Position{Line: 16, Character: 54},
		results: []definitionResult{{
			targetRange: protocol.Range{
				Start: protocol.Position{Line: 2, Character: 2},
				End:   protocol.Position{Line: 2, Character: 12},
			},
			targetSelectionRange: protocol.Range{
				Start: protocol.Position{Line: 2, Character: 2},
				End:   protocol.Position{Line: 2, Character: 6},
			},
		}},
	},
	{
		name:     "goto field through builder pattern attribute",
		filename: "testdata/goto-functions-advanced.libsonnet",
		position: protocol.Position{Line: 17, Character: 67},
		results: []definitionResult{{
			targetRange: protocol.Range{
				Start: protocol.Position{Line: 5, Character: 4},
				End:   protocol.Position{Line: 5, Character: 14},
			},
			targetSelectionRange: protocol.Range{
				Start: protocol.Position{Line: 5, Character: 4},
				End:   protocol.Position{Line: 5, Character: 8},
			},
		}},
	},
	{
		name:     "goto field through mixin attribute",
		filename: "testdata/goto-functions-advanced.libsonnet",
		position: protocol.Position{Line: 18, Character: 58},
		results: []definitionResult{{
			targetRange: protocol.Range{
				Start: protocol.Position{Line: 11, Character: 4},
				End:   protocol.Position{Line: 11, Character: 14},
			},
			targetSelectionRange: protocol.Range{
				Start: protocol.Position{Line: 11, Character: 4},
				End:   protocol.Position{Line: 11, Character: 8},
			},
		}},
	},
	{
		name:     "goto attribute of root-function library",
		filename: "testdata/goto-root-function.jsonnet",
		position: protocol.Position{Line: 5, Character: 70},
		results: []definitionResult{{
			targetFilename: "testdata/goto-root-function-lib.libsonnet",
			targetRange: protocol.Range{
				Start: protocol.Position{Line: 1, Character: 2},
				End:   protocol.Position{Line: 1, Character: 22},
			},
			targetSelectionRange: protocol.Range{
				Start: protocol.Position{Line: 1, Character: 2},
				End:   protocol.Position{Line: 1, Character: 11},
			},
		}},
	},
	{
		name:     "goto attribute of root-function library through local import",
		filename: "testdata/goto-root-function.jsonnet",
		position: protocol.Position{Line: 6, Character: 28},
		results: []definitionResult{{
			targetFilename: "testdata/goto-root-function-lib.libsonnet",
			targetRange: protocol.Range{
				Start: protocol.Position{Line: 1, Character: 2},
				End:   protocol.Position{Line: 1, Character: 22},
			},
			targetSelectionRange: protocol.Range{
				Start: protocol.Position{Line: 1, Character: 2},
				End:   protocol.Position{Line: 1, Character: 11},
			},
		}},
	},
	{
		name:     "goto attribute of root-function library through local resolved import",
		filename: "testdata/goto-root-function.jsonnet",
		position: protocol.Position{Line: 7, Character: 36},
		results: []definitionResult{{
			targetFilename: "testdata/goto-root-function-lib.libsonnet",
			targetRange: protocol.Range{
				Start: protocol.Position{Line: 1, Character: 2},
				End:   protocol.Position{Line: 1, Character: 22},
			},
			targetSelectionRange: protocol.Range{
				Start: protocol.Position{Line: 1, Character: 2},
				End:   protocol.Position{Line: 1, Character: 11},
			},
		}},
	},
	{
		name:     "goto nested attribute of root-function library",
		filename: "testdata/goto-root-function.jsonnet",
		position: protocol.Position{Line: 9, Character: 98},
		results: []definitionResult{{
			targetFilename: "testdata/goto-root-function-lib.libsonnet",
			targetRange: protocol.Range{
				Start: protocol.Position{Line: 4, Character: 4},
				End:   protocol.Position{Line: 4, Character: 36},
			},
			targetSelectionRange: protocol.Range{
				Start: protocol.Position{Line: 4, Character: 4},
				End:   protocol.Position{Line: 4, Character: 19},
			},
		}},
	},
	{
		name:     "goto nested attribute of root-function library through local import",
		filename: "testdata/goto-root-function.jsonnet",
		position: protocol.Position{Line: 10, Character: 55},
		results: []definitionResult{{
			targetFilename: "testdata/goto-root-function-lib.libsonnet",
			targetRange: protocol.Range{
				Start: protocol.Position{Line: 4, Character: 4},
				End:   protocol.Position{Line: 4, Character: 36},
			},
			targetSelectionRange: protocol.Range{
				Start: protocol.Position{Line: 4, Character: 4},
				End:   protocol.Position{Line: 4, Character: 19},
			},
		}},
	},
	{
		name:     "goto nested attribute of root-function library through local resolved import",
		filename: "testdata/goto-root-function.jsonnet",
		position: protocol.Position{Line: 11, Character: 64},
		results: []definitionResult{{
			targetFilename: "testdata/goto-root-function-lib.libsonnet",
			targetRange: protocol.Range{
				Start: protocol.Position{Line: 4, Character: 4},
				End:   protocol.Position{Line: 4, Character: 36},
			},
			targetSelectionRange: protocol.Range{
				Start: protocol.Position{Line: 4, Character: 4},
				End:   protocol.Position{Line: 4, Character: 19},
			},
		}},
	},
	{
		name:     "goto function nested multiple times in different ways",
		filename: "testdata/goto-multilevel-library-main.jsonnet",
		position: protocol.Position{Line: 2, Character: 34},
		results: []definitionResult{{
			targetFilename: "testdata/goto-multilevel-library-sub-1.1.libsonnet",
			targetRange: protocol.Range{
				Start: protocol.Position{Line: 1, Character: 2},
				End:   protocol.Position{Line: 1, Character: 28},
			},
			targetSelectionRange: protocol.Range{
				Start: protocol.Position{Line: 1, Character: 2},
				End:   protocol.Position{Line: 1, Character: 5},
			},
		}},
	},
	{
		name:     "test fix infinite recursion",
		filename: "./testdata/goto-infinite-recursion-bug-1.jsonnet",
		position: protocol.Position{Line: 2, Character: 26},
		results: []definitionResult{{
			targetFilename: "testdata/goto-infinite-recursion-bug-3.libsonnet",
			targetRange: protocol.Range{
				Start: protocol.Position{Line: 5, Character: 10},
				End:   protocol.Position{Line: 5, Character: 36},
			},
			targetSelectionRange: protocol.Range{
				Start: protocol.Position{Line: 5, Character: 10},
				End:   protocol.Position{Line: 5, Character: 22},
			},
		}},
	},
	{
		name:     "grafonnet: eval variable selection options",
		filename: "./testdata/grafonnet/eval-variable-options.jsonnet",
		position: protocol.Position{Line: 5, Character: 54},
		results: []definitionResult{{
			targetFilename: "testdata/grafonnet/vendor/github.com/grafana/grafonnet/gen/grafonnet-v10.0.0/custom/dashboard/variable.libsonnet",
			targetRange: protocol.Range{
				Start: protocol.Position{Line: 101, Character: 12},
				End:   protocol.Position{Line: 103, Character: 13},
			},
			targetSelectionRange: protocol.Range{
				Start: protocol.Position{Line: 101, Character: 12},
				End:   protocol.Position{Line: 101, Character: 21},
			},
		}},
	},
}

func TestDefinition(t *testing.T) {
	for _, tc := range definitionTestCases {
		t.Run(tc.name, func(t *testing.T) {
			params := &protocol.DefinitionParams{
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
			response, err := server.definitionLink(params)
			require.NoError(t, err)

			var expected []protocol.DefinitionLink
			for _, r := range tc.results {
				// Defaults
				if r.targetSelectionRange.End.Character == 0 {
					r.targetSelectionRange = r.targetRange
				}
				if r.targetFilename == "" {
					r.targetFilename = tc.filename
				}
				expected = append(expected, protocol.DefinitionLink{
					TargetURI:            absURI(t, r.targetFilename),
					TargetRange:          r.targetRange,
					TargetSelectionRange: r.targetSelectionRange,
				})
			}

			assert.Equal(t, expected, response)
		})
	}
}

func BenchmarkDefinition(b *testing.B) {
	for _, tc := range definitionTestCases {
		b.Run(tc.name, func(b *testing.B) {
			params := &protocol.DefinitionParams{
				TextDocumentPositionParams: protocol.TextDocumentPositionParams{
					TextDocument: protocol.TextDocumentIdentifier{
						URI: protocol.URIFromPath(tc.filename),
					},
					Position: tc.position,
				},
			}
			server := NewServer("any", "test version", nil, Configuration{
				JPaths: []string{"testdata"},
			})
			serverOpenTestFile(b, server, tc.filename)

			for i := 0; i < b.N; i++ {
				// We don't care about the response for the benchmark
				// nolint:errcheck
				server.definitionLink(params)
			}
		})
	}
}

func TestDefinitionFail(t *testing.T) {
	testCases := []struct {
		name     string
		filename string
		position protocol.Position
		expected error
	}{
		{
			name:     "goto local keyword fails",
			filename: "testdata/goto-basic-object.jsonnet",
			position: protocol.Position{Line: 0, Character: 3},
			expected: fmt.Errorf("cannot find definition"),
		},

		{
			name:     "goto index of std fails",
			filename: "testdata/goto-std.jsonnet",
			position: protocol.Position{Line: 1, Character: 20},
			expected: fmt.Errorf("cannot get definition of std lib"),
		},
		{
			name:     "goto comment fails",
			filename: "testdata/goto-comment.jsonnet",
			position: protocol.Position{Line: 0, Character: 1},
			expected: fmt.Errorf("cannot find definition"),
		},
		{
			name:     "goto super fails as no LHS object exists",
			filename: "testdata/goto-local-function.libsonnet",
			position: protocol.Position{Line: 33, Character: 23},
			expected: fmt.Errorf("could not find a lhs object"),
		},
		{
			name:     "goto self fails when out of scope",
			filename: "testdata/goto-self-complex-scoping.jsonnet",
			position: protocol.Position{Line: 3, Character: 18},
			expected: fmt.Errorf("field test was not found in ast.DesugaredObject"),
		},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			params := &protocol.DefinitionParams{
				TextDocumentPositionParams: protocol.TextDocumentPositionParams{
					TextDocument: protocol.TextDocumentIdentifier{
						URI: protocol.URIFromPath(tc.filename),
					},
					Position: tc.position,
				},
			}

			server := NewServer("any", "test version", nil, Configuration{
				JPaths: []string{"testdata"},
			})
			serverOpenTestFile(t, server, tc.filename)
			got, err := server.definitionLink(params)

			require.Error(t, err)
			assert.Equal(t, tc.expected.Error(), err.Error())
			assert.Nil(t, got)
		})
	}
}
