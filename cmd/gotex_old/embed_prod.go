// embed_prod.go
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
