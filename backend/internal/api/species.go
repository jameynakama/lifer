package api

import (
	"log"
	"net/http"

	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jameynakama/lifer/internal/store"
)

func (h *Handler) searchSpecies(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query().Get("q")

	results, err := h.queries.SearchSpecies(r.Context(), pgtype.Text{String: q, Valid: true})
	if err != nil {
		log.Printf("SearchSpecies error: %v", err)
		http.Error(w, "server error", http.StatusInternalServerError)
		return
	}
	if results == nil {
		results = []store.SearchSpeciesRow{}
	}
	writeJSON(w, http.StatusOK, results)
}
