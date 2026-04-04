package handler

import (
	"log/slog"
	"net/http"
	"regexp"
	"strings"

	"myboardgamecollection/internal/viewmodel"
)

// bggUsernameRE allows the characters BGG usernames realistically contain.
// This is a conservative allow-list; BGG itself is more permissive but these
// are the characters we trust for URL-safe identity strings.
var bggUsernameRE = regexp.MustCompile(`^[a-zA-Z0-9_.+\- ]{1,60}$`)

func (h *Handler) HandleLoginPage(w http.ResponseWriter, r *http.Request) {
	if err := h.Renderer.Page(w, "login", "Sign In", viewmodel.LoginPageData{}, ""); err != nil {
		http.Error(w, "failed to render page", http.StatusInternalServerError)
	}
}

func (h *Handler) HandleLogin(w http.ResponseWriter, r *http.Request) {
	username := strings.TrimSpace(r.FormValue("username"))
	if !bggUsernameRE.MatchString(username) {
		renderLoginError(w, r, h, "Please enter a valid BGG username (1–60 characters).")
		return
	}

	userID, err := h.Store.FindOrCreateUser(username)
	if err != nil {
		slog.Error("FindOrCreateUser", "username", username, "error", err)
		renderLoginError(w, r, h, "Something went wrong. Please try again.")
		return
	}

	token, err := h.Store.CreateSession(userID)
	if err != nil {
		slog.Error("CreateSession", "userID", userID, "error", err)
		renderLoginError(w, r, h, "Something went wrong. Please try again.")
		return
	}

	secure := r.TLS != nil || strings.EqualFold(r.Header.Get("X-Forwarded-Proto"), "https")
	http.SetCookie(w, &http.Cookie{
		Name:     "sid",
		Value:    token,
		Path:     "/",
		MaxAge:   30 * 24 * 3600, // 30 days
		HttpOnly: true,
		SameSite: http.SameSiteStrictMode,
		Secure:   secure,
	})
	http.Redirect(w, r, "/games", http.StatusSeeOther)
}

func (h *Handler) HandleLogout(w http.ResponseWriter, r *http.Request) {
	cookie, err := r.Cookie("sid")
	if err == nil {
		if err := h.Store.DeleteSession(cookie.Value); err != nil {
			slog.Error("DeleteSession", "error", err)
		}
	}
	http.SetCookie(w, &http.Cookie{Name: "sid", Path: "/", MaxAge: -1})
	http.Redirect(w, r, "/login", http.StatusSeeOther)
}

func renderLoginError(w http.ResponseWriter, r *http.Request, h *Handler, msg string) {
	w.WriteHeader(http.StatusUnprocessableEntity)
	if err := h.Renderer.Page(w, "login", "Sign In", viewmodel.LoginPageData{Error: msg}, ""); err != nil {
		http.Error(w, "failed to render page", http.StatusInternalServerError)
	}
}
