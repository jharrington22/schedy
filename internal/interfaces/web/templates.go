package web

import (
	"embed"
	"html/template"
)

//go:embed templates/*.html
var templatesFS embed.FS

func ParseTemplates() (*template.Template, error) {
	return template.ParseFS(templatesFS, "templates/*.html")
}
