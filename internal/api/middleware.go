package api

import (
	"context"
	"net/http"
	"strings"
	"time"

	"github.com/ErickLopezDev/cwlb-server/internal/auth"
	"github.com/google/uuid"
)

type ctxKey string

const accountIDKey ctxKey = "accountID"

// requireAuth validates the Bearer access token and injects the account id.
func (s *Server) requireAuth(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		token, ok := strings.CutPrefix(r.Header.Get("Authorization"), "Bearer ")
		if !ok || token == "" {
			writeError(w, http.StatusUnauthorized, "missing bearer token")
			return
		}
		id, err := s.tokens.Parse(token, auth.TokenAccess)
		if err != nil {
			writeError(w, http.StatusUnauthorized, "invalid or expired token")
			return
		}
		ctx := context.WithValue(r.Context(), accountIDKey, id)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// accountID returns the authenticated account id from the request context.
func accountID(r *http.Request) uuid.UUID {
	id, _ := r.Context().Value(accountIDKey).(uuid.UUID)
	return id
}

func (s *Server) logRequests(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		next.ServeHTTP(w, r)
		s.log.Info("request",
			"method", r.Method,
			"path", r.URL.Path,
			"dur", time.Since(start).String(),
		)
	})
}

func (s *Server) recoverer(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if rec := recover(); rec != nil {
				s.log.Error("panic", "err", rec, "path", r.URL.Path)
				writeError(w, http.StatusInternalServerError, "internal error")
			}
		}()
		next.ServeHTTP(w, r)
	})
}
