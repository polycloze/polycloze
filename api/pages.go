// Copyright (c) 2022 Levi Gruspe
// License: GNU AGPLv3 or later

package api

import (
	"embed"
	"html/template"
	"log"
	"net/http"
	"path/filepath"

	"github.com/go-chi/chi/v5"
)

//go:embed js/dist/index.* templates/*.html js/public/* js/dist/serviceworker.*
var fs embed.FS

var templates *template.Template = template.Must(template.ParseFS(fs, "templates/*.html"))

func init() {
	// Check templates.
	names := []string{"home.html", "study.html"}
	for _, name := range names {
		if t := templates.Lookup(name); t == nil {
			log.Fatal("missing template:", name)
		}
	}
}

func serveFile(path string, contentType string) func(http.ResponseWriter, *http.Request) {
	bytes, err := fs.ReadFile(path)
	if err != nil {
		log.Fatal("file not found:", path)
	}
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", contentType)
		if _, err := w.Write(bytes); err != nil {
			log.Println(err)
		}
	}
}

func serveServiceWorker() func(http.ResponseWriter, *http.Request) {
	return serveFile(filepath.Join("js", "dist", "serviceworker.js"), "application/javascript")
}

func serveServiceWorkerSourceMap() func(http.ResponseWriter, *http.Request) {
	return serveFile(filepath.Join("js", "dist", "serviceworker.js.map"), "application/json")
}

func serveDist(w http.ResponseWriter, r *http.Request) {
	filename := chi.URLParam(r, "filename")
	bytes, err := fs.ReadFile(filepath.Join("js", "dist", filename))
	if err != nil {
		http.NotFound(w, r)
		return
	}

	switch filename {
	case "index.js":
		w.Header().Set("Content-Type", "application/javascript")
	case "index.css":
		w.Header().Set("Content-Type", "text/css")
	}

	if _, err := w.Write(bytes); err != nil {
		log.Println(err)
	}
}

func showPage(name string) func(http.ResponseWriter, *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		if err := templates.ExecuteTemplate(w, name, nil); err != nil {
			log.Println(err)
		}
	}
}
