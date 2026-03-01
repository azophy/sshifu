# Go Web Template Example

A minimal Go web application demonstrating HTML templating with `html/template` and `embed`.

## Features

- Layout-based templating (shared header, footer, navigation)
- Embedded static files (CSS, templates) - single binary deployment
- Configurable port via `PORT` environment variable
- Minimal CSS using [mvp.css](https://github.com/andybrewer/mvp)

## Project Structure

```
go-web/
├── main.go              # Application entry point
├── templates/
│   ├── layout.html      # Shared layout template
│   ├── home.html        # Home page
│   └── about.html       # About page
└── static/
    └── mvp.css          # Stylesheet
```

## Usage

### Run with Go

```bash
go run main.go
```

### Build and run

```bash
go build -o myapp
./myapp
```

### Configure port

```bash
PORT=3000 ./myapp
```

Default port is `8080`.

## Pages

- **Home**: http://localhost:8080/
- **About**: http://localhost:8080/about

## How it works

1. `//go:embed` bundles all templates and static files into the binary
2. Templates are parsed per-request: `layout.html` + page template together
3. Each page defines a `content` block that fills the layout's `{{template "content" .}}`
4. The compiled binary contains everything - no external files needed at runtime

## Adding a new page

1. Create `templates/newpage.html`:
   ```html
   {{template "layout.html" .}}
   
   {{define "content"}}
   <h1>New Page</h1>
   <p>Your content here.</p>
   {{end}}
   ```

2. Add a handler in `main.go`:
   ```go
   http.HandleFunc("/newpage", func(w http.ResponseWriter, r *http.Request) {
       render(w, "newpage.html", PageData{Title: "New Page"})
   })
   ```
