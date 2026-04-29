package main

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"io"
	"os"
)

type streamMessage struct {
	Type string `json:"type"`
	Data string `json:"data,omitempty"`
	URL  string `json:"url,omitempty"`
}

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

func resolvePaths(target, binary, tree string) (string, string, error) {
	path := os.Getenv("GOTEX_PATH")
	binEnv := os.Getenv("GOTEX_BIN")
	treeEnv := os.Getenv("GOTEX_TEXMF")

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

func newID() (string, error) {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}

func sendFile(w io.Writer, path string) error {
	f, err := os.Open(path)
	if err != nil {
		return err
	}
	defer f.Close()

	_, err = io.Copy(w, f)
	if err != nil {
		return err
	}
	return nil
}

func writeStreamMessage(w io.Writer, msg streamMessage) error {
	return json.NewEncoder(w).Encode(msg)
}
