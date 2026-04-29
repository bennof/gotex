package main

import (
	"flag"
	"fmt"
	"log"

	"github.com/bennof/gotex/gotex"
)

func runInstall(args []string) {
	fs := flag.NewFlagSet("install", flag.ExitOnError)
	target := fs.String("t", "", "target directory")
	binary := fs.String("b", "", "binary directory")
	tree := fs.String("m", "", "TeXMF Tree directory")
	fs.Parse(args)

	binpath, treepath, err := resolvePaths(*target, *binary, *tree)
	if err != nil {
		log.Fatalln(err)
	}

	_, err = gotex.NewProcessor(binpath, treepath)
	if err != nil {
		log.Fatalln(err)
	}

	fmt.Println("Installation complete.")
	fmt.Println("Set the following environment variables as needed:")

	if treepath == binpath {
		fmt.Printf("export GOTEX_PATH=%q\n", treepath)
	} else {
		fmt.Printf("export GOTEX_BIN=%q\n", binpath)
		fmt.Printf("export GOTEX_TEXMF=%q\n", treepath)
	}
}
