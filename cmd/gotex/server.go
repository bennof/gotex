// Copyright (c) 2026 Benjamin Benno Falkner
// SPDX-License-Identifier: MIT
// server.go implements the gotex HTTP server and session endpoints.
package main

import (
	"encoding/json"
	"flag"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/bennof/gotex/simpleserver"
	"github.com/bennof/gotex/simpleserver/handler"
	"github.com/bennof/gotex/simpleserver/session"
	"github.com/bennof/gotex/simpleserver/stream"
	"github.com/bennof/gotex/tex"
)

func runServe(args []string) {
	// get wd
	wd, err := os.Getwd()
	if err != nil {
		log.Fatalln(err)
	}

	fs := flag.NewFlagSet("build", flag.ExitOnError)
	dotdir := fs.String("d", "", "dot directory (used as base for binary and TeXMF tree if not set individually)")
	tempdir := fs.String("t", filepath.Join(wd, "tmp"), "working directory for build sessions")
	port := fs.String("addr", ":8080", "listen address (e.g. :8080)")
	maxFileSize := fs.Int64("max-file", 2<<20, "max uploaded file size in bytes (default 2 MiB)")
	maxSessionSize := fs.Int64("max-session-size", 20<<20, "max total session size in bytes (default 20 MiB)")
	maxSession := fs.Int("max-sessions", 20, "max concurrent sessions")
	storeTtl := fs.Duration("ttl", 15*time.Minute, "session lifetime (e.g. 15m, 1h)")
	fs.Parse(args)

	defaultDotPath, err := tex.DefaultDotPath()
	if err != nil {
		log.Fatalln("failed to resolve default dot directory:", err)
	}

	dotpath, err := resolvePath(*dotdir, os.Getenv(VAR_GOTEX_PATH), defaultDotPath)
	if err != nil {
		log.Fatalln(err)
	}
	log.Printf("using dotpath: %s", dotpath)

	p, err := tex.NewProcessor(dotpath)
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

	s.Handle("/", http.FileServer(http.FS(wwwFiles)))

	// /info returns basic metadata about the server and its capabilities.
	s.HandleFunc("/info", func(w http.ResponseWriter, r *http.Request) {
		writeInfo(s, w, *maxFileSize, *maxSessionSize, *storeTtl)
	})

	ts := &TexSessionHandler{
		urlbase:        "/session/",
		maxFileSize:    *maxFileSize,
		maxSessionSize: *maxSessionSize,
		spool:          ps,
		processor:      p,
	}

	s.HandleFunc("/session", func(w http.ResponseWriter, r *http.Request) {
		ts.handleSessionCreate(w, r)
	})

	s.HandleFunc("/session/{id}", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodDelete {
			ts.handleSessionDelete(w, r)
		} else {
			ts.handleSessionGet(w, r)
		}
	})

	s.HandleFunc("/session/{id}/upload", func(w http.ResponseWriter, r *http.Request) {
		ts.handleSessionUpload(w, r)
	})

	s.HandleFunc("/session/{id}/compile", func(w http.ResponseWriter, r *http.Request) {
		ts.handleSessionBuild(w, r)
	})

	if err = s.Listen(); err != nil {
		log.Fatalln(err)
	}
}

type TexSessionHandler struct {
	urlbase        string
	maxFileSize    int64
	maxSessionSize int64
	spool          *session.PoolStore[*session.SessionFS]
	processor      *tex.Processor
}

func (ts *TexSessionHandler) handleSessionCreate(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, http.StatusText(http.StatusMethodNotAllowed), http.StatusMethodNotAllowed)
		return
	}

	sfs, err := ts.spool.Create(r.Context())
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	if err := json.NewEncoder(w).Encode(map[string]any{
		"id":   sfs.ID(),
		"path": ts.urlbase + sfs.ID(),
	}); err != nil {
		log.Printf("write response: %v", err)
	}
}

func (ts *TexSessionHandler) handleSessionDelete(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodDelete {
		http.Error(w, http.StatusText(http.StatusMethodNotAllowed), http.StatusMethodNotAllowed)
		return
	}

	id := r.PathValue("id")

	sfs, ok := ts.spool.Get(id)
	if !ok {
		http.Error(w, "session not found", http.StatusNotFound)
		return
	}

	sfs.Close()
	ts.spool.Delete(id)

	w.WriteHeader(http.StatusNoContent)
}

