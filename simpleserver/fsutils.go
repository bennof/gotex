// Copyright (c) 2026 Benjamin Benno Falkner
// SPDX-License-Identifier: MIT
package simpleserver

import (
	"os"
	"path/filepath"
)

func DirSize(root string) (int64, error) {
	var size int64
	err := filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !info.IsDir() {
			size += info.Size()
		}
		return nil
	})
	if err != nil {
		return 0, err
	}
	return size, nil
}
