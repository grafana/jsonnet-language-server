package server

import (
	"context"
	"fmt"
	"reflect"
	"strconv"
	"strings"

	"github.com/google/go-jsonnet"
	"github.com/google/go-jsonnet/ast"
	"github.com/google/go-jsonnet/formatter"
	tankaNative "github.com/grafana/tanka/pkg/jsonnet/native"
	"github.com/jdbaldry/go-language-server-protocol/jsonrpc2"
	"github.com/jdbaldry/go-language-server-protocol/lsp/protocol"
	"github.com/mitchellh/mapstructure"
	log "github.com/sirupsen/logrus"
)

type Configuration struct {
	ResolvePathsWithTanka bool
	JPaths                []string
	ExtVars               map[string]string
	ExtCode               map[string]string
	// NativeFunctions lists the native functions the host application registers
	// at runtime, as a map of function name to parameter count. They are
	// registered as stubs that return null so that evaluation diagnostics do
	// not fail on std.native calls.
	NativeFunctions   map[string]int
	FormattingOptions formatter.Options

	EnableEvalDiagnostics     bool
	EnableLintDiagnostics     bool
	ShowDocstringInCompletion bool

	// EvalBinary is the external command for evaluation (e.g. "jsonnet"). When non-empty, jsonnet.evalFile and jsonnet.evalExpression use it instead of the in-process VM.
	EvalBinary string
}

func (s *Server) DidChangeConfiguration(_ context.Context, params *protocol.DidChangeConfigurationParams) error {
	settingsMap, ok := params.Settings.(map[string]interface{})
	if !ok {
		return fmt.Errorf("%w: unsupported settings payload. expected json object, got: %T", jsonrpc2.ErrInvalidParams, params.Settings)
	}

	for sk, sv := range settingsMap {
		switch sk {
		case "log_level":
			svStr, ok := sv.(string)
			if !ok {
				return fmt.Errorf("%w: unsupported settings value for log_level. expected string. got: %T", jsonrpc2.ErrInvalidParams, sv)
			}

			level, err := log.ParseLevel(svStr)
			if err != nil {
				return fmt.Errorf("%w: %v", jsonrpc2.ErrInvalidParams, err)
			}
			log.SetLevel(level)
		case "resolve_paths_with_tanka":
			if boolVal, ok := sv.(bool); ok {
				s.configuration.ResolvePathsWithTanka = boolVal
			} else {
				return fmt.Errorf("%w: unsupported settings value for resolve_paths_with_tanka. expected boolean. got: %T", jsonrpc2.ErrInvalidParams, sv)
			}
		case "jpath":
			if svList, ok := sv.([]interface{}); ok {
				s.configuration.JPaths = make([]string, len(svList))
				for i, v := range svList {
					if strVal, ok := v.(string); ok {
						s.configuration.JPaths[i] = strVal
					} else {
						return fmt.Errorf("%w: unsupported settings value for jpath. expected string. got: %T", jsonrpc2.ErrInvalidParams, v)
					}
				}
			} else {
				return fmt.Errorf("%w: unsupported settings value for jpath. expected array of strings. got: %T", jsonrpc2.ErrInvalidParams, sv)
			}

		case "enable_eval_diagnostics":
			if boolVal, ok := sv.(bool); ok {
				s.configuration.EnableEvalDiagnostics = boolVal
			} else {
				return fmt.Errorf("%w: unsupported settings value for enable_eval_diagnostics. expected boolean. got: %T", jsonrpc2.ErrInvalidParams, sv)
			}
		case "enable_lint_diagnostics":
			if boolVal, ok := sv.(bool); ok {
				s.configuration.EnableLintDiagnostics = boolVal
			} else {
				return fmt.Errorf("%w: unsupported settings value for enable_lint_diagnostics. expected boolean. got: %T", jsonrpc2.ErrInvalidParams, sv)
			}
		case "show_docstring_in_completion":
			if boolVal, ok := sv.(bool); ok {
				s.configuration.ShowDocstringInCompletion = boolVal
			} else {
				return fmt.Errorf("%w: unsupported settings value for show_docstring_in_completion. expected boolean. got: %T", jsonrpc2.ErrInvalidParams, sv)
			}
		case "ext_vars":
			newVars, err := s.parseExtVars(sv)
			if err != nil {
				return fmt.Errorf("%w: ext_vars parsing failed: %v", jsonrpc2.ErrInvalidParams, err)
			}
			s.configuration.ExtVars = newVars
		case "formatting":
			newFmtOpts, err := s.parseFormattingOpts(sv)
			if err != nil {
				return fmt.Errorf("%w: formatting options parsing failed: %v", jsonrpc2.ErrInvalidParams, err)
			}
			s.configuration.FormattingOptions = newFmtOpts

		case "ext_code":
			newCode, err := s.parseExtCode(sv)
			if err != nil {
				return fmt.Errorf("%w: ext_code parsing failed: %v", jsonrpc2.ErrInvalidParams, err)
			}
			s.configuration.ExtCode = newCode
		case "native_functions":
			newFuncs, err := parseNativeFunctions(sv)
			if err != nil {
				return fmt.Errorf("%w: native_functions parsing failed: %v", jsonrpc2.ErrInvalidParams, err)
			}
			s.configuration.NativeFunctions = newFuncs
		case "eval_binary":
			if cmd, ok := sv.(string); ok {
				s.configuration.EvalBinary = cmd
			} else {
				return fmt.Errorf("%w: unsupported settings value for eval_binary. expected string. got: %T", jsonrpc2.ErrInvalidParams, sv)
			}

		default:
			return fmt.Errorf("%w: unsupported settings key: %q", jsonrpc2.ErrInvalidParams, sk)
		}
	}
	log.Infof("configuration updated: %+v", s.configuration)

	return nil
}

