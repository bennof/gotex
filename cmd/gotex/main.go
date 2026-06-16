// Copyright (c) 2026 Benjamin Benno Falkner
// SPDX-License-Identifier: MIT

package main

import (
	"flag"
	"fmt"
	"log"
	"os"

	"github.com/bennof/gotex/tex"
)

const VAR_GOTEX_PATH = "GOTEX_PATH"

// main dispatches the selected gotex subcommand.
func main() {
	if len(os.Args) < 2 {
		usage()
		os.Exit(1)
	}

	mode := os.Args[1]
	switch mode {
	case "build":
		runBuild(os.Args[2:])
	case "serve":
		runServe(os.Args[2:])
	case "clean":
		clean(os.Args[2:])
	default:
		usage()
		os.Exit(1)
	}
}

// usage prints CLI help text for gotex.
func usage() {
	fmt.Fprintf(os.Stderr, `Usage:
  gotex build [args] inputfile
  gotex serve [args]
  gotex clean [args]

Modes:
  build      Build a TeX document
  serve      Start the gotex server
  clean      Remove the gotex installation directory or dotpath
`)
}

// clean removes the local gotex installation directory or configured dotpath.
func clean(args []string) {
	fs := flag.NewFlagSet("clean", flag.ExitOnError)
	dotdir := fs.String("d", "", "dot directory (used as base for binary and TeXMF tree if not set individually)")
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

	err = os.RemoveAll(dotpath)
	if err != nil {
		log.Fatalln(err)
	}
}
