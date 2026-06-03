package api

import (
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"strconv"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	fsrs "github.com/open-spaced-repetition/go-fsrs/v3"

	"github.com/jameynakama/flockdeck/internal/auth"
	"github.com/jameynakama/flockdeck/internal/store"
)

type nextCardResponse struct {
	EbirdCode      string `json:"ebird_code"`
	CommonName     string `json:"common_name"`
	ScientificName string `json:"scientific_name"`
	MediaURL       string `json:"media_url"`
	PhotoURL       string `json:"photo_url"`
	Lane           string `json:"lane"`
	RecordingType  string `json:"recording_type"`
	DueRemaining   int64  `json:"due_remaining"`
}

func (h *Handler) getNextCard(w http.ResponseWriter, r *http.Request) {
	userID := auth.UserIDFromCtx(r.Context())

	deckID, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		http.Error(w, "invalid deck id", http.StatusBadRequest)
		return
	}

	lane := r.URL.Query().Get("lane")
	if lane != "audio" && lane != "image" {
		http.Error(w, "lane must be audio or image", http.StatusBadRequest)
		return
	}

	card, err := h.queries.GetNextDueCard(r.Context(), store.GetNextDueCardParams{
		UserID: userID,
		DeckID: deckID,
		Lane:   lane,
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

	dueRemaining, err := h.queries.CountDueCards(r.Context(), store.CountDueCardsParams{
		UserID: userID,
		DeckID: deckID,
		Lane:   lane,
	})
	if err != nil {
		log.Printf("CountDueCards error: %v", err)
		http.Error(w, "server error", http.StatusInternalServerError)
		return
	}

	var mediaURL, recordingType string
	if lane == "audio" {
		rec, recErr := h.queries.GetRandomRecording(r.Context(), card.SpeciesCode)
		if errors.Is(recErr, pgx.ErrNoRows) {
			log.Printf("no media for species %s lane %s", card.SpeciesCode, lane)
			http.Error(w, "no media available", http.StatusInternalServerError)
			return
		}
		if recErr != nil {
			log.Printf("GetRandom media error: %v", recErr)
			http.Error(w, "server error", http.StatusInternalServerError)
			return
		}
		mediaURL = rec.FilePath
		recordingType = rec.Type
	} else {
		mediaURL, err = h.queries.GetRandomImage(r.Context(), card.SpeciesCode)
		if errors.Is(err, pgx.ErrNoRows) {
			log.Printf("no media for species %s lane %s", card.SpeciesCode, lane)
			http.Error(w, "no media available", http.StatusInternalServerError)
			return
		}
		if err != nil {
			log.Printf("GetRandom media error: %v", err)
			http.Error(w, "server error", http.StatusInternalServerError)
			return
		}
	}

	photoURL := mediaURL
	if lane == "audio" {
		photoURL, err = h.queries.GetRandomImage(r.Context(), card.SpeciesCode)
		if errors.Is(err, pgx.ErrNoRows) {
			photoURL = ""
		} else if err != nil {
			log.Printf("GetRandomImage error: %v", err)
			http.Error(w, "server error", http.StatusInternalServerError)
			return
		}
	}

	writeJSON(w, http.StatusOK, nextCardResponse{
		EbirdCode:      card.SpeciesCode,
		CommonName:     card.CommonName,
		ScientificName: card.ScientificName,
		MediaURL:       mediaURL,
		PhotoURL:       photoURL,
		Lane:           lane,
		RecordingType:  recordingType,
		DueRemaining:   dueRemaining,
	})
}

func (h *Handler) getPracticeCards(w http.ResponseWriter, r *http.Request) {
	userID := auth.UserIDFromCtx(r.Context())

	deckID, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		http.Error(w, "invalid deck id", http.StatusBadRequest)
		return
	}

	lane := r.URL.Query().Get("lane")
	if lane != "audio" && lane != "image" {
		http.Error(w, "lane must be audio or image", http.StatusBadRequest)
		return
	}

	deck, err := h.queries.GetDeck(r.Context(), deckID)
	if errors.Is(err, pgx.ErrNoRows) {
		http.Error(w, "not found", http.StatusNotFound)
		return
	}
	if err != nil {
		log.Printf("GetDeck error: %v", err)
		http.Error(w, "server error", http.StatusInternalServerError)
		return
	}
	if deck.OwnerID.Valid && deck.OwnerID.Int64 != userID {
		http.Error(w, "forbidden", http.StatusForbidden)
		return
	}

	rows, err := h.queries.GetDeckPracticeCards(r.Context(), deckID)
	if err != nil {
		log.Printf("GetDeckPracticeCards error: %v", err)
		http.Error(w, "server error", http.StatusInternalServerError)
		return
	}

	cards := make([]nextCardResponse, 0, len(rows))
	for _, row := range rows {
		var mediaURL, photoURL string
		if lane == "audio" {
			if row.AudioUrl == "" {
				continue
			}
			mediaURL = row.AudioUrl
			photoURL = row.ImageUrl
		} else {
			if row.ImageUrl == "" {
				continue
			}
			mediaURL = row.ImageUrl
			photoURL = row.ImageUrl
		}
		cards = append(cards, nextCardResponse{
			EbirdCode:      row.EbirdCode,
			CommonName:     row.CommonName,
			ScientificName: row.ScientificName,
			MediaURL:       mediaURL,
			PhotoURL:       photoURL,
			Lane:           lane,
		})
	}

	writeJSON(w, http.StatusOK, cards)
}

