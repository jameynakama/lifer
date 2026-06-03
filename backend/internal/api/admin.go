package api

import (
	"crypto/rand"
	"errors"
	"fmt"
	"log"
	"net/http"
	"path/filepath"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5"
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
	code := chi.URLParam(r, "ebird_code")
	if err := r.ParseMultipartForm(32 << 20); err != nil {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}
	file, header, err := r.FormFile("file")
	if err != nil {
		http.Error(w, "missing file", http.StatusBadRequest)
		return
	}
	defer file.Close()

	credit := r.FormValue("credit")
	if credit == "" {
		credit = "admin upload"
	}

	ext := strings.ToLower(filepath.Ext(header.Filename))
	contentType := extContentType[ext]
	if contentType == "" {
		contentType = "application/octet-stream"
	}

	id := adminID()
	key := fmt.Sprintf("images/%s/%s%s", code, id, ext)
	fileURL, err := h.r2Client.Upload(r.Context(), key, contentType, file)
	if err != nil {
		log.Printf("admin: upload image R2 %s: %v", key, err)
		http.Error(w, "server error", http.StatusInternalServerError)
		return
	}

	img, err := h.queries.UpsertSpeciesImage(r.Context(), store.UpsertSpeciesImageParams{
		MacaulayID:  id,
		SpeciesCode: code,
		FilePath:    fileURL,
		Credit:      credit,
	})
	if err != nil {
		log.Printf("admin: insert image DB %s: %v", id, err)
		http.Error(w, "server error", http.StatusInternalServerError)
		return
	}
	writeJSON(w, http.StatusCreated, img)
}

func (h *Handler) adminUploadRecording(w http.ResponseWriter, r *http.Request) {
	code := chi.URLParam(r, "ebird_code")
	if err := r.ParseMultipartForm(32 << 20); err != nil {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}
	file, header, err := r.FormFile("file")
	if err != nil {
		http.Error(w, "missing file", http.StatusBadRequest)
		return
	}
	defer file.Close()

	quality := r.FormValue("quality")
	if quality == "" {
		quality = "A"
	}
	recType := r.FormValue("type")
	if recType == "" {
		recType = "song"
	}

	ext := strings.ToLower(filepath.Ext(header.Filename))
	contentType := extContentType[ext]
	if contentType == "" {
		contentType = "application/octet-stream"
	}

	id := adminID()
	key := fmt.Sprintf("recordings/%s/%s%s", code, id, ext)
	fileURL, err := h.r2Client.Upload(r.Context(), key, contentType, file)
	if err != nil {
		log.Printf("admin: upload recording R2 %s: %v", key, err)
		http.Error(w, "server error", http.StatusInternalServerError)
		return
	}

	rec, err := h.queries.UpsertRecording(r.Context(), store.UpsertRecordingParams{
		XenoCantoID: id,
		SpeciesCode: code,
		FilePath:    fileURL,
		Quality:     quality,
		Type:        recType,
	})
	if err != nil {
		log.Printf("admin: insert recording DB %s: %v", id, err)
		http.Error(w, "server error", http.StatusInternalServerError)
		return
	}
	writeJSON(w, http.StatusCreated, rec)
}

func (h *Handler) adminDeleteImage(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "macaulay_id")

	img, err := h.queries.GetImageByID(r.Context(), id)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			http.Error(w, "not found", http.StatusNotFound)
			return
		}
		log.Printf("admin: get image %s: %v", id, err)
		http.Error(w, "server error", http.StatusInternalServerError)
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
		if errors.Is(err, pgx.ErrNoRows) {
			http.Error(w, "not found", http.StatusNotFound)
			return
		}
		log.Printf("admin: get recording %s: %v", id, err)
		http.Error(w, "server error", http.StatusInternalServerError)
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
