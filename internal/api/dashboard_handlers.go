package api

import (
	"net/http"
	"strconv"
	"time"

	"github.com/ErickLopezDev/cwlb-server/internal/store"
	"github.com/google/uuid"
)

// ownChild parses {id}, loads the child and verifies it belongs to the caller.
// On failure it writes the response and returns ok=false.
func (s *Server) ownChild(w http.ResponseWriter, r *http.Request) (store.Child, bool) {
	id, err := uuid.Parse(r.PathValue("id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid child id")
		return store.Child{}, false
	}
	child, err := s.q.GetChild(r.Context(), id)
	if err != nil || child.AccountID != accountID(r) {
		writeError(w, http.StatusNotFound, "child not found")
		return store.Child{}, false
	}
	return child, true
}

type overviewResponse struct {
	Child             store.Child                     `json:"child"`
	TotalInteractions int64                           `json:"total_interactions"`
	Areas             []store.AreaBreakdownByChildRow `json:"areas"`
	TopTopics         []store.ListChildTopicGraphRow  `json:"top_topics"`
}

func (s *Server) handleChildOverview(w http.ResponseWriter, r *http.Request) {
	child, ok := s.ownChild(w, r)
	if !ok {
		return
	}
	ctx := r.Context()

	total, err := s.q.CountInteractionsByChild(ctx, child.ID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "count interactions")
		return
	}
	areas, err := s.q.AreaBreakdownByChild(ctx, pgUUID(child.ID))
	if err != nil {
		writeError(w, http.StatusInternalServerError, "area breakdown")
		return
	}
	topics, err := s.q.ListChildTopicGraph(ctx, child.ID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "topic graph")
		return
	}
	if len(topics) > 5 {
		topics = topics[:5]
	}

	writeJSON(w, http.StatusOK, overviewResponse{
		Child:             child,
		TotalInteractions: total,
		Areas:             orEmpty(areas),
		TopTopics:         orEmpty(topics),
	})
}

func (s *Server) handleChildUsage(w http.ResponseWriter, r *http.Request) {
	child, ok := s.ownChild(w, r)
	if !ok {
		return
	}
	to := parseDateParam(r, "to", time.Now())
	from := parseDateParam(r, "from", to.AddDate(0, 0, -7))

	buckets, err := s.q.ChildUsage(r.Context(), child.ID, from, to)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "usage")
		return
	}
	writeJSON(w, http.StatusOK, orEmpty(buckets))
}

func (s *Server) handleChildTopics(w http.ResponseWriter, r *http.Request) {
	child, ok := s.ownChild(w, r)
	if !ok {
		return
	}
	topics, err := s.q.ListChildTopicGraph(r.Context(), child.ID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "topics")
		return
	}
	writeJSON(w, http.StatusOK, orEmpty(topics))
}

func (s *Server) handleChildAreas(w http.ResponseWriter, r *http.Request) {
	child, ok := s.ownChild(w, r)
	if !ok {
		return
	}
	areas, err := s.q.AreaBreakdownByChild(r.Context(), pgUUID(child.ID))
	if err != nil {
		writeError(w, http.StatusInternalServerError, "areas")
		return
	}
	writeJSON(w, http.StatusOK, orEmpty(areas))
}

func (s *Server) handleChildRecommendations(w http.ResponseWriter, r *http.Request) {
	child, ok := s.ownChild(w, r)
	if !ok {
		return
	}
	recs, err := s.q.ListRecommendationsByChild(r.Context(), child.ID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "recommendations")
		return
	}
	writeJSON(w, http.StatusOK, orEmpty(recs))
}

func (s *Server) handleChildInteractions(w http.ResponseWriter, r *http.Request) {
	child, ok := s.ownChild(w, r)
	if !ok {
		return
	}
	limit := parseIntParam(r, "limit", 50, 1, 200)
	offset := parseIntParam(r, "offset", 0, 0, 1<<30)

	items, err := s.q.ListInteractionsByChild(r.Context(), store.ListInteractionsByChildParams{
		ChildID: pgUUID(child.ID),
		Limit:   limit,
		Offset:  offset,
	})
	if err != nil {
		writeError(w, http.StatusInternalServerError, "interactions")
		return
	}
	writeJSON(w, http.StatusOK, orEmpty(items))
}

// orEmpty replaces a nil slice with an empty one so JSON encodes [] not null.
func orEmpty[T any](s []T) []T {
	if s == nil {
		return []T{}
	}
	return s
}

// parseDateParam reads a query param as RFC3339 or YYYY-MM-DD, falling back to def.
func parseDateParam(r *http.Request, key string, def time.Time) time.Time {
	v := r.URL.Query().Get(key)
	if v == "" {
		return def
	}
	if t, err := time.Parse(time.RFC3339, v); err == nil {
		return t
	}
	if t, err := time.Parse("2006-01-02", v); err == nil {
		return t
	}
	return def
}

func parseIntParam(r *http.Request, key string, def, min, max int32) int32 {
	v := r.URL.Query().Get(key)
	if v == "" {
		return def
	}
	n, err := strconv.Atoi(v)
	if err != nil {
		return def
	}
	switch {
	case int32(n) < min:
		return min
	case int32(n) > max:
		return max
	default:
		return int32(n)
	}
}
