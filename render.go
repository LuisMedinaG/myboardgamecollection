package main

import (
	"embed"
	"fmt"
	"html/template"
	"io"
	"strings"
)

//go:embed templates
var templateFS embed.FS

var tmpl map[string]*template.Template

var funcMap = template.FuncMap{
	"split": strings.Split,
	"add":   func(a, b int) int { return a + b },
	"playerAidsData": func(gameID int64, aids []PlayerAid) PlayerAidsListData {
		return PlayerAidsListData{GameID: gameID, Aids: aids}
	},
}

// PageData wraps a title and arbitrary data for full-page renders.
type PageData struct {
	Title string
	Data  any
}

func initTemplates() {
	layout := template.Must(
		template.New("layout.html").Funcs(funcMap).ParseFS(templateFS, "templates/layout.html"),
	)

	tmpl = map[string]*template.Template{
		// Full pages (with layout)
		"home": template.Must(template.Must(layout.Clone()).ParseFS(templateFS,
			"templates/home.html")),
		"games": template.Must(template.Must(layout.Clone()).ParseFS(templateFS,
			"templates/games.html", "templates/games_result.html")),
		"game_detail": template.Must(template.Must(layout.Clone()).ParseFS(templateFS,
			"templates/game_detail.html")),
		"rules": template.Must(template.Must(layout.Clone()).ParseFS(templateFS,
			"templates/rules.html", "templates/rules_content.html", "templates/player_aids_list.html")),
		"import": template.Must(template.Must(layout.Clone()).ParseFS(templateFS,
			"templates/import.html")),

		// Partials (no layout, for HTMX responses)
		"games_result": template.Must(
			template.New("").Funcs(funcMap).ParseFS(templateFS, "templates/games_result.html")),
		"rules_content": template.Must(
			template.New("").Funcs(funcMap).ParseFS(templateFS, "templates/rules_content.html", "templates/player_aids_list.html")),
		"player_aids_list": template.Must(
			template.New("").Funcs(funcMap).ParseFS(templateFS, "templates/player_aids_list.html")),
		"import_result": template.Must(
			template.New("").Funcs(funcMap).ParseFS(templateFS, "templates/import_result.html")),
	}
}

func renderPage(w io.Writer, name string, title string, data any) error {
	t, ok := tmpl[name]
	if !ok {
		return fmt.Errorf("template %q not found", name)
	}
	return t.ExecuteTemplate(w, "layout.html", PageData{Title: title, Data: data})
}

func renderPartial(w io.Writer, name string, data any) error {
	t, ok := tmpl[name]
	if !ok {
		return fmt.Errorf("template %q not found", name)
	}
	return t.ExecuteTemplate(w, name, data)
}
