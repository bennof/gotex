// Copyright (c) 2026 Benjamin Benno Falkner
// SPDX-License-Identifier: MIT

package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"
)

// SimpleServer wraps an HTTP server with a multiplexer, structured logger,
// graceful shutdown handling, and convenience methods for common handler patterns.
type SimpleServer struct {
	mux    *http.ServeMux
	server *http.Server
	logger *log.Logger
}

// NewSimpleServer creates a new SimpleServer with a fresh multiplexer
// and a logger writing to stdout with a "server: " prefix.
func NewSimpleServer() *SimpleServer {
	logger := log.New(os.Stdout, "server: ", log.LstdFlags)
	mux := http.NewServeMux()
	return &SimpleServer{
		mux:    mux,
		logger: logger,
	}
}

// Handle registers an http.Handler for the given pattern.
func (s *SimpleServer) Handle(pattern string, handler http.Handler) {
	s.mux.Handle(pattern, handler)
}

// HandleFunc registers a handler function for the given pattern.
func (s *SimpleServer) HandleFunc(pattern string, handler func(http.ResponseWriter, *http.Request)) {
	s.mux.HandleFunc(pattern, handler)
}

// Mux returns the underlying http.ServeMux for direct registration if needed.
func (s *SimpleServer) Mux() *http.ServeMux {
	return s.mux
}

// Listen starts the HTTP server on the given port or address and blocks until shutdown.
// It handles SIGINT and SIGTERM gracefully, allowing up to 10 seconds for active
// connections to complete before the server exits.
func (s *SimpleServer) Listen(port string) error {
	addr := normalizeAddr(port)
	s.server = &http.Server{
		Addr:              addr,
		Handler:           s.loggingMiddleware(s.mux),
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       15 * time.Second,
		WriteTimeout:      200 * time.Second,
		IdleTimeout:       60 * time.Second,
		BaseContext: func(net.Listener) context.Context {
			return context.Background()
		},
	}

	// Listen for OS signals to trigger graceful shutdown.
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, os.Interrupt, syscall.SIGTERM)
	go func() {
		<-quit
		s.logger.Println("shutting down...")
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		s.server.Shutdown(ctx)
	}()

	s.logger.Printf("listening on %s", addr)
	err := s.server.ListenAndServe()
	if err == http.ErrServerClosed {
		return nil
	}
	return err
}

// WriteJSON writes a JSON-encoded payload with the given HTTP status code.
// The Content-Type header is set to application/json automatically.
func (s *SimpleServer) WriteJSON(w http.ResponseWriter, status int, payload any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(payload); err != nil {
		s.logger.Printf("write response: %v", err)
	}
}

// loggingMiddleware wraps a handler and logs the HTTP method, path,
// and elapsed time for every incoming request.
func (s *SimpleServer) loggingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		next.ServeHTTP(w, r)
		s.logger.Printf("%s %s %s", r.Method, r.URL.Path, time.Since(start))
	})
}

// normalizeAddr ensures the address is in host:port format.
// If port is empty, it defaults to ":8080".
// If port contains a colon it is returned as-is, otherwise ":" is prepended.
func normalizeAddr(port string) string {
	if port == "" {
		return ":8080"
	}
	if strings.Contains(port, ":") {
		return port
	}
	return fmt.Sprintf(":%s", port)
}
