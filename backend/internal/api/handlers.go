package api

import (
	"crypto/rand"
	"encoding/base64"
	"log"
	"net/http"
	"time"

	"github.com/jameynakama/lifer/internal/auth"
	"github.com/jameynakama/lifer/internal/store"
)

const (
	stateCookieName = "oauth_state"
	authCookieName  = "lifer_token"
)

func (h *Handler) googleLogin(w http.ResponseWriter, r *http.Request) {
	state := randomState()
	http.SetCookie(w, &http.Cookie{
		Name:     stateCookieName,
		Value:    state,
		MaxAge:   300,
		HttpOnly: true,
		Secure:   true,
		SameSite: http.SameSiteLaxMode,
		Path:     "/",
	})
	http.Redirect(w, r, h.oauthConfig.AuthCodeURL(state), http.StatusTemporaryRedirect)
}

func (h *Handler) googleCallback(w http.ResponseWriter, r *http.Request) {
	stateCookie, err := r.Cookie(stateCookieName)
	if err != nil || stateCookie.Value != r.URL.Query().Get("state") {
		http.Error(w, "invalid state", http.StatusBadRequest)
		return
	}

	googleUser, err := auth.GetGoogleUser(r.Context(), h.oauthConfig, r.URL.Query().Get("code"))
	if err != nil {
		log.Printf("google callback error: %v", err)
		http.Error(w, "authentication failed", http.StatusInternalServerError)
		return
	}

	user, err := h.queries.UpsertUser(r.Context(), store.UpsertUserParams{
		GoogleID: googleUser.ID,
		Email:    googleUser.Email,
		Name:     googleUser.Name,
		Picture:  googleUser.Picture,
	})
	if err != nil {
		log.Printf("upsert user error: %v", err)
		http.Error(w, "server error", http.StatusInternalServerError)
		return
	}

	token, err := auth.SignToken(user.ID, user.IsAdmin, h.jwtSecret)
	if err != nil {
		log.Printf("sign token error: %v", err)
		http.Error(w, "server error", http.StatusInternalServerError)
		return
	}

	http.SetCookie(w, &http.Cookie{
		Name:     authCookieName,
		Value:    token,
		Expires:  time.Now().Add(30 * 24 * time.Hour),
		HttpOnly: true,
		Secure:   true,
		SameSite: http.SameSiteLaxMode,
		Path:     "/",
	})
	http.Redirect(w, r, h.frontendURL, http.StatusTemporaryRedirect)
}

func (h *Handler) getMe(w http.ResponseWriter, r *http.Request) {
	userID := auth.UserIDFromCtx(r.Context())
	user, err := h.queries.GetUserByID(r.Context(), userID)
	if err != nil {
		http.Error(w, "user not found", http.StatusNotFound)
		return
	}
	writeJSON(w, http.StatusOK, user)
}

func randomState() string {
	b := make([]byte, 16)
	rand.Read(b) //nolint:errcheck
	return base64.URLEncoding.EncodeToString(b)
}


