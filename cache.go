package main

import (
	"errors"
	"sync"

	"github.com/google/go-jsonnet/ast"
	"github.com/jdbaldry/go-language-server-protocol/lsp/protocol"
)

type document struct {
	item protocol.TextDocumentItem
	val  string
	ast  ast.Node
	err  error
	// Symbols are hierarchical and there is only ever a single root symbol.
	symbols protocol.DocumentSymbol
}

// newCache returns a document cache.
func newCache() *cache {
	return &cache{
		mu:   sync.RWMutex{},
		docs: make(map[protocol.DocumentURI]document),
	}
}

// cache caches documents.
type cache struct {
	mu   sync.RWMutex
	docs map[protocol.DocumentURI]document
}

// put adds or replaces a document in the cache.
// Documents are only replaced if the new document version is greater than the currently
// cached version.
func (c *cache) put(new document) error {
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
func (c *cache) get(uri protocol.DocumentURI) (document, error) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	doc, ok := c.docs[uri]
	if !ok {
		return document{}, errors.New("document not found")
	}

	return doc, nil
}
