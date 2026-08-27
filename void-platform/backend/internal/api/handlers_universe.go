package api

import (
	"fmt"
	"net/http"
	"time"

	"void-platform/backend/internal/behavior"
	"void-platform/backend/internal/entity"
	"void-platform/backend/internal/generator"
	"void-platform/backend/internal/security"
	"void-platform/backend/internal/simulation"
)

func (s *Server) handleCreateUniverse(w http.ResponseWriter, r *http.Request) {
	var body struct {
		ID   string `json:"id"`
		Name string `json:"name"`
		Seed int64  `json:"seed"`
	}
	if err := decodeJSON(r, &body); err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	if body.ID == "" {
		body.ID = fmt.Sprintf("universe-%d", time.Now().UnixNano())
	}
	if err := security.ValidateInput(body.ID); err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	if _, exists := s.getHandle(body.ID); exists {
		writeError(w, http.StatusConflict, fmt.Errorf("universe %q already exists", body.ID))
		return
	}
	u := simulation.NewUniverse(body.ID, body.Name, body.Seed)
	handle := &UniverseHandle{Universe: u, Engine: simulation.NewEngine(u)}
	s.putHandle(handle)
	s.Audit.Record(claimsFrom(r).Subject, "universe.create", body.ID, body.Name)
	writeJSON(w, http.StatusCreated, universeView(u))
}

func universeView(u *simulation.Universe) map[string]interface{} {
	return map[string]interface{}{
		"id": u.ID, "name": u.Name, "seed": u.Seed, "status": u.Status,
		"entityCounts": u.EntityCounts(),
	}
}

func (s *Server) handleListUniverses(w http.ResponseWriter, r *http.Request) {
	handles := s.listHandles()
	out := make([]map[string]interface{}, 0, len(handles))
	for _, h := range handles {
		out = append(out, universeView(h.Universe))
	}
	writeJSON(w, http.StatusOK, out)
}

func (s *Server) handleGetUniverse(w http.ResponseWriter, r *http.Request) {
	h, ok := s.getHandle(r.PathValue("id"))
	if !ok {
		writeError(w, http.StatusNotFound, fmt.Errorf("universe not found"))
		return
	}
	writeJSON(w, http.StatusOK, universeView(h.Universe))
}

func (s *Server) handleAddSchema(w http.ResponseWriter, r *http.Request) {
	h, ok := s.getHandle(r.PathValue("id"))
	if !ok {
		writeError(w, http.StatusNotFound, fmt.Errorf("universe not found"))
		return
	}
	var sch entity.Schema
	if err := decodeJSON(r, &sch); err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	if err := h.Universe.AddSchema(&sch); err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	s.Audit.Record(claimsFrom(r).Subject, "schema.add", h.Universe.ID, sch.Name)
	writeJSON(w, http.StatusCreated, sch)
}

func (s *Server) handleListSchemas(w http.ResponseWriter, r *http.Request) {
	h, ok := s.getHandle(r.PathValue("id"))
	if !ok {
		writeError(w, http.StatusNotFound, fmt.Errorf("universe not found"))
		return
	}
	writeJSON(w, http.StatusOK, h.Universe.SchemasSnapshot())
}

func (s *Server) handleAddBehavior(w http.ResponseWriter, r *http.Request) {
	h, ok := s.getHandle(r.PathValue("id"))
	if !ok {
		writeError(w, http.StatusNotFound, fmt.Errorf("universe not found"))
		return
	}
	var graph behavior.Graph
	if err := decodeJSON(r, &graph); err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	if graph.Name == "" || graph.Entry == "" {
		writeError(w, http.StatusBadRequest, fmt.Errorf("behavior graph requires name and entry"))
		return
	}
	h.Universe.AddBehavior(&graph)
	s.Audit.Record(claimsFrom(r).Subject, "behavior.add", h.Universe.ID, graph.Name)
	writeJSON(w, http.StatusCreated, graph)
}

func (s *Server) handleSpawn(w http.ResponseWriter, r *http.Request) {
	h, ok := s.getHandle(r.PathValue("id"))
	if !ok {
		writeError(w, http.StatusNotFound, fmt.Errorf("universe not found"))
		return
	}
	var body struct {
		Type  string `json:"type"`
		Count int    `json:"count"`
	}
	if err := decodeJSON(r, &body); err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	if body.Count <= 0 || body.Count > 5_000_000 {
		writeError(w, http.StatusBadRequest, fmt.Errorf("count must be between 1 and 5,000,000"))
		return
	}
	entities, err := h.Universe.SpawnEntities(body.Type, body.Count, nil)
	if err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	s.Audit.Record(claimsFrom(r).Subject, "entities.spawn", h.Universe.ID, fmt.Sprintf("%s x%d", body.Type, body.Count))
	writeJSON(w, http.StatusOK, map[string]interface{}{"spawned": len(entities), "type": body.Type})
}

func (s *Server) handleListEntities(w http.ResponseWriter, r *http.Request) {
	h, ok := s.getHandle(r.PathValue("id"))
	if !ok {
		writeError(w, http.StatusNotFound, fmt.Errorf("universe not found"))
		return
	}
	typeName := r.URL.Query().Get("type")
	if typeName == "" {
		writeJSON(w, http.StatusOK, h.Universe.EntityCounts())
		return
	}
	limit := 100
	if v := r.URL.Query().Get("limit"); v != "" {
		fmt.Sscanf(v, "%d", &limit)
	}
	all := h.Universe.Collection(typeName).All()
	if limit > 0 && limit < len(all) {
		all = all[:limit]
	}
	writeJSON(w, http.StatusOK, all)
}

// keep generator import used if handlers evolve to reference it directly
var _ = generator.NewEngine
