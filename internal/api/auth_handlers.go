package api

import (
	"net/http"

	"github.com/ErickLopezDev/cwlb-server/internal/auth"
	"github.com/ErickLopezDev/cwlb-server/internal/store"
	"github.com/google/uuid"
)

type tokenResponse struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	ExpiresIn    int    `json:"expires_in"`
}

func (s *Server) respondTokens(w http.ResponseWriter, status int, accountID uuid.UUID) {
	access, err := s.tokens.AccessToken(accountID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "issue token")
		return
	}
	refresh, err := s.tokens.RefreshToken(accountID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "issue token")
		return
	}
	writeJSON(w, status, tokenResponse{
		AccessToken:  access,
		RefreshToken: refresh,
		ExpiresIn:    int(s.tokens.AccessTTL().Seconds()),
	})
}

func (s *Server) handleSignup(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Email    string `json:"email"`
		Password string `json:"password"`
		FullName string `json:"full_name"`
	}
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid body")
		return
	}
	if req.Email == "" || len(req.Password) < 8 {
		writeError(w, http.StatusBadRequest, "email and password (min 8 chars) required")
		return
	}
	hash, err := auth.HashPassword(req.Password)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "hash password")
		return
	}
	acc, err := s.q.CreateAccount(r.Context(), store.CreateAccountParams{
		Email:        req.Email,
		PasswordHash: hash,
		FullName:     req.FullName,
	})
	if err != nil {
		if isUniqueViolation(err) {
			writeError(w, http.StatusConflict, "email already registered")
			return
		}
		writeError(w, http.StatusInternalServerError, "create account")
		return
	}
	s.respondTokens(w, http.StatusCreated, acc.ID)
}

func (s *Server) handleLogin(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Email    string `json:"email"`
		Password string `json:"password"`
	}
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid body")
		return
	}
	acc, err := s.q.GetAccountByEmail(r.Context(), req.Email)
	if err != nil || !auth.CheckPassword(acc.PasswordHash, req.Password) {
		writeError(w, http.StatusUnauthorized, "invalid credentials")
		return
	}
	s.respondTokens(w, http.StatusOK, acc.ID)
}

func (s *Server) handleRefresh(w http.ResponseWriter, r *http.Request) {
	var req struct {
		RefreshToken string `json:"refresh_token"`
	}
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid body")
		return
	}
	id, err := s.tokens.Parse(req.RefreshToken, auth.TokenRefresh)
	if err != nil {
		writeError(w, http.StatusUnauthorized, "invalid refresh token")
		return
	}
	if _, err := s.q.GetAccountByID(r.Context(), id); err != nil {
		writeError(w, http.StatusUnauthorized, "account not found")
		return
	}
	s.respondTokens(w, http.StatusOK, id)
}
