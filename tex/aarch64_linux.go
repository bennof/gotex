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

// aarch64_linux.go provides the embedded Tectonic binary for Linux on arm64.
//go:build linux && arm64

package tex

import (
	"embed"
	"os"
	"path/filepath"
)

//go:embed bin/tectonic-aarch64-unknown-linux-musl
var FS embed.FS

// BinaryPath is the embedded Tectonic binary path for Linux on arm64.
const BinaryPath = "bin/tectonic-aarch64-unknown-linux-musl"

// BinaryName is the executable name used when installing the embedded binary.
const BinaryName = "tectonic"

// DefaultDotFolder is the default edotex data directory for Unix-like systems.
const DefaultDotFolder = ".gotex"

// DefaultDotPath resolves the default edotex data directory for the current user.
func DefaultDotPath() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, DefaultDotFolder), nil
}
