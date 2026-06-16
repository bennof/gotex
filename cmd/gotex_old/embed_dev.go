// embed_dev.go
//go:build dev

package main

import (
	"io/fs"
	"os"
)

var wwwFiles fs.FS = os.DirFS("cmd/gotex/www")
