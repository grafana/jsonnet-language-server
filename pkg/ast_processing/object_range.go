package ast_processing

import (
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
			Column: field.LocRange.Begin.Column + len(field.Name.(*ast.LiteralString).Value),
		},
	}
	return ObjectRange{
		Filename:       field.LocRange.FileName,
		SelectionRange: selectionRange,
		FullRange:      field.LocRange,
	}
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
