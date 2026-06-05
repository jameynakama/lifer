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

type adminDeckInfo struct {
	ID         int64  `json:"id"`
	Name       string `json:"name"`
	OwnerName  string `json:"owner_name"`
	OwnerEmail string `json:"owner_email"`
}

type adminDeckDetailResponse struct {
	Deck    adminDeckInfo                    `json:"deck"`
	Species []store.ListDeckSpeciesSimpleRow `json:"species"`
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
		writeError(w, http.StatusInternalServerError, "server error")
		return
	}
	recordings, err := h.queries.GetSpeciesRecordings(r.Context(), code)
	if err != nil {
		log.Printf("admin: get recordings for %s: %v", code, err)
		writeError(w, http.StatusInternalServerError, "server error")
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
		writeError(w, http.StatusBadRequest, "bad request")
		return
	}
	file, header, err := r.FormFile("file")
	if err != nil {
		writeError(w, http.StatusBadRequest, "missing file")
		return
	}
	defer file.Close()

	credit := r.FormValue("credit")

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
		writeError(w, http.StatusInternalServerError, "server error")
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
		writeError(w, http.StatusInternalServerError, "server error")
		return
	}
	writeJSON(w, http.StatusCreated, img)
}

func (h *Handler) adminUploadRecording(w http.ResponseWriter, r *http.Request) {
	code := chi.URLParam(r, "ebird_code")
	if err := r.ParseMultipartForm(32 << 20); err != nil {
		writeError(w, http.StatusBadRequest, "bad request")
		return
	}
	file, header, err := r.FormFile("file")
	if err != nil {
		writeError(w, http.StatusBadRequest, "missing file")
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
	credit := r.FormValue("credit")

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
		writeError(w, http.StatusInternalServerError, "server error")
		return
	}

	rec, err := h.queries.UpsertRecording(r.Context(), store.UpsertRecordingParams{
		XenoCantoID: id,
		SpeciesCode: code,
		FilePath:    fileURL,
		Quality:     quality,
		Type:        recType,
		Credit:      credit,
	})
	if err != nil {
		log.Printf("admin: insert recording DB %s: %v", id, err)
		writeError(w, http.StatusInternalServerError, "server error")
		return
	}
	writeJSON(w, http.StatusCreated, rec)
}

func (h *Handler) adminDeleteImage(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "macaulay_id")

	img, err := h.queries.GetImageByID(r.Context(), id)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			writeError(w, http.StatusNotFound, "not found")
			return
		}
		log.Printf("admin: get image %s: %v", id, err)
		writeError(w, http.StatusInternalServerError, "server error")
		return
	}
	if img.Locked {
		writeError(w, http.StatusConflict, "locked")
		return
	}
	// DB first: an orphaned R2 object is harmless (and logged); a DB row
	// pointing at a deleted object would break the app.
	if err := h.queries.DeleteImage(r.Context(), id); err != nil {
		log.Printf("admin: delete image DB %s: %v", id, err)
		writeError(w, http.StatusInternalServerError, "server error")
		return
	}
	if err := h.r2Client.Delete(r.Context(), h.r2Client.KeyFor(img.FilePath)); err != nil {
		log.Printf("admin: delete image R2 %s (orphaned object): %v", id, err)
	}
	w.WriteHeader(http.StatusNoContent)
}

type setLockedRequest struct {
	Locked bool `json:"locked"`
}

type setIsAdminRequest struct {
	IsAdmin bool `json:"is_admin"`
}

func (h *Handler) adminSetImageLocked(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "macaulay_id")
	req, ok := decodeJSON[setLockedRequest](w, r)
	if !ok {
		return
	}
	if err := h.queries.SetImageLocked(r.Context(), store.SetImageLockedParams{
		MacaulayID: id,
		Locked:     req.Locked,
	}); err != nil {
		log.Printf("admin: set image locked %s: %v", id, err)
		writeError(w, http.StatusInternalServerError, "server error")
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *Handler) adminCreatePresetDeck(w http.ResponseWriter, r *http.Request) {
	req, ok := decodeJSON[deckRequest](w, r)
	if !ok {
		return
	}
	if req.Name == "" {
		writeError(w, http.StatusBadRequest, "name is required")
		return
	}
	var desc string
	if req.Description != nil {
		desc = *req.Description
	}
	deck, err := h.queries.CreatePresetDeck(r.Context(), store.CreatePresetDeckParams{
		Name:        req.Name,
		Description: desc,
	})
	if err != nil {
		log.Printf("adminCreatePresetDeck error: %v", err)
		writeError(w, http.StatusInternalServerError, "server error")
		return
	}
	writeJSON(w, http.StatusCreated, deck)
}

func (h *Handler) adminUpdatePresetDeck(w http.ResponseWriter, r *http.Request) {
	deckID, ok := parseID(w, r, "id", "deck")
	if !ok {
		return
	}

	deck, err := h.queries.GetDeck(r.Context(), deckID)
	if errors.Is(err, pgx.ErrNoRows) {
		writeError(w, http.StatusNotFound, "not found")
		return
	}
	if err != nil {
		log.Printf("adminUpdatePresetDeck GetDeck error: %v", err)
		writeError(w, http.StatusInternalServerError, "server error")
		return
	}
	if deck.OwnerID.Valid {
		writeError(w, http.StatusBadRequest, "not a preset deck")
		return
	}

	req, ok := decodeJSON[deckRequest](w, r)
	if !ok {
		return
	}
	if req.Name == "" {
		writeError(w, http.StatusBadRequest, "name is required")
		return
	}
	var desc string
	if req.Description != nil {
		desc = *req.Description
	}
	updated, err := h.queries.UpdateDeck(r.Context(), store.UpdateDeckParams{
		ID:          deckID,
		Name:        req.Name,
		Description: desc,
	})
	if err != nil {
		log.Printf("adminUpdatePresetDeck UpdateDeck error: %v", err)
		writeError(w, http.StatusInternalServerError, "server error")
		return
	}
	writeJSON(w, http.StatusOK, updated)
}

