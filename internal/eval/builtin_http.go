// SPDX-License-Identifier: AGPL-3.0-or-later
// Copyright (c) 2023-2026 Nicholas R. Perez

package eval

import (
	"io"
	"net/http"
	"strings"
	"time"

	"nickandperla.net/losp/internal/expr"
)

// httpClient is the client used by the HTTP builtin. Package-level so tests
// can substitute a client if needed.
var httpClient = &http.Client{Timeout: 30 * time.Second}

func builtinHTTP(e *Evaluator, argsRaw string) (expr.Expr, error) {
	// HTTP method uri [headers] [data]
	// method  - HTTP method (GET, POST, PUT, DELETE, ...)
	// uri     - request URI
	// headers - newline-separated "Key: Value" pairs (may be empty)
	// data    - request body (may be empty)
	//
	// headers and data are single expression arguments. Multi-line values
	// can only arrive via retrieval (▲Name) — a literal multi-line block at
	// the call site would parse as separate arguments. See PRIMER.md.
	//
	// Returns the response body as text, or EMPTY for an empty body.
	// Request/transport errors return "ERROR: <message>".
	args, err := e.parseArgs(argsRaw)
	if err != nil {
		return nil, err
	}

	if len(args) < 2 {
		return expr.Empty{}, nil
	}

	method := strings.ToUpper(strings.TrimSpace(args[0]))
	uri := strings.TrimSpace(args[1])
	var headers, data string
	if len(args) >= 3 {
		headers = args[2]
	}
	if len(args) >= 4 {
		data = args[3]
	}

	var body io.Reader
	if data != "" {
		body = strings.NewReader(data)
	}

	req, err := http.NewRequest(method, uri, body)
	if err != nil {
		return expr.Stored{Body: "ERROR: " + err.Error()}, nil
	}

	for line := range strings.SplitSeq(headers, "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		key, value, found := strings.Cut(line, ":")
		if !found {
			continue
		}
		req.Header.Set(strings.TrimSpace(key), strings.TrimSpace(value))
	}

	resp, err := httpClient.Do(req)
	if err != nil {
		return expr.Stored{Body: "ERROR: " + err.Error()}, nil
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return expr.Stored{Body: "ERROR: " + err.Error()}, nil
	}

	text := strings.TrimSpace(string(respBody))
	if text == "" {
		return expr.Empty{}, nil
	}
	return expr.Stored{Body: text}, nil
}
