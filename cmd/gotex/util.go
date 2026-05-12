// Copyright (c) 2026 Benjamin Benno Falkner
// SPDX-License-Identifier: MIT

package main

import (
	"os"
)

// resolvePathValue returns the first non-empty value among explicit, envValue, and shared.
// If all are empty, it falls back to the current working directory.
func resolvePathValue(explicit string, envValue string, shared string) (string, error) {
	if explicit != "" {
		return explicit, nil
	}
	if envValue != "" {
		return envValue, nil
	}
	if shared != "" {
		return shared, nil
	}
	return os.Getwd()
}

// resolvePaths determines the binary and TeXMF tree paths from flags and environment variables.
// The resolution order is: explicit flag > individual environment variable > shared target/GOTEX_PATH > working directory.
func resolvePaths(target, binary, tree string) (string, string, error) {
	path := os.Getenv("GOTEX_PATH")
	binEnv := os.Getenv("GOTEX_BIN")
	treeEnv := os.Getenv("GOTEX_TEXMF")

	// An explicit target flag overrides the GOTEX_PATH environment variable.
	if target != "" {
		path = target
	}

	binpath, err := resolvePathValue(binary, binEnv, path)
	if err != nil {
		return "", "", err
	}

	treepath, err := resolvePathValue(tree, treeEnv, path)
	if err != nil {
		return "", "", err
	}

	return binpath, treepath, nil
}
