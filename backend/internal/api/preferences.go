package api

import (
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

	pref, err := h.queries.UpsertPreferences(r.Context(), store.UpsertPreferencesParams{
		UserID:       userID,
		SpeciesCode:  ebirdCode,
		AudioEnabled: req.AudioEnabled,
		ImageEnabled: req.ImageEnabled,
	})
	if err != nil {
		log.Printf("UpsertPreferences error: %v", err)
		writeError(w, http.StatusInternalServerError, "server error")
		return
	}

	for _, lane := range []string{"audio", "image"} {
		enabled := (lane == "audio" && req.AudioEnabled) || (lane == "image" && req.ImageEnabled)
		if enabled {
			if err := h.queries.UpsertCard(r.Context(), store.UpsertCardParams{
				UserID:      userID,
				SpeciesCode: ebirdCode,
				Lane:        lane,
			}); err != nil {
				log.Printf("UpsertCard error: %v", err)
				writeError(w, http.StatusInternalServerError, "server error")
				return
			}
		} else {
			if err := h.queries.DeleteCard(r.Context(), store.DeleteCardParams{
				UserID:      userID,
				SpeciesCode: ebirdCode,
				Lane:        lane,
			}); err != nil {
				log.Printf("DeleteCard error: %v", err)
				writeError(w, http.StatusInternalServerError, "server error")
				return
			}
		}
	}

	writeJSON(w, http.StatusOK, pref)
}
