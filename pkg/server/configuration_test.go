package server

import (
	"context"
	"errors"
	"testing"

	"github.com/google/go-jsonnet/formatter"
	"github.com/jdbaldry/go-language-server-protocol/lsp/protocol"
	"github.com/stretchr/testify/assert"
)

func TestConfiguration(t *testing.T) {
	type kase struct {
		name        string
		settings    interface{}
		fileContent string

		expectedErr        error
		expectedFileOutput string
	}

	testCases := []kase{
		{
			name:        "settings is not an object",
			settings:    []string{""},
			fileContent: `[]`,
			expectedErr: errors.New("JSON RPC invalid params: unsupported settings payload. expected json object, got: []string"),
		},
		{
			name: "settings has unsupported key",
			settings: map[string]interface{}{
				"foo_bar": map[string]interface{}{},
			},
			fileContent: `[]`,
			expectedErr: errors.New("JSON RPC invalid params: unsupported settings key: \"foo_bar\""),
		},
		{
			name: "ext_var config is empty",
			settings: map[string]interface{}{
				"ext_vars": map[string]interface{}{},
			},
			fileContent:        `[]`,
			expectedFileOutput: `[]`,
		},
		{
			name:               "ext_var config is missing",
			settings:           map[string]interface{}{},
			fileContent:        `[]`,
			expectedFileOutput: `[]`,
		},
		{
			name: "ext_var config is not an object",
			settings: map[string]interface{}{
				"ext_vars": []string{},
			},
			fileContent: `[]`,
			expectedErr: errors.New("JSON RPC invalid params: ext_vars parsing failed: unsupported settings value for ext_vars. expected json object. got: []string"),
		},
		{
			name: "ext_var config value is not a string",
			settings: map[string]interface{}{
				"ext_vars": map[string]interface{}{
					"foo": true,
				},
			},
			fileContent: `[]`,
			expectedErr: errors.New("JSON RPC invalid params: ext_vars parsing failed: unsupported settings value for ext_vars.foo. expected string. got: bool"),
		},
		{
			name: "ext_var config is valid",
			settings: map[string]interface{}{
				"ext_vars": map[string]interface{}{
					"hello": "world",
				},
			},
			fileContent: `
{
	hello: std.extVar("hello"),
}
			`,
			expectedFileOutput: `
{
	"hello": "world"
}
			`,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			s, fileURI := testServerWithFile(t, nil, tc.fileContent)

			err := s.DidChangeConfiguration(
				context.TODO(),
				&protocol.DidChangeConfigurationParams{
					Settings: tc.settings,
				},
			)
			if tc.expectedErr != nil {
				assert.EqualError(t, err, tc.expectedErr.Error())
				return
			}
			assert.NoError(t, err)

			vm := s.getVM("any")

			doc, err := s.cache.get(fileURI)
			assert.NoError(t, err)

			json, err := vm.Evaluate(doc.ast)
			assert.NoError(t, err)
			assert.JSONEq(t, tc.expectedFileOutput, json)
		})
	}
}

