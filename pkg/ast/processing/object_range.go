package processing

import (
	"fmt"
	"os"
	"strings"

	"github.com/google/go-jsonnet/ast"
)

type ObjectRange struct {
	Filename       string
	SelectionRange ast.LocationRange
	FullRange      ast.LocationRange
	FieldName      string
	Node           ast.Node
}

func (r *ObjectRange) ReadFromFile(fullRange bool) (string, error) {
	fileContent, err := os.ReadFile(r.Filename)
	if err != nil {
		return "", err
	}
	lines := strings.Split(string(fileContent), "\n")

	rangeToExtract := r.SelectionRange
	if fullRange {
		rangeToExtract = r.FullRange
	}

	// Extract the range from the file
	// First line: Replace trimmed columns with spaces
	// Middle lines: Keep all columns
	// Last line: Trim columns after the end of the range
	firstLine := lines[rangeToExtract.Begin.Line-1]
	firstLine = strings.Repeat(" ", rangeToExtract.Begin.Column-1) + firstLine[rangeToExtract.Begin.Column-1:]
	lastLine := lines[rangeToExtract.End.Line-1]
	lastLine = lastLine[:rangeToExtract.End.Column-1]
	rangeLines := []string{firstLine}
	if rangeToExtract.End.Line > rangeToExtract.Begin.Line+1 {
		rangeLines = append(rangeLines, lines[rangeToExtract.Begin.Line:rangeToExtract.End.Line-1]...)
	}
	if rangeToExtract.End.Line > rangeToExtract.Begin.Line {
		rangeLines = append(rangeLines, lastLine)
	}
	return strings.Join(rangeLines, "\n"), nil
}

func FieldToRange(field ast.DesugaredObjectField) ObjectRange {
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
		FieldName:      FieldNameToString(field.Name),
		Node:           field.Body,
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

func LocalBindToRange(bind ast.LocalBind) ObjectRange {
	locRange := bind.LocRange
	if !locRange.Begin.IsSet() {
		locRange = *bind.Body.Loc()
	}
	filename := locRange.FileName
	return ObjectRange{
		Node:      bind.Body,
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
