package cache

import (
	"errors"
	"fmt"
	"os"
	"strings"
	"sync"

	"github.com/google/go-jsonnet/ast"
	"github.com/jdbaldry/go-language-server-protocol/lsp/protocol"
)

type Document struct {
	// From DidOpen and DidChange
	Item protocol.TextDocumentItem

	// Contains the last successfully parsed AST. If doc.err is not nil, it's out of date.
	AST                  ast.Node
	LinesChangedSinceAST map[int]bool

	// From diagnostics
	Val         string
	Err         error
	Diagnostics []protocol.Diagnostic
}

// Cache caches documents.
type Cache struct {
	mu              sync.RWMutex
	docs            map[protocol.DocumentURI]*Document
	topLevelObjects map[string][]*ast.DesugaredObject
}

// New returns a document cache.
func New() *Cache {
	return &Cache{
		mu:              sync.RWMutex{},
		docs:            make(map[protocol.DocumentURI]*Document),
		topLevelObjects: make(map[string][]*ast.DesugaredObject),
	}
}

// Put adds or replaces a document in the cache.
func (c *Cache) Put(doc *Document) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	uri := doc.Item.URI
	if old, ok := c.docs[uri]; ok {
		if old.Item.Version > doc.Item.Version {
			return errors.New("newer version of the document is already in the cache")
		}
	}
	c.docs[uri] = doc

	// Invalidate the TopLevelObject cache
	// We can't easily invalidate the cache for a single file (hard to figure out where the import actually leads),
	// so we just clear the whole thing
	c.topLevelObjects = make(map[string][]*ast.DesugaredObject)

	return nil
}

// Get retrieves a document from the cache.
func (c *Cache) Get(uri protocol.DocumentURI) (*Document, error) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	doc, ok := c.docs[uri]
	if !ok {
		return nil, fmt.Errorf("document %s not found in cache", uri)
	}

	return doc, nil
}

func (c *Cache) GetContents(uri protocol.DocumentURI, position protocol.Range) (string, error) {
	text := ""
	doc, err := c.Get(uri)
	if err == nil {
		text = doc.Item.Text
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
			if i == position.End.Line {
				contentBuilder.WriteString(lines[i][position.Start.Character:position.End.Character])
			} else {
				contentBuilder.WriteString(lines[i][position.Start.Character:])
			}
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

func (c *Cache) GetTopLevelObject(filename, importedFrom string) ([]*ast.DesugaredObject, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	cacheKey := importedFrom + ":" + filename
	v, ok := c.topLevelObjects[cacheKey]
	return v, ok
}

func (c *Cache) PutTopLevelObject(filename, importedFrom string, objects []*ast.DesugaredObject) {
	c.mu.Lock()
	defer c.mu.Unlock()

	cacheKey := importedFrom + ":" + filename
	c.topLevelObjects[cacheKey] = objects
}
