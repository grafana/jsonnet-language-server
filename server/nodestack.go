package server

import (
	"sort"

	"github.com/google/go-jsonnet/ast"
)

type NodeStack struct {
	from  ast.Node
	stack []ast.Node
}

func NewNodeStack(from ast.Node) *NodeStack {
	return &NodeStack{
		from:  from,
		stack: []ast.Node{from},
	}
}

func (s *NodeStack) Push(n ast.Node) *NodeStack {
	s.stack = append(s.stack, n)
	return s
}

func (s *NodeStack) Pop() (*NodeStack, ast.Node) {
	l := len(s.stack)
	if l == 0 {
		return s, nil
	}
	n := s.stack[l-1]
	s.stack = s.stack[:l-1]
	return s, n
}

func (s *NodeStack) IsEmpty() bool {
	return len(s.stack) == 0
}

func (s *NodeStack) reorderDesugaredObjects() *NodeStack {
	sort.SliceStable(s.stack, func(i, j int) bool {
		_, iIsDesugared := s.stack[i].(*ast.DesugaredObject)
		_, jIsDesugared := s.stack[j].(*ast.DesugaredObject)
		if !iIsDesugared && !jIsDesugared {
			return false
		}

		iLoc, jLoc := s.stack[i].Loc(), s.stack[j].Loc()
		if iLoc.Begin.Line < jLoc.Begin.Line && iLoc.End.Line > jLoc.End.Line {
			return true
		}

		return false
	})
	return s
}
