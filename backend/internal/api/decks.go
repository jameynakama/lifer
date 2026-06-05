package api

import (
	"errors"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jameynakama/flockdeck/internal/auth"
	"github.com/jameynakama/flockdeck/internal/store"
)

// deckRequest is the body for both deck create and update.
type deckRequest struct {
	Name        string  `json:"name"`
	Description *string `json:"description"`
}

type addSpeciesRequest struct {
	EbirdCode string `json:"ebird_code"`
}

// deckOwnerCheck fetches the deck, writes 404/403 and returns false if the
// requesting user does not own it. Admins bypass the ownership check.
func (h *Handler) deckOwnerCheck(w http.ResponseWriter, r *http.Request, deckID, userID int64) bool {
	deck, err := h.queries.GetDeck(r.Context(), deckID)
	if errors.Is(err, pgx.ErrNoRows) {
		writeError(w, http.StatusNotFound, "not found")
		return false
	}
	if err != nil {
		log.Printf("GetDeck error: %v", err)
		writeError(w, http.StatusInternalServerError, "server error")
		return false
	}
	isAdmin := auth.IsAdminFromCtx(r.Context())
	if !isAdmin && (!deck.OwnerID.Valid || deck.OwnerID.Int64 != userID) {
		writeError(w, http.StatusForbidden, "forbidden")
		return false
	}
	return true
}

func (h *Handler) listPresetDecks(w http.ResponseWriter, r *http.Request) {
	decks, err := h.queries.ListPresetDecks(r.Context())
	if err != nil {
		log.Printf("ListPresetDecks error: %v", err)
		writeError(w, http.StatusInternalServerError, "server error")
		return
	}
	writeJSON(w, http.StatusOK, orEmpty(decks))
}

func (h *Handler) cloneDeck(w http.ResponseWriter, r *http.Request) {
	userID := auth.UserIDFromCtx(r.Context())

	deckID, ok := parseID(w, r, "id", "deck")
	if !ok {
		return
	}

	src, err := h.queries.GetDeck(r.Context(), deckID)
	if errors.Is(err, pgx.ErrNoRows) {
		writeError(w, http.StatusNotFound, "not found")
		return
	}
	if err != nil {
		log.Printf("GetDeck error: %v", err)
		writeError(w, http.StatusInternalServerError, "server error")
		return
	}
	if src.OwnerID.Valid {
		writeError(w, http.StatusBadRequest, "can only clone preset decks")
		return
	}

	var newDeck store.Deck
	err = h.inTx(r.Context(), func(q store.Querier) error {
		var err error
		newDeck, err = q.CreateDeck(r.Context(), store.CreateDeckParams{
			Name:        src.Name,
			Description: src.Description,
			OwnerID:     pgtype.Int8{Int64: userID, Valid: true},
		})
		if err != nil {
			return fmt.Errorf("CreateDeck (clone): %w", err)
		}
		if err := q.CloneDeckSpecies(r.Context(), store.CloneDeckSpeciesParams{
			DeckID:   deckID,
			DeckID_2: newDeck.ID,
		}); err != nil {
			return fmt.Errorf("CloneDeckSpecies: %w", err)
		}
		if err := q.UpsertCardsForDeck(r.Context(), store.UpsertCardsForDeckParams{
			UserID: userID,
			DeckID: newDeck.ID,
		}); err != nil {
			return fmt.Errorf("UpsertCardsForDeck: %w", err)
		}
		return nil
	})
	if err != nil {
		log.Printf("clone deck: %v", err)
		writeError(w, http.StatusInternalServerError, "server error")
		return
	}

	writeJSON(w, http.StatusCreated, newDeck)
}

func (h *Handler) getDeckDetail(w http.ResponseWriter, r *http.Request) {
	userID := auth.UserIDFromCtx(r.Context())

	deckID, ok := parseID(w, r, "id", "deck")
	if !ok {
		return
	}

	deck, err := h.queries.GetDeckWithDue(r.Context(), store.GetDeckWithDueParams{
		ID:     deckID,
		UserID: userID,
	})
	if errors.Is(err, pgx.ErrNoRows) {
		writeError(w, http.StatusNotFound, "not found")
		return
	}
	if err != nil {
		log.Printf("GetDeckWithDue error: %v", err)
		writeError(w, http.StatusInternalServerError, "server error")
		return
	}

	writeJSON(w, http.StatusOK, deck)
}

type listDecksResponse struct {
	Decks     []store.ListUserDecksRow `json:"decks"`
	NextDueAt *string                  `json:"next_due_at"`
}

func (h *Handler) listDecks(w http.ResponseWriter, r *http.Request) {
	userID := auth.UserIDFromCtx(r.Context())
	decks, err := h.queries.ListUserDecks(r.Context(), userID)
	if err != nil {
		log.Printf("ListUserDecks error: %v", err)
		writeError(w, http.StatusInternalServerError, "server error")
		return
	}
	nextDue, err := h.queries.GetNextDueAt(r.Context(), userID)
	if errors.Is(err, pgx.ErrNoRows) {
		// no future cards -- next_due_at stays nil
	} else if err != nil {
		log.Printf("GetNextDueAt error: %v", err)
		writeError(w, http.StatusInternalServerError, "server error")
		return
	} else if nextDue.Valid {
		t := nextDue.Time.UTC().Format(time.RFC3339)
		resp := listDecksResponse{Decks: orEmpty(decks), NextDueAt: &t}
		writeJSON(w, http.StatusOK, resp)
		return
	}
	writeJSON(w, http.StatusOK, listDecksResponse{Decks: orEmpty(decks)})
}