func (s *Server) parseExtVars(unparsed interface{}) (map[string]string, error) {
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

// ParseNativeFunction parses a command line native function spec of the form
// "name" or "name:parameters", where parameters is the number of arguments the
// function takes (zero when omitted).
func ParseNativeFunction(spec string) (string, int, error) {
	name, count, hasCount := strings.Cut(spec, ":")
	if name == "" {
		return "", 0, fmt.Errorf("invalid native function %q: missing name", spec)
	}
	if !hasCount {
		return name, 0, nil
	}
	n, err := strconv.Atoi(count)
	if err != nil || n < 0 {
		return "", 0, fmt.Errorf("invalid native function %q: parameter count must be a non-negative integer", spec)
	}
	return name, n, nil
}

func parseNativeFunctions(unparsed interface{}) (map[string]int, error) {
	newFuncs, ok := unparsed.(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("unsupported settings value for native_functions. expected json object. got: %T", unparsed)
	}

	funcs := make(map[string]int, len(newFuncs))
	for name, value := range newFuncs {
		var count int
		switch v := value.(type) {
		case int:
			count = v
		case float64:
			if v != float64(int(v)) {
				return nil, fmt.Errorf("unsupported settings value for native_functions.%s. expected integer. got: %v", name, v)
			}
			count = int(v)
		default:
			return nil, fmt.Errorf("unsupported settings value for native_functions.%s. expected integer. got: %T", name, value)
		}
		if count < 0 {
			return nil, fmt.Errorf("unsupported settings value for native_functions.%s. expected a non-negative integer. got: %d", name, count)
		}
		funcs[name] = count
	}
	return funcs, nil
}

// registerNativeStubs registers the configured native functions on the VM.
// Each stub takes the configured number of arguments and returns null, which is
// enough for the language server to evaluate code that calls std.native.
// Functions the VM already provides (the Tanka ones) are left untouched.
func (s *Server) registerNativeStubs(vm *jsonnet.VM) {
	for name, count := range s.configuration.NativeFunctions {
		if s.configuration.ResolvePathsWithTanka && isTankaNativeFunc(name) {
			continue
		}
		params := make(ast.Identifiers, count)
		for i := range params {
			params[i] = ast.Identifier("arg" + strconv.Itoa(i))
		}
		vm.NativeFunction(&jsonnet.NativeFunction{
			Name:   name,
			Params: params,
			Func:   func([]interface{}) (interface{}, error) { return nil, nil },
		})
	}
}

func isTankaNativeFunc(name string) bool {
	for _, f := range tankaNative.Funcs() {
		if f.Name == name {
			return true
		}
	}
	return false
}

func (s *Server) parseFormattingOpts(unparsed interface{}) (formatter.Options, error) {
	newOpts, ok := unparsed.(map[string]interface{})
	if !ok {
		return formatter.Options{}, fmt.Errorf("unsupported settings value for formatting. expected json object. got: %T", unparsed)
	}

	opts := formatter.DefaultOptions()
	config := mapstructure.DecoderConfig{
		Result: &opts,
		DecodeHook: mapstructure.ComposeDecodeHookFunc(
			stringStyleDecodeFunc,
			commentStyleDecodeFunc,
		),
	}
	decoder, err := mapstructure.NewDecoder(&config)
	if err != nil {
		return formatter.Options{}, fmt.Errorf("decoder construction failed: %v", err)
	}

	if err := decoder.Decode(newOpts); err != nil {
		return formatter.Options{}, fmt.Errorf("map decode failed: %v", err)
	}
	return opts, nil
}

func (s *Server) parseExtCode(unparsed interface{}) (map[string]string, error) {
	newVars, ok := unparsed.(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("unsupported settings value for ext_code. expected json object. got: %T", unparsed)
	}

	vm := s.getVM(".")

	extCode := make(map[string]string, len(newVars))
	for varKey, varValue := range newVars {
		vv, ok := varValue.(string)
		if !ok {
			return nil, fmt.Errorf("unsupported settings value for ext_code.%s. expected string. got: %T", varKey, varValue)
		}
		jsonResult, _ := vm.EvaluateAnonymousSnippet("ext-code", vv)
		extCode[varKey] = jsonResult
	}

	return extCode, nil
}

func resetExtVars(vm *jsonnet.VM, vars map[string]string, code map[string]string) {
	vm.ExtReset()
	for vk, vv := range vars {
		vm.ExtVar(vk, vv)
	}
	for vk, vv := range code {
		vm.ExtCode(vk, vv)
	}
}

func stringStyleDecodeFunc(_, to reflect.Type, unparsed interface{}) (interface{}, error) {
	if to != reflect.TypeOf(formatter.StringStyleDouble) {
		return unparsed, nil
	}

	str, ok := unparsed.(string)
	if !ok {
		return nil, fmt.Errorf("expected string, got: %T", unparsed)
	}
	// will not panic because of the kind == string check above
	switch str {
	case "double":
		return formatter.StringStyleDouble, nil
	case "single":
		return formatter.StringStyleSingle, nil
	case "leave":
		return formatter.StringStyleLeave, nil
	default:
		return nil, fmt.Errorf("expected one of 'double', 'single', 'leave', got: %q", str)
	}
}

func commentStyleDecodeFunc(_, to reflect.Type, unparsed interface{}) (interface{}, error) {
	if to != reflect.TypeOf(formatter.CommentStyleHash) {
		return unparsed, nil
	}

	str, ok := unparsed.(string)
	if !ok {
		return nil, fmt.Errorf("expected string, got: %T", unparsed)
	}
	switch str {
	case "hash":
		return formatter.CommentStyleHash, nil
	case "slash":
		return formatter.CommentStyleSlash, nil
	case "leave":
		return formatter.CommentStyleLeave, nil
	default:
		return nil, fmt.Errorf("expected one of 'hash', 'slash', 'leave', got: %q", str)
	}
}
