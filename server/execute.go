package server

import (
	"context"
	"encoding/json"
	"fmt"
	"reflect"

	"github.com/grafana/jsonnet-language-server/utils"
	"github.com/jdbaldry/go-language-server-protocol/lsp/protocol"
	log "github.com/sirupsen/logrus"
)

func (s *server) ExecuteCommand(ctx context.Context, params *protocol.ExecuteCommandParams) (interface{}, error) {
	switch params.Command {
	case "jsonnet.evalItem":
		// WIP
		return s.evalItem(ctx, params)
	case "jsonnet.evalFile":
		params.Arguments = append(params.Arguments, json.RawMessage("\"\""))
		return s.evalExpression(ctx, params)
	case "jsonnet.evalExpression":
		return s.evalExpression(ctx, params)
	}

	return nil, fmt.Errorf("unknown command: %s", params.Command)
}

func (s *server) evalItem(ctx context.Context, params *protocol.ExecuteCommandParams) (interface{}, error) {
	args := params.Arguments
	if len(args) != 2 {
		return nil, fmt.Errorf("expected 2 arguments, got %d", len(args))
	}

	var fileName string
	if err := json.Unmarshal(args[0], &fileName); err != nil {
		return nil, fmt.Errorf("failed to unmarshal file name: %v", err)
	}
	var position protocol.Position
	if err := json.Unmarshal(args[1], &position); err != nil {
		return nil, fmt.Errorf("failed to unmarshal position: %v", err)
	}

	doc, err := s.cache.get(protocol.URIFromPath(fileName))
	if err != nil {
		return nil, utils.LogErrorf("evalItem: %s: %w", errorRetrievingDocument, err)
	}

	stack, err := findNodeByPosition(doc.ast, position)
	if err != nil {
		return nil, err
	}

	if stack.IsEmpty() {
		return nil, fmt.Errorf("no node found at position %v", position)
	}

	log.Infof("fileName: %s", fileName)
	log.Infof("position: %+v", position)

	_, node := stack.Pop()

	return nil, fmt.Errorf("%v: %+v", reflect.TypeOf(node), node)
}

func (s *server) evalExpression(ctx context.Context, params *protocol.ExecuteCommandParams) (interface{}, error) {
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
	vm, err := s.getVM(fileName)
	if err != nil {
		return nil, err
	}

	script := fmt.Sprintf("local main = (import '%s');\nmain", fileName)
	if expression != "" {
		script += "." + expression

	}

	return vm.EvaluateAnonymousSnippet(fileName, script)
}
