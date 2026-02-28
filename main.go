package main

import (
	"embed"
	"html/template"
	"net/http"
)

//go:embed templates/* static/*
var files embed.FS

var tmpl = template.Must(template.ParseFS(files, "templates/*.html"))

type PageData struct {
	Title   string
	Message string
}

func render(w http.ResponseWriter, name string, data PageData) {
	if err := tmpl.ExecuteTemplate(w, name, data); err != nil {
		http.Error(w, err.Error(), 500)
	}
}

func main() {
	static := http.FileServer(http.FS(files))
	http.Handle("/static/", static)

	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		render(w, "home.html", PageData{Title: "Home", Message: "Hello from Go!"})
	})

	http.HandleFunc("/about", func(w http.ResponseWriter, r *http.Request) {
		render(w, "about.html", PageData{Title: "About"})
	})

	http.ListenAndServe(":8080", nil)
}
