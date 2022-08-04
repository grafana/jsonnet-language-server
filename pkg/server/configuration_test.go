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
			if tc.expectedErr == nil && err != nil {
				t.Fatalf("DidChangeConfiguration produced unexpected error: %v", err)
			} else if tc.expectedErr != nil && err == nil {
				t.Fatalf("expected DidChangeConfiguration to produce error but it did not")
			} else if tc.expectedErr != nil && err != nil {
				assert.EqualError(t, err, tc.expectedErr.Error())
				return
			}

			vm, err := s.getVM("any")
			assert.NoError(t, err)

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
		name            string
		settings        interface{}
		expectedOptions formatter.Options
		expectedErr     error
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
			expectedOptions: func() formatter.Options {
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
			name: "does not override default values",
			settings: map[string]interface{}{
				"formatting": map[string]interface{}{},
			},
			expectedOptions: formatter.DefaultOptions(),
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
			if tc.expectedErr == nil && err != nil {
				t.Fatalf("DidChangeConfiguration produced unexpected error: %v", err)
			} else if tc.expectedErr != nil && err == nil {
				t.Fatalf("expected DidChangeConfiguration to produce error but it did not")
			} else if tc.expectedErr != nil && err != nil {
				assert.EqualError(t, err, tc.expectedErr.Error())
				return
			}

			assert.Equal(t, tc.expectedOptions, s.fmtOpts)
		})
	}
}
