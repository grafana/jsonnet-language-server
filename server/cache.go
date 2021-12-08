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

package server

import (
	"errors"
	"fmt"
	"sync"

	"github.com/google/go-jsonnet/ast"
	"github.com/jdbaldry/go-language-server-protocol/lsp/protocol"
)

type document struct {
	// From DidOpen and DidChange
	item protocol.TextDocumentItem
	ast  ast.Node

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
// Documents are only replaced if the new document version is greater than the currently
// cached version.
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
