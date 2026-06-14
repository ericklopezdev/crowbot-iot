// Package api implements the parent-facing HTTP API (PRODUCT.md P1).
package api

import (
	"log/slog"
	"net/http"

	"github.com/ErickLopezDev/cwlb-server/internal/auth"
	"github.com/ErickLopezDev/cwlb-server/internal/store"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Server struct {
	pool   *pgxpool.Pool
	q      *store.Queries
	tokens *auth.TokenManager
	log    *slog.Logger
}

func NewServer(pool *pgxpool.Pool, tokens *auth.TokenManager, log *slog.Logger) *Server {
	return &Server{pool: pool, q: store.New(pool), tokens: tokens, log: log}
}

// Routes builds the router with global middleware applied.
func (s *Server) Routes() http.Handler {
	mux := http.NewServeMux()

	mux.HandleFunc("GET /healthz", s.handleHealthz)

	mux.HandleFunc("POST /api/auth/signup", s.handleSignup)
	mux.HandleFunc("POST /api/auth/login", s.handleLogin)
	mux.HandleFunc("POST /api/auth/refresh", s.handleRefresh)

	mux.Handle("POST /api/devices/claim", s.requireAuth(http.HandlerFunc(s.handleClaimDevice)))
	mux.Handle("GET /api/devices", s.requireAuth(http.HandlerFunc(s.handleListDevices)))
	mux.Handle("POST /api/devices/{id}/assign", s.requireAuth(http.HandlerFunc(s.handleAssignDevice)))

	mux.Handle("POST /api/children", s.requireAuth(http.HandlerFunc(s.handleCreateChild)))
	mux.Handle("GET /api/children", s.requireAuth(http.HandlerFunc(s.handleListChildren)))
	mux.Handle("GET /api/children/{id}", s.requireAuth(http.HandlerFunc(s.handleGetChild)))

	// Dashboard reads (PRODUCT.md P3).
	mux.Handle("GET /api/children/{id}/overview", s.requireAuth(http.HandlerFunc(s.handleChildOverview)))
	mux.Handle("GET /api/children/{id}/usage", s.requireAuth(http.HandlerFunc(s.handleChildUsage)))
	mux.Handle("GET /api/children/{id}/topics", s.requireAuth(http.HandlerFunc(s.handleChildTopics)))
	mux.Handle("GET /api/children/{id}/areas", s.requireAuth(http.HandlerFunc(s.handleChildAreas)))
	mux.Handle("GET /api/children/{id}/recommendations", s.requireAuth(http.HandlerFunc(s.handleChildRecommendations)))
	mux.Handle("GET /api/children/{id}/interactions", s.requireAuth(http.HandlerFunc(s.handleChildInteractions)))

	return s.recoverer(s.logRequests(mux))
}

func (s *Server) handleHealthz(w http.ResponseWriter, r *http.Request) {
	if err := s.pool.Ping(r.Context()); err != nil {
		writeError(w, http.StatusServiceUnavailable, "db unavailable")
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}
