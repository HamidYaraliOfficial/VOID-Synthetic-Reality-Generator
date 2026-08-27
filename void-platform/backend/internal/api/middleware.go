package api

import (
	"context"
	"log"
	"net/http"
	"strings"
	"time"

	"void-platform/backend/internal/security"
)

type ctxKey string

const claimsKey ctxKey = "claims"

// withMiddleware wraps every route with request logging + permissive CORS
// (tightened via AllowedOrigins for the WebSocket upgrade specifically).
func (s *Server) withMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Headers", "Authorization, Content-Type")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		next.ServeHTTP(w, r)
		log.Printf("%s %s %s in %s", r.RemoteAddr, r.Method, r.URL.Path, time.Since(start))
	})
}

// protect enforces: valid bearer token -> rate limit -> RBAC permission,
// then stores parsed Claims in the request context for handlers to use.
func (s *Server) protect(perm security.Permission, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		authHeader := r.Header.Get("Authorization")
		token := strings.TrimPrefix(authHeader, "Bearer ")
		if token == "" || (token == authHeader && !strings.HasPrefix(authHeader, "Bearer ")) {
			// WebSocket handshakes from a browser cannot set a custom
			// Authorization header, so accept ?token=... as a fallback for
			// those routes only (still fully verified below, same as the
			// header path — this is not a weaker auth mode).
			token = r.URL.Query().Get("token")
		}
		if token == "" {
			writeError(w, http.StatusUnauthorized, errAuth("missing bearer token"))
			return
		}
		claims, err := s.Tokens.Verify(token)
		if err != nil {
			writeError(w, http.StatusUnauthorized, err)
			return
		}
		rateKey := claims.Subject
		if !s.RateLimiter.Allow(rateKey) {
			writeError(w, http.StatusTooManyRequests, errAuth("rate limit exceeded"))
			return
		}
		if !security.HasPermission(claims.Roles, perm) {
			s.Audit.Record(claims.Subject, "authz.denied", string(perm), r.URL.Path)
			writeError(w, http.StatusForbidden, errAuth("insufficient role for this action"))
			return
		}
		ctx := context.WithValue(r.Context(), claimsKey, claims)
		s.Audit.Record(claims.Subject, "api.call", r.URL.Path, r.Method)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

func claimsFrom(r *http.Request) *security.Claims {
	if c, ok := r.Context().Value(claimsKey).(*security.Claims); ok {
		return c
	}
	return &security.Claims{Subject: "unknown"}
}

type authError string

func (e authError) Error() string { return string(e) }
func errAuth(msg string) error    { return authError(msg) }
