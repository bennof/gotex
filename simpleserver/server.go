// Copyright (c) 2026 Benjamin Benno Falkner
// SPDX-License-Identifier: MIT

package simpleserver

import (
	"context"
	"encoding/json"
	"log"
	"net"
	"net/http"
	"os"
	"os/signal"
	"reflect"
	"syscall"
	"time"
)

// SimpleServer wraps an HTTP server with a mux, logger, and graceful shutdown.
type SimpleServer struct {
	mux    *http.ServeMux
	server *http.Server
	logger *log.Logger
	cfg    Config
}

// Option configures a SimpleServer.
type Option func(*SimpleServer)

// WithConfig sets a custom Config.
func WithConfig(cfg Config) Option {
	return func(s *SimpleServer) {
		d := reflect.ValueOf(DefaultConfig())
		c := reflect.ValueOf(&cfg).Elem()
		for i := 0; i < c.NumField(); i++ {
			if c.Field(i).IsZero() {
				c.Field(i).Set(d.Field(i))
			}
		}
		s.cfg = cfg
	}
}

// WithLogger sets a custom logger.
func WithLogger(logger *log.Logger) Option {
	return func(s *SimpleServer) {
		s.logger = logger
	}
}

// NewSimpleServer creates a SimpleServer with defaults and applies opts.
func NewSimpleServer(opts ...Option) *SimpleServer {
	s := &SimpleServer{
		mux:    http.NewServeMux(),
		logger: log.New(os.Stdout, "server: ", log.LstdFlags),
		cfg:    DefaultConfig(),
	}
	for _, opt := range opts {
		opt(s)
	}
	return s
}

// Handle registers an http.Handler for the given pattern.
func (s *SimpleServer) Handle(pattern string, handler http.Handler) {
	s.mux.Handle(pattern, handler)
}

// HandleFunc registers a handler function for the given pattern.
func (s *SimpleServer) HandleFunc(pattern string, handler func(http.ResponseWriter, *http.Request)) {
	s.mux.HandleFunc(pattern, handler)
}

// Mux returns the underlying http.ServeMux for direct registration.
func (s *SimpleServer) Mux() *http.ServeMux {
	return s.mux
}

// Log writes a message to the server logger.
func (s *SimpleServer) Log(msg string) {
	s.logger.Println(msg)
}

// Listen starts the HTTP server and blocks until shutdown.
// Handles SIGINT and SIGTERM gracefully with a 10s drain timeout.
func (s *SimpleServer) Listen() error {
	s.server = &http.Server{
		Addr:              s.cfg.Addr,
		Handler:           s.loggingMiddleware(s.mux),
		ReadTimeout:       s.cfg.ReadTimeout.Duration,
		ReadHeaderTimeout: s.cfg.ReadHeaderTimeout.Duration,
		WriteTimeout:      s.cfg.WriteTimeout.Duration,
		IdleTimeout:       s.cfg.IdleTimeout.Duration,
		ErrorLog:          s.logger,
		BaseContext: func(_ net.Listener) context.Context {
			return context.Background()
		},
	}

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, os.Interrupt, syscall.SIGTERM)
	go func() {
		<-quit
		s.logger.Println("shutting down...")
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		s.server.Shutdown(ctx)
	}()

	s.logger.Printf("listening on %s", s.cfg.Addr)
	if err := s.server.ListenAndServe(); err != http.ErrServerClosed {
		return err
	}
	return nil
}

// WriteJSON writes a JSON-encoded payload with the given status code.
func (s *SimpleServer) WriteJSON(w http.ResponseWriter, status int, payload any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(payload); err != nil {
		s.logger.Printf("write response: %v", err)
	}
}

// loggingMiddleware logs method, path, status code, and elapsed time per request.
func (s *SimpleServer) loggingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		rec := &statusRecorder{ResponseWriter: w, status: http.StatusOK}
		next.ServeHTTP(rec, r)
		s.logger.Printf("%s %s %d %s", r.Method, r.URL.RequestURI(), rec.status, time.Since(start).Round(time.Millisecond))
	})
}

func SecurityHeaders(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.Header().Set("X-Frame-Options", "DENY")
		w.Header().Set("Referrer-Policy", "strict-origin-when-cross-origin")
		w.Header().Set("Cross-Origin-Opener-Policy", "same-origin")
		next.ServeHTTP(w, r)
	})
}
