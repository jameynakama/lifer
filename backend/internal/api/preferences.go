package api

import (
	"fmt"
	"log"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/jameynakama/flockdeck/internal/auth"
	"github.com/jameynakama/flockdeck/internal/store"
)

type preferencesRequest struct {
	AudioEnabled bool `json:"audio_enabled"`
	ImageEnabled bool `json:"image_enabled"`
}

func (h *Handler) updatePreferences(w http.ResponseWriter, r *http.Request) {
	userID := auth.UserIDFromCtx(r.Context())
	ebirdCode := chi.URLParam(r, "ebird_code")

	req, ok := decodeJSON[preferencesRequest](w, r)
	if !ok {
		return
	}

	var pref store.UserSpeciesPreference
	err := h.inTx(r.Context(), func(q store.Querier) error {
		var err error
		pref, err = q.UpsertPreferences(r.Context(), store.UpsertPreferencesParams{
			UserID:       userID,
			SpeciesCode:  ebirdCode,
			AudioEnabled: req.AudioEnabled,
			ImageEnabled: req.ImageEnabled,
		})
		if err != nil {
			return fmt.Errorf("UpsertPreferences: %w", err)
		}
		for _, lane := range []string{"audio", "image"} {
			enabled := (lane == "audio" && req.AudioEnabled) || (lane == "image" && req.ImageEnabled)
			if enabled {
				if err := q.UpsertCard(r.Context(), store.UpsertCardParams{
					UserID:      userID,
					SpeciesCode: ebirdCode,
					Lane:        lane,
				}); err != nil {
					return fmt.Errorf("UpsertCard: %w", err)
				}
			} else if err := q.DeleteCard(r.Context(), store.DeleteCardParams{
				UserID:      userID,
				SpeciesCode: ebirdCode,
				Lane:        lane,
			}); err != nil {
				return fmt.Errorf("DeleteCard: %w", err)
			}
		}
		return nil
	})
	if err != nil {
		log.Printf("update preferences: %v", err)
		writeError(w, http.StatusInternalServerError, "server error")
		return
	}

	writeJSON(w, http.StatusOK, pref)
}
