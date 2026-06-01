package api

import (
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/jameynakama/lifer/internal/auth"
	"github.com/jameynakama/lifer/internal/store"
	"golang.org/x/oauth2"
)

type RouterConfig struct {
	Queries     store.Querier
	OAuthConfig *oauth2.Config
	JWTSecret   []byte
	FrontendURL string
}

type Handler struct {
	queries     store.Querier
	oauthConfig *oauth2.Config
	jwtSecret   []byte
	frontendURL string
}

func NewRouter(cfg RouterConfig) http.Handler {
	h := &Handler{
		queries:     cfg.Queries,
		oauthConfig: cfg.OAuthConfig,
		jwtSecret:   cfg.JWTSecret,
		frontendURL: cfg.FrontendURL,
	}

	r := chi.NewRouter()
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)
	r.Use(middleware.RequestID)

	r.Get("/health", h.healthCheck)

	r.Route("/api/v1", func(r chi.Router) {
		r.Get("/auth/google", h.googleLogin)
		r.Get("/auth/google/callback", h.googleCallback)

		r.Group(func(r chi.Router) {
			r.Use(auth.RequireAuth(cfg.JWTSecret))
			r.Get("/me", h.getMe)

			r.Get("/groups", h.listGroups)
			r.Post("/groups", h.createGroup)
			r.Get("/groups/{id}", h.getGroupDetail)
			r.Patch("/groups/{id}", h.updateGroup)
			r.Delete("/groups/{id}", h.deleteGroup)

			r.Get("/groups/{id}/species", h.listGroupSpecies)
			r.Post("/groups/{id}/species", h.addSpeciesToGroup)
			r.Delete("/groups/{id}/species/{ebird_code}", h.removeSpeciesFromGroup)

			r.Get("/groups/{id}/next", h.getNextCard)
			r.Post("/groups/{id}/rate", h.rateCard)
			r.Get("/groups/{id}/practice", h.getPracticeCards)

			r.Get("/species", h.listSpecies)
			r.Get("/species/{ebird_code}", h.getSpeciesDetail)
			r.Get("/species/{ebird_code}/groups", h.getSpeciesGroups)
			r.Put("/species/{ebird_code}/preferences", h.updatePreferences)
		})
	})

	return r
}

func (h *Handler) healthCheck(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(v) //nolint:errcheck
}
