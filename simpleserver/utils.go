// Copyright (c) 2026 Benjamin Benno Falkner
// SPDX-License-Identifier: MIT

// utils.go provides ID generation and file transfer helpers for the simpleserver package.
package simpleserver

import (
	"crypto/rand"
	"encoding/hex"
	"io"
	"os"
)

// newID generates a cryptographically random 16-byte identifier
// and returns it as a 32-character lowercase hex string.
func NewID() (string, error) {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}

// sendFile copies the contents of the file at path to the given writer.
func SendFile(w io.Writer, path string) error {
	f, err := os.Open(path)
	if err != nil {
		return err
	}
	defer f.Close()

	_, err = io.Copy(w, f)
	return err
}
