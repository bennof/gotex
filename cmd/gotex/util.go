// Copyright (c) 2026 Benjamin Benno Falkner
// SPDX-License-Identifier: MIT

package main

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"io"
	"os"
)

// streamMessage represents a server-sent event message used to communicate
// build status, data payloads, or URLs to the client over a streaming response.
type streamMessage struct {
	Type string `json:"type"`
	Data string `json:"data,omitempty"`
	URL  string `json:"url,omitempty"`
}

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

// newID generates a cryptographically random 16-byte identifier
// and returns it as a 32-character lowercase hex string.
func newID() (string, error) {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}

// sendFile copies the contents of the file at path to the given writer.
func sendFile(w io.Writer, path string) error {
	f, err := os.Open(path)
	if err != nil {
		return err
	}
	defer f.Close()

	_, err = io.Copy(w, f)
	return err
}

// writeStreamMessage encodes a streamMessage as JSON and writes it to the given writer.
// Each call writes exactly one newline-terminated JSON object.
func writeStreamMessage(w io.Writer, msg streamMessage) error {
	return json.NewEncoder(w).Encode(msg)
}
