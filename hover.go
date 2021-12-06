package main

import (
	"github.com/jdbaldry/go-language-server-protocol/lsp/protocol"
)

func handleHover(doc document, params *protocol.HoverParams) (*protocol.Hover, error) {
	var aux func([]protocol.DocumentSymbol, protocol.DocumentSymbol) (*protocol.Hover, error)
	aux = func(stack []protocol.DocumentSymbol, ds protocol.DocumentSymbol) (*protocol.Hover, error) {
		if params.Position.Line == ds.Range.Start.Line &&
			params.Position.Line == ds.Range.End.Line &&
			params.Position.Character >= ds.Range.Start.Character &&
			params.Position.Character <= ds.Range.End.Character {

			if ds.Kind == protocol.Function {
				// Look before if it's a std function
			}
			if ds.Kind == protocol.Variable && ds.Name == "std" {
				return &protocol.Hover{Range: ds.Range, Contents: protocol.MarkupContent{Kind: protocol.PlainText, Value: "test"}}, nil
			}

		}
		stack = append(stack, ds.Children...)
		for i := len(ds.Children); i != 0; i-- {
			if def, err := aux(stack, ds.Children[i-1]); def != nil || err != nil {
				return def, err
			}
			stack = stack[:len(stack)-1]
		}
		return nil, nil
	}

	return aux([]protocol.DocumentSymbol{doc.symbols}, doc.symbols)
}
