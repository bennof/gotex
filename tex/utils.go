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

// utils.go provides filesystem helpers and embedded resource extraction utilities.
package tex

import (
	"embed"
	"io/fs"
	"os"
	"path/filepath"
)

// pathExists reports whether a file or directory exists at the given path.
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

// binaryEnsure installs the embedded Tectonic binary if it is missing locally.
func binaryEnsure(binary string) error {
	if ok, err := pathExists(binary); !ok {
		if err != nil {
			return err
		}
		log("binary not found at %s, extracting from embedded resources", binary)
		err = os.MkdirAll(filepath.Dir(binary), os.ModePerm)
		if err != nil {
			return err
		}
		data, err := FS.ReadFile(BinaryPath)
		if err != nil {
			return err
		}
		err = os.WriteFile(binary, data, 0755)
		if err != nil {
			return err
		}
	}
	return nil
}

// texmfEnsure installs the embedded TeX tree if it is missing locally.
func texmfEnsure(texmf string) error {
	if ok, err := pathExists(texmf); !ok {
		if err != nil {
			return err
		}
		log("texmf tree not found at %s, extracting from embedded resources", texmf)
		err = os.MkdirAll(texmf, os.ModePerm)
		if err != nil {
			return err
		}
		err = extractDir(TeXmf, TeXmfPath, texmf)
		if err != nil {
			return err
		}
		// gen font config !!! to be improved
		err := os.WriteFile(filepath.Join(texmf, "tex/latex/bflatex/fontconfig.tex"), GetFontConfig(texmf), 0644)
		if err != nil {
			return err
		}
	}
	return nil
}

// extractDir copies an embedded directory tree to a destination on disk.
func extractDir(fs embed.FS, srcDir, dstDir string) error {
	entries, err := fs.ReadDir(srcDir)
	if err != nil {
		return err
	}
	for _, entry := range entries {
		if shouldSkipEmbeddedFile(entry.Name()) {
			continue
		}
		srcPath := srcDir + "/" + entry.Name()
		dstPath := dstDir + "/" + entry.Name()
		if entry.IsDir() {
			err = os.MkdirAll(dstPath, os.ModePerm)
			if err != nil {
				return err
			}
			err = extractDir(fs, srcPath, dstPath)
			if err != nil {
				return err
			}
		} else {
			data, err := fs.ReadFile(srcPath)
			if err != nil {
				return err
			}
			err = os.WriteFile(dstPath, data, 0644)
			if err != nil {
				return err
			}
		}
	}
	return nil
}

// shouldSkipEmbeddedFile filters metadata files that should not be extracted.
func shouldSkipEmbeddedFile(name string) bool {
	return name == ".DS_Store"
}

// texmfsearch collects all local TeX directories as Tectonic search-path arguments.
func texmfsearch(texmf string) ([]string, error) {
	var searchpath []string
	err := filepath.WalkDir(texmf, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			searchpath = append(searchpath, "-Z", "search-path="+path)
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return searchpath, nil
}
