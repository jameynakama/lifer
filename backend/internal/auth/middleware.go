package auth

import (
	"context"
	"net/http"
)

type contextKey string

const userIDKey contextKey = "userID"
const isAdminKey contextKey = "isAdmin"

func RequireAuth(secret []byte) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			cookie, err := r.Cookie("flockdeck_token")
			if err != nil {
				http.Error(w, "unauthorized", http.StatusUnauthorized)
				return
			}
			claims, err := VerifyToken(cookie.Value, secret)
			if err != nil {
				http.Error(w, "unauthorized", http.StatusUnauthorized)
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

func UserIDFromCtx(ctx context.Context) int64 {
	id, _ := ctx.Value(userIDKey).(int64)
	return id
}

func IsAdminFromCtx(ctx context.Context) bool {
	v, _ := ctx.Value(isAdminKey).(bool)
	return v
}
