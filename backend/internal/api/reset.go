package api

import (
	"log"
	"net/http"

	"github.com/jameynakama/flockdeck/internal/auth"
	"github.com/jameynakama/flockdeck/internal/store"
)

type resetRequest struct {
	Scope string `json:"scope"` // "schedule" (cards only) or "everything" (cards + review_log)
}

type resetResponse struct {
	CardsDeleted   int64 `json:"cards_deleted"`
	ReviewsDeleted int64 `json:"reviews_deleted"`
}

// resetUserData irreversibly deletes the authenticated user's learning data.
// "schedule" wipes FSRS card state (history-derived stats survive);
// "everything" also wipes review_log. Decks, deck_species, and
// user_species_preferences are never touched -- cards re-create lazily on
// the next practice. Both scopes run in inTx; for "everything" atomicity
// matters (no half-nuked state), for "schedule" it's one statement either way.
func (h *Handler) resetUserData(w http.ResponseWriter, r *http.Request) {
	userID := auth.UserIDFromCtx(r.Context())

	req, ok := decodeJSON[resetRequest](w, r)
	if !ok {
		return
	}
	if req.Scope != "schedule" && req.Scope != "everything" {
		writeError(w, http.StatusBadRequest, `scope must be "schedule" or "everything"`)
		return
	}

	var resp resetResponse
	err := h.inTx(r.Context(), func(q store.Querier) error {
		var err error
		resp.CardsDeleted, err = q.DeleteAllCardsForUser(r.Context(), userID)
		if err != nil {
			return err
		}
		if req.Scope == "everything" {
			resp.ReviewsDeleted, err = q.DeleteAllReviewsForUser(r.Context(), userID)
		}
		return err
	})
	if err != nil {
		log.Printf("reset tx error: %v", err)
		writeError(w, http.StatusInternalServerError, "server error")
		return
	}

	writeJSON(w, http.StatusOK, resp)
}
