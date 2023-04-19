package server

import (
	"context"
	"encoding/json"
	"fmt"
	"reflect"

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

	doc, err := s.cache.get(protocol.URIFromPath(fileName))
	if err != nil {
		return nil, utils.LogErrorf("evalItem: %s: %w", errorRetrievingDocument, err)
	}

	stack, err := processing.FindNodeByPosition(doc.ast, position.ProtocolToAST(p))
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

	// TODO: Replace this stuff with Tanka's `eval` code
	vm := s.getVM(fileName)

	script := fmt.Sprintf("local main = (import '%s');\nmain", fileName)
	if expression != "" {
		script += "." + expression
	}

	return vm.EvaluateAnonymousSnippet(fileName, script)
}
