// Package embed provides embedded static assets for the application.
//
//go:generate sh -c "cp ../web/login.html ./login.html"
package embed

import (
	"embed"
	"html/template"
)

//go:embed login.html
var loginFile embed.FS

// LoadLoginTemplate loads the login.html template from embedded content.
func LoadLoginTemplate() (*template.Template, error) {
	return template.ParseFS(loginFile, "login.html")
}
