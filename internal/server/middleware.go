package server

import (
	"context"
	"net/http"
	"strings"
)

type contextKey string

const IsAdminContextKey = contextKey("isAdmin")

// AdminAuthMiddleware checks for the admin session cookie and adds a flag to the request context.
// This allows us to know if a user is an admin on any route, without blocking access.
func (s *Server) AdminAuthMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		cookie, err := r.Cookie("admin_session")
		isAdmin := err == nil && cookie.Value == "logged_in"

		ctx := context.WithValue(r.Context(), IsAdminContextKey, isAdmin)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// RequireAdmin middleware checks the context flag set by AdminAuthMiddleware.
// If the user is not an admin, it redirects them to the login page.
// This should be used to protect specific routes like `/admin/*`.
func (s *Server) RequireAdmin(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		isAdmin, ok := r.Context().Value(IsAdminContextKey).(bool)

		// This should only protect routes that are not the login page itself
		if strings.HasPrefix(r.URL.Path, s.cfg.AdminLoginPath) {
			next.ServeHTTP(w, r)
			return
		}

		if !ok || !isAdmin {
			http.Redirect(w, r, s.cfg.AdminLoginPath, http.StatusFound)
			return
		}

		next.ServeHTTP(w, r)
	})
}
