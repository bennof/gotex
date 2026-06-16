# GoTeX

GoTeX is a local TeX/Tectonic project with three layers:

- a shell-based local workflow for fast testing
- Go packages for reusable build and server logic
- a lightweight embedded web UI served from a local process

The project is centered on a repository-local `texmf` tree, KOMA-Script-based document classes, and full local control over fonts, packages, and document classes.

## Goals

- run Tectonic locally and reproducibly
- keep custom TeX classes, packages, and fonts inside the repository
- support German/KOMA-Script document workflows
- share build logic between shell scripts and Go code
- offer a small local server and editor interface for TeX development

## Project layout

```text
.
├── Makefile
├── edotex.mk
├── tex.sh
├── go.mod
├── texmf/
│   ├── doc/latex/bflatex/
│   ├── fonts/
│   │   ├── opentype/
│   │   └── truetype/
│   └── tex/latex/
│       ├── bflatex/
│       └── edolatex/
├── cmd/gotex/
│   ├── main.go
│   ├── build.go
│   ├── server.go
│   ├── version.go
│   └── www/
│       ├── index.html
│       ├── app.js
│       └── styles.css
├── tex/
│   ├── tex.go
│   ├── utils.go
│   ├── texmf.go
│   ├── pipe.go
│   ├── log.go
│   └── *.go (platform-specific binaries)
└── simpleserver/
    ├── server.go
    ├── config.go
    ├── fsutils.go
    ├── utils.go
    ├── statusrecorder.go
    ├── session/
    └── stream/
```

## Build and install

The repository includes two build entry points:

- `Makefile` for the default local workflow
- `edotex.mk` as an alternate makefile with the same core targets

Build the CLI binary:

```sh
make
```

Install the binary to the system path:

```sh
sudo make install
```

## Shell workflow with `tex.sh`

`tex.sh` is a small helper script for testing Tectonic with the local gotex runtime layout.
It uses `~/.gotex` by default and expects the Tectonic binary at `~/.gotex/bin/tectonic`.

### Install the runtime

```sh
./tex.sh install
```

This command downloads Tectonic into `~/.gotex/bin`, creates the local TeX tree directories, and writes a `fontconfig.tex` file into `~/.gotex/texmf/tex/latex/bflatex`.

### Compile a document

```sh
./tex.sh worksheet.tex
```

This runs Tectonic with:

- `TECTONIC_CACHE_DIR=~/.gotex/.tectonic-cache`
- all directories under `~/.gotex/texmf` as `-Z search-path=...`
- `-p` enabled
- output written to the input file directory by default

You can also pass arbitrary Tectonic arguments through the script:

```sh
./tex.sh --help
./tex.sh --shell-escape document.tex
```

### Cleanup

```sh
./tex.sh clean
```

This removes runtime log files and the local Tectonic cache under `~/.gotex`.

## Gotex CLI

The `cmd/gotex` command provides a reusable Go-based toolchain.

Build or run it with:

```sh
go run ./cmd/gotex build worksheet.tex
```

Key commands:

- `gotex build [args] <inputfile>`
- `gotex serve [args]`
- `gotex clean`

The CLI resolves the local dotpath from `GOTEX_PATH` and the platform-specific default path, then uses `tex.NewProcessor` to prepare Tectonic and the TeX tree.

## Go packages

### `tex`

The `tex` package contains the core runtime logic that:

- ensures the embedded Tectonic binary is available
- verifies the local TeX tree exists
- collects all `texmf` subdirectories as Tectonic `-Z search-path` arguments
- runs Tectonic with `TECTONIC_CACHE_DIR` set to the local cache folder

### `simpleserver`

The `simpleserver` package provides a small HTTP server framework with:

- request routing
- graceful shutdown
- session management
- streaming JSON responses

## Web interface

The web UI is stored in `cmd/gotex/www/` and embedded into the binary using Go `embed`.

It provides a minimal interface for editing LaTeX source and triggering builds through the local server.

## TeX structure

### `bflatex`

`texmf/tex/latex/bflatex/` contains the base LaTeX layer used for general documents:

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

Included fonts currently contain:

- EB Garamond
- Libertinus
- Fira Sans
- Fira Math
- Inter
- Inter Display

## Examples and documentation

Example and test files live under `texmf/doc/latex/bflatex/`, including:

- `article.tex`
- `book.tex`
- `literature.bib`
- `tikz.tex`

## Local cache and runtime data

The local Tectonic cache is stored under:

```text
~/.gotex/.tectonic-cache
```

It stores:

- downloaded bundles
- extracted package contents
- format files (`.fmt`)

This cache is runtime data and not part of the source tree.

## Current state

GoTeX is intentionally local-first:

- local fonts and packages stored in source control
- a repository-local `texmf` tree
- a self-contained local Tectonic runtime
- a minimal embedded web interface

Future improvements can include:

- configurable texmf search-path models
- asset upload and processing in the server
- AI-assisted LaTeX generation
- a richer editor and document workflow
