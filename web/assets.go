package web

import (
	"embed"
	"html/template"
)

//go:embed templates/index.html
var templates embed.FS

func IndexTemplate() *template.Template {
	return template.Must(template.ParseFS(templates, "templates/index.html"))
}
