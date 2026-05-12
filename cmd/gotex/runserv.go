// Copyright (c) 2026 Benjamin Benno Falkner
// SPDX-License-Identifier: MIT

package main

import (
	"errors"
	"flag"
	"io"
	"log"
	"mime/multipart"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/bennof/gotex/gotex"
	"github.com/bennof/gotex/simpleserver"
	"github.com/bennof/gotex/simpleserver/session"
	"github.com/bennof/gotex/simpleserver/stream"
)

var (
	ErrUnsupportedFileType      = errors.New("unsupported file type")
	ErrSessionSizeLimitExceeded = errors.New("session size limit exceeded")
	ErrFileSizeLimitExceeded    = errors.New("file size limit exceeded")
	ErrFailedToOpenUpload       = errors.New("failed to open upload")
	ErrFailedToCreateFile       = errors.New("failed to create file")
	ErrFailedToSaveUpload       = errors.New("failed to save upload")
	ErrFailedToCheckSize        = errors.New("failed to check session size")
)

// runServe handles the "serve" subcommand.
// It initializes the tectonic processor, creates a temporary directory for build output,
// registers HTTP handlers, and starts the server on the given port.
func runServe(args []string) {

	// get wd
	wd, err := os.Getwd()
	if err != nil {
		log.Fatalln(err)
	}

	flags := flag.NewFlagSet("serve", flag.ExitOnError)
	target := flags.String("d", "", "tectonic installation directory")
	binary := flags.String("binary", "", "tectonic binary directory")
	tree := flags.String("texmf", "", "TeXMF tree directory")
	tempdir := flags.String("t", filepath.Join(wd, "tmp"), "working directory for build sessions")
	port := flags.String("addr", ":8080", "listen address (e.g. :8080)")
	maxFileSize := flags.Int64("max-file", 2<<20, "max uploaded file size in bytes (default 2 MiB)")
	maxSessionSize := flags.Int64("max-session-size", 20<<20, "max total session size in bytes (default 20 MiB)")
	maxSession := flags.Int("max-sessions", 20, "max concurrent sessions")
	storeTtl := flags.Duration("ttl", 15*time.Minute, "session lifetime (e.g. 15m, 1h)")
	flags.Parse(args)

	binpath, treepath, err := resolvePaths(*target, *binary, *tree)
	if err != nil {
		log.Fatalln(err)
	}

	// Initialize the tectonic processor, downloading the binary if necessary.
	p, err := gotex.NewProcessor(binpath, treepath)
	if err != nil {
		log.Fatalln(err)
	}

	// Create the temporary build directory and remove it on exit.
	err = os.MkdirAll(*tempdir, 0700)
	if err != nil {
		log.Fatalln(err)
	}
	defer func() {
		os.RemoveAll(*tempdir)
	}()

	s := simpleserver.NewSimpleServer(simpleserver.WithConfig(simpleserver.Config{Addr: simpleserver.NormalizeAddr(*port)}))

	// create session manager
	ps, err := session.NewPoolStore[*session.SessionFS](*maxSession, *storeTtl, session.NewSessionFS(*tempdir, *maxSessionSize))
	if err != nil {
		log.Fatalln(err)
	}
	defer ps.Close()

	// Serve embedded static files from the www directory.
	s.Handle("/", http.FileServer(http.FS(wwwFiles)))

	// /info returns basic metadata about the server and its capabilities.
	s.HandleFunc("/info", func(w http.ResponseWriter, r *http.Request) {
		s.WriteJSON(w, http.StatusOK, map[string]any{
			"server":    ServerName,
			"version":   Version,
			"mode":      ServerMode,
			"extension": ServerExtensions,
			"capabilities": map[string]any{
				"tikz":        true,
				"assets":      true,
				"streaming":   true,
				"max_file":    *maxFileSize,
				"max_session": *maxSessionSize,
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
	})

	s.HandleFunc("/new", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, http.StatusText(http.StatusMethodNotAllowed), http.StatusMethodNotAllowed)
			return
		}

		sfs, err := ps.Create(r.Context())
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		s.WriteJSON(w, http.StatusOK, map[string]any{
			"id":   sfs.ID(),
			"path": "/files/" + sfs.ID(),
		})
	})

	// /build accepts a POST request with a LaTeX document body,
	// compiles it using tectonic, and streams build log lines as NDJSON.
	// On success it sends a final "done" message with the URL to retrieve the PDF.
	s.HandleFunc("/build", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, http.StatusText(http.StatusMethodNotAllowed), http.StatusMethodNotAllowed)
			return
		}

		// Generate a unique ID and create a dedicated output directory for this build.
		sfs, err := ps.Create(r.Context())
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		// Run the tectonic processor with the request body as input.
		res, err := p.Process(r.Body, sfs.Path())
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		defer res.Reader.Close()

		xstream, err := stream.NewSimpleXndJSON(w)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		// stream process output
		err = xstream.StreamReader(res.Reader)
		if err != nil {
			log.Println(err)
		}

		if err = res.Wait(); err != nil {
			xstream.Error(err)
			log.Println(err)
			return
		}
		err = xstream.Write(stream.SimpleXndJSONMsg{Type: "done", URL: "/files/" + sfs.ID()})
		if err != nil {
			log.Println(err)
			return
		}
	})

	s.HandleFunc("/build/", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, http.StatusText(http.StatusMethodNotAllowed), http.StatusMethodNotAllowed)
			return
		}

		parts := strings.Split(strings.Trim(r.URL.Path, "/"), "/")
		if len(parts) != 2 || parts[0] != "build" || parts[1] == "" {
			http.Error(w, "invalid build path", http.StatusBadRequest)
			return
		}
		id := parts[1]

		sfs, ok := ps.Get(id)
		if !ok {
			http.Error(w, "session not found", http.StatusBadRequest)
			return
		}

		info, err := sfs.Stat("")
		if err != nil || !info.IsDir() {
			http.Error(w, "invalid session path", http.StatusBadRequest)
			if err = ps.Delete(id); err != nil {
				log.Println(err)
			}
			return
		}

		res, err := p.Process(r.Body, sfs.Path())
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		defer res.Reader.Close()

		xstream, err := stream.NewSimpleXndJSON(w)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		// stream process output
		err = xstream.StreamReader(res.Reader)
		if err != nil {
			log.Println(err)
		}

		if err = res.Wait(); err != nil {
			xstream.Error(err)
			log.Println(err)
			return
		}
		err = xstream.Write(stream.SimpleXndJSONMsg{Type: "done", URL: "/files/" + id})
		if err != nil {
			log.Println(err)
			return
		}
	})

	// /files/{id} serves the compiled PDF for the given build ID.
	// By default the PDF is served inline for direct display in the browser.
	// If the query parameter "dl=1" is set, it is sent as a download attachment instead.
	// The file is deleted from the temporary directory after it is served.
	s.HandleFunc("/files/", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, http.StatusText(http.StatusMethodNotAllowed), http.StatusMethodNotAllowed)
			return
		}

		// Validate the URL structure: must be exactly /files/{id}.
		parts := strings.Split(strings.Trim(r.URL.Path, "/"), "/")
		if len(parts) != 2 || parts[0] != "files" || parts[1] == "" {
			http.Error(w, "invalid file path", http.StatusBadRequest)
			return
		}
		id := parts[1]

		sfs, ok := ps.Get(id)
		if !ok {
			http.Error(w, "session not found", http.StatusNotFound)
			return
		}

		// Verify the output PDF exists and is not a directory.
		info, err := sfs.Stat("texput.pdf")
		if err != nil {
			http.Error(w, "file not found", http.StatusBadRequest)
			return
		}
		if info.IsDir() {
			http.Error(w, "invalid file path", http.StatusBadRequest)
			return
		}

		// Determine content disposition: download if "dl=1", inline otherwise.
		download := r.URL.Query().Get("dl") == "1"
		disposition := "inline"
		if download {
			disposition = `attachment; filename="gotex.pdf"`
		}

		// Stream the PDF to the client with appropriate headers.
		w.Header().Set("Content-Type", "application/pdf")
		w.Header().Set("Content-Disposition", disposition)
		w.Header().Set("Cache-Control", "no-cache")
		if err = simpleserver.SendFile(w, sfs.PathJoin("texput.pdf")); err != nil {
			http.Error(w, "file not accessible", http.StatusInternalServerError)
			return
		}

		// Clean up the build directory only after an explicit download.
		if download {
			if err = ps.Delete(id); err != nil {
				s.Log(err.Error())
			}
		}
	})

	s.HandleFunc("/assets/", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, http.StatusText(http.StatusMethodNotAllowed), http.StatusMethodNotAllowed)
			return
		}

		parts := strings.Split(strings.Trim(r.URL.Path, "/"), "/")
		if len(parts) != 2 || parts[0] != "assets" || parts[1] == "" {
			http.Error(w, "invalid asset path", http.StatusBadRequest)
			return
		}
		id := parts[1]

		sfs, ok := ps.Get(id)
		if !ok {
			http.Error(w, "session not found", http.StatusNotFound)
			return
		}

		r.Body = http.MaxBytesReader(w, r.Body, *maxFileSize+(1<<20))
		if err := r.ParseMultipartForm(*maxFileSize + (1 << 20)); err != nil {
			http.Error(w, "invalid upload", http.StatusBadRequest)
			return
		}

		if len(r.MultipartForm.File) == 0 {
			http.Error(w, "missing upload file", http.StatusBadRequest)
			return
		}

		type result struct {
			Name string `json:"name"`
			Path string `json:"path"`
			Size int64  `json:"size"`
		}
		var results []result

		for _, headers := range r.MultipartForm.File {
			for _, header := range headers {
				name, written, err := handleAssetUpload(sfs, header, *maxFileSize, *maxSessionSize)
				if err != nil {
					switch {
					case errors.Is(err, ErrSessionSizeLimitExceeded), errors.Is(err, ErrFileSizeLimitExceeded):
						http.Error(w, err.Error(), http.StatusRequestEntityTooLarge)
					case errors.Is(err, ErrUnsupportedFileType):
						http.Error(w, err.Error(), http.StatusBadRequest)
					default:
						http.Error(w, err.Error(), http.StatusInternalServerError)
					}
					return
				}
				results = append(results, result{
					Name: name,
					Path: "/assets/" + id + "/" + name,
					Size: written,
				})
			}
		}

		s.WriteJSON(w, http.StatusOK, map[string]any{
			"id":    id,
			"files": results,
		})
	})

	if err = s.Listen(); err != nil {
		log.Fatalln(err)
	}
}

