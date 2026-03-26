package web

import (
	"embed"
	"io/fs"
	"log"
	"net/http"
)

//go:embed static/*
var staticFiles embed.FS

func Handler() http.Handler {
	f, err := fs.Sub(staticFiles, "static")
	if err != nil {
		log.Printf("[Error] Sub FS: %v", err)
	}
	
	fileServer := http.FileServer(http.FS(f))
	
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		log.Printf("[Web] %s %s from %s", r.Method, r.URL.Path, r.RemoteAddr)
		fileServer.ServeHTTP(w, r)
	})
}
