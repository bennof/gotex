// Copyright (c) 2026 Benjamin Benno Falkner
//
// Permission is hereby granted, free of charge, to any person obtaining a copy
// of this software and associated documentation files (the "Software"), to deal
// in the Software without restriction, including without limitation the rights
// to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
// copies of the Software, and to permit persons to whom the Software is
// furnished to do so, subject to the following conditions:
//
// The above copyright notice and this permission notice shall be included in all
// copies or substantial portions of the Software.
//
// THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
// IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
// FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
// AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
// LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
// OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE
// SOFTWARE.

package main

import (
	"fmt"
	"os"
)

// main is the entry point of the gotex command line tool.
// It reads the first argument as the mode and dispatches to the
// corresponding handler. If no mode is given or the mode is unknown,
// the usage message is printed and the process exits with code 1.
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

// usage prints a short description of the available modes and their arguments to stderr.
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
