// Copyright (c) 2026 Benjamin Benno Falkner
// SPDX-License-Identifier: MIT
// version.go exposes server metadata and the /info endpoint payload.
package main

import (
	"net/http"
	"time"

	"github.com/bennof/gotex/simpleserver"
)

const Version = "0.08"
const ServerName = "gotex"
const ServerMode = "tectonic - XeTeX - laTeX"

var ServerExtensions = []string{"bflatex", "golatex"}

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
				"method":      "POST",
				"path":        "/session",
				"description": "Creates a new session with a dedicated working directory for uploads and compilation.",
				"response":    `{"id": "<session-id>", "path": "/session/<session-id>"}`,
			},
			{
				"method":      "GET",
				"path":        "/session/{id}",
				"description": "Returns the compiled PDF for the given session. Add query parameter dl=1 to force download and remove the session afterwards.",
				"response":    "PDF file (Content-Type: application/pdf)",
			},
			{
				"method":      "DELETE",
				"path":        "/session/{id}",
				"description": "Deletes a session and removes its working directory and generated files.",
				"response":    "Empty response with status 204 on success.",
			},
			{
				"method":      "POST",
				"path":        "/session/{id}/upload",
				"description": "Uploads one asset file into the session working directory. Uploaded files can be referenced from the LaTeX source.",
				"body":        "multipart/form-data with a single field named file. Allowed types: jpg, jpeg, png, pdf.",
				"response":    `{"id":"<session-id>","name":"<filename>","path":"<server-path>","bytes":<bytes>}`,
			},
			{
				"method":      "POST",
				"path":        "/session/{id}/compile",
				"description": "Compiles a LaTeX document inside an existing session. Use this after uploading any required assets.",
				"body":        "LaTeX source as plain text in the request body.",
				"response":    "NDJSON stream with build log messages and a final done message whose url points to /session/{id}.",
			},
		},
	})
}
