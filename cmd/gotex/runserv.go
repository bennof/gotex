// Copyright (c) 2026 Benjamin Benno Falkner
// SPDX-License-Identifier: MIT

package main

import (
	"bufio"
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
)

const MaxImageSize int64 = 10 << 20
const MaxSessionSize int64 = 50 << 20

// runServe handles the "serve" subcommand.
// It initializes the tectonic processor, creates a temporary directory for build output,
// registers HTTP handlers, and starts the server on the given port.
func runServe(args []string) {
	wd, err := os.Getwd()
	if err != nil {
		log.Fatalln(err)
	}

	flags := flag.NewFlagSet("serve", flag.ExitOnError)
	target := flags.String("t", "", "target directory")
	binary := flags.String("b", "", "binary directory")
	tree := flags.String("m", "", "TeXMF Tree directory")
	tempdir := flags.String("d", filepath.Join(wd, "tmp"), "temporary directory")
	port := flags.String("p", ":8080", "server port")
	//MaxImageSize_ := flags.String("MaxImageSize", "", "target directory")
	//MaxSessionSize_ := flags.String("MaxSessionSize", "", "target directory")
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
	err = os.MkdirAll(*tempdir, os.ModePerm)
	if err != nil {
		log.Fatalln(err)
	}
	defer func() {
		os.RemoveAll(*tempdir)
	}()

	s := simpleserver.NewSimpleServer(simpleserver.WithConfig(simpleserver.Config{Addr: simpleserver.NormalizeAddr(*port)}))

	newBuildJob := func() (string, string, error) {
		id, err := newID()
		if err != nil {
			return "", "", err
		}

		path := filepath.Join(*tempdir, id)
		if err = os.MkdirAll(path, os.ModePerm); err != nil {
			return "", "", err
		}

		return id, path, nil
	}

	// Serve embedded static files from the www directory.
	//sub, err := fs.Sub(wwwFiles, "www")
	//if err != nil {
	//	log.Fatalln(err)
	//}
	s.Handle("/", http.FileServer(http.FS(wwwFiles)))

	// /info returns basic metadata about the server and its capabilities.
	s.HandleFunc("/info", func(w http.ResponseWriter, r *http.Request) {
		s.WriteJSON(w, http.StatusOK, map[string]any{
			"server":    ServerName,
			"version":   Version,
			"mode":      ServerMode,
			"extension": ServerExtensions,
		})
	})

	s.HandleFunc("/new", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, http.StatusText(http.StatusMethodNotAllowed), http.StatusMethodNotAllowed)
			return
		}

		id, _, err := newBuildJob()
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		s.WriteJSON(w, http.StatusOK, map[string]any{
			"id":   id,
			"path": "/files/" + id,
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
		id, path, err := newBuildJob()
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		// Run the tectonic processor with the request body as input.
		res, err := p.Process(r.Body, path)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		defer res.Reader.Close()

		// Set up streaming NDJSON response headers.
		w.Header().Set("Content-Type", "application/x-ndjson")
		w.Header().Set("Cache-Control", "no-cache")
		flusher, ok := w.(http.Flusher)
		if !ok {
			http.Error(w, "streaming unsupported", http.StatusInternalServerError)
			return
		}

		// Stream each line of tectonic output as a "log" message to the client.
		scanner := bufio.NewScanner(res.Reader)
		for scanner.Scan() {
			line := scanner.Text()
			err = writeStreamMessage(w, streamMessage{Type: "log", Data: line})
			if err != nil {
				_ = writeStreamMessage(w, streamMessage{Type: "error", Data: err.Error()})
				flusher.Flush()
				return
			}
			flusher.Flush()
		}
		if err := scanner.Err(); err != nil {
			_ = writeStreamMessage(w, streamMessage{Type: "error", Data: err.Error()})
			flusher.Flush()
			return
		}

		// Wait for the tectonic process to finish and check for errors.
		if err = res.Wait(); err != nil {
			_ = writeStreamMessage(w, streamMessage{Type: "error", Data: err.Error()})
			flusher.Flush()
			return
		}

		// Send the final "done" message with the URL to retrieve the PDF.
		err = writeStreamMessage(w, streamMessage{Type: "done", URL: "/files/" + id})
		if err != nil {
			_ = writeStreamMessage(w, streamMessage{Type: "error", Data: err.Error()})
		}
		flusher.Flush()
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
		path := filepath.Join(*tempdir, id)

		info, err := os.Stat(path)
		if err != nil {
			http.Error(w, "session not found", http.StatusBadRequest)
			return
		}
		if !info.IsDir() {
			http.Error(w, "invalid session path", http.StatusBadRequest)
			return
		}

		res, err := p.Process(r.Body, path)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		defer res.Reader.Close()

		w.Header().Set("Content-Type", "application/x-ndjson")
		w.Header().Set("Cache-Control", "no-cache")
		flusher, ok := w.(http.Flusher)
		if !ok {
			http.Error(w, "streaming unsupported", http.StatusInternalServerError)
			return
		}

		scanner := bufio.NewScanner(res.Reader)
		for scanner.Scan() {
			line := scanner.Text()
			err = writeStreamMessage(w, streamMessage{Type: "log", Data: line})
			if err != nil {
				_ = writeStreamMessage(w, streamMessage{Type: "error", Data: err.Error()})
				flusher.Flush()
				return
			}
			flusher.Flush()
		}
		if err := scanner.Err(); err != nil {
			_ = writeStreamMessage(w, streamMessage{Type: "error", Data: err.Error()})
			flusher.Flush()
			return
		}

		if err = res.Wait(); err != nil {
			_ = writeStreamMessage(w, streamMessage{Type: "error", Data: err.Error()})
			flusher.Flush()
			return
		}

		err = writeStreamMessage(w, streamMessage{Type: "done", URL: "/files/" + id})
		if err != nil {
			_ = writeStreamMessage(w, streamMessage{Type: "error", Data: err.Error()})
		}
		flusher.Flush()
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
		id := parts[len(parts)-1]

		// Verify the output PDF exists and is not a directory.
		path := filepath.Join(*tempdir, id, "texput.pdf")
		info, err := os.Stat(path)
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
		if err = sendFile(w, path); err != nil {
			http.Error(w, "file not accessible", http.StatusInternalServerError)
			return
		}

		// Clean up the build directory only after an explicit download.
		if download {
			if err = os.RemoveAll(filepath.Join(*tempdir, id)); err != nil {
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

		sessionDir := filepath.Join(*tempdir, id)
		info, err := os.Stat(sessionDir)
		if err != nil {
			http.Error(w, "session not found", http.StatusBadRequest)
			return
		}
		if !info.IsDir() {
			http.Error(w, "invalid session path", http.StatusBadRequest)
			return
		}

		currentSize, err := dirSize(sessionDir)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		if currentSize >= MaxSessionSize {
			http.Error(w, "session size limit exceeded", http.StatusRequestEntityTooLarge)
			return
		}

		r.Body = http.MaxBytesReader(w, r.Body, MaxImageSize+(1<<20))
		if err = r.ParseMultipartForm(MaxImageSize + (1 << 20)); err != nil {
			http.Error(w, "invalid upload", http.StatusBadRequest)
			return
		}

		header := firstUploadedFile(r.MultipartForm)
		if header == nil {
			http.Error(w, "missing upload file", http.StatusBadRequest)
			return
		}

		name := filepath.Base(header.Filename)
		ext := strings.ToLower(filepath.Ext(name))
		switch ext {
		case ".jpg", ".jpeg", ".png", ".pdf":
		default:
			http.Error(w, "unsupported file type", http.StatusBadRequest)
			return
		}

		remainingSize := MaxSessionSize - currentSize
		if remainingSize <= 0 {
			http.Error(w, "session size limit exceeded", http.StatusRequestEntityTooLarge)
			return
		}

		limit := MaxImageSize
		if remainingSize < limit {
			limit = remainingSize
		}

		src, err := header.Open()
		if err != nil {
			http.Error(w, "failed to open upload", http.StatusInternalServerError)
			return
		}
		defer src.Close()

		targetPath := filepath.Join(sessionDir, name)
		dst, err := os.Create(targetPath)
		if err != nil {
			http.Error(w, "failed to save upload", http.StatusInternalServerError)
			return
		}

		written, copyErr := io.Copy(dst, io.LimitReader(src, limit+1))
		closeErr := dst.Close()
		if copyErr != nil {
			_ = os.Remove(targetPath)
			http.Error(w, "failed to save upload", http.StatusInternalServerError)
			return
		}
		if closeErr != nil {
			_ = os.Remove(targetPath)
			http.Error(w, "failed to save upload", http.StatusInternalServerError)
			return
		}
		if written > limit {
			_ = os.Remove(targetPath)
			if limit < MaxImageSize {
				http.Error(w, "session size limit exceeded", http.StatusRequestEntityTooLarge)
				return
			}
			http.Error(w, "image size limit exceeded", http.StatusRequestEntityTooLarge)
			return
		}

		s.WriteJSON(w, http.StatusOK, map[string]any{
			"id":   id,
			"name": name,
			"path": "/assets/" + id + "/" + name,
			"size": written,
		})
	})

	if err = s.Listen(); err != nil {
		log.Fatalln(err)
	}
}

// startCleanupLoop launches a background goroutine that periodically scans
// the session root and removes directories older than storeDuration.
// It uses the directory timestamp from os.Stat as the portable age marker.
func startCleanupLoop(root string, cleanupTimer, storeDuration time.Duration, logger *log.Logger) {
	go func() {
		ticker := time.NewTicker(cleanupTimer)
		defer ticker.Stop()

		for range ticker.C {
			cutoff := time.Now().Add(-storeDuration)

			entries, err := os.ReadDir(root)
			if err != nil {
				if logger != nil {
					logger.Printf("cleanup: cannot read %s: %v", root, err)
				}
				continue
			}

			for _, entry := range entries {
				if !entry.IsDir() {
					continue
				}

				path := filepath.Join(root, entry.Name())
				info, err := entry.Info()
				if err != nil {
					if logger != nil {
						logger.Printf("cleanup: cannot stat %s: %v", path, err)
					}
					continue
				}

				if info.ModTime().Before(cutoff) {
					if err := os.RemoveAll(path); err != nil {
						if logger != nil {
							logger.Printf("cleanup: cannot remove %s: %v", path, err)
						}
					}
				}
			}
		}
	}()
}

func firstUploadedFile(form *multipart.Form) *multipart.FileHeader {
	if form == nil {
		return nil
	}
	for _, headers := range form.File {
		if len(headers) > 0 {
			return headers[0]
		}
	}
	return nil
}

func dirSize(root string) (int64, error) {
	var size int64
	err := filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !info.IsDir() {
			size += info.Size()
		}
		return nil
	})
	if err != nil {
		return 0, err
	}
	return size, nil
}
