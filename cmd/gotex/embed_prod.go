// Copyright (c) 2026 Benjamin Benno Falkner
// SPDX-License-Identifier: MIT
// embed_prod.go embeds the compiled web assets for production mode.
//go:build !dev

package main

import (
	"embed"
	"io/fs"
	"log"
)

//go:embed dist/*
var embeddedFiles embed.FS

var wwwFiles fs.FS

func init() {
	var err error
	wwwFiles, err = fs.Sub(embeddedFiles, "dist")
	if err != nil {
		log.Fatalln("failed to load embedded files:", err)
	}
}
