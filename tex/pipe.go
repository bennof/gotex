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

// pipe.go provides helpers for managing external process pipelines and streams.
package tex

import (
	"bytes"
	"errors"
	"io"
	"sync"
)

// PipeResult represents a running pipeline step with stdout, stderr, and a wait hook.
type PipeResult struct {
	Reader io.ReadCloser
	Stderr io.ReadCloser
	Wait   func() error
}

// PrefixWriter prefixes each stderr line before writing it to the target writer.
type PrefixWriter struct {
	Writer io.Writer
	Prefix string

	atLineStart bool
}

type synchronizedWriter struct {
	writer io.Writer
	mu     sync.Mutex
}

// Write serializes access to the wrapped writer.
func (w *synchronizedWriter) Write(p []byte) (int, error) {
	w.mu.Lock()
	defer w.mu.Unlock()
	return w.writer.Write(p)
}

// CopyTo copies stdout and stderr to a shared writer and waits for the process to finish.
func (p *PipeResult) CopyTo(w io.Writer) error {
	var wg sync.WaitGroup
	var stdoutErr error
	var stderrErr error

	locked := &synchronizedWriter{writer: w}

	if p.Reader != nil {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_, stdoutErr = io.Copy(locked, p.Reader)
		}()
	}

	if p.Stderr != nil {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_, stderrErr = io.Copy(&PrefixWriter{
				Writer:      locked,
				Prefix:      "[ERROR] ",
				atLineStart: true,
			}, p.Stderr)
		}()
	}

	wg.Wait()

	waitErr := error(nil)
	if p.Wait != nil {
		waitErr = p.Wait()
	}

	return errors.Join(stdoutErr, stderrErr, waitErr)
}

// Write adds the configured prefix to each new line written through the wrapper.
func (w PrefixWriter) Write(p []byte) (int, error) {
	written := 0
	for len(p) > 0 {
		if w.atLineStart {
			if _, err := io.WriteString(w.Writer, w.Prefix); err != nil {
				return written, err
			}
			w.atLineStart = false
		}

		newline := bytes.IndexByte(p, '\n')
		chunkEnd := len(p)
		if newline >= 0 {
			chunkEnd = newline + 1
		}

		n, err := w.Writer.Write(p[:chunkEnd])
		written += n
		if err != nil {
			return written, err
		}
		if newline >= 0 {
			w.atLineStart = true
		}
		p = p[chunkEnd:]
	}
	return written, nil
}
