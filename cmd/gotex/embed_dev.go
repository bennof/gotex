// Copyright (c) 2026 Benjamin Benno Falkner
// SPDX-License-Identifier: MIT
// embed_dev.go is used in development mode to serve local static www files.
//go:build dev

package main

import (
	"io/fs"
	"os"
)

var wwwFiles fs.FS = os.DirFS("cmd/gotex/www")
