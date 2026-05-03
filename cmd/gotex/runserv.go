// Copyright (c) 2026 Benjamin Benno Falkner
// SPDX-License-Identifier: MIT

package main

import (
	"bufio"
	"flag"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/bennof/gotex/gotex"
)

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

	s := NewSimpleServer()

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

	// /build accepts a POST request with a LaTeX document body,
	// compiles it using tectonic, and streams build log lines as NDJSON.
	// On success it sends a final "done" message with the URL to retrieve the PDF.
	s.HandleFunc("/build", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, http.StatusText(http.StatusMethodNotAllowed), http.StatusMethodNotAllowed)
			return
		}

		// Generate a unique ID for this build job.
		id, err := newID()
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		path := filepath.Join(*tempdir, id)

		// Create a dedicated output directory for this build.
		err = os.MkdirAll(path, os.ModePerm)
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
		disposition := "inline"
		if r.URL.Query().Get("dl") == "1" {
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

		// Clean up the build directory after the file has been served.
		if err = os.RemoveAll(filepath.Join(*tempdir, id)); err != nil {
			s.logger.Println(err)
		}
	})

	if err = s.Listen(*port); err != nil {
		log.Fatalln(err)
	}
}
