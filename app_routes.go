package main

import (
	"embed"
	"io/fs"
	"net/http"
	"path/filepath"

	"myboardgamecollection/internal/handler"
	"myboardgamecollection/internal/httpx"
	"myboardgamecollection/internal/store"
)

type routeDeps struct {
	store         *store.Store
	legacy        *handler.Handler
	games         *handler.GameHandler
	loginLimiter  *httpx.LoginLimiter
	sessionSecret string
	sessionKey    []byte
	dataDir       string
	staticFiles   embed.FS
}

func registerRoutes(mux *http.ServeMux, deps routeDeps) {
	registerAssetRoutes(mux, deps)
	registerAPIRoutes(mux, deps)
	registerPageRoutes(mux, deps)
}

func registerAssetRoutes(mux *http.ServeMux, deps routeDeps) {
	staticFS, _ := fs.Sub(deps.staticFiles, "static")
	mux.Handle("GET /static/", http.StripPrefix("/static/", http.FileServer(http.FS(staticFS))))

	uploads := http.StripPrefix("/uploads/", http.FileServer(http.Dir(filepath.Join(deps.dataDir, "uploads"))))
	mux.Handle("GET /uploads/", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-Content-Type-Options", "nosniff")
		uploads.ServeHTTP(w, r)
	}))
}

func registerAPIRoutes(mux *http.ServeMux, deps routeDeps) {
	mw := newRouteMiddleware(deps.store, deps.sessionKey, deps.sessionSecret, deps.loginLimiter)

	for _, route := range []struct {
		pattern string
		handler http.Handler
	}{
		{"POST /api/v1/auth/login", mw.apiPOSTPublic(deps.legacy.HandleAPILogin)},
		{"POST /api/v1/auth/refresh", mw.apiPOSTPublic(deps.legacy.HandleAPIRefresh)},
		{"POST /api/v1/auth/logout", mw.apiPOSTPublic(deps.legacy.HandleAPILogout)},
		{"GET /api/v1/ping", mw.apiGET(deps.legacy.HandleAPIPing)},
		{"GET /api/v1/games", mw.apiGET(deps.legacy.HandleAPIListGames)},
		{"GET /api/v1/games/{id}", mw.apiGET(deps.legacy.HandleAPIGetGame)},
		{"DELETE /api/v1/games/{id}", mw.apiDELETE(deps.legacy.HandleAPIDeleteGame)},
		{"POST /api/v1/games/{id}/vibes", mw.apiPOST(deps.legacy.HandleAPISetGameVibes)},
		{"POST /api/v1/games/bulk-vibes", mw.apiPOST(deps.legacy.HandleAPIBulkVibes)},
		{"GET /api/v1/vibes", mw.apiGET(deps.legacy.HandleAPIListVibes)},
		{"POST /api/v1/vibes", mw.apiPOST(deps.legacy.HandleAPICreateVibe)},
		{"PUT /api/v1/vibes/{id}", mw.apiPUT(deps.legacy.HandleAPIUpdateVibe)},
		{"DELETE /api/v1/vibes/{id}", mw.apiDELETE(deps.legacy.HandleAPIDeleteVibe)},
		{"POST /api/v1/import/sync", mw.apiPOST(deps.legacy.HandleAPIImportSync)},
		{"POST /api/v1/import/csv/preview", mw.apiPOST(deps.legacy.HandleAPIImportCSVPreview)},
		{"POST /api/v1/import/csv", mw.apiPOST(deps.legacy.HandleAPIImportCSV)},
		{"GET /api/v1/profile", mw.apiGET(deps.legacy.HandleAPIGetProfile)},
		{"PUT /api/v1/profile/bgg-username", mw.apiPUT(deps.legacy.HandleAPISetBGGUsername)},
		{"PUT /api/v1/profile/password", mw.apiPUT(deps.legacy.HandleAPIChangePassword)},
		{"PUT /api/v1/games/{id}/rules-url", mw.apiPUT(deps.legacy.HandleAPIUpdateRulesURL)},
		{"POST /api/v1/games/{id}/player-aids", mw.apiPOST(deps.legacy.HandleAPIUploadPlayerAid)},
		{"DELETE /api/v1/games/{id}/player-aids/{aid_id}", mw.apiDELETE(deps.legacy.HandleAPIDeletePlayerAid)},
		{"GET /api/v1/discover", mw.apiGET(deps.legacy.HandleAPIDiscover)},
	} {
		mux.Handle(route.pattern, route.handler)
	}
}

