package ast_processing

import (
	"fmt"
	"strings"

	"github.com/google/go-jsonnet/ast"
)

type ObjectRange struct {
	Filename       string
	SelectionRange ast.LocationRange
	FullRange      ast.LocationRange
}

func FieldToRange(field *ast.DesugaredObjectField) ObjectRange {
	selectionRange := ast.LocationRange{
		Begin: ast.Location{
			Line:   field.LocRange.Begin.Line,
			Column: field.LocRange.Begin.Column,
		},
		End: ast.Location{
			Line:   field.LocRange.Begin.Line,
			Column: field.LocRange.Begin.Column + len(FieldNameToString(field.Name)),
		},
	}
	return ObjectRange{
		Filename:       field.LocRange.FileName,
		SelectionRange: selectionRange,
		FullRange:      field.LocRange,
	}
}

func FieldNameToString(fieldName ast.Node) string {
	if fieldName, ok := fieldName.(*ast.LiteralString); ok {
		return fieldName.Value
	}
	if fieldName, ok := fieldName.(*ast.Index); ok {
		// We only want to wrap in brackets at the top level, so we trim at all step and then rewrap
		return fmt.Sprintf("[%s.%s]",
			strings.Trim(FieldNameToString(fieldName.Target), "[]"),
			strings.Trim(FieldNameToString(fieldName.Index), "[]"),
		)
	}
	if fieldName, ok := fieldName.(*ast.Var); ok {
		return string(fieldName.Id)
	}
	return ""
}

func LocalBindToRange(bind *ast.LocalBind) ObjectRange {
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
