package server

import (
	"bytes"
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/google/go-jsonnet/linter"
	"github.com/jdbaldry/go-language-server-protocol/lsp/protocol"
	log "github.com/sirupsen/logrus"
)

func (s *server) queueDiagnostics(uri protocol.DocumentURI) {
	s.cache.diagMutex.Lock()
	defer s.cache.diagMutex.Unlock()
	s.cache.diagQueue[uri] = struct{}{}
}

func (s *server) diagnosticsLoop() {
	go func() {
		for {
			s.cache.diagMutex.Lock()
			for uri := range s.cache.diagQueue {
				if _, ok := s.cache.diagRunning.Load(uri); ok {
					continue
				}

				go func() {
					s.cache.diagRunning.Store(uri, true)

					log.Debug("Publishing diagnostics for ", uri)
					doc, err := s.cache.get(uri)
					if err != nil {
						log.Errorf("publishDiagnostics: %s: %v\n", errorRetrievingDocument, err)
						return
					}

					diags := []protocol.Diagnostic{}
					evalChannel := make(chan []protocol.Diagnostic, 1)
					go func() {
						evalChannel <- s.getEvalDiags(doc)
					}()

					lintChannel := make(chan []protocol.Diagnostic, 1)
					if s.LintDiags {
						go func() {
							lintChannel <- s.getLintDiags(doc)
						}()
					}

					diags = append(diags, <-evalChannel...)

					if s.LintDiags {
						err = s.client.PublishDiagnostics(context.Background(), &protocol.PublishDiagnosticsParams{
							URI:         uri,
							Diagnostics: diags,
						})
						if err != nil {
							log.Errorf("publishDiagnostics: unable to publish diagnostics: %v\n", err)
						}

						diags = append(diags, <-lintChannel...)
					}

					if len(diags) == 0 {
						diags = []protocol.Diagnostic{
							{
								Source:   "jsonnet",
								Message:  "No errors or warnings",
								Severity: protocol.SeverityInformation,
							},
						}
					}

					err = s.client.PublishDiagnostics(context.Background(), &protocol.PublishDiagnosticsParams{
						URI:         uri,
						Diagnostics: diags,
					})
					if err != nil {
						log.Errorf("publishDiagnostics: unable to publish diagnostics: %v\n", err)
					}

					doc.diagnostics = diags

					log.Debug("Done publishing diagnostics for ", uri)

					s.cache.diagRunning.Delete(uri)
				}()
				delete(s.cache.diagQueue, uri)
			}
			s.cache.diagMutex.Unlock()

			time.Sleep(1 * time.Second)
		}
	}()
}

func (s *server) getEvalDiags(doc *document) (diags []protocol.Diagnostic) {
	if doc.err == nil && s.EvalDiags {
		vm, err := s.getVM(doc.item.URI.SpanURI().Filename())
		if err != nil {
			log.Errorf("getEvalDiags: %s: %v\n", errorRetrievingDocument, err)
			return
		}
		doc.val, doc.err = vm.EvaluateAnonymousSnippet(doc.item.URI.SpanURI().Filename(), doc.item.Text)
	}

	// Initialize with 1 because we indiscriminately subtract one to map error ranges to LSP ranges.
	if doc.err != nil {
		line, col, endLine, endCol := 1, 1, 1, 1
		diag := protocol.Diagnostic{Source: "jsonnet evaluation"}
		lines := strings.Split(doc.err.Error(), "\n")
		if len(lines) == 0 {
			log.Errorf("publishDiagnostics: expected at least two lines of Jsonnet evaluation error output, got: %v\n", lines)
			return
		}

		var match []string
		// TODO(#22): Runtime errors that come from imported files report an incorrect location
		runtimeErr := strings.HasPrefix(lines[0], "RUNTIME ERROR:")
		if runtimeErr {
			match = errRegexp.FindStringSubmatch(lines[1])
		} else {
			match = errRegexp.FindStringSubmatch(lines[0])
		}
		if len(match) == 10 {
			if match[1] != "" {
				line, _ = strconv.Atoi(match[1])
				endLine = line + 1
			}
			if match[2] != "" {
				line, _ = strconv.Atoi(match[2])
				col, _ = strconv.Atoi(match[3])
				endLine = line
				endCol, _ = strconv.Atoi(match[4])
			}
			if match[5] != "" {
				line, _ = strconv.Atoi(match[5])
				col, _ = strconv.Atoi(match[6])
				endLine, _ = strconv.Atoi(match[7])
				endCol, _ = strconv.Atoi(match[8])
			}
		}

		if runtimeErr {
			diag.Message = doc.err.Error()
			diag.Severity = protocol.SeverityWarning
		} else {
			diag.Message = match[9]
			diag.Severity = protocol.SeverityError
		}

		diag.Range = protocol.Range{
			Start: protocol.Position{Line: uint32(line - 1), Character: uint32(col - 1)},
			End:   protocol.Position{Line: uint32(endLine - 1), Character: uint32(endCol - 1)},
		}
		diags = append(diags, diag)
	}

	return diags
}

func (s *server) getLintDiags(doc *document) (diags []protocol.Diagnostic) {
	result, err := s.lintWithRecover(doc)
	if err != nil {
		log.Errorf("getLintDiags: %s: %v\n", errorRetrievingDocument, err)
	} else {
		for _, match := range errRegexp.FindAllStringSubmatch(result, -1) {
			line, col, endLine, endCol := 1, 1, 1, 1
			diag := protocol.Diagnostic{Source: "lint", Severity: protocol.SeverityWarning}

			if len(match) == 10 {
				if match[1] != "" {
					line, _ = strconv.Atoi(match[1])
					endLine = line + 1
				}
				if match[2] != "" {
					line, _ = strconv.Atoi(match[2])
					col, _ = strconv.Atoi(match[3])
					endLine = line
					endCol, _ = strconv.Atoi(match[4])
				}
				if match[5] != "" {
					line, _ = strconv.Atoi(match[5])
					col, _ = strconv.Atoi(match[6])
					endLine, _ = strconv.Atoi(match[7])
					endCol, _ = strconv.Atoi(match[8])
				}
			}

			diag.Message = match[9]

			diag.Range = protocol.Range{
				Start: protocol.Position{Line: uint32(line - 1), Character: uint32(col - 1)},
				End:   protocol.Position{Line: uint32(endLine - 1), Character: uint32(endCol - 1)},
			}
			diags = append(diags, diag)
		}
	}

	return diags
}

func (s *server) lintWithRecover(doc *document) (result string, err error) {
	defer func() {
		if r := recover(); r != nil {
			err = fmt.Errorf("error linting: %v", r)
		}
	}()

	vm, err := s.getVM(doc.item.URI.SpanURI().Filename())
	if err != nil {
		return "", err
	}

	buf := &bytes.Buffer{}
	linter.LintSnippet(vm, buf, doc.item.URI.SpanURI().Filename(), doc.item.Text)
	result = buf.String()

	return
}
