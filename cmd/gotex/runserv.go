package main

import (
	"bufio"
	"embed"
	"flag"
	"io/fs"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/bennof/gotex/gotex"
)

//go:embed www/*
var wwwFiles embed.FS

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

	p, err := gotex.NewProcessor(binpath, treepath)
	if err != nil {
		log.Fatalln(err)
	}

	// mk tempdir
	err = os.MkdirAll(*tempdir, os.ModePerm)
	if err != nil {
		log.Fatalln(err)
	}
	defer func() {
		os.RemoveAll(*tempdir)
	}()

	s := NewSimpleServer()

	sub, err := fs.Sub(wwwFiles, "www")
	if err != nil {
		log.Fatalln(err)
	}

	s.Handle("/", http.FileServer(http.FS(sub)))

	s.HandleFunc("/info", func(w http.ResponseWriter, r *http.Request) {
		s.WriteJSON(w, http.StatusOK, map[string]any{
			"server":    "gotex",
			"version":   "0.01",
			"mode":      "latex",
			"extension": []string{"bflatex", "edolatex"},
		})
	})

	s.HandleFunc("/build", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, http.StatusText(http.StatusMethodNotAllowed), http.StatusMethodNotAllowed)
			return
		}

		// gen id
		id, err := newID()
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		path := filepath.Join(*tempdir, id)

		// create and run processor with outputfile/{id}/
		err = os.MkdirAll(path, os.ModePerm)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		res, err := p.Process(r.Body, path)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		defer res.Reader.Close()

		// return stream
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
		err = res.Wait()
		if err != nil {
			_ = writeStreamMessage(w, streamMessage{Type: "error", Data: err.Error()})
			flusher.Flush()
			return
		}

		err = writeStreamMessage(w, streamMessage{Type: "done", URL: "/files/" + id})
		if err != nil {
			_ = writeStreamMessage(w, streamMessage{Type: "error", Data: err.Error()})
			flusher.Flush()
			return
		}
		flusher.Flush()
	})

	s.HandleFunc("/files/", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, http.StatusText(http.StatusMethodNotAllowed), http.StatusMethodNotAllowed)
			return
		}

		parts := strings.Split(strings.Trim(r.URL.Path, "/"), "/")
		if len(parts) != 2 || parts[0] != "files" || parts[1] == "" {
			http.Error(w, "invalid file path", http.StatusBadRequest)
			return
		}
		id := parts[len(parts)-1]

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

		w.Header().Set("Content-Type", "application/pdf")
		w.Header().Set("Content-Disposition", `attachment; filename="gotex.pdf"`)
		w.Header().Set("Cache-Control", "no-cache")
		err = sendFile(w, path)
		if err != nil {
			http.Error(w, "file not accessible", http.StatusInternalServerError)
			return
		}
		err = os.RemoveAll(filepath.Join(*tempdir, id))
		if err != nil {
			s.logger.Println(err)
		}
	})

	err = s.Listen(*port)
	if err != nil {
		log.Fatalln(err)
	}
}
