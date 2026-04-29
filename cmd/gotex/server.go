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

type SimpleServer struct {
	mux    *http.ServeMux
	server *http.Server
	logger *log.Logger
}

func NewSimpleServer() *SimpleServer {
	logger := log.New(os.Stdout, "server: ", log.LstdFlags)
	mux := http.NewServeMux()

	s := &SimpleServer{
		mux:    mux,
		logger: logger,
	}

	return s
}

func (s *SimpleServer) Handle(pattern string, handler http.Handler) {
	s.mux.Handle(pattern, handler)
}

func (s *SimpleServer) HandleFunc(pattern string, handler func(http.ResponseWriter, *http.Request)) {
	s.mux.HandleFunc(pattern, handler)
}

func (s *SimpleServer) Mux() *http.ServeMux {
	return s.mux
}

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

	// Signal-Handling
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, os.Interrupt, syscall.SIGTERM)

	go func() {
		<-quit
		s.logger.Println("shutting down...")
		// Hier cleanup aufrufen
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

func (s *SimpleServer) WriteJSON(w http.ResponseWriter, status int, payload any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)

	if err := json.NewEncoder(w).Encode(payload); err != nil {
		s.logger.Printf("write response: %v", err)
	}
}

func (s *SimpleServer) loggingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		next.ServeHTTP(w, r)
		s.logger.Printf("%s %s %s", r.Method, r.URL.Path, time.Since(start))
	})
}

func normalizeAddr(port string) string {
	if port == "" {
		return ":8080"
	}
	if strings.Contains(port, ":") {
		return port
	}
	return fmt.Sprintf(":%s", port)
}
