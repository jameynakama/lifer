package api

import (
	"crypto/rand"
	"fmt"
	"log"
	"net/http"

	"github.com/go-chi/chi/v5"
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
	id := chi.URLParam(r, "macaulay_id")

	img, err := h.queries.GetImageByID(r.Context(), id)
	if err != nil {
		http.Error(w, "not found", http.StatusNotFound)
		return
	}
	if err := h.r2Client.Delete(r.Context(), h.r2Client.KeyFor(img.FilePath)); err != nil {
		log.Printf("admin: delete image R2 %s: %v", id, err)
		http.Error(w, "server error", http.StatusInternalServerError)
		return
	}
	if err := h.queries.DeleteImage(r.Context(), id); err != nil {
		log.Printf("admin: delete image DB %s: %v", id, err)
		http.Error(w, "server error", http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *Handler) adminDeleteRecording(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "xeno_canto_id")

	rec, err := h.queries.GetRecordingByID(r.Context(), id)
	if err != nil {
		http.Error(w, "not found", http.StatusNotFound)
		return
	}
	if err := h.r2Client.Delete(r.Context(), h.r2Client.KeyFor(rec.FilePath)); err != nil {
		log.Printf("admin: delete recording R2 %s: %v", id, err)
		http.Error(w, "server error", http.StatusInternalServerError)
		return
	}
	if err := h.queries.DeleteRecording(r.Context(), id); err != nil {
		log.Printf("admin: delete recording DB %s: %v", id, err)
		http.Error(w, "server error", http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
