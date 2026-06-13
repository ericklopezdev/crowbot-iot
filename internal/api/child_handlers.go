package api

import (
	"net/http"
	"time"

	"github.com/ErickLopezDev/cwlb-server/internal/store"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
)

func (s *Server) handleCreateChild(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Name      string `json:"name"`
		Birthdate string `json:"birthdate"` // optional, YYYY-MM-DD
		Avatar    string `json:"avatar"`
	}
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid body")
		return
	}
	if req.Name == "" {
		writeError(w, http.StatusBadRequest, "name is required")
		return
	}
	var bd pgtype.Date
	if req.Birthdate != "" {
		t, err := time.Parse("2006-01-02", req.Birthdate)
		if err != nil {
			writeError(w, http.StatusBadRequest, "birthdate must be YYYY-MM-DD")
			return
		}
		bd = pgtype.Date{Time: t, Valid: true}
	}

	child, err := s.q.CreateChild(r.Context(), store.CreateChildParams{
		AccountID: accountID(r),
		Name:      req.Name,
		Birthdate: bd,
		Avatar:    req.Avatar,
	})
	if err != nil {
		writeError(w, http.StatusInternalServerError, "create child")
		return
	}
	writeJSON(w, http.StatusCreated, child)
}

func (s *Server) handleListChildren(w http.ResponseWriter, r *http.Request) {
	children, err := s.q.ListChildrenByAccount(r.Context(), accountID(r))
	if err != nil {
		writeError(w, http.StatusInternalServerError, "list children")
		return
	}
	if children == nil {
		children = []store.Child{}
	}
	writeJSON(w, http.StatusOK, children)
}

func (s *Server) handleGetChild(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(r.PathValue("id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid child id")
		return
	}
	child, err := s.q.GetChild(r.Context(), id)
	if err != nil || child.AccountID != accountID(r) {
		writeError(w, http.StatusNotFound, "child not found")
		return
	}
	writeJSON(w, http.StatusOK, child)
}
