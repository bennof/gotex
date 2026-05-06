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

package gotex

import (
	"fmt"
	"io"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
)

// URL_TECTONIC is the URL to the tectonic installer shell script.
const URL_TECTONIC = "https://drop-sh.fullyjustified.net"

// TECTONIC_BIN is the platform-dependent binary name, defined in tectonic_unix.go / tectonic_windows.go.

// LOCALE_TEXMF is the name of the local TeX tree directory.
const LOCALE_TEXMF = "texmf"

// VAR_TECTONIC_CACHE_DIR is the environment variable name for the tectonic cache directory.
const VAR_TECTONIC_CACHE_DIR = "TECTONIC_CACHE_DIR"

// TECTONIC_CACHE_FOLDER is the default folder name for the tectonic cache.
const TECTONIC_CACHE_FOLDER = ".tectonic-cache"

// VAR_GOTEX_PATH is the environment variable for the base path used by gotex.
const VAR_GOTEX_PATH = "GOTEX_PATH"

// VAR_GOTEX_BIN is the environment variable for the directory containing the tectonic binary.
const VAR_GOTEX_BIN = "GOTEX_BIN"

// VAR_GOTEX_TEXMF is the environment variable for the TeX tree path used by gotex.
const VAR_GOTEX_TEXMF = "GOTEX_TEXMF"

// Processor holds the configuration for a tectonic LaTeX processor instance,
// including the binary path, TeX tree path, and precomputed search path arguments.
type Processor struct {
	binpath    string
	treepath   string
	searchpath []string
	binary     string
}

// NewProcessor creates a new Processor with the given binary directory and TeX tree path.
// It verifies that the tectonic binary exists (downloading it if necessary),
// checks that the TeX tree path is accessible, and collects all subdirectories
// of the local texmf tree as search paths for tectonic.
func NewProcessor(binpath, treepath string) (*Processor, error) {
	err := checkBinary(binpath)
	if err != nil {
		return nil, err
	}

	if ok, err := pathExists(treepath); !ok {
		if err != nil {
			return nil, err
		}
		return nil, fmt.Errorf("TeXMF path not found: %s", treepath)
	}

	var searchpath []string
	err = filepath.WalkDir(filepath.Join(treepath, LOCALE_TEXMF), func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			searchpath = append(searchpath, "-Z", "search-path="+path)
		}
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("failed to collect search paths: %w", err)
	}

	return &Processor{
		binpath:    binpath,
		treepath:   treepath,
		searchpath: searchpath,
		binary:     filepath.Join(binpath, TECTONIC_BIN),
	}, nil
}

// checkBinary verifies that the tectonic binary exists in the given directory.
// If it is not found, it creates the directory and downloads tectonic
// by fetching the installer script from URL_TECTONIC and piping it to sh.
func checkBinary(binpath string) error {
	binary := filepath.Join(binpath, TECTONIC_BIN)
	if ok, err := pathExists(binary); !ok {
		if err != nil {
			return err
		}
		log("tectonic binary not found, downloading to %s", binpath)
		err = os.MkdirAll(binpath, os.ModePerm)
		if err != nil {
			return err
		}
		dl, err := Download(URL_TECTONIC)
		if err != nil {
			return err
		}
		defer dl.Reader.Close()

		sh, err := ShellCmdDir(dl.Reader, binpath, "sh")
		if err != nil {
			return err
		}
		defer sh.Reader.Close()

		if err := dl.Wait(); err != nil {
			return err
		}
		if err := sh.Wait(); err != nil {
			return err
		}
	}
	return nil
}

// NewProcessorSimple creates a Processor using environment variables for configuration.
// It reads GOTEX_BIN for the binary directory, GOTEX_TEXMF for the TeX tree path,
// and GOTEX_PATH as a fallback for both. If none are set, the current working directory is used.
func NewProcessorSimple() (*Processor, error) {
	var err error
	path := os.Getenv(VAR_GOTEX_PATH)
	binpath := os.Getenv(VAR_GOTEX_BIN)
	treepath := os.Getenv(VAR_GOTEX_TEXMF)

	if binpath == "" {
		if path != "" {
			binpath = path
		} else {
			binpath, err = os.Getwd()
			if err != nil {
				return nil, err
			}
		}
	}

	if treepath == "" {
		if path != "" {
			treepath = path
		} else {
			treepath, err = os.Getwd()
			if err != nil {
				return nil, err
			}
		}
	}

	return NewProcessor(binpath, treepath)
}

// Process compiles a LaTeX document from the given input reader using tectonic.
// The output PDF is written to outdir. It sets the tectonic cache directory
// via environment variable and returns a PipeResult for the command output.
func (p *Processor) Process(input io.Reader, outdir string) (*PipeResult, error) {
	args := []string{"-p"}
	args = append(args, p.searchpath...)
	args = append(args, "-o", outdir, "-")

	cmd := exec.Command(p.binary, args...)
	cmd.Env = append(os.Environ(),
		VAR_TECTONIC_CACHE_DIR+"="+filepath.Join(p.treepath, TECTONIC_CACHE_FOLDER),
	)
	cmd.Stdin = input
	cmd.Dir = outdir

	log("running tectonic: %s %v", cmd.Path, cmd.Args)
	log("output dir: %s", outdir)

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return nil, err
	}
	if err := cmd.Start(); err != nil {
		return nil, err
	}

	return &PipeResult{
		Reader: stdout,
		Wait:   cmd.Wait,
	}, nil
}

// pathExists checks whether a file or directory exists at the given path.
// Returns (true, nil) if it exists, (false, nil) if it does not,
// or (false, err) if the check failed for another reason.
func pathExists(path string) (bool, error) {
	_, err := os.Stat(path)
	if err == nil {
		return true, nil
	}
	if os.IsNotExist(err) {
		return false, nil
	}
	return false, err
}
