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

	// fullPage clones the layout and parses page-specific templates into it.
	fullPage := func(files ...string) *template.Template {
		return template.Must(template.Must(layout.Clone()).ParseFS(templateFS, files...))
	}
	// partial parses a standalone template (no layout) for HTMX responses.
	partial := func(files ...string) *template.Template {
		return template.Must(template.New("").Funcs(funcMap).ParseFS(templateFS, files...))
	}

	tmpl := map[string]*template.Template{
		// Full pages (with layout)
		"home":            fullPage("templates/home.html"),
		"games":           fullPage("templates/games.html", "templates/games_result.html"),
		"game_detail":     fullPage("templates/game_detail.html"),
		"game_edit":       fullPage("templates/game_edit.html"),
		"rules":           fullPage("templates/rules.html", "templates/rules_content.html", "templates/player_aids_list.html"),
		"import":          fullPage("templates/import.html"),
		"import_csv":      fullPage("templates/import_csv.html"),
		"discover":        fullPage("templates/discover.html", "templates/discover_result.html"),
		"vibes":           fullPage("templates/vibes.html"),
		"login":           fullPage("templates/login.html"),
		"signup":          fullPage("templates/signup.html"),
		"change_password": fullPage("templates/change_password.html"),

		// Partials (no layout, for HTMX responses)
		"games_result":      partial("templates/games_result.html"),
		"discover_result":   partial("templates/discover_result.html"),
		"rules_content":     partial("templates/rules_content.html", "templates/player_aids_list.html"),
		"player_aids_list":  partial("templates/player_aids_list.html"),
		"import_result":     partial("templates/import_result.html"),
		"import_csv_preview": partial("templates/import_csv_preview.html"),
		"import_csv_result": partial("templates/import_csv_result.html"),
	}

	return &Renderer{tmpl: tmpl}
}

// Page renders a full page (with layout) to the writer.
// It buffers the output so a template error never produces a partial response.
func (r *Renderer) Page(w io.Writer, name, title string, data any, username, csrfToken string) error {
	t, ok := r.tmpl[name]
	if !ok {
		return fmt.Errorf("template %q not found", name)
	}

	var buf bytes.Buffer
	if err := t.ExecuteTemplate(&buf, "layout.html", viewmodel.PageData{
		Title:     title,
		User:      username,
		CSRFToken: csrfToken,
		Data:      data,
	}); err != nil {
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
