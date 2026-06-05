package api

import (
	"errors"
	"log"
	"net/http"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	fsrs "github.com/open-spaced-repetition/go-fsrs/v3"

	"github.com/jameynakama/flockdeck/internal/auth"
	"github.com/jameynakama/flockdeck/internal/store"
)

type nextCardResponse struct {
	EbirdCode       string `json:"ebird_code"`
	CommonName      string `json:"common_name"`
	ScientificName  string `json:"scientific_name"`
	MediaURL        string `json:"media_url"`
	PhotoURL        string `json:"photo_url"`
	Lane            string `json:"lane"`
	RecordingType   string `json:"recording_type"`
	RecordingCredit string `json:"recording_credit"`
	PhotoCredit     string `json:"photo_credit"`
	DueRemaining    int64  `json:"due_remaining"`
}

func (h *Handler) getNextCard(w http.ResponseWriter, r *http.Request) {
	userID := auth.UserIDFromCtx(r.Context())

	lane := r.URL.Query().Get("lane")
	if lane != "audio" && lane != "image" {
		writeError(w, http.StatusBadRequest, "lane must be audio or image")
		return
	}

	deckID, ok := h.ownedDeckID(w, r)
	if !ok {
		return
	}

	// due_before pins the session: cards FSRS re-dues mid-session (short
	// learning steps) don't repeat within the same quiz run.
	var dueBefore pgtype.Timestamptz
	if raw := r.URL.Query().Get("due_before"); raw != "" {
		ts, err := time.Parse(time.RFC3339, raw)
		if err != nil {
			writeError(w, http.StatusBadRequest, "due_before must be RFC3339")
			return
		}
		if err := dueBefore.Scan(ts); err != nil {
			writeError(w, http.StatusBadRequest, "due_before must be RFC3339")
			return
		}
	}

	card, err := h.queries.GetNextDueCard(r.Context(), store.GetNextDueCardParams{
		UserID:    userID,
		DeckID:    deckID,
		Lane:      lane,
		DueBefore: dueBefore,
	})
	if errors.Is(err, pgx.ErrNoRows) {
		w.WriteHeader(http.StatusNoContent)
		return
	}
	if err != nil {
		log.Printf("GetNextDueCard error: %v", err)
		writeError(w, http.StatusInternalServerError, "server error")
		return
	}

	media, err := h.queries.GetRandomMediaForSpecies(r.Context(), card.SpeciesCode)
	if err != nil {
		log.Printf("GetRandomMediaForSpecies error: %v", err)
		writeError(w, http.StatusInternalServerError, "server error")
		return
	}

	var mediaURL, recordingType, recordingCredit, photoURL, photoCredit string
	if lane == "audio" {
		if media.AudioPath == "" {
			// Defensive: GetNextDueCard's media filter should prevent this.
			log.Printf("no media for species %s lane %s", card.SpeciesCode, lane)
			writeError(w, http.StatusNotFound, "no media available")
			return
		}
		mediaURL = media.AudioPath
		recordingType = media.AudioType
		recordingCredit = media.AudioCredit
		photoURL = media.ImagePath
		photoCredit = media.ImageCredit
	} else {
		if media.ImagePath == "" {
			log.Printf("no media for species %s lane %s", card.SpeciesCode, lane)
			writeError(w, http.StatusNotFound, "no media available")
			return
		}
		mediaURL = media.ImagePath
		photoURL = media.ImagePath
		photoCredit = media.ImageCredit
	}

	writeJSON(w, http.StatusOK, nextCardResponse{
		EbirdCode:       card.SpeciesCode,
		CommonName:      card.CommonName,
		ScientificName:  card.ScientificName,
		MediaURL:        mediaURL,
		PhotoURL:        photoURL,
		Lane:            lane,
		RecordingType:   recordingType,
		RecordingCredit: recordingCredit,
		PhotoCredit:     photoCredit,
		DueRemaining:    card.DueRemaining,
	})
}

func (h *Handler) getPracticeCards(w http.ResponseWriter, r *http.Request) {
	lane := r.URL.Query().Get("lane")
	if lane != "audio" && lane != "image" {
		writeError(w, http.StatusBadRequest, "lane must be audio or image")
		return
	}

	deckID, ok := h.ownedDeckID(w, r)
	if !ok {
		return
	}

	rows, err := h.queries.GetDeckPracticeCards(r.Context(), deckID)
	if err != nil {
		log.Printf("GetDeckPracticeCards error: %v", err)
		writeError(w, http.StatusInternalServerError, "server error")
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
		var recordingCredit, photoCredit string
		if lane == "audio" {
			recordingCredit = row.AudioCredit
			photoCredit = row.ImageCredit
		} else {
			photoCredit = row.ImageCredit
		}
		cards = append(cards, nextCardResponse{
			EbirdCode:       row.EbirdCode,
			CommonName:      row.CommonName,
			ScientificName:  row.ScientificName,
			MediaURL:        mediaURL,
			PhotoURL:        photoURL,
			Lane:            lane,
			RecordingCredit: recordingCredit,
			PhotoCredit:     photoCredit,
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

	if _, ok := h.ownedDeckID(w, r); !ok {
		return
	}

	req, ok := decodeJSON[rateCardRequest](w, r)
	if !ok {
		return
	}
	if req.Rating < 1 || req.Rating > 4 {
		writeError(w, http.StatusBadRequest, "rating must be 1-4")
		return
	}
	if req.Lane != "audio" && req.Lane != "image" {
		writeError(w, http.StatusBadRequest, "lane must be audio or image")
		return
	}
	if req.EbirdCode == "" {
		writeError(w, http.StatusBadRequest, "ebird_code is required")
		return
	}

	current, err := h.queries.GetCard(r.Context(), store.GetCardParams{
		UserID:      userID,
		SpeciesCode: req.EbirdCode,
		Lane:        req.Lane,
	})
	if errors.Is(err, pgx.ErrNoRows) {
		writeError(w, http.StatusNotFound, "card not found")
		return
	}
	if err != nil {
		log.Printf("GetCard error: %v", err)
		writeError(w, http.StatusInternalServerError, "server error")
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
		writeError(w, http.StatusInternalServerError, "server error")
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
		writeError(w, http.StatusInternalServerError, "server error")
		return
	}

	writeJSON(w, http.StatusOK, updated)
}