func TestConfiguration_Formatting(t *testing.T) {
	type kase struct {
		name                  string
		settings              interface{}
		expectedConfiguration Configuration
		expectedErr           error
	}

	testCases := []kase{
		{
			name: "formatting opts",
			settings: map[string]interface{}{
				"formatting": map[string]interface{}{
					"Indent":           4,
					"MaxBlankLines":    10,
					"StringStyle":      "single",
					"CommentStyle":     "leave",
					"PrettyFieldNames": true,
					"PadArrays":        false,
					"PadObjects":       true,
					"SortImports":      false,
					"UseImplicitPlus":  true,
					"StripEverything":  false,
					"StripComments":    false,
					// not setting StripAllButComments
				},
			},
			expectedConfiguration: Configuration{
				FormattingOptions: func() formatter.Options {
					opts := formatter.DefaultOptions()
					opts.Indent = 4
					opts.MaxBlankLines = 10
					opts.StringStyle = formatter.StringStyleSingle
					opts.CommentStyle = formatter.CommentStyleLeave
					opts.PrettyFieldNames = true
					opts.PadArrays = false
					opts.PadObjects = true
					opts.SortImports = false
					opts.UseImplicitPlus = true
					opts.StripEverything = false
					opts.StripComments = false
					return opts
				}(),
			},
		},
		{
			name: "invalid string style",
			settings: map[string]interface{}{
				"formatting": map[string]interface{}{
					"StringStyle": "invalid",
				},
			},
			expectedErr: errors.New("JSON RPC invalid params: formatting options parsing failed: map decode failed: 1 error(s) decoding:\n\n* error decoding 'StringStyle': expected one of 'double', 'single', 'leave', got: \"invalid\""),
		},
		{
			name: "invalid comment style",
			settings: map[string]interface{}{
				"formatting": map[string]interface{}{
					"CommentStyle": "invalid",
				},
			},
			expectedErr: errors.New("JSON RPC invalid params: formatting options parsing failed: map decode failed: 1 error(s) decoding:\n\n* error decoding 'CommentStyle': expected one of 'hash', 'slash', 'leave', got: \"invalid\""),
		},
		{
			name: "invalid comment style type",
			settings: map[string]interface{}{
				"formatting": map[string]interface{}{
					"CommentStyle": 123,
				},
			},
			expectedErr: errors.New("JSON RPC invalid params: formatting options parsing failed: map decode failed: 1 error(s) decoding:\n\n* error decoding 'CommentStyle': expected string, got: int"),
		},
		{
			name: "does not override default values",
			settings: map[string]interface{}{
				"formatting": map[string]interface{}{},
			},
			expectedConfiguration: Configuration{FormattingOptions: formatter.DefaultOptions()},
		},
		{
			name: "invalid jpath type",
			settings: map[string]interface{}{
				"jpath": 123,
			},
			expectedErr: errors.New("JSON RPC invalid params: unsupported settings value for jpath. expected array of strings. got: int"),
		},
		{
			name: "invalid jpath item type",
			settings: map[string]interface{}{
				"jpath": []interface{}{123},
			},
			expectedErr: errors.New("JSON RPC invalid params: unsupported settings value for jpath. expected string. got: int"),
		},
		{
			name: "invalid bool",
			settings: map[string]interface{}{
				"resolve_paths_with_tanka": "true",
			},
			expectedErr: errors.New("JSON RPC invalid params: unsupported settings value for resolve_paths_with_tanka. expected boolean. got: string"),
		},
		{
			name: "invalid log level",
			settings: map[string]interface{}{
				"log_level": "bad",
			},
			expectedErr: errors.New(`JSON RPC invalid params: not a valid logrus Level: "bad"`),
		},
		{
			name: "all settings",
			settings: map[string]interface{}{
				"log_level": "error",
				"formatting": map[string]interface{}{
					"Indent":              4,
					"MaxBlankLines":       10,
					"StringStyle":         "double",
					"CommentStyle":        "slash",
					"PrettyFieldNames":    false,
					"PadArrays":           true,
					"PadObjects":          false,
					"SortImports":         false,
					"UseImplicitPlus":     false,
					"StripEverything":     true,
					"StripComments":       true,
					"StripAllButComments": true,
				},
				"ext_vars": map[string]interface{}{
					"hello": "world",
				},
				"resolve_paths_with_tanka": false,
				"jpath":                    []interface{}{"blabla", "blabla2"},
				"enable_eval_diagnostics":  false,
				"enable_lint_diagnostics":  true,
			},
			expectedConfiguration: Configuration{
				FormattingOptions: func() formatter.Options {
					opts := formatter.DefaultOptions()
					opts.Indent = 4
					opts.MaxBlankLines = 10
					opts.StringStyle = formatter.StringStyleDouble
					opts.CommentStyle = formatter.CommentStyleSlash
					opts.PrettyFieldNames = false
					opts.PadArrays = true
					opts.PadObjects = false
					opts.SortImports = false
					opts.UseImplicitPlus = false
					opts.StripEverything = true
					opts.StripComments = true
					opts.StripAllButComments = true
					return opts
				}(),
				ExtVars: map[string]string{
					"hello": "world",
				},
				ResolvePathsWithTanka: false,
				JPaths:                []string{"blabla", "blabla2"},
				EnableEvalDiagnostics: false,
				EnableLintDiagnostics: true,
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			s, _ := testServerWithFile(t, nil, "")

			err := s.DidChangeConfiguration(
				context.TODO(),
				&protocol.DidChangeConfigurationParams{
					Settings: tc.settings,
				},
			)
			if tc.expectedErr != nil {
				assert.EqualError(t, err, tc.expectedErr.Error())
				return
			}
			assert.NoError(t, err)

			assert.Equal(t, tc.expectedConfiguration, s.configuration)
		})
	}
}
