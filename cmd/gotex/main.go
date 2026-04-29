package main

import (
	"flag"
	"fmt"
	"log"
	"os"

	"github.com/bennof/gotex/gotex"
)

func main() {

	if len(os.Args) < 2 {
		usage()
		os.Exit(1)
	}

	mode := os.Args[1]

	switch mode {
	case "build":
		runBuild(os.Args[2:])
	case "install":
		runInstall(os.Args[2:])
	case "serve":
		runServe(os.Args[2:])
	default:
		usage()
		os.Exit(1)
	}

}

func usage() {
	fmt.Fprintf(os.Stderr, `Usage:
  gotex build [args] inputfile
  gotex install [args]
  gotex serve [args]

Modes:
  build      Build a TeX document
  install    Install tectonic and local runtime dependencies
  serve      Start the gotex server
`)
}

func runBuild(args []string) {
	fs := flag.NewFlagSet("build", flag.ExitOnError)
	outdir := fs.String("o", ".", "output directory")
	target := fs.String("t", "", "target directory")
	binary := fs.String("b", "", "binary directory")
	tree := fs.String("m", "", "TeXMF Tree directory")
	fs.Parse(args)

	binpath, treepath, err := resolvePaths(*target, *binary, *tree)
	if err != nil {
		log.Fatalln(err)
	}

	if fs.NArg() < 1 {
		log.Fatalln("missing inputfile")
	}

	inputfile := fs.Arg(0)

	p, err := gotex.NewProcessor(binpath, treepath)
	if err != nil {
		log.Fatalln(err)
	}

	input, err := gotex.ReadFile(inputfile)
	if err != nil {
		log.Fatalln("failed open inputfile:", inputfile)
	}
	defer input.Reader.Close()

	if *outdir == "." {
		*outdir, err = os.Getwd()
		if err != nil {
			log.Fatalln(err)
		}
	}

	res, err := p.Process(input.Reader, *outdir)
	if err != nil {
		log.Fatalln("Loading Tex Environment:", err)
	}
	defer res.Reader.Close()

	err = gotex.FileWriter(res.Reader, "-")
	if err != nil {
		log.Fatalln(err)
	}
	if err := input.Wait(); err != nil {
		log.Fatalln(err)
	}
	if err := res.Wait(); err != nil {
		log.Fatalln(err)
	}

}
