// jsonnet-language-server: A Language Server Protocol server for Jsonnet.
// Copyright (C) 2021 Jack Baldry

// This program is free software: you can redistribute it and/or modify
// it under the terms of the GNU Affero General Public License as published
// by the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.

// This program is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
// GNU Affero General Public License for more details.

// You should have received a copy of the GNU Affero General Public License
// along with this program.  If not, see <https://www.gnu.org/licenses/>.

package main

import (
	"fmt"
	"os"

	"github.com/google/go-jsonnet/ast"
	"github.com/jdbaldry/go-language-server-protocol/lsp/protocol"
)

// locationRangeToProtocolRange translates a ast.LocationRange to a protocol.Range.
// The former is one indexed and the latter is zero indexed.
func locationRangeToProtocolRange(lr ast.LocationRange) protocol.Range {
	return protocol.Range{
		Start: protocol.Position{Line: uint32(lr.Begin.Line - 1), Character: uint32(lr.Begin.Column - 1)},
		End:   protocol.Position{Line: uint32(lr.End.Line - 1), Character: uint32(lr.End.Column - 1)},
	}
}

// analyseSymbols traverses the Jsonnet AST and produces a hierarchy of LSP symbols.
// TODO(#4): Implement symbol analysis for all AST nodes.
func analyseSymbols(n ast.Node) (symbols []protocol.DocumentSymbol) {
	switch n := n.(type) {

	case *ast.Apply:
		args := make([]protocol.DocumentSymbol, len(n.Arguments.Named))
		for i, bind := range n.Arguments.Named {
			args[i] = protocol.DocumentSymbol{
				Name:           string(bind.Name),
				Kind:           protocol.Variable,
				Range:          locationRangeToProtocolRange(*n.Loc()),
				SelectionRange: locationRangeToProtocolRange(*n.Loc()),
				Tags:           []protocol.SymbolTag{symbolTagDefinition},
				Children:       analyseSymbols(bind.Arg),
			}
		}
		for _, bind := range n.Arguments.Positional {
			// TODO: Evaluate the bind expression to know what the argument is.
			args = append(args, protocol.DocumentSymbol{
				Name:           "expr",
				Kind:           protocol.Variable,
				Range:          locationRangeToProtocolRange(*n.Loc()),
				SelectionRange: locationRangeToProtocolRange(*n.Loc()),
				Tags:           []protocol.SymbolTag{symbolTagDefinition},
				Children:       analyseSymbols(bind.Expr),
			})
		}
		symbols = append(symbols, protocol.DocumentSymbol{
			Name:           "apply",
			Kind:           protocol.Function,
			Range:          locationRangeToProtocolRange(*n.Loc()),
			SelectionRange: locationRangeToProtocolRange(*n.Loc()),
			Children:       append(args, analyseSymbols(n.Target)...),
		})

	case *ast.Array:
		children := []protocol.DocumentSymbol{}
		for _, elem := range n.Elements {
			children = append(children, analyseSymbols(elem.Expr)...)
		}
		symbols = append(symbols, protocol.DocumentSymbol{
			Name:           "array",
			Kind:           protocol.Array,
			Range:          locationRangeToProtocolRange(*n.Loc()),
			SelectionRange: locationRangeToProtocolRange(*n.Loc()),
			Children:       children,
		})

	case *ast.Binary:
		children := analyseSymbols(n.Left)
		children = append(children, analyseSymbols(n.Right)...)
		symbols = append(symbols, protocol.DocumentSymbol{
			Name:           n.Op.String(),
			Kind:           protocol.Operator,
			Range:          locationRangeToProtocolRange(*n.Loc()),
			SelectionRange: locationRangeToProtocolRange(*n.Loc()),
			Children:       children,
		})

	case *ast.DesugaredObject:
		locals := make([]protocol.DocumentSymbol, len(n.Locals))
		for i, bind := range n.Locals {
			// This variable is where `$` references for all children of this object.
			// Although this local has children, that is only a self reference and is currently ignored.
			if string(bind.Variable) == "$" {
				locals[i] = protocol.DocumentSymbol{
					Name:           string(bind.Variable),
					Kind:           protocol.Variable,
					Range:          locationRangeToProtocolRange(*n.Loc()),
					SelectionRange: locationRangeToProtocolRange(*n.Loc()),
					Tags:           []protocol.SymbolTag{symbolTagDefinition},
				}
			} else {
				locals[i] = protocol.DocumentSymbol{
					Name:           string(bind.Variable),
					Kind:           protocol.Variable,
					Range:          locationRangeToProtocolRange(bind.LocRange),
					SelectionRange: locationRangeToProtocolRange(bind.LocRange),
					Tags:           []protocol.SymbolTag{symbolTagDefinition},
					Children:       analyseSymbols(bind.Body),
				}
			}
		}

		fields := make([]protocol.DocumentSymbol, len(n.Fields))
		for i, field := range n.Fields {
			fields[i] = protocol.DocumentSymbol{
				Name:           "field",
				Kind:           protocol.Field,
				Range:          locationRangeToProtocolRange(field.LocRange),
				SelectionRange: locationRangeToProtocolRange(field.LocRange),
				Children:       append(analyseSymbols(field.Name), analyseSymbols(field.Body)...),
			}
		}
		symbols = append(symbols, protocol.DocumentSymbol{
			Name:           "object",
			Kind:           protocol.Object,
			Range:          locationRangeToProtocolRange(*n.Loc()),
			SelectionRange: locationRangeToProtocolRange(*n.Loc()),
			Children:       append(locals, fields...),
		})

	case *ast.Error:
		// Do nothing for now.

	case *ast.Function:
		params := make([]protocol.DocumentSymbol, len(n.Parameters))
		for i, param := range n.Parameters {
			params[i] = protocol.DocumentSymbol{
				Name:           string(param.Name),
				Kind:           protocol.Variable,
				Range:          locationRangeToProtocolRange(param.LocRange),
				SelectionRange: locationRangeToProtocolRange(param.LocRange),
				Tags:           []protocol.SymbolTag{symbolTagDefinition},
			}
			if param.DefaultArg != nil {
				params[i].Children = analyseSymbols(param.DefaultArg)
			}
		}
		symbols = append(symbols, protocol.DocumentSymbol{
			Name:           "function",
			Kind:           protocol.Function,
			Range:          locationRangeToProtocolRange(*n.Loc()),
			SelectionRange: locationRangeToProtocolRange(*n.Loc()),
		})

	case *ast.Import:
		// Although the literal string is a child of an Import, it is ignored and the keyword
		// and the literal string are captured by a single File document.
		// This is one less symbol in the hierarchy and it means that go to definition works
		// from either the keyword or the literal string.
		// The same is true for ImportStr.
		symbols = append(symbols, protocol.DocumentSymbol{
			Name:           n.File.Value,
			Kind:           protocol.File,
			Range:          locationRangeToProtocolRange(*n.Loc()),
			SelectionRange: locationRangeToProtocolRange(*n.Loc()),
		})

	case *ast.ImportStr:
		// See comment associated with Import.
		symbols = append(symbols, protocol.DocumentSymbol{
			Name:           n.File.Value,
			Kind:           protocol.File,
			Range:          locationRangeToProtocolRange(*n.Loc()),
			SelectionRange: locationRangeToProtocolRange(*n.Loc()),
		})

	case *ast.Index:
		symbols = append(symbols, protocol.DocumentSymbol{
			Name:           "index",
			Kind:           protocol.Field,
			Range:          locationRangeToProtocolRange(*n.Loc()),
			SelectionRange: locationRangeToProtocolRange(*n.Loc()),
			Tags:           []protocol.SymbolTag{symbolTagDefinition},
			Children:       append(analyseSymbols(n.Target), analyseSymbols(n.Index)...),
		})

	case *ast.LiteralBoolean:
		symbols = append(symbols, protocol.DocumentSymbol{
			Name:           fmt.Sprint(n.Value),
			Kind:           protocol.Boolean,
			Range:          locationRangeToProtocolRange(*n.Loc()),
			SelectionRange: locationRangeToProtocolRange(*n.Loc()),
		})

	case *ast.LiteralNull:
		symbols = append(symbols, protocol.DocumentSymbol{
			Name:           "null",
			Kind:           protocol.Null,
			Range:          locationRangeToProtocolRange(*n.Loc()),
			SelectionRange: locationRangeToProtocolRange(*n.Loc()),
		})

	case *ast.LiteralNumber:
		symbols = append(symbols, protocol.DocumentSymbol{
			Name:           n.OriginalString,
			Kind:           protocol.Number,
			Range:          locationRangeToProtocolRange(*n.Loc()),
			SelectionRange: locationRangeToProtocolRange(*n.Loc()),
		})

	case *ast.LiteralString:
		symbols = append(symbols, protocol.DocumentSymbol{
			Name:           n.Value,
			Kind:           protocol.String,
			Range:          locationRangeToProtocolRange(*n.Loc()),
			SelectionRange: locationRangeToProtocolRange(*n.Loc()),
		})

	case *ast.Local:
		binds := make([]protocol.DocumentSymbol, len(n.Binds))
		for i, bind := range n.Binds {
			binds[i] = protocol.DocumentSymbol{
				Name:     string(bind.Variable),
				Kind:     protocol.Variable,
				Tags:     []protocol.SymbolTag{symbolTagDefinition},
				Children: analyseSymbols(bind.Body),
			}
			// If the line is zero, it must be unset as Jsonnet location ranges are indexed at one.
			// This seems to only happen with local definitions of functions which are preceded with the token "local".
			// Adding five (five minus the one for zero indexing plus one for a space) to the location range of the local
			// symbol gets closer to the real location but any amount of whitespace could be inbetween.
			// Assuming a single space, this works perfectly.
			// TODO(#7): Understand why identifiers in local function binds are missing.
			if bind.LocRange.Begin.Line == 0 {
				binds[i].Range = protocol.Range{
					Start: protocol.Position{Line: uint32(n.Loc().Begin.Line - 1), Character: uint32(n.Loc().Begin.Column + 5)},
					End:   protocol.Position{Line: uint32(n.Loc().End.Line - 1), Character: uint32(n.Loc().End.Column + 5)},
				}
				binds[i].SelectionRange = binds[i].Range
			} else {
				binds[i].Range = locationRangeToProtocolRange(bind.LocRange)
				binds[i].SelectionRange = locationRangeToProtocolRange(bind.LocRange)
			}
		}
		symbols = append(symbols, protocol.DocumentSymbol{
			Name:           "local",
			Kind:           protocol.Namespace,
			Range:          locationRangeToProtocolRange(*n.Loc()),
			SelectionRange: locationRangeToProtocolRange(*n.Loc()),
			Children:       append(binds, analyseSymbols(n.Body)...),
		})

	case *ast.Self:
		symbols = append(symbols, protocol.DocumentSymbol{
			Name:           "self",
			Kind:           protocol.Variable,
			Range:          locationRangeToProtocolRange(*n.Loc()),
			SelectionRange: locationRangeToProtocolRange(*n.Loc()),
			Tags:           []protocol.SymbolTag{symbolTagDefinition},
		})

	case *ast.SuperIndex:
		symbols = append(symbols, protocol.DocumentSymbol{
			Name:           "index",
			Kind:           protocol.Field,
			Range:          locationRangeToProtocolRange(*n.Loc()),
			SelectionRange: locationRangeToProtocolRange(*n.Loc()),
			Children: append([]protocol.DocumentSymbol{{
				Name:           "super",
				Kind:           protocol.Variable,
				Range:          locationRangeToProtocolRange(*n.Loc()),
				SelectionRange: locationRangeToProtocolRange(*n.Loc()),
			}}, analyseSymbols(n.Index)...),
		})

	case *ast.Var:
		symbols = append(symbols, protocol.DocumentSymbol{
			Name:           string(n.Id),
			Kind:           protocol.Variable,
			Range:          locationRangeToProtocolRange(*n.Loc()),
			SelectionRange: locationRangeToProtocolRange(*n.Loc()),
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
