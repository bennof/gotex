// Copyright (c) 2026 Benjamin Benno Falkner
// SPDX-License-Identifier: MIT

// session.go defines the session interface and base session state implementation.
package session

import (
	"context"
	"errors"
	"sync"
	"time"

	"github.com/bennof/gotex/simpleserver"
)

// ErrSessionNotFound is returned when a session id is unknown.
var ErrSessionNotFound = errors.New("session: not found")

// NewID is the session ID generator. Replace to use a custom strategy.
var NewID func() (string, error) = simpleserver.NewID

// SessionInterface must be implemented by all session types used with Store.
type SessionInterface interface {
	ID() string
	Close() error
	Touch()
	CreatedAt() time.Time
	LastSeen() time.Time
}

// Session is the base type to embed in concrete session types.
type Session struct {
	mu        sync.RWMutex
	id        string
	createdAt time.Time
	lastSeen  time.Time
}

func (s *Session) ID() string {
	return s.id
}

func (s *Session) CreatedAt() time.Time {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.createdAt
}

func (s *Session) LastSeen() time.Time {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.lastSeen
}

func (s *Session) Touch() {
	s.mu.Lock()
	s.lastSeen = time.Now()
	s.mu.Unlock()
}

// Close is a no-op on the base type. Override in concrete session types.
func (s *Session) Close() error {
	return nil
}

// Store is the generic interface for a session collection.
type Store[T SessionInterface] interface {
	Create(ctx context.Context) (T, error)
	Get(id string) (T, bool)
	Delete(id string)
	Len() int
}
