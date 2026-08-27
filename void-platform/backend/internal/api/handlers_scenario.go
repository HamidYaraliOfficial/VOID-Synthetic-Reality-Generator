package api

import (
	"fmt"
	"net/http"

	"void-platform/backend/internal/chaos"
	"void-platform/backend/internal/network"
	"void-platform/backend/internal/scenario"
)

func (s *Server) handleAddNode(w http.ResponseWriter, r *http.Request) {
	h, ok := s.getHandle(r.PathValue("id"))
	if !ok {
		writeError(w, http.StatusNotFound, fmt.Errorf("universe not found"))
		return
	}
	var n network.Node
	if err := decodeJSON(r, &n); err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	h.Universe.Network.AddNode(&n)
	writeJSON(w, http.StatusCreated, n)
}

func (s *Server) handleAddLink(w http.ResponseWriter, r *http.Request) {
	h, ok := s.getHandle(r.PathValue("id"))
	if !ok {
		writeError(w, http.StatusNotFound, fmt.Errorf("universe not found"))
		return
	}
	var l network.Link
	if err := decodeJSON(r, &l); err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	h.Universe.Network.AddLink(&l)
	writeJSON(w, http.StatusCreated, l)
}

func (s *Server) handleGetNetwork(w http.ResponseWriter, r *http.Request) {
	h, ok := s.getHandle(r.PathValue("id"))
	if !ok {
		writeError(w, http.StatusNotFound, fmt.Errorf("universe not found"))
		return
	}
	nodes, links := h.Universe.Network.Snapshot()
	writeJSON(w, http.StatusOK, map[string]interface{}{"nodes": nodes, "links": links})
}

func (s *Server) handleRunScenario(w http.ResponseWriter, r *http.Request) {
	h, ok := s.getHandle(r.PathValue("id"))
	if !ok {
		writeError(w, http.StatusNotFound, fmt.Errorf("universe not found"))
		return
	}
	var sc scenario.Scenario
	if err := decodeJSON(r, &sc); err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	if sc.Name == "" {
		writeError(w, http.StatusBadRequest, fmt.Errorf("scenario name is required"))
		return
	}
	if err := h.Engine.RunScenario(r.Context(), &sc); err != nil {
		writeError(w, http.StatusConflict, err)
		return
	}
	s.Audit.Record(claimsFrom(r).Subject, "scenario.run", h.Universe.ID, sc.Name)
	writeJSON(w, http.StatusAccepted, map[string]interface{}{
		"status": "running", "totalDuration": sc.TotalDuration().String(),
	})
}

func (s *Server) handlePause(w http.ResponseWriter, r *http.Request) {
	h, ok := s.getHandle(r.PathValue("id"))
	if !ok {
		writeError(w, http.StatusNotFound, fmt.Errorf("universe not found"))
		return
	}
	h.Engine.Pause()
	s.Audit.Record(claimsFrom(r).Subject, "scenario.pause", h.Universe.ID, "")
	writeJSON(w, http.StatusOK, map[string]string{"status": string(h.Universe.Status)})
}

func (s *Server) handleResume(w http.ResponseWriter, r *http.Request) {
	h, ok := s.getHandle(r.PathValue("id"))
	if !ok {
		writeError(w, http.StatusNotFound, fmt.Errorf("universe not found"))
		return
	}
	h.Engine.Resume()
	s.Audit.Record(claimsFrom(r).Subject, "scenario.resume", h.Universe.ID, "")
	writeJSON(w, http.StatusOK, map[string]string{"status": string(h.Universe.Status)})
}

func (s *Server) handleStop(w http.ResponseWriter, r *http.Request) {
	h, ok := s.getHandle(r.PathValue("id"))
	if !ok {
		writeError(w, http.StatusNotFound, fmt.Errorf("universe not found"))
		return
	}
	h.Engine.Stop()
	s.Audit.Record(claimsFrom(r).Subject, "scenario.stop", h.Universe.ID, "")
	writeJSON(w, http.StatusOK, map[string]string{"status": string(h.Universe.Status)})
}

func (s *Server) handleChaos(w http.ResponseWriter, r *http.Request) {
	h, ok := s.getHandle(r.PathValue("id"))
	if !ok {
		writeError(w, http.StatusNotFound, fmt.Errorf("universe not found"))
		return
	}
	var f chaos.Fault
	if err := decodeJSON(r, &f); err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	if f.ID == "" {
		f.ID = fmt.Sprintf("fault-%d", len(h.Universe.Chaos.Active)+1)
	}
	h.Universe.Chaos.Inject(&f)
	s.Audit.Record(claimsFrom(r).Subject, "chaos.inject", h.Universe.ID, string(f.Kind))
	writeJSON(w, http.StatusAccepted, f)
}
