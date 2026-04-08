package handler

import (
	"log/slog"
	"net/http"
	"regexp"
	"strings"

	"myboardgamecollection/internal/httpx"
	"myboardgamecollection/internal/viewmodel"
)

// usernameRE allows alphanumeric characters, dots, underscores, hyphens, and spaces.
var usernameRE = regexp.MustCompile(`^[a-zA-Z0-9_.+\- ]{1,60}$`)

func (h *Handler) HandleLoginPage(w http.ResponseWriter, r *http.Request) {
	if err := h.renderPage(w, r, "login", "Sign In", viewmodel.AuthPageData{}); err != nil {
		http.Error(w, "failed to render page", http.StatusInternalServerError)
	}
}

func (h *Handler) HandleSignupPage(w http.ResponseWriter, r *http.Request) {
	if err := h.renderPage(w, r, "signup", "Create Account", viewmodel.AuthPageData{}); err != nil {
		http.Error(w, "failed to render page", http.StatusInternalServerError)
	}
}

func (h *Handler) HandleSignup(w http.ResponseWriter, r *http.Request) {
	username := strings.TrimSpace(r.FormValue("username"))
	password := r.FormValue("password")
	password2 := r.FormValue("password2")
	bggUsername := strings.TrimSpace(r.FormValue("bgg_username"))
	email := strings.TrimSpace(r.FormValue("email"))

	if !usernameRE.MatchString(username) {
		h.recordLoginFailure(r)
		renderAuthError(w, r, h, "signup", "Create Account", "Invalid username (1–60 characters, letters, numbers, dots, hyphens, underscores).")
		return
	}
	if len(password) < 8 {
		h.recordLoginFailure(r)
		renderAuthError(w, r, h, "signup", "Create Account", "Password must be at least 8 characters.")
		return
	}
	if password != password2 {
		h.recordLoginFailure(r)
		renderAuthError(w, r, h, "signup", "Create Account", "Passwords do not match.")
		return
	}

	userID, err := h.Store.RegisterUser(username, password, bggUsername, email)
	if err != nil {
		slog.Error("RegisterUser", "username", username, "error", err)
		h.recordLoginFailure(r)
		renderAuthError(w, r, h, "signup", "Create Account", err.Error())
		return
	}

	h.createSessionAndRedirect(w, r, userID)
}

func (h *Handler) HandleLogin(w http.ResponseWriter, r *http.Request) {
	username := strings.TrimSpace(r.FormValue("username"))
	password := r.FormValue("password")

	userID, err := h.Store.AuthenticateUser(username, password)
	if err != nil {
		h.recordLoginFailure(r)
		renderAuthError(w, r, h, "login", "Sign In", "Invalid username or password.")
		return
	}

	// Session rotation: remove any existing sessions for this user.
	if err := h.Store.DeleteUserSessions(userID); err != nil {
		slog.Warn("DeleteUserSessions", "userID", userID, "error", err)
	}

	h.createSessionAndRedirect(w, r, userID)
}

func (h *Handler) createSessionAndRedirect(w http.ResponseWriter, r *http.Request, userID int64) {
	token, err := h.Store.CreateSession(userID)
	if err != nil {
		slog.Error("CreateSession", "userID", userID, "error", err)
		http.Error(w, "internal error", http.StatusInternalServerError)
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
	http.Redirect(w, r, "/", http.StatusSeeOther)
}

func (h *Handler) HandleChangePasswordPage(w http.ResponseWriter, r *http.Request) {
	data := viewmodel.AuthPageData{Success: r.URL.Query().Get("success") == "1"}
	if err := h.renderPage(w, r, "change_password", "Change Password", data); err != nil {
		http.Error(w, "failed to render page", http.StatusInternalServerError)
	}
}

func (h *Handler) HandleChangePassword(w http.ResponseWriter, r *http.Request) {
	userID, ok := h.requireUserID(w, r)
	if !ok {
		return
	}

	current := r.FormValue("current_password")
	newPass := r.FormValue("new_password")
	newPass2 := r.FormValue("new_password2")

	if len(newPass) < 8 {
		renderAuthError(w, r, h, "change_password", "Change Password", "New password must be at least 8 characters.")
		return
	}
	if newPass != newPass2 {
		renderAuthError(w, r, h, "change_password", "Change Password", "New passwords do not match.")
		return
	}

	if err := h.Store.ChangePassword(userID, current, newPass); err != nil {
		renderAuthError(w, r, h, "change_password", "Change Password", err.Error())
		return
	}

	http.Redirect(w, r, "/profile/change-password?success=1", http.StatusSeeOther)
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

// recordLoginFailure records a failed login attempt for rate limiting.
func (h *Handler) recordLoginFailure(r *http.Request) {
	if h.LoginLimiter != nil {
		h.LoginLimiter.Record(httpx.ClientIP(r))
	}
}

func renderAuthError(w http.ResponseWriter, r *http.Request, h *Handler, template, title, msg string) {
	w.WriteHeader(http.StatusUnprocessableEntity)
	if err := h.renderPage(w, r, template, title, viewmodel.AuthPageData{Error: msg}); err != nil {
		http.Error(w, "failed to render page", http.StatusInternalServerError)
	}
}
