# Go Web Template Example

A minimal Go web application demonstrating HTML templating with `html/template` and `embed`, featuring automatic routing based on page filenames.

## Features

- Layout-based templating (shared header, footer, navigation)
- **Automatic routing** - pages are discovered and routed based on filenames
- Subfolder support for organizing pages
- Embedded static files (CSS, images, templates) - single binary deployment
- Configurable port via `APP_PORT` environment variable
- Minimal CSS using [mvp.css](https://github.com/andybrewer/mvp)

## Project Structure

```
go-web/
├── main.go              # Application entry point
├── layouts/
│   └── layout.html      # Shared layout template(s)
├── pages/
│   ├── home.html        # Home page (served at /)
│   ├── about.html       # About page (served at /about)
│   └── blog/            # Subfolder support
│       └── post.html    # Blog post (served at /blog/post)
└── assets/
    ├── mvp.css          # Stylesheet
    └── images/          # Images and other static files
```

## Conventions

### Folders

| Folder    | Purpose                              |
|-----------|--------------------------------------|
| `layouts/` | Shared layout templates (header, footer, navigation) |
| `pages/`   | Page templates - automatically routed based on filename |
| `assets/`  | Static files (CSS, images, JS, etc.) served under `/static/` |

### Automatic Routing

Routes are generated automatically from filenames in the `pages/` directory:

| Filename              | URL Path         |
|-----------------------|------------------|
| `home.html`           | `/`              |
| `about.html`          | `/about`         |
| `contact.html`        | `/contact`       |
| `blog/post.html`      | `/blog/post`     |
| `docs/guide.html`     | `/docs/guide`    |

### Creating a Layout

Create `layouts/layout.html` with a `content` placeholder:

```html
<!DOCTYPE html>
<html lang="en">
<head>
  <meta charset="UTF-8">
  <title>{{.Title}}</title>
  <link rel="stylesheet" href="/static/mvp.css">
</head>
<body>
  <header>
    <nav>
      <a href="/">Home</a>
      <a href="/about">About</a>
    </nav>
  </header>
  <main>
    {{template "content" .}}
  </main>
  <footer><p>My App</p></footer>
</body>
</html>
```

### Creating a Page

Create a page in `pages/` with a `content` block:

```html
{{template "layout.html" .}}

{{define "content"}}
<h1>Page Title</h1>
<p>Your content here.</p>
{{end}}
```

That's it! The route is automatically registered based on the filename.

### Subfolder Support

Organize pages in subfolders:

```
pages/
└── blog/
    └── my-post.html
```

This creates a route at `/blog/my-post`.

### Static Files

Place all static files in `assets/`. They are served under `/static/`:

```html
<link rel="stylesheet" href="/static/mvp.css">
<img src="/static/images/logo.png" alt="Logo">
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
APP_PORT=3000 ./myapp
```

Default port is `8080`.

## Pages

- **Home**: http://localhost:8080/
- **About**: http://localhost:8080/about

## How it works

1. `//go:embed` bundles all layouts, pages, and assets into the binary
2. On startup, the app scans `pages/` directory for all `.html` files
3. Routes are automatically registered based on filenames
4. Each request renders the page with the shared layout
5. The compiled binary contains everything - no external files needed at runtime

## Adding a new page

1. Create `pages/newpage.html`:
   ```html
   {{template "layout.html" .}}

   {{define "content"}}
   <h1>New Page</h1>
   <p>Your content here.</p>
   {{end}}
   ```

2. That's it! The route `/newpage` is automatically available.
