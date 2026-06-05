package api

import (
	"crypto/rand"
	"encoding/base64"
	"log"
	"net/http"
	"time"

	"github.com/jameynakama/flockdeck/internal/auth"
	"github.com/jameynakama/flockdeck/internal/store"
)

const (
	// stateCookiePrefix + <state> names a per-flow cookie, so concurrent
	// login flows (e.g. a browser tab and an installed PWA window) cannot
	// clobber each other's state.
	stateCookiePrefix = "oauth_state_"
)

// authCookieName aliases the shared auth-cookie constant.
const authCookieName = auth.CookieName

func (h *Handler) googleLogin(w http.ResponseWriter, r *http.Request) {
	state, err := randomState()
	if err != nil {
		log.Printf("oauth state generation: %v", err)
		writeError(w, http.StatusInternalServerError, "server error")
		return
	}
	http.SetCookie(w, &http.Cookie{
		Name:     stateCookiePrefix + state,
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
	state := r.URL.Query().Get("state")
	stateCookie, err := r.Cookie(stateCookiePrefix + state)
	if state == "" || err != nil || stateCookie.Value != state {
		// This is a browser navigation: redirect to the frontend with an
		// error param rather than white-screening on a bare 400.
		log.Printf("oauth callback: state cookie missing or mismatched (err=%v)", err)
		http.Redirect(w, r, h.frontendURL+"/?error=auth_state", http.StatusTemporaryRedirect)
		return
	}
	// The state is one-time use: expire its cookie immediately.
	http.SetCookie(w, &http.Cookie{
		Name:     stateCookiePrefix + state,
		Value:    "",
		MaxAge:   -1,
		HttpOnly: true,
		Secure:   true,
		SameSite: http.SameSiteLaxMode,
		Path:     "/",
	})

	googleUser, err := auth.GetGoogleUser(r.Context(), h.oauthConfig, r.URL.Query().Get("code"))
	if err != nil {
		log.Printf("google callback error: %v", err)
		writeError(w, http.StatusInternalServerError, "authentication failed")
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
		writeError(w, http.StatusInternalServerError, "server error")
		return
	}

	token, err := auth.SignToken(user.ID, user.IsAdmin, h.jwtSecret)
	if err != nil {
		log.Printf("sign token error: %v", err)
		writeError(w, http.StatusInternalServerError, "server error")
		return
	}

	http.SetCookie(w, &http.Cookie{
		Name:     authCookieName,
		Value:    token,
		Expires:  time.Now().Add(auth.TokenDuration),
		HttpOnly: true,
		Secure:   true,
		SameSite: http.SameSiteLaxMode,
		Path:     "/",
	})
	http.Redirect(w, r, h.frontendURL, http.StatusTemporaryRedirect)
}

func (h *Handler) logout(w http.ResponseWriter, r *http.Request) {
	http.SetCookie(w, &http.Cookie{
		Name:     authCookieName,
		Value:    "",
		MaxAge:   -1,
		HttpOnly: true,
		Secure:   true,
		SameSite: http.SameSiteLaxMode,
		Path:     "/",
	})
	w.WriteHeader(http.StatusNoContent)
}

func (h *Handler) getMe(w http.ResponseWriter, r *http.Request) {
	userID := auth.UserIDFromCtx(r.Context())
	user, err := h.queries.GetUserByID(r.Context(), userID)
	if err != nil {
		writeError(w, http.StatusNotFound, "user not found")
		return
	}
	if user.IsAdmin != auth.IsAdminFromCtx(r.Context()) {
		token, err := auth.SignToken(user.ID, user.IsAdmin, h.jwtSecret)
		if err != nil {
			log.Printf("getMe: re-sign token: %v", err)
		} else {
			http.SetCookie(w, &http.Cookie{
				Name:     authCookieName,
				Value:    token,
				Expires:  time.Now().Add(auth.TokenDuration),
				HttpOnly: true,
				Secure:   true,
				SameSite: http.SameSiteLaxMode,
				Path:     "/",
			})
		}
	}
	writeJSON(w, http.StatusOK, user)
}

// randomState returns a URL- and cookie-name-safe random state value.
func randomState() (string, error) {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(b), nil
}
