package api

import (
	"net/http"

	"void-platform/backend/internal/ai"
)

func (s *Server) handleAIGenerate(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Prompt string `json:"prompt"`
	}
	if err := decodeJSON(r, &body); err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	cfg, err := s.AI.Interpret(r.Context(), body.Prompt)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	s.Audit.Record(claimsFrom(r).Subject, "ai.generate", "", body.Prompt)
	writeJSON(w, http.StatusOK, cfg)
}

func (s *Server) handleAIAnalyze(w http.ResponseWriter, r *http.Request) {
	var input ai.AnalyzeInput
	if err := decodeJSON(r, &input); err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	findings := s.AI.Analyze(input)
	writeJSON(w, http.StatusOK, map[string]interface{}{"findings": findings})
}
