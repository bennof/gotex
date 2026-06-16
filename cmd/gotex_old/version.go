// Copyright (c) 2026 Benjamin Benno Falkner
// SPDX-License-Identifier: MIT
package main

import (
	"net/http"
	"time"

	"github.com/bennof/gotex/simpleserver"
)

const Version = "0.02"
const ServerName = "gotex"
const ServerMode = "latex"

var ServerExtensions = []string{"bflatex", "edolatex"}

func writeInfo(s *simpleserver.SimpleServer, w http.ResponseWriter, maxFileSize int64, maxSessionSize int64, storeTtl time.Duration) {
	s.WriteJSON(w, http.StatusOK, map[string]any{
		"server":    ServerName,
		"version":   Version,
		"mode":      ServerMode,
		"extension": ServerExtensions,
		"capabilities": map[string]any{
			"tikz":        true,
			"assets":      true,
			"streaming":   true,
			"max_file":    maxFileSize,
			"max_session": maxSessionSize,
			"ttl":         storeTtl.String(),
		},
		"hints": []string{
			"Prefer TikZ over external images where possible",
			"Use \\includegraphics only for photographs and scans",
			"Assets support: jpg, jpeg, png, pdf",
		},
		"endpoints": []map[string]any{
			{
				"method":      "GET",
				"path":        "/info",
				"description": "Returns server capabilities and endpoint documentation.",
				"response":    "JSON object with server info, capabilities, hints and endpoints.",
			},
			{
				"method":      "GET",
				"path":        "/new",
				"description": "Creates a new session with a dedicated working directory. Use this before uploading assets or building with /build/{id}.",
				"response":    `{"id": "<session-id>", "path": "/files/<session-id>"}`,
			},
			{
				"method":      "POST",
				"path":        "/build",
				"description": "Compiles a LaTeX document from the request body. Creates a temporary session automatically. Use when no assets are needed.",
				"body":        "LaTeX source as plain text (Content-Type: text/plain)",
				"response":    "NDJSON stream. Each line is a JSON object with field 'type': 'log' (build output), 'error' (build error), or 'done' (url field contains PDF path).",
			},
			{
				"method":      "POST",
				"path":        "/build/{id}",
				"description": "Compiles a LaTeX document in an existing session. Use after uploading assets via /assets/{id}. The session working directory is available to the compiler.",
				"body":        "LaTeX source as plain text (Content-Type: text/plain)",
				"response":    "NDJSON stream. Same format as /build.",
			},
			{
				"method":      "GET",
				"path":        "/files/{id}",
				"description": "Returns the compiled PDF for the given session. Add query parameter dl=1 to force download. The session is deleted after a download.",
				"response":    "PDF file (Content-Type: application/pdf)",
			},
			{
				"method":      "POST",
				"path":        "/assets/{id}",
				"description": "Uploads one or more files to the session working directory. Uploaded files can be referenced in LaTeX via \\includegraphics{filename}.",
				"body":        "multipart/form-data with one or more files. Allowed types: jpg, jpeg, png, pdf.",
				"response":    `{"id": "<session-id>", "files": [{"name": "<filename>", "path": "/assets/<id>/<name>", "size": <bytes>}]}`,
			},
		},
	})
}
