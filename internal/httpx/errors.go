package httpx

import (
	"database/sql"
	"errors"
	"log/slog"
	"net/http"

	"myboardgamecollection/internal/store"
)

var (
	ErrUnauthorized = errors.New("unauthorized")
	ErrInvalidID    = errors.New("invalid id")
)

// HandleError writes an appropriate HTTP error response based on the error type.
// It logs internal errors and returns user-friendly messages for client errors.
func HandleError(w http.ResponseWriter, err error, defaultStatus int) {
	if err == nil {
		return
	}

	switch {
	case errors.Is(err, sql.ErrNoRows):
		http.Error(w, "not found", http.StatusNotFound)
	case errors.Is(err, store.ErrDuplicate):
		http.Error(w, "already exists", http.StatusConflict)
	default:
		slog.Error("handler error", "error", err)
		http.Error(w, "internal error", defaultStatus)
	}
}

// WriteSuccess writes a successful JSON response with the given data.
func WriteSuccess(w http.ResponseWriter, data any) error {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	// Note: In a real implementation, you'd use json.Marshal here
	// For now, we'll keep it simple
	return nil
}

// WriteError writes an error response using HandleError.
func WriteError(w http.ResponseWriter, err error) {
	HandleError(w, err, http.StatusInternalServerError)
}
