// Copyright (c) 2026 Benjamin Benno Falkner
// SPDX-License-Identifier: MIT
package stream

import (
	"bufio"
	"encoding/json"
	"errors"
	"io"
	"net/http"
)

var ErrStreamingUnsupported = errors.New("x-ndjson: streaming unsupported")

type SimpleXndJSONMsg struct {
	Type string `json:"type"`
	Data string `json:"data,omitempty"`
	URL  string `json:"url,omitempty"`
}

type XndJSON[T any] struct {
	w       http.ResponseWriter
	flusher http.Flusher
	enc     *json.Encoder
}

func NewXndJSON[T any](w http.ResponseWriter) (*XndJSON[T], error) {
	w.Header().Set("Content-Type", "application/x-ndjson")
	w.Header().Set("Cache-Control", "no-cache")
	w.WriteHeader(http.StatusOK)

	flusher, ok := w.(http.Flusher)
	if !ok {
		return nil, ErrStreamingUnsupported
	}

	enc := json.NewEncoder(w)

	return &XndJSON[T]{
		w:       w,
		flusher: flusher,
		enc:     enc,
	}, nil
}

func (x *XndJSON[T]) Error(err error) {
	_ = x.enc.Encode(SimpleXndJSONMsg{Type: "error", Data: err.Error()})
}

func (x *XndJSON[T]) Write(msg T) error {
	err := x.enc.Encode(msg)
	if err != nil {
		x.Error(err)
	}
	x.flusher.Flush()
	return err
}

type SimpleXndJSON struct {
	*XndJSON[SimpleXndJSONMsg]
}

func NewSimpleXndJSON(w http.ResponseWriter) (*SimpleXndJSON, error) {
	x, err := NewXndJSON[SimpleXndJSONMsg](w)
	if err != nil {
		return nil, err
	}
	return &SimpleXndJSON{x}, nil
}

func (x *SimpleXndJSON) StreamReader(r io.Reader) error {
	scanner := bufio.NewScanner(r)
	for scanner.Scan() {
		if err := x.Write(SimpleXndJSONMsg{Type: "log", Data: scanner.Text()}); err != nil {
			return err
		}
	}
	if err := scanner.Err(); err != nil {
		x.Error(err)
		return err
	}
	return nil
}
