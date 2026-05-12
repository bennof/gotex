// Copyright (c) 2026 Benjamin Benno Falkner
// SPDX-License-Identifier: MIT

package session

import (
	"context"
	"errors"
	"log"
	"sync"
	"time"
)

// PoolStore is a bounded, thread-safe session store with TTL-based eviction.
type PoolStore[T SessionInterface] struct {
	mu       sync.RWMutex
	sessions map[string]T
	cap      int
	ttl      time.Duration
	init     func(id string, now time.Time) (T, error)
	done     chan struct{}
}

// NewPoolStore creates a new PoolStore with the given capacity, TTL and session factory.
func NewPoolStore[T SessionInterface](
	cap int,
	ttl time.Duration,
	init func(id string, now time.Time) (T, error),
) (*PoolStore[T], error) {
	if cap <= 0 {
		return nil, errors.New("session: cap must be > 0")
	}
	if ttl <= 0 {
		return nil, errors.New("session: ttl must be > 0")
	}
	p := &PoolStore[T]{
		sessions: make(map[string]T, cap),
		cap:      cap,
		ttl:      ttl,
		init:     init,
		done:     make(chan struct{}),
	}
	go p.gc(ttl / 2)
	return p, nil
}

// Create allocates a new session. Blocks until a slot is available or ctx expires.
func (p *PoolStore[T]) Create(ctx context.Context) (T, error) {
	for {
		p.mu.Lock()
		if len(p.sessions) < p.cap {
			id, err := NewID()
			if err != nil {
				p.mu.Unlock()
				var zero T
				return zero, err
			}
			t, err := p.init(id, time.Now())
			if err != nil {
				p.mu.Unlock()
				var zero T
				return zero, err
			}
			p.sessions[id] = t
			p.mu.Unlock()
			return t, nil
		}
		p.mu.Unlock()
		select {
		case <-ctx.Done():
			var zero T
			return zero, ctx.Err()
		case <-time.After(50 * time.Millisecond):
		}
	}
}

// Get returns the session for the given id and updates its last-seen timestamp.
func (p *PoolStore[T]) Get(id string) (T, bool) {
	p.mu.RLock()
	defer p.mu.RUnlock()
	t, ok := p.sessions[id]
	if ok {
		t.Touch()
	}
	return t, ok
}

// Delete removes the session and calls its Close method.
func (p *PoolStore[T]) Delete(id string) error {
	var err error
	p.mu.Lock()
	if t, ok := p.sessions[id]; ok {
		delete(p.sessions, id)
		err = t.Close()
	}
	p.mu.Unlock()
	return err
}

// Len returns the current number of active sessions.
func (p *PoolStore[T]) Len() int {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return len(p.sessions)
}

// Close stops the GC goroutine and clears all sessions.
func (p *PoolStore[T]) Close() {
	close(p.done)
}

// gc runs the background eviction loop at the given interval.
func (p *PoolStore[T]) gc(interval time.Duration) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	for {
		select {
		case <-p.done:
			p.mu.Lock()
			for _, t := range p.sessions {
				if err := t.Close(); err != nil {
					log.Println("session: close error:", err)
				}
			}
			clear(p.sessions)
			p.mu.Unlock()
			return
		case <-ticker.C:
			p.sweep()
		}
	}
}

// sweep removes all sessions whose last-seen time exceeds the TTL.
func (p *PoolStore[T]) sweep() {
	now := time.Now()
	p.mu.Lock()
	defer p.mu.Unlock()
	for id, t := range p.sessions {
		if now.Sub(t.LastSeen()) > p.ttl {
			delete(p.sessions, id)
			if err := t.Close(); err != nil {
				log.Println("session: close error:", err)
			}
		}
	}
}
