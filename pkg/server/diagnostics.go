package server

import (
	"bytes"
	"context"
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/google/go-jsonnet/linter"
	position "github.com/grafana/jsonnet-language-server/pkg/position_conversion"
	"github.com/jdbaldry/go-language-server-protocol/lsp/protocol"
	log "github.com/sirupsen/logrus"
)

var (
	// errRegexp matches the various Jsonnet location formats in errors.
	// 1. file:line msg
	// 2. file:line:col msg
	// 3. file:line:col-endCol msg
	// 4. file:(line:col)-(endLine:endCol) msg
	// https://regex101.com/r/tL5VWi/2
	errRegexp = regexp.MustCompile(`/[^:]*:` +
		`(?:(?P<startLine1>\d+)` +
		`|(?P<startLine2>\d+):(?P<startCol2>\d+)` +
		`|(?:(?P<startLine3>\d+):(?P<startCol3>\d+)-(?P<endCol3>\d+))` +
		`|(?:\((?P<startLine4>\d+):(?P<startCol4>\d+)\)-\((?P<endLine4>\d+):(?P<endCol4>\d+))\))` +
		`\s(?P<message>.*)`)
)

func parseErrRegexpMatch(match []string) (string, protocol.Range) {
	get := func(name string) string {
		idx := errRegexp.SubexpIndex(name)
		if len(match) <= idx {
			return ""
		}
		return match[idx]
	}

	message, line, col, endLine, endCol := "", 1, 1, 1, 1
	if len(match) > 1 {
		if lineStr := get("startLine1"); lineStr != "" {
			line, _ = strconv.Atoi(lineStr)
			endLine = line
		}

		if lineStr := get("startLine2"); lineStr != "" {
			line, _ = strconv.Atoi(lineStr)
			endLine = line
			col, _ = strconv.Atoi(get("startCol2"))
			endCol = col
		}

		if lineStr := get("startLine3"); lineStr != "" {
			line, _ = strconv.Atoi(lineStr)
			endLine = line
			col, _ = strconv.Atoi(get("startCol3"))
			endCol, _ = strconv.Atoi(get("endCol3"))
		}

		if lineStr := get("startLine4"); lineStr != "" {
			line, _ = strconv.Atoi(lineStr)
			endLine, _ = strconv.Atoi(get("endLine4"))
			col, _ = strconv.Atoi(get("startCol4"))
			endCol, _ = strconv.Atoi(get("endCol4"))
		}

		message = get("message")
	}

	return message, position.NewProtocolRange(line-1, col-1, endLine-1, endCol-1)
}

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
					if s.configuration.EnableLintDiagnostics {
						go func() {
							lintChannel <- s.getLintDiags(doc)
						}()
					}

					diags = append(diags, <-evalChannel...)

					if s.configuration.EnableLintDiagnostics {
						err = s.client.PublishDiagnostics(context.Background(), &protocol.PublishDiagnosticsParams{
							URI:         uri,
							Diagnostics: diags,
						})
						if err != nil {
							log.Errorf("publishDiagnostics: unable to publish diagnostics: %v\n", err)
						}

						diags = append(diags, <-lintChannel...)
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
	if doc.err == nil && s.configuration.EnableEvalDiagnostics {
		vm := s.getVM(doc.item.URI.SpanURI().Filename())
		doc.val, doc.err = vm.EvaluateAnonymousSnippet(doc.item.URI.SpanURI().Filename(), doc.item.Text)
	}

	if doc.err != nil {
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

		message, rang := parseErrRegexpMatch(match)
		if runtimeErr {
			diag.Message = doc.err.Error()
			diag.Severity = protocol.SeverityWarning
		} else {
			diag.Message = message
			diag.Severity = protocol.SeverityError
		}

		diag.Range = rang
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
			diag := protocol.Diagnostic{Source: "lint", Severity: protocol.SeverityWarning}
			diag.Message, diag.Range = parseErrRegexpMatch(match)
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

	vm := s.getVM(doc.item.URI.SpanURI().Filename())

	buf := &bytes.Buffer{}
	linter.LintSnippet(vm, buf, []linter.Snippet{
		{FileName: doc.item.URI.SpanURI().Filename(), Code: doc.item.Text},
	})
	result = buf.String()

	return
}