type rateCardRequest struct {
	EbirdCode string `json:"ebird_code"`
	Lane      string `json:"lane"`
	Rating    int    `json:"rating"`
}

func (h *Handler) rateCard(w http.ResponseWriter, r *http.Request) {
	userID := auth.UserIDFromCtx(r.Context())

	var req rateCardRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}
	if req.Rating < 1 || req.Rating > 4 {
		http.Error(w, "rating must be 1-4", http.StatusBadRequest)
		return
	}
	if req.Lane != "audio" && req.Lane != "image" {
		http.Error(w, "lane must be audio or image", http.StatusBadRequest)
		return
	}
	if req.EbirdCode == "" {
		http.Error(w, "ebird_code is required", http.StatusBadRequest)
		return
	}

	current, err := h.queries.GetCard(r.Context(), store.GetCardParams{
		UserID:      userID,
		SpeciesCode: req.EbirdCode,
		Lane:        req.Lane,
	})
	if errors.Is(err, pgx.ErrNoRows) {
		http.Error(w, "card not found", http.StatusNotFound)
		return
	}
	if err != nil {
		log.Printf("GetCard error: %v", err)
		http.Error(w, "server error", http.StatusInternalServerError)
		return
	}

	fsrsCard := fsrs.Card{
		Stability:  current.Stability,
		Difficulty: current.Difficulty,
		Reps:       uint64(current.Reps),
		Lapses:     uint64(current.Lapses),
		State:      fsrs.State(current.State),
	}
	if current.LastReview.Valid {
		fsrsCard.LastReview = current.LastReview.Time
	}
	if current.Due.Valid {
		fsrsCard.Due = current.Due.Time
	}

	f := fsrs.NewFSRS(fsrs.DefaultParam())
	result := f.Next(fsrsCard, time.Now(), fsrs.Rating(req.Rating)).Card

	due := pgtype.Timestamptz{}
	if err := due.Scan(result.Due); err != nil {
		log.Printf("scan due error: %v", err)
		http.Error(w, "server error", http.StatusInternalServerError)
		return
	}

	updated, err := h.queries.UpdateCardSchedule(r.Context(), store.UpdateCardScheduleParams{
		UserID:      userID,
		SpeciesCode: req.EbirdCode,
		Lane:        req.Lane,
		Stability:   result.Stability,
		Difficulty:  result.Difficulty,
		Due:         due,
		Lapses:      int32(result.Lapses),
		State:       int16(result.State),
	})
	if err != nil {
		log.Printf("UpdateCardSchedule error: %v", err)
		http.Error(w, "server error", http.StatusInternalServerError)
		return
	}

	writeJSON(w, http.StatusOK, updated)
}
