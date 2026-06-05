package auth

import (
	"context"
	"net/http"
)

// writeJSONError mirrors the api package's error envelope so middleware
// rejections parse the same as handler errors.
func writeJSONError(w http.ResponseWriter, status int, msg string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_, _ = w.Write([]byte(`{"error":"` + msg + `"}`))
}

type contextKey string

const userIDKey contextKey = "userID"
const isAdminKey contextKey = "isAdmin"

// CookieName is the auth-token cookie, shared with the api package.
const CookieName = "flockdeck_token"

func RequireAuth(secret []byte) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			cookie, err := r.Cookie(CookieName)
			if err != nil {
				writeJSONError(w, http.StatusUnauthorized, "unauthorized")
				return
			}
			claims, err := VerifyToken(cookie.Value, secret)
			if err != nil {
				writeJSONError(w, http.StatusUnauthorized, "unauthorized")
				return
			}
			ctx := context.WithValue(r.Context(), userIDKey, claims.UserID)
			ctx = context.WithValue(ctx, isAdminKey, claims.IsAdmin)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// UserIDKey returns the context key used by RequireAuth so tests can inject a user ID.
func UserIDKey() any {
	return userIDKey
}

// IsAdminKey returns the context key used by RequireAuth so tests can inject isAdmin.
func IsAdminKey() any {
	return isAdminKey
}

func UserIDFromCtx(ctx context.Context) int64 {
	id, _ := ctx.Value(userIDKey).(int64)
	return id
}

func IsAdminFromCtx(ctx context.Context) bool {
	v, _ := ctx.Value(isAdminKey).(bool)
	return v
}

// RequireAdmin returns 403 if the request context does not have isAdmin=true.
// Must be used after RequireAuth.
func RequireAdmin(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !IsAdminFromCtx(r.Context()) {
			writeJSONError(w, http.StatusForbidden, "forbidden")
			return
		}
		next.ServeHTTP(w, r)
	})
}
