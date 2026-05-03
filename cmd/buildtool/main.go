// Copyright (c) 2026 Benjamin Benno Falkner
// SPDX-License-Identifier: MIT

// buildtool minifies and bundles the web assets for the gotex server.
// It parses index.html, extracts local CSS and JS references,
// processes them through esbuild, and writes the output to dist/.
package main

import (
	"log"
	"os"
	"path/filepath"
	"strings"

	"github.com/evanw/esbuild/pkg/api"
	"golang.org/x/net/html"
)

const indexHtml = "cmd/gotex/www/index.html"
const distDir = "cmd/gotex/dist"

var srcDir = filepath.Dir(indexHtml)

func main() {
	log.Println("Reading html", indexHtml, "...")
	doc, err := readHTML(indexHtml)
	if err != nil {
		log.Fatalln(err)
	}

	var styles []*html.Node
	var scripts []*html.Node

	// walk traverses the HTML node tree recursively.
	// It collects local <link> and <script> nodes into the styles and scripts slices.
	var walk func(*html.Node)
	walk = func(n *html.Node) {
		if n.Type == html.ElementNode {
			switch n.Data {
			case "link":
				// Collect local stylesheet references.
				for _, a := range n.Attr {
					if a.Key == "href" && isLocalPath(a.Val) {
						log.Println("CSS found:", a.Val)
						styles = append(styles, n)
					}
				}
			case "script":
				// Collect local script references.
				for _, a := range n.Attr {
					if a.Key == "src" && isLocalPath(a.Val) {
						log.Println("JS found:", a.Val)
						scripts = append(scripts, n)
					}
				}
			}
		}
		// Recurse into child nodes.
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			walk(c)
		}
	}
	walk(doc)

	styles = validateNodes(styles, "href", srcDir)
	scripts = validateNodes(scripts, "src", srcDir)

	entrypoints := append(genEntryPoints(srcDir, scripts...), genEntryPoints(srcDir, styles...)...)

	// Ensure the dist directory exists.
	err = os.MkdirAll(distDir, os.ModePerm)
	if err != nil {
		log.Fatalln("failed to create dist directory:", err)
	}

	result := api.Build(api.BuildOptions{
		EntryPoints:       entrypoints,
		Outdir:            distDir,
		Bundle:            true,
		MinifyWhitespace:  true,
		MinifyIdentifiers: true,
		MinifySyntax:      true,
		Write:             true,
		Loader: map[string]api.Loader{
			".css": api.LoaderCSS,
			".js":  api.LoaderJS,
		},
	})

	if len(result.Errors) > 0 {
		for _, e := range result.Errors {
			log.Println("error:", e.Text)
		}
		os.Exit(1)
	}

	if len(result.Warnings) > 0 {
		for _, w := range result.Warnings {
			log.Println("warning:", w.Text)
		}
	}

	// Update HTML attributes to point to the generated output files,
	// then write the modified index.html to dist/.
	updateAndWriteHTML(doc, result.OutputFiles, distDir)

	log.Println("build complete →", distDir)
}

// readHTML opens and parses an HTML file, returning the root node.
func readHTML(filename string) (*html.Node, error) {
	f, err := os.Open(filename)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	return html.Parse(f)
}

// isLocalPath returns true if the path is a local file reference,
// i.e. not an absolute URL with a scheme like http:// or https://.
func isLocalPath(path string) bool {
	return !strings.HasPrefix(path, "http://") &&
		!strings.HasPrefix(path, "https://") &&
		!strings.HasPrefix(path, "//") &&
		path != ""
}

// validateNodes checks whether the files referenced by the given nodes exist on disk.
// Nodes pointing to missing files are skipped with a warning.
// Returns only the nodes whose files were found locally.
func validateNodes(nodes []*html.Node, attrKey string, srcDir string) []*html.Node {
	var valid []*html.Node
	for _, n := range nodes {
		for _, a := range n.Attr {
			if a.Key == attrKey {
				fullPath := filepath.Join(srcDir, a.Val)
				if _, err := os.Stat(fullPath); err != nil {
					if os.IsNotExist(err) {
						log.Printf("warning: referenced file not found, skipping: %s", fullPath)
					} else {
						log.Printf("warning: could not stat file, skipping: %s (%v)", fullPath, err)
					}
				} else {
					log.Println("file found:", fullPath)
					valid = append(valid, n)
				}
			}
		}
	}
	return valid
}

// genEntryPoints collects the file paths referenced by the given HTML nodes.
// It reads both href (stylesheets) and src (scripts) attributes
// and resolves each value relative to srcDir.
func genEntryPoints(srcDir string, nodes ...*html.Node) []string {
	var entryPoints []string
	for _, n := range nodes {
		for _, a := range n.Attr {
			if a.Key == "href" || a.Key == "src" {
				entryPoints = append(entryPoints, filepath.Join(srcDir, a.Val))
			}
		}
	}
	return entryPoints
}

// updateAndWriteHTML updates href and src attributes in the HTML tree
// to point to the esbuild output files, then writes the result to dist/index.html.
func updateAndWriteHTML(doc *html.Node, outputFiles []api.OutputFile, distDir string) {
	// Build a map from original base name to output file base name.
	outMap := make(map[string]string)
	for _, f := range outputFiles {
		base := filepath.Base(f.Path)
		// Strip hash suffix if present: style-A1B2C3.css → style.css
		outMap[base] = base
	}

	// walk updates href and src attributes to the new output file names.
	var walk func(*html.Node)
	walk = func(n *html.Node) {
		if n.Type == html.ElementNode {
			for i, a := range n.Attr {
				if a.Key == "href" || a.Key == "src" {
					base := filepath.Base(a.Val)
					if newName, ok := outMap[base]; ok {
						n.Attr[i].Val = newName
					}
				}
			}
		}
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			walk(c)
		}
	}
	walk(doc)

	// Write the updated HTML to dist/index.html.
	out, err := os.Create(filepath.Join(distDir, "index.html"))
	if err != nil {
		log.Fatalln("failed to write index.html:", err)
	}
	defer out.Close()

	if err := html.Render(out, doc); err != nil {
		log.Fatalln("failed to render index.html:", err)
	}
	log.Println("HTML written → dist/index.html")
}
