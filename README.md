# GoTeX

GoTeX is a local TeX/Tectonic project with three layers:

- a simple shell setup for local builds
- a Go library for reusable build logic
- a small web interface backed by a local build server

The main focus is a custom `texmf` tree, KOMA-Script-based classes, and full local control over fonts, packages, and document classes.

## Goals

- use Tectonic locally and reproducibly
- keep custom classes, packages, and fonts inside the repository
- maintain German documents based on KOMA-Script
- reuse build logic from both shell scripts and Go code
- later process documents, images, and AI-generated LaTeX through a local server

## Project Layout

```text
.
├── tex.sh
├── tectonic
├── texmf/
│   ├── doc/latex/bflatex/
│   ├── fonts/
│   └── tex/latex/
│       ├── bflatex/
│       └── edolatex/
├── gotex/
│   ├── cmd.go
│   └── processor.go
├── cmd/gotex/
│   ├── main.go
│   ├── runinstall.go
│   ├── runserv.go
│   ├── server.go
│   ├── util.go
│   └── www/
│       ├── index.html
│       ├── app.js
│       └── styles.css
└── go.mod
```

## Shell Workflow

The shell script `tex.sh` is the simplest entry point.

### Installation

```sh
./tex.sh install
```

The script:

- downloads Tectonic
- creates `texmf/tex/latex/bflatex/fontconfig.tex`
- prepares font configuration for the local tree

### Compile

```sh
./tex.sh worksheet.tex
```

During compilation:

- all subdirectories under `texmf/` are added as `search-path` entries
- the local `.tectonic-cache` is used
- output is written to the directory of the input file

### Cleanup

```sh
./tex.sh clean
```

This removes typical build artifacts and the local Tectonic cache.

## Go Library

The `gotex` package contains the reusable build logic.

Important building blocks:

- `Download(...)`: downloads content as a stream
- `ReadFile(...)`: opens files as a stream
- `ShellCmdDir(...)`: runs commands with `stdin`/`stdout` piping
- `FileWriter(...)`: writes a stream to a file or to `stdout`
- `NewProcessor(...)`: creates a processor for a binary path and a `texmf` path
- `(*Processor).Process(...)`: starts Tectonic with all collected local search paths

When a `Processor` is created, it walks the local `texmf` tree and passes all discovered directories to Tectonic as `-Z search-path=...`.

That means the current approach is not limited to only two TeX directories anymore, but can load the full local tree.

## Go CLI

The CLI binary lives in `cmd/gotex`.

Run it with:

```sh
go run ./cmd/gotex
```

Available modes:

```sh
gotex build [args] inputfile
gotex install [args]
gotex serve [args]
```

### Build

```sh
go run ./cmd/gotex build worksheet.tex
go run ./cmd/gotex build -o out worksheet.tex
```

### Install

```sh
go run ./cmd/gotex install
```

After a successful installation, the CLI prints matching hints for:

- `GOTEX_PATH`
- `GOTEX_BIN`
- `GOTEX_TEXMF`

### Serve

```sh
go run ./cmd/gotex serve
```

Optional:

```sh
go run ./cmd/gotex serve -p 8080 -d ./tmp
```

## Web Interface

The small SPA lives in `cmd/gotex/www/` and is embedded into the binary using Go `embed`.

It currently provides:

- a text field for LaTeX source
- a log output area
- a `Build` button
- a `Download` button for the generated PDF

The browser sends the text via `POST /build` to the server. The server responds with an NDJSON stream containing messages such as:

```json
{"type":"log","data":"Running TeX ..."}
{"type":"done","url":"/files/<id>"}
```

The final PDF is then served through `GET /files/<id>`.

## TeX Structure

### `bflatex`

`texmf/tex/latex/bflatex/` contains the general base layer:

- `bfarticle.cls`
- `bfbook.cls`
- `bfletter.cls`
- `addfonts.sty`

### `edolatex`

`texmf/tex/latex/edolatex/` contains education-oriented extensions:

- `edoxbase.sty`
- `edoxexercise.sty`
- `edoxgrades.sty`
- `edoxworksheet.sty`
- `edoxarticle.cls`
- `edoxbook.cls`

### Fonts

Custom fonts are stored in the repository under:

- `texmf/fonts/opentype/...`
- `texmf/fonts/truetype/...`

The current set includes, among others:

- EB Garamond
- Libertinus
- Fira Sans
- Fira Math
- Inter
- Inter Display

## Examples and Documents

Under `texmf/doc/latex/bflatex/` you will find example and test files, including:

- `article.tex`
- `book.tex`
- `literature.bib`
- `tikz.tex`



## Cache and Local Runtime Data

The local Tectonic cache is stored in:

```text
.tectonic-cache/
```

Tectonic stores there:

- downloaded bundles
- resolved bundle contents
- format files (`.fmt`)

This cache is runtime material and not a logical part of the main source tree.

## Current State

The current state is intentionally local-first:

- local fonts
- a local `texmf` tree
- an embedded web interface
- Tectonic as a local binary

Useful future extensions include:

- multiple configurable search-path models for `texmf`
- image upload and processing on the server side
- AI-assisted generation of valid LaTeX
- stronger Python/data-workflow or server integration

## Notes

- The project is clearly designed for local usage.
- The server currently works with temporary build directories on purpose.
- The web interface is intentionally minimal and not yet a full document management system.
