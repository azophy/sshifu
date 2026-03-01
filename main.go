package main

import (
	"embed"
	"html/template"
	"log"
	"net/http"
	"os"
)

//go:embed templates/* static/*
var files embed.FS

type PageData struct {
	Title   string
	Message string
}

func render(w http.ResponseWriter, name string, data PageData) {
	tmpl, err := template.ParseFS(files, "templates/layout.html", "templates/"+name)
	if err != nil {
		http.Error(w, err.Error(), 500)
		return
	}
	if err := tmpl.ExecuteTemplate(w, name, data); err != nil {
		http.Error(w, err.Error(), 500)
	}
}

func main() {
	port := os.Getenv("APP_PORT")
	if port == "" {
		port = "8080"
	}

	static := http.FileServer(http.FS(files))
	http.Handle("/static/", static)

	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		render(w, "home.html", PageData{Title: "Home", Message: "Hello from Go!"})
	})

	http.HandleFunc("/about", func(w http.ResponseWriter, r *http.Request) {
		render(w, "about.html", PageData{Title: "About"})
	})

	log.Printf("Server starting on http://localhost:%s", port)
	log.Fatal(http.ListenAndServe(":"+port, nil))
}
