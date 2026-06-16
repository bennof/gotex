// Copyright (c) 2026 Benjamin Benno Falkner
// SPDX-License-Identifier: MIT
// build.go implements the local gotex build command and resolves the runtime dotpath.
package main

import (
	"flag"
	"log"
	"os"

	"github.com/bennof/gotex/tex"
)

func runBuild(args []string) {
	fs := flag.NewFlagSet("build", flag.ExitOnError)
	outdir := fs.String("o", ".", "output directory")
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

	// Require at least one positional argument as the input file.
	if fs.NArg() < 1 {
		log.Fatalln("missing inputfile")
	}
	inputfile := fs.Arg(0)

	p, err := tex.NewProcessor(dotpath)
	if err != nil {
		log.Fatalln(err)
	}

	f, err := os.Open(inputfile)
	if err != nil {
		log.Fatalln("failed to open inputfile:", inputfile)
	}
	defer f.Close()

	if *outdir == "." {
		*outdir, err = os.Getwd()
		if err != nil {
			log.Fatalln(err)
		}
	}

	res, err := p.Process(f, *outdir)
	if err != nil {
		log.Fatalln("failed to load TeX environment:", err)
	}
	defer res.Reader.Close()

	if err := res.CopyTo(os.Stdout); err != nil {
		log.Fatalln("edotex process failed:", err)
	}
}

func resolvePath(explicit string, envValue string, fallback string) (string, error) {
	if explicit != "" {
		return explicit, nil
	}
	if envValue != "" {
		return envValue, nil
	}
	if fallback == "." || fallback == "" {
		return os.Getwd()
	}
	return fallback, nil
}