func handleAssetUpload(sfs *session.SessionFS, header *multipart.FileHeader, maxFileSize, maxSessionSize int64) (string, int64, error) {
	name := filepath.Base(header.Filename)
	ext := strings.ToLower(filepath.Ext(name))
	switch ext {
	case ".jpg", ".jpeg", ".png", ".pdf":
	default:
		return "", 0, ErrUnsupportedFileType
	}

	currentSize, err := sfs.DirSize()
	if err != nil {
		log.Println("assets: DirSize:", err)
		return "", 0, ErrFailedToCheckSize
	}

	remainingSize := maxSessionSize - currentSize
	if remainingSize <= 0 {
		return "", 0, ErrSessionSizeLimitExceeded
	}

	limit := maxFileSize
	if remainingSize < limit {
		limit = remainingSize
	}

	src, err := header.Open()
	if err != nil {
		log.Println("assets: open upload:", err)
		return "", 0, ErrFailedToOpenUpload
	}
	defer src.Close()

	targetPath := sfs.PathJoin(name)
	dst, err := os.Create(targetPath)
	if err != nil {
		log.Println("assets: create file:", err)
		return "", 0, ErrFailedToCreateFile
	}

	written, copyErr := io.Copy(dst, io.LimitReader(src, limit+1))
	closeErr := dst.Close()
	if copyErr != nil {
		_ = os.Remove(targetPath)
		log.Println("assets: copy:", copyErr)
		return "", 0, ErrFailedToSaveUpload
	}
	if closeErr != nil {
		_ = os.Remove(targetPath)
		log.Println("assets: close:", closeErr)
		return "", 0, ErrFailedToSaveUpload
	}
	if written > limit {
		_ = os.Remove(targetPath)
		if limit < maxSessionSize {
			return "", 0, ErrSessionSizeLimitExceeded
		}
		return "", 0, ErrFileSizeLimitExceeded
	}
	return name, written, nil
}
