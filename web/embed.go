package web

import (
	"embed"
	"io/fs"
)

//go:embed dist
var assets embed.FS

// Dist is rooted at the generated Vite distribution directory.
var Dist fs.FS = mustSub(assets, "dist")

func mustSub(source fs.FS, dir string) fs.FS {
	sub, err := fs.Sub(source, dir)
	if err != nil {
		panic(err)
	}
	return sub
}
