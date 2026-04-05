package render

import (
	"bytes"
	"embed"
	"fmt"
	"html/template"
	"io"
	"strings"

	"myboardgamecollection/internal/model"
	"myboardgamecollection/internal/viewmodel"
)

// Renderer manages parsed templates and provides page/partial rendering.
type Renderer struct {
	tmpl map[string]*template.Template
}

// New parses all templates from the embedded filesystem and returns a Renderer.
func New(templateFS embed.FS) *Renderer {
	funcMap := template.FuncMap{
		"split": strings.Split,
		"add":   func(a, b int) int { return a + b },
		"playerAidsData": func(gameID int64, aids []model.PlayerAid) viewmodel.PlayerAidsListData {
			return viewmodel.PlayerAidsListData{GameID: gameID, Aids: aids}
		},
	}

	layout := template.Must(
		template.New("layout.html").Funcs(funcMap).ParseFS(templateFS, "templates/layout.html"),
	)

	tmpl := map[string]*template.Template{
		// Full pages (with layout)
		"home": template.Must(template.Must(layout.Clone()).ParseFS(templateFS,
			"templates/home.html")),
		"games": template.Must(template.Must(layout.Clone()).ParseFS(templateFS,
			"templates/games.html", "templates/games_result.html")),
		"game_detail": template.Must(template.Must(layout.Clone()).ParseFS(templateFS,
			"templates/game_detail.html")),
		"game_edit": template.Must(template.Must(layout.Clone()).ParseFS(templateFS,
			"templates/game_edit.html")),
		"rules": template.Must(template.Must(layout.Clone()).ParseFS(templateFS,
			"templates/rules.html", "templates/rules_content.html", "templates/player_aids_list.html")),
		"import": template.Must(template.Must(layout.Clone()).ParseFS(templateFS,
			"templates/import.html")),
		"discover": template.Must(template.Must(layout.Clone()).ParseFS(templateFS,
			"templates/discover.html", "templates/discover_result.html")),
		"vibes": template.Must(template.Must(layout.Clone()).ParseFS(templateFS,
			"templates/vibes.html")),
		"login": template.Must(template.Must(layout.Clone()).ParseFS(templateFS,
			"templates/login.html")),

		// Partials (no layout, for HTMX responses)
		"games_result": template.Must(
			template.New("").Funcs(funcMap).ParseFS(templateFS, "templates/games_result.html")),
		"discover_result": template.Must(
			template.New("").Funcs(funcMap).ParseFS(templateFS, "templates/discover_result.html")),
		"rules_content": template.Must(
			template.New("").Funcs(funcMap).ParseFS(templateFS, "templates/rules_content.html", "templates/player_aids_list.html")),
		"player_aids_list": template.Must(
			template.New("").Funcs(funcMap).ParseFS(templateFS, "templates/player_aids_list.html")),
		"import_result": template.Must(
			template.New("").Funcs(funcMap).ParseFS(templateFS, "templates/import_result.html")),
	}

	return &Renderer{tmpl: tmpl}
}

// Page renders a full page (with layout) to the writer.
// username is the BGG username of the logged-in user; pass "" for the login page.
// It buffers the output so a template error never produces a partial response.
func (r *Renderer) Page(w io.Writer, name, title string, data any, username, csrfToken string) error {
	t, ok := r.tmpl[name]
	if !ok {
		return fmt.Errorf("template %q not found", name)
	}
	var buf bytes.Buffer
	if err := t.ExecuteTemplate(&buf, "layout.html", viewmodel.PageData{Title: title, User: username, CSRFToken: csrfToken, Data: data}); err != nil {
		return err
	}
	_, err := buf.WriteTo(w)
	return err
}

// Partial renders a template partial (no layout) to the writer.
// It buffers the output so a template error never produces a partial response.
func (r *Renderer) Partial(w io.Writer, name string, data any) error {
	t, ok := r.tmpl[name]
	if !ok {
		return fmt.Errorf("template %q not found", name)
	}
	var buf bytes.Buffer
	if err := t.ExecuteTemplate(&buf, name, data); err != nil {
		return err
	}
	_, err := buf.WriteTo(w)
	return err
}
