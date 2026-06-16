// Copyright (c) 2026 Benjamin Benno Falkner
// SPDX-License-Identifier: MIT

package main

import (
	"flag"
	"log"
	"os"

	"github.com/bennof/gotex/gotex"
)

// runBuild handles the "build" subcommand.
// It compiles a LaTeX input file using tectonic and writes the resulting
// PDF output to the specified output directory.
func runBuild(args []string) {
	fs := flag.NewFlagSet("build", flag.ExitOnError)
	outdir := fs.String("o", ".", "output directory")
	target := fs.String("t", "", "target directory (used as base for binary and TeXMF tree if not set individually)")
	binary := fs.String("b", "", "binary directory (overrides -t for the tectonic binary location)")
	tree := fs.String("m", "", "TeXMF tree directory (overrides -t for the TeX tree location)")
	fs.Parse(args)

	binpath, treepath, err := resolvePaths(*target, *binary, *tree)
	if err != nil {
		log.Fatalln(err)
	}

	// Require at least one positional argument as the input file.
	if fs.NArg() < 1 {
		log.Fatalln("missing inputfile")
	}
	inputfile := fs.Arg(0)

	// Initialize the tectonic processor with the resolved paths.
	p, err := gotex.NewProcessor(binpath, treepath)
	if err != nil {
		log.Fatalln(err)
	}

	// Open the LaTeX input file as a pipeline source.
	input, err := gotex.ReadFile(inputfile)
	if err != nil {
		log.Fatalln("failed to open inputfile:", inputfile)
	}
	defer input.Reader.Close()

	// Resolve output directory to an absolute path if "." was given.
	if *outdir == "." {
		*outdir, err = os.Getwd()
		if err != nil {
			log.Fatalln(err)
		}
	}

	// Run the tectonic processor and stream the result to stdout.
	res, err := p.Process(input.Reader, *outdir)
	if err != nil {
		log.Fatalln("failed to load TeX environment:", err)
	}
	defer res.Reader.Close()

	// Write the tectonic output to stdout.
	err = gotex.FileWriter(res.Reader, "-")
	if err != nil {
		log.Fatalln(err)
	}

	// Wait for both pipeline stages to complete cleanly.
	if err := input.Wait(); err != nil {
		log.Fatalln(err)
	}
	if err := res.Wait(); err != nil {
		log.Fatalln(err)
	}
}