func registerPageRoutes(mux *http.ServeMux, deps routeDeps) {
	mw := newRouteMiddleware(deps.store, deps.sessionKey, deps.sessionSecret, deps.loginLimiter)

	for _, route := range []struct {
		pattern string
		handler http.Handler
	}{
		{"GET /login", mw.publicGET(deps.legacy.HandleLoginPage)},
		{"POST /login", mw.loginPOST(deps.legacy.HandleLogin)},
		{"GET /signup", mw.publicGET(deps.legacy.HandleSignupPage)},
		{"POST /signup", mw.signupPOST(deps.legacy.HandleSignup)},
		{"POST /logout", mw.logoutPOST(deps.legacy.HandleLogout)},
		{"GET /images/{bgg_id}", http.HandlerFunc(deps.legacy.HandleImage)},
		{"GET /{$}", mw.authGET(deps.legacy.HandleHome)},
		{"GET /games", mw.authGET(deps.games.HandleGames)},
		{"GET /games/{id}", mw.authGetWithID(deps.games.HandleGameDetail)},
		{"POST /games/bulk-vibes", mw.authPOST(deps.games.HandleBulkVibeAssign)},
		{"POST /games/{id}/delete", mw.authPostWithID(deps.games.HandleGameDelete)},
		{"GET /games/{id}/edit", mw.authGET(deps.legacy.HandleGameEdit)},
		{"POST /games/{id}/vibes", mw.authPOST(deps.legacy.HandleGameVibesSave)},
		{"GET /discover", mw.authGET(deps.legacy.HandleDiscover)},
		{"GET /vibes", mw.authGET(deps.legacy.HandleVibes)},
		{"POST /vibes", mw.authPOST(deps.legacy.HandleVibeCreate)},
		{"POST /vibes/batch-update", mw.authPOST(deps.legacy.HandleVibeBatchUpdate)},
		{"POST /vibes/{id}", mw.authPOST(deps.legacy.HandleVibeUpdate)},
		{"POST /vibes/{id}/delete", mw.authPOST(deps.legacy.HandleVibeDelete)},
		{"GET /games/{id}/rules", mw.authGET(deps.legacy.HandleRules)},
		{"POST /games/{id}/rules/url", mw.authPOST(deps.legacy.HandleRulesURLUpdate)},
		{"POST /games/{id}/rules/upload", mw.authPOST(deps.legacy.HandlePlayerAidUpload)},
		{"POST /games/{id}/rules/aids/{aid_id}/delete", mw.authPOST(deps.legacy.HandlePlayerAidDelete)},
		{"GET /import", mw.authGET(deps.legacy.HandleImport)},
		{"POST /import", mw.authPOST(deps.legacy.HandleImportSync)},
		{"GET /import/csv", mw.authGET(deps.legacy.HandleImportCSVPage)},
		{"POST /import/csv/preview", mw.authPOST(deps.legacy.HandleImportCSVPreview)},
		{"POST /import/csv", mw.authPOST(deps.legacy.HandleImportCSV)},
		{"POST /profile/bgg-username", mw.authPOST(deps.legacy.HandleSetBGGUsername)},
		{"GET /profile/change-password", mw.authGET(deps.legacy.HandleChangePasswordPage)},
		{"POST /profile/change-password", mw.authPOST(deps.legacy.HandleChangePassword)},
	} {
		mux.Handle(route.pattern, route.handler)
	}
}

type routeMiddleware struct {
	store         *store.Store
	sessionKey    []byte
	sessionSecret string
	loginLimiter  *httpx.LoginLimiter
}

func newRouteMiddleware(store *store.Store, sessionKey []byte, sessionSecret string, loginLimiter *httpx.LoginLimiter) routeMiddleware {
	return routeMiddleware{
		store:         store,
		sessionKey:    sessionKey,
		sessionSecret: sessionSecret,
		loginLimiter:  loginLimiter,
	}
}

func (m routeMiddleware) publicGET(hf http.HandlerFunc) http.Handler {
	return httpx.Chain(hf, httpx.MethodGuard(http.MethodGet))
}

func (m routeMiddleware) loginPOST(hf http.HandlerFunc) http.Handler {
	return httpx.Chain(hf, httpx.MethodGuard(http.MethodPost), httpx.SameOrigin(), httpx.RateLimit(m.loginLimiter))
}

func (m routeMiddleware) signupPOST(hf http.HandlerFunc) http.Handler {
	return httpx.Chain(hf, httpx.MethodGuard(http.MethodPost), httpx.SameOrigin(), httpx.RateLimit(m.loginLimiter))
}

func (m routeMiddleware) logoutPOST(hf http.HandlerFunc) http.Handler {
	return httpx.Chain(hf, httpx.MethodGuard(http.MethodPost), httpx.SameOrigin())
}

func (m routeMiddleware) authGET(hf http.HandlerFunc) http.Handler {
	return httpx.Chain(hf, httpx.MethodGuard(http.MethodGet), httpx.RequireAuth(m.store, m.sessionKey))
}

func (m routeMiddleware) authPOST(hf http.HandlerFunc) http.Handler {
	return httpx.Chain(hf, httpx.MethodGuard(http.MethodPost), httpx.RequireAuth(m.store, m.sessionKey), httpx.SameOrigin(), httpx.VerifyCSRF())
}

func (m routeMiddleware) authGetWithID(hf http.HandlerFunc) http.Handler {
	return httpx.Chain(hf, httpx.ExtractID(), httpx.RequireAuth(m.store, m.sessionKey))
}

func (m routeMiddleware) authPostWithID(hf http.HandlerFunc) http.Handler {
	return httpx.Chain(hf, httpx.ExtractID(), httpx.MethodGuard(http.MethodPost), httpx.RequireAuth(m.store, m.sessionKey), httpx.SameOrigin(), httpx.VerifyCSRF())
}

func (m routeMiddleware) apiGET(hf http.HandlerFunc) http.Handler {
	return httpx.Chain(hf, httpx.MethodGuard(http.MethodGet), httpx.RequireJWT(m.sessionSecret))
}

func (m routeMiddleware) apiPOSTPublic(hf http.HandlerFunc) http.Handler {
	return httpx.Chain(hf, httpx.MethodGuard(http.MethodPost))
}

func (m routeMiddleware) apiPOST(hf http.HandlerFunc) http.Handler {
	return httpx.Chain(hf, httpx.MethodGuard(http.MethodPost), httpx.RequireJWT(m.sessionSecret))
}

func (m routeMiddleware) apiPUT(hf http.HandlerFunc) http.Handler {
	return httpx.Chain(hf, httpx.MethodGuard(http.MethodPut), httpx.RequireJWT(m.sessionSecret))
}

func (m routeMiddleware) apiDELETE(hf http.HandlerFunc) http.Handler {
	return httpx.Chain(hf, httpx.MethodGuard(http.MethodDelete), httpx.RequireJWT(m.sessionSecret))
}
