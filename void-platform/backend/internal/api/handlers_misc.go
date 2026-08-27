package api

import (
	"fmt"
	"net/http"

	"void-platform/backend/internal/simulation"
)

func (s *Server) handleMetrics(w http.ResponseWriter, r *http.Request) {
	h, ok := s.getHandle(r.PathValue("id"))
	if !ok {
		writeError(w, http.StatusNotFound, fmt.Errorf("universe not found"))
		return
	}
	snap := h.Universe.Metrics.Snapshot()
	snap.Gauges["event_bus.queue_len"] = float64(h.Universe.EventBus.Stats().QueueLen)
	snap.Gauges["event_bus.processed"] = float64(h.Universe.EventBus.Stats().Processed)
	snap.Gauges["event_bus.dropped"] = float64(h.Universe.EventBus.Stats().Dropped)
	writeJSON(w, http.StatusOK, snap)
}

func (s *Server) handleConsole(w http.ResponseWriter, r *http.Request) {
	h, ok := s.getHandle(r.PathValue("id"))
	if !ok {
		writeError(w, http.StatusNotFound, fmt.Errorf("universe not found"))
		return
	}
	writeJSON(w, http.StatusOK, h.Universe.ConsoleTail(500))
}

func (s *Server) handlePrometheus(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/plain; version=0.0.4")
	total := ""
	for _, h := range s.listHandles() {
		total += h.Universe.Metrics.PrometheusText()
	}
	_, _ = w.Write([]byte(total))
}

func (s *Server) handleDiff(w http.ResponseWriter, r *http.Request) {
	var body struct {
		A map[string]float64 `json:"a"`
		B map[string]float64 `json:"b"`
	}
	if err := decodeJSON(r, &body); err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	writeJSON(w, http.StatusOK, simulation.DiffMetrics(body.A, body.B))
}

func (s *Server) handleListPlugins(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, s.Plugins.List())
}

func (s *Server) handleAudit(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, s.Audit.Recent(500))
}

func (s *Server) handleListTemplates(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, builtinTemplates())
}
