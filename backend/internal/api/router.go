package api

import (
	"context"
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/jackc/pgx/v5"
	"github.com/jameynakama/flockdeck/internal/auth"
	"github.com/jameynakama/flockdeck/internal/r2"
	"github.com/jameynakama/flockdeck/internal/store"
	"golang.org/x/oauth2"
)

// txBeginner is the slice of *pgxpool.Pool the handlers need to open
// transactions; nil in unit tests.
type txBeginner interface {
	Begin(ctx context.Context) (pgx.Tx, error)
}

type RouterConfig struct {
	Queries     store.Querier
	DB          txBeginner // optional: enables transactional handlers
	OAuthConfig *oauth2.Config
	JWTSecret   []byte
	FrontendURL string
	R2Client    *r2.Client
}

type Handler struct {
	queries     store.Querier
	db          txBeginner
	oauthConfig *oauth2.Config
	jwtSecret   []byte
	frontendURL string
	r2Client    *r2.Client
}

func NewRouter(cfg RouterConfig) http.Handler {
	h := &Handler{
		queries:     cfg.Queries,
		db:          cfg.DB,
		oauthConfig: cfg.OAuthConfig,
		jwtSecret:   cfg.JWTSecret,
		frontendURL: cfg.FrontendURL,
		r2Client:    cfg.R2Client,
	}

	r := chi.NewRouter()
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)
	r.Use(middleware.RequestID)

	r.Get("/health", h.healthCheck)

	r.Route("/api/v1", func(r chi.Router) {
		r.Get("/auth/google", h.googleLogin)
		r.Get("/auth/google/callback", h.googleCallback)
		r.Post("/auth/logout", h.logout)

		r.Get("/decks/presets", h.listPresetDecks)

		r.Group(func(r chi.Router) {
			r.Use(auth.RequireAuth(cfg.JWTSecret))
			r.Get("/me", h.getMe)
			r.Get("/stats", h.getStats)

			r.Get("/decks", h.listDecks)
			r.Post("/decks", h.createDeck)
			r.Get("/decks/{id}", h.getDeckDetail)
			r.Patch("/decks/{id}", h.updateDeck)
			r.Delete("/decks/{id}", h.deleteDeck)
			r.Post("/decks/{id}/clone", h.cloneDeck)

			r.Get("/decks/{id}/species", h.listDeckSpecies)
			r.Post("/decks/{id}/species", h.addSpeciesToDeck)
			r.Post("/decks/{id}/species/bulk", h.bulkAddSpeciesToDeck)
			r.Delete("/decks/{id}/species/{ebird_code}", h.removeSpeciesFromDeck)

			r.Get("/decks/{id}/next", h.getNextCard)
			r.Post("/decks/{id}/rate", h.rateCard)
			r.Get("/decks/{id}/practice", h.getPracticeCards)

			r.Get("/species", h.listSpecies)
			r.Get("/species/all", h.listAllSpecies)
			r.Get("/species/{ebird_code}", h.getSpeciesDetail)
			r.Get("/species/{ebird_code}/decks", h.getSpeciesDecks)
			r.Put("/species/{ebird_code}/preferences", h.updatePreferences)
		})

		r.With(auth.RequireAuth(cfg.JWTSecret), auth.RequireAdmin).Route("/admin", func(r chi.Router) {
			r.Get("/species/{ebird_code}", h.adminGetSpeciesDetail)
			r.Post("/species/{ebird_code}/images", h.adminUploadImage)
			r.Post("/species/{ebird_code}/recordings", h.adminUploadRecording)
			r.Delete("/species/{ebird_code}/images/{macaulay_id}", h.adminDeleteImage)
			r.Patch("/species/{ebird_code}/images/{macaulay_id}/locked", h.adminSetImageLocked)
			r.Delete("/species/{ebird_code}/recordings/{xeno_canto_id}", h.adminDeleteRecording)
			r.Patch("/species/{ebird_code}/recordings/{xeno_canto_id}/locked", h.adminSetRecordingLocked)

			r.Get("/decks", h.adminListUserDecks)
			r.Get("/decks/{id}/species", h.adminGetDeckSpecies)
			r.Post("/decks", h.adminCreatePresetDeck)
			r.Patch("/decks/{id}", h.adminUpdatePresetDeck)
			r.Delete("/decks/{id}", h.adminDeletePresetDeck)

			r.Get("/users", h.adminGetUsers)
			r.Patch("/users/{id}", h.adminSetUserIsAdmin)
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
