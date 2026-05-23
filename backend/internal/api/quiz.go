package api

import (
	"errors"
	"log"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
	"github.com/jameynakama/lifer/internal/auth"
	"github.com/jameynakama/lifer/internal/store"
	"github.com/jackc/pgx/v5"
)

type nextCardResponse struct {
	SpeciesID      int64  `json:"species_id"`
	CommonName     string `json:"common_name"`
	ScientificName string `json:"scientific_name"`
	MediaURL       string `json:"media_url"`
	PhotoURL       string `json:"photo_url"`
	Lane           string `json:"lane"`
}

func (h *Handler) getNextCard(w http.ResponseWriter, r *http.Request) {
	userID := auth.UserIDFromCtx(r.Context())

	groupID, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		http.Error(w, "invalid group id", http.StatusBadRequest)
		return
	}

	lane := r.URL.Query().Get("lane")
	if lane != "audio" && lane != "image" {
		http.Error(w, "lane must be audio or image", http.StatusBadRequest)
		return
	}

	card, err := h.queries.GetNextDueCard(r.Context(), store.GetNextDueCardParams{
		UserID:  userID,
		GroupID: groupID,
		Lane:    lane,
	})
	if errors.Is(err, pgx.ErrNoRows) {
		w.WriteHeader(http.StatusNoContent)
		return
	}
	if err != nil {
		log.Printf("GetNextDueCard error: %v", err)
		http.Error(w, "server error", http.StatusInternalServerError)
		return
	}

	var mediaURL string
	if lane == "audio" {
		mediaURL, err = h.queries.GetRandomRecording(r.Context(), card.SpeciesID)
	} else {
		mediaURL, err = h.queries.GetRandomImage(r.Context(), card.SpeciesID)
	}
	if errors.Is(err, pgx.ErrNoRows) {
		log.Printf("no media for species %d lane %s", card.SpeciesID, lane)
		http.Error(w, "no media available", http.StatusInternalServerError)
		return
	}
	if err != nil {
		log.Printf("GetRandom media error: %v", err)
		http.Error(w, "server error", http.StatusInternalServerError)
		return
	}

	// For audio lane, fetch a separate photo for the reveal.
	// For image lane, the reveal shows the same photo the user just saw.
	photoURL := mediaURL
	if lane == "audio" {
		photoURL, err = h.queries.GetRandomImage(r.Context(), card.SpeciesID)
		if errors.Is(err, pgx.ErrNoRows) {
			photoURL = ""
		} else if err != nil {
			log.Printf("GetRandomImage error: %v", err)
			http.Error(w, "server error", http.StatusInternalServerError)
			return
		}
	}

	writeJSON(w, http.StatusOK, nextCardResponse{
		SpeciesID:      card.SpeciesID,
		CommonName:     card.CommonName,
		ScientificName: card.ScientificName,
		MediaURL:       mediaURL,
		PhotoURL:       photoURL,
		Lane:           lane,
	})
}
