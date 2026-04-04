package handler

import (
	"net/http"
	"strconv"

	"myboardgamecollection/internal/bgg"
	"myboardgamecollection/internal/render"
	"myboardgamecollection/internal/store"
)

// Handler holds dependencies for all HTTP handlers.
type Handler struct {
	Store    *store.Store
	Renderer *render.Renderer
	BGG      *bgg.Client // may be nil if BGG is not configured
}

func parseID(r *http.Request) (int64, error) {
	return strconv.ParseInt(r.PathValue("id"), 10, 64)
}

func isHTMX(r *http.Request) bool {
	return r.Header.Get("HX-Request") == "true"
}