func (h *Handler) createDeck(w http.ResponseWriter, r *http.Request) {
	userID := auth.UserIDFromCtx(r.Context())

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
	deck, err := h.queries.CreateDeck(r.Context(), store.CreateDeckParams{
		Name:        req.Name,
		Description: desc,
		OwnerID:     pgtype.Int8{Int64: userID, Valid: true},
	})
	if err != nil {
		log.Printf("CreateDeck error: %v", err)
		writeError(w, http.StatusInternalServerError, "server error")
		return
	}

	writeJSON(w, http.StatusCreated, deck)
}

func (h *Handler) updateDeck(w http.ResponseWriter, r *http.Request) {
	deckID, ok := h.ownedDeckID(w, r)
	if !ok {
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
	deck, err := h.queries.UpdateDeck(r.Context(), store.UpdateDeckParams{
		ID:          deckID,
		Name:        req.Name,
		Description: desc,
	})
	if err != nil {
		log.Printf("UpdateDeck error: %v", err)
		writeError(w, http.StatusInternalServerError, "server error")
		return
	}

	writeJSON(w, http.StatusOK, deck)
}

func (h *Handler) deleteDeck(w http.ResponseWriter, r *http.Request) {
	deckID, ok := h.ownedDeckID(w, r)
	if !ok {
		return
	}

	if err := h.queries.DeleteDeck(r.Context(), deckID); err != nil {
		log.Printf("DeleteDeck error: %v", err)
		writeError(w, http.StatusInternalServerError, "server error")
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func (h *Handler) listDeckSpecies(w http.ResponseWriter, r *http.Request) {
	userID := auth.UserIDFromCtx(r.Context())

	deckID, ok := h.ownedDeckID(w, r)
	if !ok {
		return
	}

	species, err := h.queries.ListDeckSpeciesWithPrefs(r.Context(), store.ListDeckSpeciesWithPrefsParams{
		DeckID: deckID,
		UserID: userID,
	})
	if err != nil {
		log.Printf("ListDeckSpeciesWithPrefs error: %v", err)
		writeError(w, http.StatusInternalServerError, "server error")
		return
	}
	writeJSON(w, http.StatusOK, orEmpty(species))
}

func (h *Handler) addSpeciesToDeck(w http.ResponseWriter, r *http.Request) {
	userID := auth.UserIDFromCtx(r.Context())

	deckID, ok := h.ownedDeckID(w, r)
	if !ok {
		return
	}

	req, ok := decodeJSON[addSpeciesRequest](w, r)
	if !ok {
		return
	}
	if req.EbirdCode == "" {
		writeError(w, http.StatusBadRequest, "ebird_code is required")
		return
	}

	err := h.inTx(r.Context(), func(q store.Querier) error {
		if err := q.AddSpeciesToDeck(r.Context(), store.AddSpeciesToDeckParams{
			DeckID:      deckID,
			SpeciesCode: req.EbirdCode,
		}); err != nil {
			return fmt.Errorf("AddSpeciesToDeck: %w", err)
		}
		for _, lane := range []string{"audio", "image"} {
			if err := q.UpsertCard(r.Context(), store.UpsertCardParams{
				UserID:      userID,
				SpeciesCode: req.EbirdCode,
				Lane:        lane,
			}); err != nil {
				return fmt.Errorf("UpsertCard: %w", err)
			}
		}
		return nil
	})
	if err != nil {
		log.Printf("add species to deck: %v", err)
		writeError(w, http.StatusInternalServerError, "server error")
		return
	}

	w.WriteHeader(http.StatusCreated)
}

type bulkAddSpeciesRequest struct {
	SpeciesCodes []string `json:"species_codes"`
}

func (h *Handler) bulkAddSpeciesToDeck(w http.ResponseWriter, r *http.Request) {
	userID := auth.UserIDFromCtx(r.Context())

	deckID, ok := h.ownedDeckID(w, r)
	if !ok {
		return
	}

	req, ok := decodeJSON[bulkAddSpeciesRequest](w, r)
	if !ok {
		return
	}
	if len(req.SpeciesCodes) == 0 {
		writeError(w, http.StatusBadRequest, "species_codes must not be empty")
		return
	}

	var added int64
	err := h.inTx(r.Context(), func(q store.Querier) error {
		var err error
		added, err = q.BulkAddSpeciesToDeck(r.Context(), store.BulkAddSpeciesToDeckParams{
			DeckID:  deckID,
			Column2: req.SpeciesCodes,
		})
		if err != nil {
			return fmt.Errorf("BulkAddSpeciesToDeck: %w", err)
		}
		if err := q.BulkUpsertCards(r.Context(), store.BulkUpsertCardsParams{
			UserID:  userID,
			Column2: req.SpeciesCodes,
		}); err != nil {
			return fmt.Errorf("BulkUpsertCards: %w", err)
		}
		return nil
	})
	if err != nil {
		log.Printf("bulk add species: %v", err)
		writeError(w, http.StatusInternalServerError, "server error")
		return
	}

	writeJSON(w, http.StatusOK, map[string]int64{"added": added})
}

func (h *Handler) removeSpeciesFromDeck(w http.ResponseWriter, r *http.Request) {
	deckID, ok := h.ownedDeckID(w, r)
	if !ok {
		return
	}

	ebirdCode := chi.URLParam(r, "ebird_code")

	if err := h.queries.RemoveSpeciesFromDeck(r.Context(), store.RemoveSpeciesFromDeckParams{
		DeckID:      deckID,
		SpeciesCode: ebirdCode,
	}); err != nil {
		log.Printf("RemoveSpeciesFromDeck error: %v", err)
		writeError(w, http.StatusInternalServerError, "server error")
		return
	}

	w.WriteHeader(http.StatusNoContent)
}
