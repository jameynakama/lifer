package api

import (
	"encoding/json"
	"log"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
	"github.com/jameynakama/lifer/internal/auth"
	"github.com/jameynakama/lifer/internal/store"
)

type preferencesRequest struct {
	AudioEnabled bool `json:"audio_enabled"`
	ImageEnabled bool `json:"image_enabled"`
}

func (h *Handler) updatePreferences(w http.ResponseWriter, r *http.Request) {
	userID := auth.UserIDFromCtx(r.Context())

	speciesID, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		http.Error(w, "invalid species id", http.StatusBadRequest)
		return
	}

	var req preferencesRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}

	pref, err := h.queries.UpsertPreferences(r.Context(), store.UpsertPreferencesParams{
		UserID:       userID,
		SpeciesID:    speciesID,
		AudioEnabled: req.AudioEnabled,
		ImageEnabled: req.ImageEnabled,
	})
	if err != nil {
		log.Printf("UpsertPreferences error: %v", err)
		http.Error(w, "server error", http.StatusInternalServerError)
		return
	}

	for _, lane := range []string{"audio", "image"} {
		enabled := (lane == "audio" && req.AudioEnabled) || (lane == "image" && req.ImageEnabled)
		if enabled {
			if err := h.queries.UpsertCard(r.Context(), store.UpsertCardParams{
				UserID:    userID,
				SpeciesID: speciesID,
				Lane:      lane,
			}); err != nil {
				log.Printf("UpsertCard error: %v", err)
				http.Error(w, "server error", http.StatusInternalServerError)
				return
			}
		} else {
			if err := h.queries.DeleteCard(r.Context(), store.DeleteCardParams{
				UserID:    userID,
				SpeciesID: speciesID,
				Lane:      lane,
			}); err != nil {
				log.Printf("DeleteCard error: %v", err)
				http.Error(w, "server error", http.StatusInternalServerError)
				return
			}
		}
	}

	writeJSON(w, http.StatusOK, pref)
}
