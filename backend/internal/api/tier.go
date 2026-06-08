package api

import (
	"log"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/jameynakama/flockdeck/internal/auth"
	"github.com/jameynakama/flockdeck/internal/store"
)

type tierBird struct {
	statsSpecies
	Lane      string  `json:"lane"`
	Stability float64 `json:"stability"`
}

func (h *Handler) getStatsTier(w http.ResponseWriter, r *http.Request) {
	userID := auth.UserIDFromCtx(r.Context())
	stage := chi.URLParam(r, "stage")
	win, ok := tierWindowFor(stage)
	if !ok {
		writeError(w, http.StatusBadRequest, "unknown stage")
		return
	}
	lane := r.URL.Query().Get("lane")
	if lane != "" && lane != "audio" && lane != "image" {
		writeError(w, http.StatusBadRequest, "lane must be audio or image")
		return
	}

	rows, err := h.queries.GetCardsInTier(r.Context(), store.GetCardsInTierParams{
		UserID:       userID,
		Lane:         laneArg(lane),
		Egg:          win.egg,
		MinStability: win.min,
		Unbounded:    win.unbounded,
		MaxStability: win.max,
	})
	if err != nil {
		log.Printf("getStatsTier GetCardsInTier: %v", err)
		writeError(w, http.StatusInternalServerError, "server error")
		return
	}

	birds := make([]tierBird, 0, len(rows))
	for _, c := range rows {
		birds = append(birds, tierBird{
			statsSpecies: statsSpecies{
				EbirdCode:      c.SpeciesCode,
				CommonName:     c.CommonName,
				ScientificName: c.ScientificName,
			},
			Lane:      c.Lane,
			Stability: c.Stability,
		})
	}
	writeJSON(w, http.StatusOK, birds)
}
