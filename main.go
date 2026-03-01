package main

import (
	"embed"
	"fmt"
	"html/template"
	"io/fs"
	"log"
	"net/http"
	"os"
	"path"
	"strings"
)

//go:embed layouts/* pages/* assets/*
var files embed.FS

type PageData struct {
	Title   string
	Message string
}

func render(w http.ResponseWriter, name string, data PageData) {
	tmpl, err := template.ParseFS(files, "layouts/layout.html", "pages/"+name)
	if err != nil {
		http.Error(w, err.Error(), 500)
		return
	}
	if err := tmpl.ExecuteTemplate(w, name, data); err != nil {
		http.Error(w, err.Error(), 500)
	}
}

// discoverPages scans the pages directory and returns all page filenames
func discoverPages() ([]string, error) {
	var pages []string
	err := fs.WalkDir(files, "pages", func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if !d.IsDir() && strings.HasSuffix(path, ".html") {
			pages = append(pages, d.Name())
		}
		return nil
	})
	return pages, err
}

// pathToPage converts a URL path to a page filename
func pathToPage(urlPath string) string {
	if urlPath == "/" {
		return "home.html"
	}
	// Remove leading/trailing slashes and convert to filename
	cleanPath := strings.Trim(urlPath, "/")
	return cleanPath + ".html"
}

// registerRoutes automatically registers routes for all pages
func registerRoutes() error {
	pages, err := discoverPages()
	if err != nil {
		return fmt.Errorf("failed to discover pages: %w", err)
	}

	for _, page := range pages {
		pageName := strings.TrimSuffix(page, ".html")
		routePath := "/" + pageName
		
		// Handle subfolder pages (e.g., pages/blog/post.html -> /blog/post)
		// The filename itself may contain subfolder structure when embedded
		
		http.HandleFunc(routePath, func(w http.ResponseWriter, r *http.Request) {
			pageFile := pathToPage(r.URL.Path)
			
			// Check if page exists
			if _, err := files.Open("pages/" + pageFile); err != nil {
				http.NotFound(w, r)
				return
			}
			
			// Extract title from page name
			title := strings.Title(strings.TrimSuffix(path.Base(pageFile), ".html"))
			
			render(w, pageFile, PageData{Title: title})
		})
	}
	
	return nil
}

func main() {
	port := os.Getenv("APP_PORT")
	if port == "" {
		port = "8080"
	}

	// Serve static assets from assets/ directory
	static := http.FileServer(http.FS(files))
	http.Handle("/static/", http.StripPrefix("/static/", static))

	// Auto-register routes for all pages
	if err := registerRoutes(); err != nil {
		log.Fatalf("Failed to register routes: %v", err)
	}

	log.Printf("Server starting on http://localhost:%s", port)
	log.Fatal(http.ListenAndServe(":"+port, nil))
}
