package processing

import (
	"fmt"
	"strings"

	"github.com/google/go-jsonnet/ast"
	position "github.com/grafana/jsonnet-language-server/pkg/position_conversion"
	"github.com/jdbaldry/go-language-server-protocol/lsp/protocol"
)

type ObjectRange struct {
	Filename       string
	SelectionRange ast.LocationRange
	FullRange      ast.LocationRange
	FieldName      string
	Node           ast.Node
}

func (p *Processor) FieldToRange(field ast.DesugaredObjectField) ObjectRange {
	selectionRange := ast.LocationRange{
		Begin: ast.Location{
			Line:   field.LocRange.Begin.Line,
			Column: field.LocRange.Begin.Column,
		},
		End: ast.Location{
			Line:   field.LocRange.Begin.Line,
			Column: field.LocRange.Begin.Column + len(p.FieldNameToString(field.Name)),
		},
	}
	return ObjectRange{
		Filename:       field.LocRange.FileName,
		SelectionRange: selectionRange,
		FullRange:      field.LocRange,
		FieldName:      p.FieldNameToString(field.Name),
		Node:           field.Body,
	}
}

func (p *Processor) FieldNameToString(fieldName ast.Node) string {
	const unknown = "<unknown>"

	switch fieldName := fieldName.(type) {
	case *ast.LiteralString:
		return fieldName.Value
	case *ast.Index:
		// We only want to wrap in brackets at the top level, so we trim at all step and then rewrap
		return fmt.Sprintf("[%s.%s]",
			strings.Trim(p.FieldNameToString(fieldName.Target), "[]"),
			strings.Trim(p.FieldNameToString(fieldName.Index), "[]"),
		)
	case *ast.Var:
		return string(fieldName.Id)
	default:
		loc := fieldName.Loc()
		if loc == nil {
			return unknown
		}
		fname := loc.FileName
		if fname == "" {
			return unknown
		}

		content, err := p.cache.GetContents(protocol.URIFromPath(fname), position.RangeASTToProtocol(*loc))
		if err != nil {
			return unknown
		}
		return content
	}
}

func LocalBindToRange(bind ast.LocalBind) ObjectRange {
	locRange := bind.LocRange
	if !locRange.Begin.IsSet() {
		locRange = *bind.Body.Loc()
	}
	filename := locRange.FileName
	return ObjectRange{
		Filename:  filename,
		FullRange: locRange,
		SelectionRange: ast.LocationRange{
			Begin: ast.Location{
				Line:   locRange.Begin.Line,
				Column: locRange.Begin.Column,
			},
			End: ast.Location{
				Line:   locRange.Begin.Line,
				Column: locRange.Begin.Column + len(bind.Variable),
			},
		},
	}
}
