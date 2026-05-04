// Package searchapi exposes a tiny HTTP /search endpoint that the web app calls.
package searchapi

import (
	"encoding/json"
	"errors"
	"net/http"
	"strconv"

	idxos "github.com/imgacademy/ecosystem-technical-interview/apps/indexer/internal/opensearch"
)

type Server struct {
	os *idxos.Client
}

func NewServer(os *idxos.Client) *Server { return &Server{os: os} }

func (s *Server) Routes() *http.ServeMux {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /search", s.handleSearch)
	mux.HandleFunc("GET /healthz", func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte("ok"))
	})
	return mux
}

type searchResponse struct {
	Hits []idxos.SearchHit `json:"hits"`
}

func (s *Server) handleSearch(w http.ResponseWriter, r *http.Request) {
	limit := 50
	if v := r.URL.Query().Get("limit"); v != "" {
		n, err := strconv.Atoi(v)
		if err != nil || n < 1 || n > 200 {
			writeErr(w, http.StatusBadRequest, errors.New("limit must be 1..200"))
			return
		}
		limit = n
	}
	hits, err := s.os.SearchCamps(r.Context(), idxos.SearchQuery{
		Q:          r.URL.Query().Get("q"),
		Sport:      r.URL.Query().Get("sport"),
		SkillLevel: r.URL.Query().Get("skill_level"),
		Limit:      limit,
	})
	if err != nil {
		writeErr(w, http.StatusInternalServerError, err)
		return
	}
	writeJSON(w, http.StatusOK, searchResponse{Hits: hits})
}

func writeJSON(w http.ResponseWriter, code int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	_ = json.NewEncoder(w).Encode(v)
}

func writeErr(w http.ResponseWriter, code int, err error) {
	writeJSON(w, code, map[string]string{"error": err.Error()})
}
