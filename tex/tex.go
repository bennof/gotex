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

// tex.go implements the tex processor lifecycle and Tectonic build orchestration.
package tex

import (
	"io"
	"os"
	"os/exec"
	"path/filepath"
)

// VAR_TECTONIC_CACHE_DIR is the environment variable used by Tectonic for its cache.
const VAR_TECTONIC_CACHE_DIR = "TECTONIC_CACHE_DIR"

// TECTONIC_CACHE_FOLDER is the local directory name used for cached Tectonic data.
const TECTONIC_CACHE_FOLDER = ".tectonic-cache"

// Processor holds the resolved binary path, local dot directory, and TeX search paths.
type Processor struct {
	binary     string
	dotpath    string
	searchpath []string
}

// NewProcessor prepares the embedded binary and TeX tree inside the target dot directory.
func NewProcessor(dotpath string) (*Processor, error) {
	binary := filepath.Join(dotpath, "bin", BinaryName)
	err := binaryEnsure(binary)
	if err != nil {
		log("error ensuring binary: %v", err)
		return nil, err
	}

	texmf := filepath.Join(dotpath, "texmf")
	err = texmfEnsure(texmf)
	if err != nil {
		log("error ensuring texmf: %v", err)
		return nil, err
	}

	searchpath, err := texmfsearch(texmf)
	if err != nil {
		log("error searching texmf: %v", err)
		return nil, err
	}

	return &Processor{
		binary:     binary,
		dotpath:    dotpath,
		searchpath: searchpath,
	}, nil
}

// Process runs Tectonic with the prepared local search paths and output directory.
func (p *Processor) Process(input io.Reader, outdir string) (*PipeResult, error) {
	args := []string{"-p"}
	args = append(args, p.searchpath...)
	args = append(args, "-o", outdir, "-")

	cmd := exec.Command(p.binary, args...)
	cmd.Env = append(os.Environ(),
		VAR_TECTONIC_CACHE_DIR+"="+filepath.Join(p.dotpath, TECTONIC_CACHE_FOLDER),
	)
	cmd.Stdin = input
	cmd.Dir = outdir

	log("running tectonic: %s %v", cmd.Path, cmd.Args)
	log("output dir: %s", outdir)

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return nil, err
	}

	stderr, err := cmd.StderrPipe()
	if err != nil {
		return nil, err
	}

	if err := cmd.Start(); err != nil {
		return nil, err
	}

	return &PipeResult{
		Reader: stdout,
		Stderr: stderr,
		Wait:   cmd.Wait,
	}, nil
}