func (ts *TexSessionHandler) handleSessionGet(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, http.StatusText(http.StatusMethodNotAllowed), http.StatusMethodNotAllowed)
		return
	}
	id := r.PathValue("id")

	sfs, ok := ts.spool.Get(id)
	if !ok {
		http.Error(w, "session not found", http.StatusNotFound)
		return
	}

	info, err := sfs.Stat("texput.pdf")
	if err != nil {
		http.Error(w, "file not found", http.StatusNotFound)
		return
	}
	if info.IsDir() {
		http.Error(w, "invalid file path", http.StatusConflict)
		return
	}

	disposition := "inline"
	if r.URL.Query().Get("dl") == "1" {
		disposition = `attachment; filename="edotex.pdf"`
	}

	w.Header().Set("Content-Type", "application/pdf")
	w.Header().Set("Content-Disposition", disposition)
	w.Header().Set("Cache-Control", "no-cache")
	if err = simpleserver.SendFile(w, sfs.PathJoin("texput.pdf")); err != nil {
		http.Error(w, "file not accessible", http.StatusInternalServerError)
		return
	}

	if r.URL.Query().Get("dl") == "1" {
		if err = ts.spool.Delete(id); err != nil {
			log.Println(err.Error())
		}
	}
}

func (ts *TexSessionHandler) handleSessionBuild(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, http.StatusText(http.StatusMethodNotAllowed), http.StatusMethodNotAllowed)
		return
	}
	id := r.PathValue("id")

	sfs, ok := ts.spool.Get(id)
	if !ok {
		http.Error(w, "session not found", http.StatusNotFound)
		return
	}

	res, err := ts.processor.Process(r.Body, sfs.Path())
	if err != nil {
		log.Println(err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer res.Reader.Close()

	xstream, err := stream.NewSimpleXndJSON(w)
	if err != nil {
		log.Println(err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	err = xstream.StreamReader(res.Reader)
	if err != nil {
		log.Println(err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}

	if err = res.Wait(); err != nil {
		xstream.Error(err)
		log.Println(err)
		return
	}

	err = xstream.Write(stream.SimpleXndJSONMsg{Type: "done", URL: ts.urlbase + sfs.ID()})
	if err != nil {
		log.Println(err)
		http.Error(w, "XStream failed", http.StatusInternalServerError)
		return
	}
}

func (ts *TexSessionHandler) handleSessionUpload(w http.ResponseWriter, r *http.Request) {
	if !handler.MethodIs(w, r, http.MethodPost) {
		return
	}
	id := r.PathValue("id")

	sfs, ok := ts.spool.Get(id)
	if !ok {
		http.Error(w, "session not found", http.StatusNotFound)
		return
	}

	r.Body = http.MaxBytesReader(w, r.Body, ts.maxFileSize+(1<<20))
	file, header, err := r.FormFile("file")
	if err != nil {
		http.Error(w, "missing upload file", http.StatusBadRequest)
		return
	}
	defer file.Close()

	if header.Size > ts.maxFileSize {
		http.Error(w, "max file size exceeded", http.StatusRequestEntityTooLarge)
		return
	}

	currentSize, err := sfs.DirSize()
	if err != nil {
		log.Println("assets: DirSize:", err)
		http.Error(w, "error accessing session folder", http.StatusInternalServerError)
		return
	}

	remainingSize := ts.maxSessionSize - currentSize
	if remainingSize <= 0 {
		http.Error(w, "max session size exceeded", http.StatusRequestEntityTooLarge)
		return
	}

	name := filepath.Base(header.Filename)
	ext := strings.ToLower(filepath.Ext(name))
	switch ext {
	case ".jpg", ".jpeg", ".png", ".pdf":
	default:
		http.Error(w, "file type not accepted", http.StatusUnsupportedMediaType)
		return
	}

	targetPath := sfs.PathJoin(name)
	dst, err := os.Create(targetPath)
	if err != nil {
		log.Println("assets: create file:", err)
		http.Error(w, "error creating resource", http.StatusInternalServerError)
		return
	}
	defer dst.Close()

	written, err := io.Copy(dst, io.LimitReader(file, header.Size+1))
	if err != nil {
		log.Println("assets: writing data:", err)
		http.Error(w, "error writing data", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	if err := json.NewEncoder(w).Encode(map[string]any{
		"id":    sfs.ID(),
		"name":  name,
		"bytes": written,
	}); err != nil {
		log.Printf("write response: %v", err)
	}
}
