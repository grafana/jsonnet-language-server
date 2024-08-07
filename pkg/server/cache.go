package server

import (
	"errors"
	"fmt"
	"os"
	"strings"
	"sync"

	"github.com/google/go-jsonnet/ast"
	"github.com/jdbaldry/go-language-server-protocol/lsp/protocol"
)

type document struct {
	// From DidOpen and DidChange
	item protocol.TextDocumentItem

	// Contains the last successfully parsed AST. If doc.err is not nil, it's out of date.
	ast                  ast.Node
	linesChangedSinceAST map[int]bool

	// From diagnostics
	val         string
	err         error
	diagnostics []protocol.Diagnostic
}

// newCache returns a document cache.
func newCache() *cache {
	return &cache{
		mu:        sync.RWMutex{},
		docs:      make(map[protocol.DocumentURI]*document),
		diagQueue: make(map[protocol.DocumentURI]struct{}),
	}
}

// cache caches documents.
type cache struct {
	mu   sync.RWMutex
	docs map[protocol.DocumentURI]*document

	diagMutex   sync.RWMutex
	diagQueue   map[protocol.DocumentURI]struct{}
	diagRunning sync.Map
}

// put adds or replaces a document in the cache.
func (c *cache) put(new *document) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	uri := new.item.URI
	if old, ok := c.docs[uri]; ok {
		if old.item.Version > new.item.Version {
			return errors.New("newer version of the document is already in the cache")
		}
	}
	c.docs[uri] = new

	return nil
}

// get retrieves a document from the cache.
func (c *cache) get(uri protocol.DocumentURI) (*document, error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	doc, ok := c.docs[uri]
	if !ok {
		return nil, fmt.Errorf("document %s not found in cache", uri)
	}

	return doc, nil
}

func (c *cache) getContents(uri protocol.DocumentURI, position protocol.Range) (string, error) {
	text := ""
	doc, err := c.get(uri)
	if err == nil {
		text = doc.item.Text
	} else {
		// Read the file from disk (TODO: cache this)
		bytes, err := os.ReadFile(uri.SpanURI().Filename())
		if err != nil {
			return "", err
		}
		text = string(bytes)
	}

	lines := strings.Split(text, "\n")
	if int(position.Start.Line) >= len(lines) {
		return "", fmt.Errorf("line %d out of range", position.Start.Line)
	}
	if int(position.Start.Character) >= len(lines[position.Start.Line]) {
		return "", fmt.Errorf("character %d out of range", position.Start.Character)
	}
	if int(position.End.Line) >= len(lines) {
		return "", fmt.Errorf("line %d out of range", position.End.Line)
	}
	if int(position.End.Character) >= len(lines[position.End.Line]) {
		return "", fmt.Errorf("character %d out of range", position.End.Character)
	}

	contentBuilder := strings.Builder{}
	for i := position.Start.Line; i <= position.End.Line; i++ {
		switch i {
		case position.Start.Line:
			contentBuilder.WriteString(lines[i][position.Start.Character:])
		case position.End.Line:
			contentBuilder.WriteString(lines[i][:position.End.Character])
		default:
			contentBuilder.WriteString(lines[i])
		}
		if i != position.End.Line {
			contentBuilder.WriteRune('\n')
		}
	}

	return contentBuilder.String(), nil
}
