package api

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
	"github.com/jameynakama/flockdeck/internal/auth"
)

// maxBodyBytes caps JSON request bodies (multipart uploads are separate).
const maxBodyBytes = 1 << 20 // 1 MiB

// writeError writes a JSON error envelope: {"error": msg}. All error
// responses go through this so clients parse one shape.
func writeError(w http.ResponseWriter, status int, msg string) {
	writeJSON(w, status, map[string]string{"error": msg})
}

// parseID parses the named chi URL param as an int64. On failure it writes
// a 400 ("invalid <what> id") and returns false.
func parseID(w http.ResponseWriter, r *http.Request, param, what string) (int64, bool) {
	id, err := strconv.ParseInt(chi.URLParam(r, param), 10, 64)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid "+what+" id")
		return 0, false
	}
	return id, true
}

// ownedDeckID parses the deck id from the URL and enforces the ownership
// policy (deckOwnerCheck: owner or admin; presets are clone-only). Writes
// the appropriate error response and returns false on any failure.
func (h *Handler) ownedDeckID(w http.ResponseWriter, r *http.Request) (int64, bool) {
	deckID, ok := parseID(w, r, "id", "deck")
	if !ok {
		return 0, false
	}
	if !h.deckOwnerCheck(w, r, deckID, auth.UserIDFromCtx(r.Context())) {
		return 0, false
	}
	return deckID, true
}

// decodeJSON decodes the request body into T with a size cap and unknown
// fields rejected. On failure it writes a 400 and returns false.
func decodeJSON[T any](w http.ResponseWriter, r *http.Request) (T, bool) {
	var v T
	r.Body = http.MaxBytesReader(w, r.Body, maxBodyBytes)
	dec := json.NewDecoder(r.Body)
	dec.DisallowUnknownFields()
	if err := dec.Decode(&v); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return v, false
	}
	return v, true
}

// orEmpty turns a nil slice into an empty one so writeJSON emits [] not null.
func orEmpty[T any](s []T) []T {
	if s == nil {
		return []T{}
	}
	return s
}
