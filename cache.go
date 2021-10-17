package main

import (
	"errors"
	"fmt"
	"os"
	"sync"

	"github.com/jdbaldry/go-language-server-protocol/lsp/protocol"
)

// newCache returns a document cache.
func newCache() *cache {
	return &cache{
		mu:   sync.RWMutex{},
		docs: make(map[protocol.DocumentURI]protocol.TextDocumentItem),
	}
}

// cache caches documents.
type cache struct {
	mu   sync.RWMutex
	docs map[protocol.DocumentURI]protocol.TextDocumentItem
}

// add puts a document in the cache.
func (c *cache) put(doc protocol.TextDocumentItem) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if _, ok := c.docs[doc.URI]; ok {
		return errors.New("document already in cache")
	}
	c.docs[doc.URI] = doc
	fmt.Fprintf(os.Stderr, "%+v\n", c.docs)

	return nil
}

// applyChanges applies a sequence of changes to document text.
// TODO: Support incremental sync.
func applyChanges(text string, changes []protocol.TextDocumentContentChangeEvent) (string, error) {
	if len(changes) == 0 {
		return text, nil
	}
	return changes[len(changes)-1].Text, nil
}

// update applies content changes to a document in the cache.
func (c *cache) update(id protocol.VersionedTextDocumentIdentifier, changes []protocol.TextDocumentContentChangeEvent) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	doc, ok := c.docs[id.URI]
	if !ok {
		return errors.New("document not found")
	}

	text, err := applyChanges(doc.Text, changes)
	if err != nil {
		return fmt.Errorf("unable to apply changes: %w", err)
	}
	doc.Version = id.Version
	doc.Text = text
	c.docs[id.URI] = doc

	return nil
}

// get retrieves a document from the cache.
func (c *cache) get(uri protocol.DocumentURI) (protocol.TextDocumentItem, error) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	doc, ok := c.docs[uri]
	if !ok {
		return protocol.TextDocumentItem{}, errors.New("document not found")
	}

	return doc, nil
}
