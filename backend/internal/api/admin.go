package api

import (
	"crypto/rand"
	"fmt"
	"log"
	"net/http"
	"path/filepath"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/jameynakama/flockdeck/internal/store"
)

type adminSpeciesDetailResponse struct {
	Images     any `json:"images"`
	Recordings any `json:"recordings"`
}

var extContentType = map[string]string{
	".jpg":  "image/jpeg",
	".jpeg": "image/jpeg",
	".png":  "image/png",
	".webp": "image/webp",
	".mp3":  "audio/mpeg",
	".ogg":  "audio/ogg",
	".wav":  "audio/wav",
	".flac": "audio/flac",
}

func (h *Handler) adminGetSpeciesDetail(w http.ResponseWriter, r *http.Request) {
	code := chi.URLParam(r, "ebird_code")

	images, err := h.queries.GetSpeciesImages(r.Context(), code)
	if err != nil {
		log.Printf("admin: get images for %s: %v", code, err)
		http.Error(w, "server error", http.StatusInternalServerError)
		return
	}
	recordings, err := h.queries.GetSpeciesRecordings(r.Context(), code)
	if err != nil {
		log.Printf("admin: get recordings for %s: %v", code, err)
		http.Error(w, "server error", http.StatusInternalServerError)
		return
	}

	writeJSON(w, http.StatusOK, adminSpeciesDetailResponse{
		Images:     images,
		Recordings: recordings,
	})
}

func adminID() string {
	b := make([]byte, 8)
	rand.Read(b) //nolint:errcheck
	return fmt.Sprintf("admin-%x", b)
}

func (h *Handler) adminUploadImage(w http.ResponseWriter, r *http.Request) {
	http.Error(w, "not implemented", http.StatusNotImplemented)
}

func (h *Handler) adminUploadRecording(w http.ResponseWriter, r *http.Request) {
	http.Error(w, "not implemented", http.StatusNotImplemented)
}

func (h *Handler) adminDeleteImage(w http.ResponseWriter, r *http.Request) {
	http.Error(w, "not implemented", http.StatusNotImplemented)
}

func (h *Handler) adminDeleteRecording(w http.ResponseWriter, r *http.Request) {
	http.Error(w, "not implemented", http.StatusNotImplemented)
}

// Ensure imported packages are used in later tasks.
var _ = filepath.Ext
var _ = strings.HasPrefix
var _ = store.UpsertSpeciesImageParams{}
