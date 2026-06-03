package api

import (
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jameynakama/flockdeck/internal/auth"
	"github.com/jameynakama/flockdeck/internal/store"
)

type createDeckRequest struct {
	Name string `json:"name"`
}

type updateDeckRequest struct {
	Name string `json:"name"`
}

type addSpeciesRequest struct {
	EbirdCode string `json:"ebird_code"`
}

// deckOwnerCheck fetches the deck, writes 404/403 and returns false if the
// requesting user does not own it.
func (h *Handler) deckOwnerCheck(w http.ResponseWriter, r *http.Request, deckID, userID int64) bool {
	deck, err := h.queries.GetDeck(r.Context(), deckID)
	if errors.Is(err, pgx.ErrNoRows) {
		http.Error(w, "not found", http.StatusNotFound)
		return false
	}
	if err != nil {
		log.Printf("GetDeck error: %v", err)
		http.Error(w, "server error", http.StatusInternalServerError)
		return false
	}
	if !deck.OwnerID.Valid || deck.OwnerID.Int64 != userID {
		http.Error(w, "forbidden", http.StatusForbidden)
		return false
	}
	return true
}

func (h *Handler) getDeckDetail(w http.ResponseWriter, r *http.Request) {
	userID := auth.UserIDFromCtx(r.Context())

	deckID, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		http.Error(w, "invalid deck id", http.StatusBadRequest)
		return
	}

	deck, err := h.queries.GetDeckWithDue(r.Context(), store.GetDeckWithDueParams{
		ID:     deckID,
		UserID: userID,
	})
	if errors.Is(err, pgx.ErrNoRows) {
		http.Error(w, "not found", http.StatusNotFound)
		return
	}
	if err != nil {
		log.Printf("GetDeckWithDue error: %v", err)
		http.Error(w, "server error", http.StatusInternalServerError)
		return
	}

	writeJSON(w, http.StatusOK, deck)
}

func (h *Handler) listDecks(w http.ResponseWriter, r *http.Request) {
	userID := auth.UserIDFromCtx(r.Context())
	decks, err := h.queries.ListUserDecks(r.Context(), userID)
	if err != nil {
		log.Printf("ListUserDecks error: %v", err)
		http.Error(w, "server error", http.StatusInternalServerError)
		return
	}
	if decks == nil {
		decks = []store.ListUserDecksRow{}
	}
	writeJSON(w, http.StatusOK, decks)
}

func (h *Handler) createDeck(w http.ResponseWriter, r *http.Request) {
	userID := auth.UserIDFromCtx(r.Context())

	var req createDeckRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}
	if req.Name == "" {
		http.Error(w, "name is required", http.StatusBadRequest)
		return
	}

	deck, err := h.queries.CreateDeck(r.Context(), store.CreateDeckParams{
		Name:    req.Name,
		OwnerID: pgtype.Int8{Int64: userID, Valid: true},
	})
	if err != nil {
		log.Printf("CreateDeck error: %v", err)
		http.Error(w, "server error", http.StatusInternalServerError)
		return
	}

	writeJSON(w, http.StatusCreated, deck)
}

func (h *Handler) updateDeck(w http.ResponseWriter, r *http.Request) {
	userID := auth.UserIDFromCtx(r.Context())

	deckID, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		http.Error(w, "invalid deck id", http.StatusBadRequest)
		return
	}

	if !h.deckOwnerCheck(w, r, deckID, userID) {
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

	deck, err := h.queries.UpdateDeckName(r.Context(), store.UpdateDeckNameParams{
		ID:   deckID,
		Name: req.Name,
	})
	if err != nil {
		log.Printf("UpdateDeckName error: %v", err)
		http.Error(w, "server error", http.StatusInternalServerError)
		return
	}

	writeJSON(w, http.StatusOK, deck)
}

func (h *Handler) deleteDeck(w http.ResponseWriter, r *http.Request) {
	userID := auth.UserIDFromCtx(r.Context())

	deckID, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		http.Error(w, "invalid deck id", http.StatusBadRequest)
		return
	}

	if !h.deckOwnerCheck(w, r, deckID, userID) {
		return
	}

	if err := h.queries.DeleteDeck(r.Context(), deckID); err != nil {
		log.Printf("DeleteDeck error: %v", err)
		http.Error(w, "server error", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func (h *Handler) listDeckSpecies(w http.ResponseWriter, r *http.Request) {
	userID := auth.UserIDFromCtx(r.Context())

	deckID, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		http.Error(w, "invalid deck id", http.StatusBadRequest)
		return
	}

	if !h.deckOwnerCheck(w, r, deckID, userID) {
		return
	}

	species, err := h.queries.ListDeckSpeciesWithPrefs(r.Context(), store.ListDeckSpeciesWithPrefsParams{
		DeckID: deckID,
		UserID: userID,
	})
	if err != nil {
		log.Printf("ListDeckSpeciesWithPrefs error: %v", err)
		http.Error(w, "server error", http.StatusInternalServerError)
		return
	}
	if species == nil {
		species = []store.ListDeckSpeciesWithPrefsRow{}
	}
	writeJSON(w, http.StatusOK, species)
}

func (h *Handler) addSpeciesToDeck(w http.ResponseWriter, r *http.Request) {
	userID := auth.UserIDFromCtx(r.Context())

	deckID, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		http.Error(w, "invalid deck id", http.StatusBadRequest)
		return
	}

	if !h.deckOwnerCheck(w, r, deckID, userID) {
		return
	}

	var req addSpeciesRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}
	if req.EbirdCode == "" {
		http.Error(w, "ebird_code is required", http.StatusBadRequest)
		return
	}

	if err := h.queries.AddSpeciesToDeck(r.Context(), store.AddSpeciesToDeckParams{
		DeckID:      deckID,
		SpeciesCode: req.EbirdCode,
	}); err != nil {
		log.Printf("AddSpeciesToDeck error: %v", err)
		http.Error(w, "server error", http.StatusInternalServerError)
		return
	}

	for _, lane := range []string{"audio", "image"} {
		if err := h.queries.UpsertCard(r.Context(), store.UpsertCardParams{
			UserID:      userID,
			SpeciesCode: req.EbirdCode,
			Lane:        lane,
		}); err != nil {
			log.Printf("UpsertCard error: %v", err)
			http.Error(w, "server error", http.StatusInternalServerError)
			return
		}
	}

	w.WriteHeader(http.StatusCreated)
}

func (h *Handler) removeSpeciesFromDeck(w http.ResponseWriter, r *http.Request) {
	userID := auth.UserIDFromCtx(r.Context())

	deckID, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		http.Error(w, "invalid deck id", http.StatusBadRequest)
		return
	}

	ebirdCode := chi.URLParam(r, "ebird_code")

	if !h.deckOwnerCheck(w, r, deckID, userID) {
		return
	}

	if err := h.queries.RemoveSpeciesFromDeck(r.Context(), store.RemoveSpeciesFromDeckParams{
		DeckID:      deckID,
		SpeciesCode: ebirdCode,
	}); err != nil {
		log.Printf("RemoveSpeciesFromDeck error: %v", err)
		http.Error(w, "server error", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}
