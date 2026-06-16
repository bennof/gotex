// Copyright (c) 2026 Benjamin Benno Falkner
// SPDX-License-Identifier: MIT

// config.go defines configuration options and defaults for the simpleserver package.
package simpleserver

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"strings"
	"time"
)

// Config holds HTTP server tuning parameters.
type Config struct {
	Addr              string   `json:"addr"`
	ReadTimeout       Duration `json:"read_timeout"`
	ReadHeaderTimeout Duration `json:"read_header_timeout"`
	WriteTimeout      Duration `json:"write_timeout"`
	IdleTimeout       Duration `json:"idle_timeout"`
}

// Duration wraps time.Duration to support JSON strings like "15s".
type Duration struct{ time.Duration }

func (d *Duration) UnmarshalJSON(data []byte) error {
	var s string
	if err := json.Unmarshal(data, &s); err != nil {
		return err
	}
	dur, err := time.ParseDuration(s)
	if err != nil {
		return err
	}
	d.Duration = dur
	return nil
}

// DefaultConfig returns a Config with sensible defaults.
func DefaultConfig() Config {
	return Config{
		Addr:              ":8080",
		ReadTimeout:       Duration{15 * time.Second},
		ReadHeaderTimeout: Duration{5 * time.Second},
		WriteTimeout:      Duration{200 * time.Second},
		IdleTimeout:       Duration{60 * time.Second},
	}
}

func (c *Config) UnmarshalJSON(data []byte) error {
	*c = DefaultConfig()
	type plain Config
	return json.Unmarshal(data, (*plain)(c))
}

// ConfigFromFile loads a Config from a JSON file.
// Calls log.Fatal on any error.
func ConfigFromFile(path string) Config {
	cfg := DefaultConfig()
	data, err := os.ReadFile(path)
	if err != nil {
		log.Fatalf("config: cannot read %s: %v", path, err)
	}
	if err := json.Unmarshal(data, &cfg); err != nil {
		log.Fatalf("config: cannot parse %s: %v", path, err)
	}
	return cfg
}

// normalizeAddr ensures the address is in host:port format.
// If port is empty, it defaults to ":8080".
// If port contains a colon it is returned as-is, otherwise ":" is prepended.
func NormalizeAddr(port string) string {
	if port == "" {
		return ":8080"
	}
	if strings.Contains(port, ":") {
		return port
	}
	return fmt.Sprintf(":%s", port)
}
