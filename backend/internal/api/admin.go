package api

import (
	"crypto/rand"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"path/filepath"
	"strconv"
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
		http.Error(w, "server error", http.StatusInternalServerError)
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
	if img.Locked {
		http.Error(w, "locked", http.StatusConflict)
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

func (h *Handler) adminSetImageLocked(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "macaulay_id")
	var req struct {
		Locked bool `json:"locked"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}
	if err := h.queries.SetImageLocked(r.Context(), store.SetImageLockedParams{
		MacaulayID: id,
		Locked:     req.Locked,
	}); err != nil {
		log.Printf("admin: set image locked %s: %v", id, err)
		http.Error(w, "server error", http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *Handler) adminCreatePresetDeck(w http.ResponseWriter, r *http.Request) {
	var req createDeckRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}
	if req.Name == "" {
		http.Error(w, "name is required", http.StatusBadRequest)
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
		http.Error(w, "server error", http.StatusInternalServerError)
		return
	}
	writeJSON(w, http.StatusCreated, deck)
}

func (h *Handler) adminUpdatePresetDeck(w http.ResponseWriter, r *http.Request) {
	deckID, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		http.Error(w, "invalid deck id", http.StatusBadRequest)
		return
	}

	deck, err := h.queries.GetDeck(r.Context(), deckID)
	if errors.Is(err, pgx.ErrNoRows) {
		http.Error(w, "not found", http.StatusNotFound)
		return
	}
	if err != nil {
		log.Printf("adminUpdatePresetDeck GetDeck error: %v", err)
		http.Error(w, "server error", http.StatusInternalServerError)
		return
	}
	if deck.OwnerID.Valid {
		http.Error(w, "not a preset deck", http.StatusBadRequest)
		return
	}

	var req updateDeckRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}
	if req.Name == "" {
		http.Error(w, "name is required", http.StatusBadRequest)
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
		http.Error(w, "server error", http.StatusInternalServerError)
		return
	}
	writeJSON(w, http.StatusOK, updated)
}

func (h *Handler) adminDeletePresetDeck(w http.ResponseWriter, r *http.Request) {
	deckID, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		http.Error(w, "invalid deck id", http.StatusBadRequest)
		return
	}

	deck, err := h.queries.GetDeck(r.Context(), deckID)
	if errors.Is(err, pgx.ErrNoRows) {
		http.Error(w, "not found", http.StatusNotFound)
		return
	}
	if err != nil {
		log.Printf("adminDeletePresetDeck GetDeck error: %v", err)
		http.Error(w, "server error", http.StatusInternalServerError)
		return
	}
	if deck.OwnerID.Valid {
		http.Error(w, "not a preset deck", http.StatusBadRequest)
		return
	}

	if err := h.queries.DeleteDeck(r.Context(), deckID); err != nil {
		log.Printf("adminDeletePresetDeck DeleteDeck error: %v", err)
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
	if rec.Locked {
		http.Error(w, "locked", http.StatusConflict)
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

func (h *Handler) adminSetRecordingLocked(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "xeno_canto_id")
	var req struct {
		Locked bool `json:"locked"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}
	if err := h.queries.SetRecordingLocked(r.Context(), store.SetRecordingLockedParams{
		XenoCantoID: id,
		Locked:      req.Locked,
	}); err != nil {
		log.Printf("admin: set recording locked %s: %v", id, err)
		http.Error(w, "server error", http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *Handler) adminListUserDecks(w http.ResponseWriter, r *http.Request) {
	decks, err := h.queries.ListAllUserDecks(r.Context())
	if err != nil {
		log.Printf("admin: list user decks: %v", err)
		http.Error(w, "server error", http.StatusInternalServerError)
		return
	}
	if decks == nil {
		decks = []store.ListAllUserDecksRow{}
	}
	writeJSON(w, http.StatusOK, decks)
}

func (h *Handler) adminGetDeckSpecies(w http.ResponseWriter, r *http.Request) {
	deckID, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		http.Error(w, "invalid deck id", http.StatusBadRequest)
		return
	}
	deck, err := h.queries.GetDeckWithOwner(r.Context(), deckID)
	if errors.Is(err, pgx.ErrNoRows) {
		http.Error(w, "not found", http.StatusNotFound)
		return
	}
	if err != nil {
		log.Printf("admin: get deck with owner %d: %v", deckID, err)
		http.Error(w, "server error", http.StatusInternalServerError)
		return
	}
	species, err := h.queries.ListDeckSpeciesSimple(r.Context(), deckID)
	if err != nil {
		log.Printf("admin: list deck species %d: %v", deckID, err)
		http.Error(w, "server error", http.StatusInternalServerError)
		return
	}
	if species == nil {
		species = []store.ListDeckSpeciesSimpleRow{}
	}
	writeJSON(w, http.StatusOK, adminDeckDetailResponse{
		Deck: adminDeckInfo{
			ID:         deck.ID,
			Name:       deck.Name,
			OwnerName:  deck.OwnerName,
			OwnerEmail: deck.OwnerEmail,
		},
		Species: species,
	})
}

func (h *Handler) adminGetUsers(w http.ResponseWriter, r *http.Request) {
	users, err := h.queries.GetUsers(r.Context())
	if err != nil {
		log.Printf("admin: get users: %v", err)
		http.Error(w, "server error", http.StatusInternalServerError)
		return
	}

	writeJSON(w, http.StatusOK, users)
}

func (h *Handler) adminSetUserIsAdmin(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		http.Error(w, "invalid id", http.StatusBadRequest)
		return
	}
	var req struct {
		IsAdmin bool `json:"is_admin"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}
	if err := h.queries.SetUserIsAdmin(r.Context(), store.SetUserIsAdminParams{
		ID:      id,
		IsAdmin: req.IsAdmin,
	}); err != nil {
		log.Printf("admin: set user is_admin %d: %v", id, err)
		http.Error(w, "server error", http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
