package main

import (
	"fmt"
	"os"

	"github.com/google/go-jsonnet/ast"
	"github.com/jdbaldry/go-language-server-protocol/lsp/protocol"
)

// analyseSymbols traverses the Jsonnet AST and produces a hierarchy of LSP symbols.
func analyseSymbols(n ast.Node) (symbols []protocol.DocumentSymbol) {
	switch n := n.(type) {

	case *ast.Array:
		children := []protocol.DocumentSymbol{}
		for _, elem := range n.Elements {
			children = append(children, analyseSymbols(elem.Expr)...)
		}
		symbols = append(symbols, protocol.DocumentSymbol{
			Name: "array",
			Kind: protocol.Array,
			Range: protocol.Range{
				Start: protocol.Position{Line: uint32(n.Loc().Begin.Line - 1), Character: uint32(n.Loc().Begin.Column - 1)},
				End:   protocol.Position{Line: uint32(n.Loc().End.Line - 1), Character: uint32(n.Loc().End.Column - 1)},
			},
			SelectionRange: protocol.Range{
				Start: protocol.Position{Line: uint32(n.Loc().Begin.Line - 1), Character: uint32(n.Loc().Begin.Column - 1)},
				End:   protocol.Position{Line: uint32(n.Loc().End.Line - 1), Character: uint32(n.Loc().End.Column - 1)},
			},
			Children: children,
		})

	case *ast.Binary:
		children := analyseSymbols(n.Left)
		children = append(children, analyseSymbols(n.Right)...)
		symbols = append(symbols, protocol.DocumentSymbol{
			Name: n.Op.String(),
			Kind: protocol.Operator,
			Range: protocol.Range{
				Start: protocol.Position{Line: uint32(n.Loc().Begin.Line - 1), Character: uint32(n.Loc().Begin.Column - 1)},
				End:   protocol.Position{Line: uint32(n.Loc().End.Line - 1), Character: uint32(n.Loc().End.Column - 1)},
			},
			SelectionRange: protocol.Range{
				Start: protocol.Position{Line: uint32(n.Loc().Begin.Line - 1), Character: uint32(n.Loc().Begin.Column - 1)},
				End:   protocol.Position{Line: uint32(n.Loc().End.Line - 1), Character: uint32(n.Loc().End.Column - 1)},
			},
			Children: children,
		})

	case *ast.DesugaredObject:
		fields := make([]protocol.DocumentSymbol, len(n.Fields))
		locals := make([]protocol.DocumentSymbol, len(n.Locals))
		for i, bind := range n.Locals {
			locals[i] = protocol.DocumentSymbol{
				Name: string(bind.Variable),
				Kind: protocol.Variable,
				Range: protocol.Range{
					Start: protocol.Position{Line: uint32(bind.LocRange.Begin.Line - 1), Character: uint32(bind.LocRange.Begin.Column - 1)},
					End:   protocol.Position{Line: uint32(bind.LocRange.End.Line - 1), Character: uint32(bind.LocRange.End.Column - 1)},
				},
				SelectionRange: protocol.Range{
					Start: protocol.Position{Line: uint32(bind.LocRange.Begin.Line - 1), Character: uint32(bind.LocRange.Begin.Column - 1)},
					End:   protocol.Position{Line: uint32(bind.LocRange.End.Line - 1), Character: uint32(bind.LocRange.End.Column - 1)},
				},
				Tags:     []protocol.SymbolTag{symbolTagDefinition},
				Children: analyseSymbols(bind.Body),
			}
		}
		for i, field := range n.Fields {
			fields[i] = protocol.DocumentSymbol{
				Name: "field",
				Kind: protocol.Field,
				Range: protocol.Range{
					Start: protocol.Position{Line: uint32(field.LocRange.Begin.Line - 1), Character: uint32(field.LocRange.Begin.Column - 1)},
					End:   protocol.Position{Line: uint32(field.LocRange.End.Line - 1), Character: uint32(field.LocRange.End.Column - 1)},
				},
				SelectionRange: protocol.Range{
					Start: protocol.Position{Line: uint32(field.LocRange.Begin.Line - 1), Character: uint32(field.LocRange.Begin.Column - 1)},
					End:   protocol.Position{Line: uint32(field.LocRange.End.Line - 1), Character: uint32(field.LocRange.End.Column - 1)},
				},
				Children: append(analyseSymbols(field.Name), analyseSymbols(field.Body)...),
			}
		}
		symbols = append(symbols, protocol.DocumentSymbol{
			Name:       "object",
			Kind:       protocol.Object,
			Tags:       []protocol.SymbolTag{},
			Deprecated: false,
			Range: protocol.Range{
				Start: protocol.Position{Line: uint32(n.Loc().Begin.Line - 1), Character: uint32(n.Loc().Begin.Column - 1)},
				End:   protocol.Position{Line: uint32(n.Loc().End.Line - 1), Character: uint32(n.Loc().End.Column - 1)},
			},
			SelectionRange: protocol.Range{
				Start: protocol.Position{Line: uint32(n.Loc().Begin.Line - 1), Character: uint32(n.Loc().Begin.Column - 1)},
				End:   protocol.Position{Line: uint32(n.Loc().End.Line - 1), Character: uint32(n.Loc().End.Column - 1)},
			},
			Children: append(locals, fields...),
		})

	case *ast.Import:
		symbols = append(symbols, protocol.DocumentSymbol{
			Name: n.File.Value,
			Kind: protocol.File,
			Range: protocol.Range{
				Start: protocol.Position{Line: uint32(n.Loc().Begin.Line - 1), Character: uint32(n.Loc().Begin.Column - 1)},
				End:   protocol.Position{Line: uint32(n.Loc().End.Line - 1), Character: uint32(n.Loc().End.Column - 1)},
			},
			SelectionRange: protocol.Range{
				Start: protocol.Position{Line: uint32(n.Loc().Begin.Line - 1), Character: uint32(n.Loc().Begin.Column - 1)},
				End:   protocol.Position{Line: uint32(n.Loc().End.Line - 1), Character: uint32(n.Loc().End.Column - 1)},
			},
		})

	case *ast.ImportStr:
		symbols = append(symbols, protocol.DocumentSymbol{
			Name: n.File.Value,
			Kind: protocol.File,
			Range: protocol.Range{
				Start: protocol.Position{Line: uint32(n.Loc().Begin.Line - 1), Character: uint32(n.Loc().Begin.Column - 1)},
				End:   protocol.Position{Line: uint32(n.Loc().End.Line - 1), Character: uint32(n.Loc().End.Column - 1)},
			},
			SelectionRange: protocol.Range{
				Start: protocol.Position{Line: uint32(n.Loc().Begin.Line - 1), Character: uint32(n.Loc().Begin.Column - 1)},
				End:   protocol.Position{Line: uint32(n.Loc().End.Line - 1), Character: uint32(n.Loc().End.Column - 1)},
			},
		})
	case *ast.Index:
		symbols = append(symbols, protocol.DocumentSymbol{
			Name: "index",
			Kind: protocol.Field,
			Range: protocol.Range{
				Start: protocol.Position{Line: uint32(n.Loc().Begin.Line - 1), Character: uint32(n.Loc().Begin.Column - 1)},
				End:   protocol.Position{Line: uint32(n.Loc().End.Line - 1), Character: uint32(n.Loc().End.Column - 1)},
			},
			SelectionRange: protocol.Range{
				Start: protocol.Position{Line: uint32(n.Loc().Begin.Line - 1), Character: uint32(n.Loc().Begin.Column - 1)},
				End:   protocol.Position{Line: uint32(n.Loc().End.Line - 1), Character: uint32(n.Loc().End.Column - 1)},
			},
			Children: append(analyseSymbols(n.Target), analyseSymbols(n.Index)...),
			Tags:     []protocol.SymbolTag{symbolTagDefinition},
		})

	case *ast.LiteralBoolean:
		symbols = append(symbols, protocol.DocumentSymbol{
			Name: fmt.Sprint(n.Value),
			Kind: protocol.Boolean,
			Range: protocol.Range{
				Start: protocol.Position{Line: uint32(n.Loc().Begin.Line - 1), Character: uint32(n.Loc().Begin.Column - 1)},
				End:   protocol.Position{Line: uint32(n.Loc().End.Line - 1), Character: uint32(n.Loc().End.Column - 1)},
			},
			SelectionRange: protocol.Range{
				Start: protocol.Position{Line: uint32(n.Loc().Begin.Line - 1), Character: uint32(n.Loc().Begin.Column - 1)},
				End:   protocol.Position{Line: uint32(n.Loc().End.Line - 1), Character: uint32(n.Loc().End.Column - 1)},
			},
		})

	case *ast.LiteralNull:
		symbols = append(symbols, protocol.DocumentSymbol{
			Name: "null",
			Kind: protocol.Null,
			Range: protocol.Range{
				Start: protocol.Position{Line: uint32(n.Loc().Begin.Line - 1), Character: uint32(n.Loc().Begin.Column - 1)},
				End:   protocol.Position{Line: uint32(n.Loc().End.Line - 1), Character: uint32(n.Loc().End.Column - 1)},
			},
			SelectionRange: protocol.Range{
				Start: protocol.Position{Line: uint32(n.Loc().Begin.Line - 1), Character: uint32(n.Loc().Begin.Column - 1)},
				End:   protocol.Position{Line: uint32(n.Loc().End.Line - 1), Character: uint32(n.Loc().End.Column - 1)},
			},
		})

	case *ast.LiteralNumber:
		symbols = append(symbols, protocol.DocumentSymbol{
			Name: n.OriginalString,
			Kind: protocol.Number,
			Range: protocol.Range{
				Start: protocol.Position{Line: uint32(n.Loc().Begin.Line - 1), Character: uint32(n.Loc().Begin.Column - 1)},
				End:   protocol.Position{Line: uint32(n.Loc().End.Line - 1), Character: uint32(n.Loc().End.Column - 1)},
			},
			SelectionRange: protocol.Range{
				Start: protocol.Position{Line: uint32(n.Loc().Begin.Line - 1), Character: uint32(n.Loc().Begin.Column - 1)},
				End:   protocol.Position{Line: uint32(n.Loc().End.Line - 1), Character: uint32(n.Loc().End.Column - 1)},
			},
		})

	case *ast.LiteralString:
		symbols = append(symbols, protocol.DocumentSymbol{
			Name: n.Value,
			Kind: protocol.String,
			Range: protocol.Range{
				Start: protocol.Position{Line: uint32(n.Loc().Begin.Line - 1), Character: uint32(n.Loc().Begin.Column - 1)},
				End:   protocol.Position{Line: uint32(n.Loc().End.Line - 1), Character: uint32(n.Loc().End.Column - 1)},
			},
			SelectionRange: protocol.Range{
				Start: protocol.Position{Line: uint32(n.Loc().Begin.Line - 1), Character: uint32(n.Loc().Begin.Column - 1)},
				End:   protocol.Position{Line: uint32(n.Loc().End.Line - 1), Character: uint32(n.Loc().End.Column - 1)},
			},
		})

	case *ast.Local:
		binds := make([]protocol.DocumentSymbol, len(n.Binds))
		for i, bind := range n.Binds {
			binds[i] = protocol.DocumentSymbol{
				Name: string(bind.Variable),
				Kind: protocol.Variable,
				Range: protocol.Range{
					Start: protocol.Position{Line: uint32(bind.LocRange.Begin.Line - 1), Character: uint32(bind.LocRange.Begin.Column - 1)},
					End:   protocol.Position{Line: uint32(bind.LocRange.End.Line - 1), Character: uint32(bind.LocRange.End.Column - 1)},
				},
				SelectionRange: protocol.Range{
					Start: protocol.Position{Line: uint32(bind.LocRange.Begin.Line - 1), Character: uint32(bind.LocRange.Begin.Column - 1)},
					End:   protocol.Position{Line: uint32(bind.LocRange.End.Line - 1), Character: uint32(bind.LocRange.End.Column - 1)},
				},
				Children: analyseSymbols(bind.Body),
				Tags:     []protocol.SymbolTag{symbolTagDefinition},
			}
		}
		symbols = append(symbols, protocol.DocumentSymbol{
			Name:       "local",
			Kind:       protocol.Namespace,
			Tags:       []protocol.SymbolTag{},
			Deprecated: false,
			Range: protocol.Range{
				Start: protocol.Position{Line: uint32(n.Loc().Begin.Line - 1), Character: uint32(n.Loc().Begin.Column - 1)},
				End:   protocol.Position{Line: uint32(n.Loc().End.Line - 1), Character: uint32(n.Loc().End.Column - 1)},
			},
			SelectionRange: protocol.Range{
				Start: protocol.Position{Line: uint32(n.Loc().Begin.Line - 1), Character: uint32(n.Loc().Begin.Column - 1)},
				End:   protocol.Position{Line: uint32(n.Loc().End.Line - 1), Character: uint32(n.Loc().End.Column - 1)},
			},
			Children: append(binds, analyseSymbols(n.Body)...),
		})

	case *ast.Self:
		symbols = append(symbols, protocol.DocumentSymbol{
			Name: "self",
			Kind: protocol.Variable,
			Range: protocol.Range{
				Start: protocol.Position{Line: uint32(n.Loc().Begin.Line - 1), Character: uint32(n.Loc().Begin.Column - 1)},
				End:   protocol.Position{Line: uint32(n.Loc().End.Line - 1), Character: uint32(n.Loc().End.Column - 1)},
			},
			SelectionRange: protocol.Range{
				Start: protocol.Position{Line: uint32(n.Loc().Begin.Line - 1), Character: uint32(n.Loc().Begin.Column - 1)},
				End:   protocol.Position{Line: uint32(n.Loc().End.Line - 1), Character: uint32(n.Loc().End.Column - 1)},
			},
			Tags: []protocol.SymbolTag{symbolTagDefinition},
		})

	case *ast.SuperIndex:
		symbols = append(symbols, protocol.DocumentSymbol{
			Name: "super",
			Kind: protocol.Field,
			Range: protocol.Range{
				Start: protocol.Position{Line: uint32(n.Loc().Begin.Line - 1), Character: uint32(n.Loc().Begin.Column - 1)},
				End:   protocol.Position{Line: uint32(n.Loc().End.Line - 1), Character: uint32(n.Loc().End.Column - 1)},
			},
			SelectionRange: protocol.Range{
				Start: protocol.Position{Line: uint32(n.Loc().Begin.Line - 1), Character: uint32(n.Loc().Begin.Column - 1)},
				End:   protocol.Position{Line: uint32(n.Loc().End.Line - 1), Character: uint32(n.Loc().End.Column - 1)},
			},
			Children: analyseSymbols(n.Index),
			Tags:     []protocol.SymbolTag{symbolTagDefinition},
		})

	case *ast.Var:
		symbols = append(symbols, protocol.DocumentSymbol{
			Name: string(n.Id),
			Kind: protocol.Variable,
			Range: protocol.Range{
				Start: protocol.Position{Line: uint32(n.Loc().Begin.Line - 1), Character: uint32(n.Loc().Begin.Column - 1)},
				End:   protocol.Position{Line: uint32(n.Loc().End.Line - 1), Character: uint32(n.Loc().End.Column - 1)},
			},
			SelectionRange: protocol.Range{
				Start: protocol.Position{Line: uint32(n.Loc().Begin.Line - 1), Character: uint32(n.Loc().Begin.Column - 1)},
				End:   protocol.Position{Line: uint32(n.Loc().End.Line - 1), Character: uint32(n.Loc().End.Column - 1)},
			},
		})

	default:
		fmt.Fprintf(os.Stderr, "analyseSymbols: unhandled node: %T\n", n)
	}
	return
}

// isDefinition returns true if a symbol is tagged as a definition.
func isDefinition(s protocol.DocumentSymbol) bool {
	for _, t := range s.Tags {
		if t == symbolTagDefinition {
			return true
		}
	}
	return false
}
