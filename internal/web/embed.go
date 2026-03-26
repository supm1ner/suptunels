package web

import (
	"embed"
	"io/fs"
	"net/http"
)

//go:embed static/*
var staticFiles embed.FS

func Handler() http.Handler {
	f, _ := fs.Sub(staticFiles, "static")
	return http.FileServer(http.FS(f))
}