func (h *Handler) adminDeletePresetDeck(w http.ResponseWriter, r *http.Request) {
	deckID, ok := parseID(w, r, "id", "deck")
	if !ok {
		return
	}

	deck, err := h.queries.GetDeck(r.Context(), deckID)
	if errors.Is(err, pgx.ErrNoRows) {
		writeError(w, http.StatusNotFound, "not found")
		return
	}
	if err != nil {
		log.Printf("adminDeletePresetDeck GetDeck error: %v", err)
		writeError(w, http.StatusInternalServerError, "server error")
		return
	}
	if deck.OwnerID.Valid {
		writeError(w, http.StatusBadRequest, "not a preset deck")
		return
	}

	if err := h.queries.DeleteDeck(r.Context(), deckID); err != nil {
		log.Printf("adminDeletePresetDeck DeleteDeck error: %v", err)
		writeError(w, http.StatusInternalServerError, "server error")
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *Handler) adminDeleteRecording(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "xeno_canto_id")

	rec, err := h.queries.GetRecordingByID(r.Context(), id)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			writeError(w, http.StatusNotFound, "not found")
			return
		}
		log.Printf("admin: get recording %s: %v", id, err)
		writeError(w, http.StatusInternalServerError, "server error")
		return
	}
	if rec.Locked {
		writeError(w, http.StatusConflict, "locked")
		return
	}
	// DB first: an orphaned R2 object is harmless (and logged); a DB row
	// pointing at a deleted object would break the app.
	if err := h.queries.DeleteRecording(r.Context(), id); err != nil {
		log.Printf("admin: delete recording DB %s: %v", id, err)
		writeError(w, http.StatusInternalServerError, "server error")
		return
	}
	if err := h.r2Client.Delete(r.Context(), h.r2Client.KeyFor(rec.FilePath)); err != nil {
		log.Printf("admin: delete recording R2 %s (orphaned object): %v", id, err)
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *Handler) adminSetRecordingLocked(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "xeno_canto_id")
	req, ok := decodeJSON[setLockedRequest](w, r)
	if !ok {
		return
	}
	if err := h.queries.SetRecordingLocked(r.Context(), store.SetRecordingLockedParams{
		XenoCantoID: id,
		Locked:      req.Locked,
	}); err != nil {
		log.Printf("admin: set recording locked %s: %v", id, err)
		writeError(w, http.StatusInternalServerError, "server error")
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *Handler) adminListUserDecks(w http.ResponseWriter, r *http.Request) {
	decks, err := h.queries.ListAllUserDecks(r.Context())
	if err != nil {
		log.Printf("admin: list user decks: %v", err)
		writeError(w, http.StatusInternalServerError, "server error")
		return
	}
	writeJSON(w, http.StatusOK, orEmpty(decks))
}

func (h *Handler) adminGetDeckSpecies(w http.ResponseWriter, r *http.Request) {
	deckID, ok := parseID(w, r, "id", "deck")
	if !ok {
		return
	}
	deck, err := h.queries.GetDeckWithOwner(r.Context(), deckID)
	if errors.Is(err, pgx.ErrNoRows) {
		writeError(w, http.StatusNotFound, "not found")
		return
	}
	if err != nil {
		log.Printf("admin: get deck with owner %d: %v", deckID, err)
		writeError(w, http.StatusInternalServerError, "server error")
		return
	}
	species, err := h.queries.ListDeckSpeciesSimple(r.Context(), deckID)
	if err != nil {
		log.Printf("admin: list deck species %d: %v", deckID, err)
		writeError(w, http.StatusInternalServerError, "server error")
		return
	}
	writeJSON(w, http.StatusOK, adminDeckDetailResponse{
		Deck: adminDeckInfo{
			ID:         deck.ID,
			Name:       deck.Name,
			OwnerName:  deck.OwnerName,
			OwnerEmail: deck.OwnerEmail,
		},
		Species: orEmpty(species),
	})
}

func (h *Handler) adminGetUsers(w http.ResponseWriter, r *http.Request) {
	users, err := h.queries.GetUsers(r.Context())
	if err != nil {
		log.Printf("admin: get users: %v", err)
		writeError(w, http.StatusInternalServerError, "server error")
		return
	}

	writeJSON(w, http.StatusOK, users)
}

func (h *Handler) adminSetUserIsAdmin(w http.ResponseWriter, r *http.Request) {
	id, ok := parseID(w, r, "id", "user")
	if !ok {
		return
	}
	req, ok := decodeJSON[setIsAdminRequest](w, r)
	if !ok {
		return
	}
	if err := h.queries.SetUserIsAdmin(r.Context(), store.SetUserIsAdminParams{
		ID:      id,
		IsAdmin: req.IsAdmin,
	}); err != nil {
		log.Printf("admin: set user is_admin %d: %v", id, err)
		writeError(w, http.StatusInternalServerError, "server error")
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
