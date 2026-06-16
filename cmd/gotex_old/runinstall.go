// Copyright (c) 2026 Benjamin Benno Falkner
// SPDX-License-Identifier: MIT

package main

import (
	"flag"
	"fmt"
	"log"

	"github.com/bennof/gotex/gotex"
)

// runInstall handles the "install" subcommand.
// It resolves the binary and TeXMF tree paths from the given flags,
// then initializes a Processor which downloads tectonic if not already present.
// On success it prints the environment variables the user should set.
func runInstall(args []string) {
	fs := flag.NewFlagSet("install", flag.ExitOnError)
	target := fs.String("t", "", "target directory (used as base for binary and TeXMF tree if not set individually)")
	binary := fs.String("b", "", "binary directory (overrides -t for the tectonic binary location)")
	tree := fs.String("m", "", "TeXMF tree directory (overrides -t for the TeX tree location)")
	fs.Parse(args)

	binpath, treepath, err := resolvePaths(*target, *binary, *tree)
	if err != nil {
		log.Fatalln(err)
	}

	// NewProcessor will download tectonic if the binary is not found in binpath.
	_, err = gotex.NewProcessor(binpath, treepath)
	if err != nil {
		log.Fatalln(err)
	}

	fmt.Println("Installation complete.")
	fmt.Println("Set the following environment variables as needed:")

	// If both paths are identical, a single GOTEX_PATH variable is sufficient.
	if treepath == binpath {
		fmt.Printf("export GOTEX_PATH=%q\n", treepath)
	} else {
		fmt.Printf("export GOTEX_BIN=%q\n", binpath)
		fmt.Printf("export GOTEX_TEXMF=%q\n", treepath)
	}
}
