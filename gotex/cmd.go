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
	"bytes"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"strings"
)

// Verbose enables detailed logging output to stderr.
var Verbose bool = false

// LogPrefix is the prefix used for log output messages.
var LogPrefix string = "gotex"

// log writes a formatted message to stderr if Verbose is enabled.
func log(format string, args ...any) {
	if Verbose {
		fmt.Fprintf(os.Stderr, "["+LogPrefix+"] "+format+"\n", args...)
	}
}

// PipeResult represents the output of a pipeline step,
// providing a reader for stdout, a reader for stderr,
// and a Wait function to block until the step completes.
type PipeResult struct {
	Reader io.ReadCloser
	Stderr io.ReadCloser
	Wait   func() error
}

// FromString creates a PipeResult from a string value.
func FromString(s string) *PipeResult {
	return &PipeResult{
		Reader: io.NopCloser(strings.NewReader(s)),
		Stderr: io.NopCloser(strings.NewReader("")),
		Wait:   func() error { return nil },
	}
}

// FromBytes creates a PipeResult from a byte slice.
func FromBytes(b []byte) *PipeResult {
	return &PipeResult{
		Reader: io.NopCloser(bytes.NewReader(b)),
		Stderr: io.NopCloser(strings.NewReader("")),
		Wait:   func() error { return nil },
	}
}

// Download fetches the content of the given URL and returns it as a PipeResult.
// Returns an error if the request fails or the server returns a non-200 status.
func Download(url string) (*PipeResult, error) {
	log("downloading %s", url)
	resp, err := http.Get(url)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode != http.StatusOK {
		resp.Body.Close()
		return nil, fmt.Errorf("download failed: %s", resp.Status)
	}
	return &PipeResult{
		Reader: resp.Body,
		Stderr: io.NopCloser(strings.NewReader("")),
		Wait:   func() error { return nil },
	}, nil
}

// ReadFile opens a file at the given path and returns its contents as a PipeResult.
func ReadFile(path string) (*PipeResult, error) {
	log("reading file %s", path)
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	return &PipeResult{
		Reader: f,
		Stderr: io.NopCloser(strings.NewReader("")),
		Wait:   func() error { return nil },
	}, nil
}

// ShellCmdDir runs a shell command in the given directory, feeding input via stdin.
// It returns a PipeResult with access to stdout, stderr, and a Wait function.
func ShellCmdDir(input io.Reader, dir string, name string, args ...string) (*PipeResult, error) {
	log("running %s %v in %s", name, args, dir)
	cmd := exec.Command(name, args...)
	cmd.Dir = dir

	stdin, err := cmd.StdinPipe()
	if err != nil {
		return nil, err
	}
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

	go func() {
		_, _ = io.Copy(stdin, input)
		_ = stdin.Close()
	}()

	return &PipeResult{
		Reader: stdout,
		Stderr: stderr,
		Wait: func() error {
			return cmd.Wait()
		},
	}, nil
}

// ShellCmd runs a shell command in the current directory, feeding input via stdin.
// It is a convenience wrapper around ShellCmdDir with an empty directory.
func ShellCmd(input io.Reader, name string, args ...string) (*PipeResult, error) {
	return ShellCmdDir(input, "", name, args...)
}

// Pipe chains a PipeResult into a new shell command, passing its stdout as stdin.
// The returned PipeResult's Wait function waits for both the previous and new step.
func Pipe(input *PipeResult, name string, args ...string) (*PipeResult, error) {
	result, err := ShellCmd(input.Reader, name, args...)
	if err != nil {
		return nil, err
	}
	oldWait := input.Wait
	result.Wait = func() error {
		if err := oldWait(); err != nil {
			return err
		}
		return result.Wait()
	}
	return result, nil
}

// Tee writes the stream from a PipeResult to a file while passing it through unchanged.
// This allows saving intermediate pipeline output without interrupting the pipeline.
func Tee(p *PipeResult, path string) (*PipeResult, error) {
	log("tee to %s", path)
	f, err := os.Create(path)
	if err != nil {
		return nil, err
	}
	pr, pw := io.Pipe()
	go func() {
		_, _ = io.Copy(io.MultiWriter(pw, f), p.Reader)
		pw.Close()
		f.Close()
	}()
	return &PipeResult{
		Reader: pr,
		Stderr: p.Stderr,
		Wait:   p.Wait,
	}, nil
}

// Collect reads the full output of a PipeResult into a string and waits for completion.
func Collect(p *PipeResult) (string, error) {
	data, err := io.ReadAll(p.Reader)
	if err != nil {
		return "", err
	}
	if err := p.Wait(); err != nil {
		return "", err
	}
	log("collected %d bytes", len(data))
	return string(data), nil
}

// FileWriter writes the contents of a reader to a file at the given path.
// If target is empty or "-", the output is written to stdout instead.
func FileWriter(input io.Reader, target string) error {
	var w io.Writer
	var f *os.File
	var err error

	if target == "" || target == "-" {
		log("writing to stdout")
		w = os.Stdout
	} else {
		log("writing to %s", target)
		f, err = os.Create(target)
		if err != nil {
			return err
		}
		defer f.Close()
		w = f
	}

	_, err = io.Copy(w, input)
	return err
}
