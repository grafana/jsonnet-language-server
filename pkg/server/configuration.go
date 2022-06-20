package server

import (
	"context"
	"fmt"
	"math"
	"reflect"

	"github.com/google/go-jsonnet"
	"github.com/google/go-jsonnet/formatter"
	"github.com/jdbaldry/go-language-server-protocol/jsonrpc2"
	"github.com/jdbaldry/go-language-server-protocol/lsp/protocol"
)

func (s *server) DidChangeConfiguration(ctx context.Context, params *protocol.DidChangeConfigurationParams) error {
	settingsMap, ok := params.Settings.(map[string]interface{})
	if !ok {
		return fmt.Errorf("%w: unsupported settings payload. expected json object, got: %T", jsonrpc2.ErrInvalidParams, params.Settings)
	}

	for sk, sv := range settingsMap {
		switch sk {
		case "ext_vars":
			newVars, err := s.parseExtVars(sv)
			if err != nil {
				return fmt.Errorf("%w: ext_vars parsing failed: %v", jsonrpc2.ErrInvalidParams, err)
			}
			s.extVars = newVars

		case "formatting":
			newFmtOpts, err := s.parseFormattingOpts(sv)
			if err != nil {
				return fmt.Errorf("%w: formatting options parsing failed: %v", jsonrpc2.ErrInvalidParams, err)
			}
			s.fmtOpts = newFmtOpts

		default:
			return fmt.Errorf("%w: unsupported settings key: %q", jsonrpc2.ErrInvalidParams, sk)
		}
	}
	return nil
}

func (s *server) parseExtVars(unparsed interface{}) (map[string]string, error) {
	newVars, ok := unparsed.(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("unsupported settings value for ext_vars. expected json object. got: %T", unparsed)
	}

	extVars := make(map[string]string, len(newVars))
	for varKey, varValue := range newVars {
		vv, ok := varValue.(string)
		if !ok {
			return nil, fmt.Errorf("unsupported settings value for ext_vars.%s. expected string. got: %T", varKey, varValue)
		}
		extVars[varKey] = vv
	}
	return extVars, nil
}

func (s *server) parseFormattingOpts(unparsed interface{}) (formatter.Options, error) {
	newOpts, ok := unparsed.(map[string]interface{})
	if !ok {
		return formatter.Options{}, fmt.Errorf("unsupported settings value for formatting. expected json object. got: %T", unparsed)
	}

	opts := formatter.DefaultOptions()
	optsTyp, optsVal := reflect.TypeOf(opts), reflect.ValueOf(&opts).Elem()
	for optName, unparsedValue := range newOpts {
		field, ok := optsTyp.FieldByName(optName)
		if !ok {
			return opts, fmt.Errorf("unknown option: %q", optName)
		}

		var err error
		switch field.Type {
		case reflect.TypeOf(int(0)):
			dest := optsVal.FieldByIndex(field.Index).Addr().Interface().(*int)
			err = assignInt(dest, unparsedValue)

		case reflect.TypeOf(false):
			dest := optsVal.FieldByIndex(field.Index).Addr().Interface().(*bool)
			err = assignBool(dest, unparsedValue)

		case reflect.TypeOf(formatter.StringStyleDouble):
			dest := optsVal.FieldByIndex(field.Index).Addr().Interface().(*formatter.StringStyle)
			err = assignStringStyle(dest, unparsedValue)

		case reflect.TypeOf(formatter.CommentStyleHash):
			dest := optsVal.FieldByIndex(field.Index).Addr().Interface().(*formatter.CommentStyle)
			err = assignCommentStyle(dest, unparsedValue)

		default:
			err = fmt.Errorf("unknown field type: %v", field.Type)
		}
		if err != nil {
			return opts, fmt.Errorf("%s: %v", optName, err)
		}

		// switch optName {
		// case "indent":
		// 	if err := assignInt(&opts.Indent, unparsedValue); err != nil {
		// 		return opts, fmt.Errorf("indent: %v", err)
		// 	}
		// case "max_blank_lines":
		// 	if err := assignInt(&opts.MaxBlankLines, unparsedValue); err != nil {
		// 		return opts, fmt.Errorf("max_blank_lines: %v", err)
		// 	}
		// case "string_style":
		// 	if err := assignStringStyle(&opts.StringStyle, unparsedValue); err != nil {
		// 		return opts, fmt.Errorf("string_style: %v", err)
		// 	}
		// case "comment_style":
		// 	if err := assignCommentStyle(&opts.CommentStyle, unparsedValue); err != nil {
		// 		return opts, fmt.Errorf("comment_style: %v", err)
		// 	}
		// case "pretty_field_names":
		// 	if err := assignBool(&opts.PrettyFieldNames, unparsedValue); err != nil {
		// 		return opts, fmt.Errorf("pretty_field_names: %v", err)
		// 	}
		// case "pad_arrays":
		// 	if err := assignBool(&opts.PadArrays, unparsedValue); err != nil {
		// 		return opts, fmt.Errorf("pad_arrays: %v", err)
		// 	}

		// default:
		// 	return opts, fmt.Errorf("unknown option: %q", optName)
		// }
	}
	return opts, nil
}

func resetExtVars(vm *jsonnet.VM, vars map[string]string) {
	vm.ExtReset()
	for vk, vv := range vars {
		vm.ExtVar(vk, vv)
	}
}

func assignBool(dest *bool, unparsed interface{}) error {
	switch unparsed {
	case true:
		*dest = true
	case false:
		*dest = false
	default:
		return fmt.Errorf("expected bool, got: %T", unparsed)
	}
	return nil
}

func assignInt(dest *int, unparsed interface{}) error {
	switch v := unparsed.(type) {
	case int:
		*dest = v
	case float64:
		*dest = int(math.Floor(v))
	default:
		return fmt.Errorf("expected int or float, got: %T", unparsed)
	}
	return nil
}

func assignCommentStyle(dest *formatter.CommentStyle, unparsed interface{}) error {
	str, ok := unparsed.(string)
	if !ok {
		return fmt.Errorf("expected string, got: %T", unparsed)
	}
	switch str {
	case "hash":
		*dest = formatter.CommentStyleHash
	case "slash":
		*dest = formatter.CommentStyleSlash
	case "leave":
		*dest = formatter.CommentStyleLeave
	default:
		return fmt.Errorf("expected one of 'hash', 'slash', 'leave', got: %q", str)
	}
	return nil
}

func assignStringStyle(dest *formatter.StringStyle, unparsed interface{}) error {
	str, ok := unparsed.(string)
	if !ok {
		return fmt.Errorf("expected string, got: %T", unparsed)
	}
	switch str {
	case "double":
		*dest = formatter.StringStyleDouble
	case "single":
		*dest = formatter.StringStyleSingle
	case "leave":
		*dest = formatter.StringStyleLeave
	default:
		return fmt.Errorf("unknown string_style: expected one of 'double', 'single', 'leave', got: %q", str)
	}
	return nil
}
