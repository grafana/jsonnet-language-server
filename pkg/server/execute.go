package server

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"strings"

	"github.com/grafana/jsonnet-language-server/pkg/ast/processing"
	position "github.com/grafana/jsonnet-language-server/pkg/position_conversion"
	"github.com/grafana/jsonnet-language-server/pkg/utils"
	"github.com/jdbaldry/go-language-server-protocol/lsp/protocol"
	log "github.com/sirupsen/logrus"
)

func (s *Server) ExecuteCommand(_ context.Context, params *protocol.ExecuteCommandParams) (interface{}, error) {
	switch params.Command {
	case "jsonnet.evalItem":
		// WIP
		return s.evalItem(params)
	case "jsonnet.evalFile":
		params.Arguments = append(params.Arguments, json.RawMessage("\"\""))
		return s.evalExpression(params)
	case "jsonnet.evalExpression":
		return s.evalExpression(params)
	}

	return nil, fmt.Errorf("unknown command: %s", params.Command)
}

func (s *Server) evalItem(params *protocol.ExecuteCommandParams) (interface{}, error) {
	args := params.Arguments
	if len(args) != 2 {
		return nil, fmt.Errorf("expected 2 arguments, got %d", len(args))
	}

	var fileName string
	if err := json.Unmarshal(args[0], &fileName); err != nil {
		return nil, fmt.Errorf("failed to unmarshal file name: %v", err)
	}
	var p protocol.Position
	if err := json.Unmarshal(args[1], &p); err != nil {
		return nil, fmt.Errorf("failed to unmarshal position: %v", err)
	}

	doc, err := s.cache.Get(protocol.URIFromPath(fileName))
	if err != nil {
		return nil, utils.LogErrorf("evalItem: %s: %w", errorRetrievingDocument, err)
	}

	stack, err := processing.FindNodeByPosition(doc.AST, position.ProtocolToAST(p))
	if err != nil {
		return nil, err
	}

	if stack.IsEmpty() {
		return nil, fmt.Errorf("no node found at position %v", p)
	}

	log.Infof("fileName: %s", fileName)
	log.Infof("position: %+v", p)

	node := stack.Pop()

	return nil, fmt.Errorf("%v: %+v", reflect.TypeOf(node), node)
}

func (s *Server) evalExpression(params *protocol.ExecuteCommandParams) (interface{}, error) {
	args := params.Arguments
	if len(args) != 2 {
		return nil, fmt.Errorf("expected 2 arguments, got %d", len(args))
	}

	var fileName string
	if err := json.Unmarshal(args[0], &fileName); err != nil {
		return nil, fmt.Errorf("failed to unmarshal file name: %v", err)
	}
	var expression string
	if err := json.Unmarshal(args[1], &expression); err != nil {
		return nil, fmt.Errorf("failed to unmarshal expression: %v", err)
	}

	script := fmt.Sprintf("local main = (import '%s');\nmain", fileName)
	if expression != "" {
		script += "." + expression
	}

	if s.configuration.EvalBinary != "" {
		return s.runEvalExternal(fileName, expression)
	}

	log.Infof("evaluating internally: file=%s expression=%q", fileName, expression)
	vm := s.getVM(fileName)
	return vm.EvaluateAnonymousSnippet(fileName, script)
}

// runEvalExternal runs the configured external command. When ResolvePathsWithTanka is set,
// EvalBinary is invoked as a Tanka binary (e.g. "tk eval <path>"). Otherwise it is invoked
// as a jsonnet binary (temp script file with -J, -V, --ext-code).
func (s *Server) runEvalExternal(fileName, expression string) (string, error) {
	absPath, err := filepath.Abs(fileName)
	if err != nil {
		return "", fmt.Errorf("eval: resolving path: %w", err)
	}

	if s.configuration.ResolvePathsWithTanka {
		return s.runEvalTanka(absPath, expression)
	}
	return s.runEvalJsonnet(absPath, expression)
}

// runEvalTanka invokes EvalBinary as a Tanka binary: tk eval <envDir> [-e expr] [-V ...] [--ext-code ...].
// envDir is the directory containing the file (e.g. the environment directory with main.jsonnet).
func (s *Server) runEvalTanka(absPath, expression string) (string, error) {
	envDir := filepath.Dir(absPath)
	args := []string{"eval", envDir}
	if expression != "" {
		args = append(args, "-e", expression)
	}
	for k, v := range s.configuration.ExtVars {
		args = append(args, "-V", k+"="+v)
	}
	for k, v := range s.configuration.ExtCode {
		args = append(args, "--ext-code", k+"="+v)
	}
	log.Infof("evaluating with external command: %s %s", s.configuration.EvalBinary, strings.Join(args, " "))
	cmd := exec.CommandContext(context.Background(), s.configuration.EvalBinary, args...)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("eval %s: %w\n%s", s.configuration.EvalBinary, err, out)
	}
	return strings.TrimSuffix(string(out), "\n"), nil
}

// runEvalJsonnet invokes EvalBinary as a jsonnet binary: jsonnet -J ... -V ... --ext-code ... <script.jsonnet>.
func (s *Server) runEvalJsonnet(absPath, expression string) (string, error) {
	script := fmt.Sprintf("local main = (import %q);\nmain", absPath)
	if expression != "" {
		script += "." + expression
	}
	jpaths := make([]string, 0, len(s.configuration.JPaths)+1)
	jpaths = append(jpaths, s.configuration.JPaths...)
	jpaths = append(jpaths, filepath.Dir(absPath))
	args := make([]string, 0, 4+len(jpaths)*2+len(s.configuration.ExtVars)*2+len(s.configuration.ExtCode)*2+2)
	for _, p := range jpaths {
		args = append(args, "-J", p)
	}
	for k, v := range s.configuration.ExtVars {
		args = append(args, "-V", k+"="+v)
	}
	for k, v := range s.configuration.ExtCode {
		args = append(args, "--ext-code", k+"="+v)
	}
	tmp, err := os.CreateTemp("", "jsonnet-eval-*.jsonnet")
	if err != nil {
		return "", fmt.Errorf("eval: creating temp file: %w", err)
	}
	tmpPath := tmp.Name()
	defer os.Remove(tmpPath)
	if _, err := tmp.WriteString(script); err != nil {
		tmp.Close()
		return "", fmt.Errorf("eval: writing script: %w", err)
	}
	if err := tmp.Close(); err != nil {
		return "", fmt.Errorf("eval: closing temp file: %w", err)
	}
	args = append(args, tmpPath)
	log.Infof("evaluating with external command: %s %s", s.configuration.EvalBinary, strings.Join(args, " "))
	cmd := exec.CommandContext(context.Background(), s.configuration.EvalBinary, args...)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("eval %s: %w\n%s", s.configuration.EvalBinary, err, out)
	}
	return strings.TrimSuffix(string(out), "\n"), nil
}
