package server

import (
	"context"
	"errors"
	"testing"

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
