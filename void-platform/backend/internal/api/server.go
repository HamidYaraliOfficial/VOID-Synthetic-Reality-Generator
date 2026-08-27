// Package api implements the VOID API Server: REST endpoints for Universe,
// Schema, Behavior, Scenario, Entity, Snapshot, Export, Scheduler and AI
// Assistant operations, an authenticated WebSocket for real-time metrics,
// plus Authentication, RBAC, Rate Limiting and Audit Logging middleware —
// so any external test tool can drive VOID purely over HTTP/WS.
package api

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"sync"
	"time"

	"void-platform/backend/internal/ai"
	"void-platform/backend/internal/plugin"
	"void-platform/backend/internal/scheduler"
	"void-platform/backend/internal/security"
	"void-platform/backend/internal/simulation"
)

// Server holds every piece of shared, process-wide state the HTTP layer
// needs: the universe registry, security primitives and the scheduler.
type Server struct {
	mu        sync.RWMutex
	universes map[string]*UniverseHandle

	Tokens      *security.TokenIssuer
	RateLimiter *security.RateLimiter
	Audit       *security.AuditLog
	Scheduler   *scheduler.Scheduler
	AI          *ai.Assistant
	Plugins     *plugin.Registry

	AllowedOrigins []string
}

// UniverseHandle bundles a running Universe with its Engine controller.
type UniverseHandle struct {
	Universe *simulation.Universe
	Engine   *simulation.Engine
}

func NewServer(jwtSecret string) *Server {
	return &Server{
		universes:      map[string]*UniverseHandle{},
		Tokens:         security.NewTokenIssuer(jwtSecret),
		RateLimiter:    security.NewRateLimiter(50, 100),
		Audit:          security.NewAuditLog(20000),
		Scheduler:      scheduler.New(),
		AI:             ai.New(),
		Plugins:        plugin.Global(),
		AllowedOrigins: []string{"*"},
	}
}

func (s *Server) getHandle(id string) (*UniverseHandle, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	h, ok := s.universes[id]
	return h, ok
}

func (s *Server) putHandle(h *UniverseHandle) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.universes[h.Universe.ID] = h
}

func (s *Server) listHandles() []*UniverseHandle {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make([]*UniverseHandle, 0, len(s.universes))
	for _, h := range s.universes {
		out = append(out, h)
	}
	return out
}

// Routes builds the full HTTP route table using Go 1.22's method+pattern
// ServeMux, wrapped with logging/CORS/auth/rate-limit middleware.
func (s *Server) Routes() http.Handler {
	mux := http.NewServeMux()

	mux.HandleFunc("GET /api/health", s.handleHealth)
	mux.HandleFunc("POST /api/auth/login", s.handleLogin)

	mux.Handle("POST /api/universes", s.protect(security.PermSimulationWrite, http.HandlerFunc(s.handleCreateUniverse)))
	mux.Handle("GET /api/universes", s.protect(security.PermSimulationRead, http.HandlerFunc(s.handleListUniverses)))
	mux.Handle("GET /api/universes/{id}", s.protect(security.PermSimulationRead, http.HandlerFunc(s.handleGetUniverse)))

	mux.Handle("POST /api/universes/{id}/schemas", s.protect(security.PermScenarioWrite, http.HandlerFunc(s.handleAddSchema)))
	mux.Handle("GET /api/universes/{id}/schemas", s.protect(security.PermSimulationRead, http.HandlerFunc(s.handleListSchemas)))

	mux.Handle("POST /api/universes/{id}/behaviors", s.protect(security.PermScenarioWrite, http.HandlerFunc(s.handleAddBehavior)))

	mux.Handle("POST /api/universes/{id}/entities/spawn", s.protect(security.PermSimulationWrite, http.HandlerFunc(s.handleSpawn)))
	mux.Handle("GET /api/universes/{id}/entities", s.protect(security.PermSimulationRead, http.HandlerFunc(s.handleListEntities)))

	mux.Handle("POST /api/universes/{id}/network/nodes", s.protect(security.PermScenarioWrite, http.HandlerFunc(s.handleAddNode)))
	mux.Handle("POST /api/universes/{id}/network/links", s.protect(security.PermScenarioWrite, http.HandlerFunc(s.handleAddLink)))
	mux.Handle("GET /api/universes/{id}/network", s.protect(security.PermSimulationRead, http.HandlerFunc(s.handleGetNetwork)))

	mux.Handle("POST /api/universes/{id}/scenario/run", s.protect(security.PermSimulationWrite, http.HandlerFunc(s.handleRunScenario)))
	mux.Handle("POST /api/universes/{id}/scenario/pause", s.protect(security.PermSimulationWrite, http.HandlerFunc(s.handlePause)))
	mux.Handle("POST /api/universes/{id}/scenario/resume", s.protect(security.PermSimulationWrite, http.HandlerFunc(s.handleResume)))
	mux.Handle("POST /api/universes/{id}/scenario/stop", s.protect(security.PermSimulationWrite, http.HandlerFunc(s.handleStop)))

	mux.Handle("POST /api/universes/{id}/chaos", s.protect(security.PermSimulationWrite, http.HandlerFunc(s.handleChaos)))

	mux.Handle("POST /api/universes/{id}/snapshot", s.protect(security.PermSimulationWrite, http.HandlerFunc(s.handleSnapshot)))
	mux.Handle("POST /api/universes/{id}/snapshot/load", s.protect(security.PermSimulationWrite, http.HandlerFunc(s.handleSnapshotLoad)))
	mux.Handle("GET /api/universes/{id}/snapshots", s.protect(security.PermSimulationRead, http.HandlerFunc(s.handleListSnapshots)))

	mux.Handle("POST /api/universes/{id}/export", s.protect(security.PermExport, http.HandlerFunc(s.handleExport)))

	mux.Handle("GET /api/universes/{id}/metrics", s.protect(security.PermSimulationRead, http.HandlerFunc(s.handleMetrics)))
	mux.Handle("GET /api/universes/{id}/console", s.protect(security.PermSimulationRead, http.HandlerFunc(s.handleConsole)))
	mux.HandleFunc("GET /api/metrics/prometheus", s.handlePrometheus)

	mux.Handle("POST /api/experiments/diff", s.protect(security.PermSimulationRead, http.HandlerFunc(s.handleDiff)))

	mux.Handle("GET /api/scheduler/hours", s.protect(security.PermSimulationRead, http.HandlerFunc(s.handleGetHours)))
	mux.Handle("POST /api/scheduler/hours", s.protect(security.PermScenarioWrite, http.HandlerFunc(s.handleSetHours)))
	mux.Handle("GET /api/scheduler/status", s.protect(security.PermSimulationRead, http.HandlerFunc(s.handleSchedulerStatus)))
	mux.Handle("POST /api/scheduler/runs", s.protect(security.PermScenarioWrite, http.HandlerFunc(s.handleAddScheduledRun)))
	mux.Handle("GET /api/scheduler/runs", s.protect(security.PermSimulationRead, http.HandlerFunc(s.handleListScheduledRuns)))

	mux.Handle("POST /api/ai/generate", s.protect(security.PermScenarioWrite, http.HandlerFunc(s.handleAIGenerate)))
	mux.Handle("POST /api/ai/analyze", s.protect(security.PermSimulationRead, http.HandlerFunc(s.handleAIAnalyze)))

	mux.Handle("GET /api/plugins", s.protect(security.PermSimulationRead, http.HandlerFunc(s.handleListPlugins)))
	mux.Handle("GET /api/audit", s.protect(security.PermAdmin, http.HandlerFunc(s.handleAudit)))
	mux.Handle("GET /api/templates", s.protect(security.PermSimulationRead, http.HandlerFunc(s.handleListTemplates)))

	mux.Handle("GET /api/ws/metrics", s.protect(security.PermSimulationRead, http.HandlerFunc(s.handleWSMetrics)))

	return s.withMiddleware(mux)
}

// --- shared JSON helpers ----------------------------------------------------

func writeJSON(w http.ResponseWriter, status int, v interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

func writeError(w http.ResponseWriter, status int, err error) {
	writeJSON(w, status, map[string]string{"error": err.Error()})
}

func decodeJSON(r *http.Request, v interface{}) error {
	defer r.Body.Close()
	dec := json.NewDecoder(r.Body)
	dec.DisallowUnknownFields()
	return dec.Decode(v)
}

func (s *Server) handleHealth(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]interface{}{
		"status": "ok", "time": time.Now(), "service": "void-api",
	})
}

func (s *Server) handleLogin(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Username string `json:"username"`
		Role     string `json:"role"`
	}
	if err := decodeJSON(r, &body); err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	if body.Username == "" {
		writeError(w, http.StatusBadRequest, fmt.Errorf("username is required"))
		return
	}
	role := body.Role
	switch role {
	case security.RoleAdmin, security.RoleEngineer, security.RoleViewer:
	default:
		role = security.RoleEngineer
	}
	token, err := s.Tokens.Issue(body.Username, []string{role}, 12*time.Hour)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	s.Audit.Record(body.Username, "auth.login", "", "role="+role)
	writeJSON(w, http.StatusOK, map[string]string{"token": token, "role": role})
}

// Serve is a convenience entry point used by cmd/api.
func Serve(addr, jwtSecret string) error {
	srv := NewServer(jwtSecret)
	handler := srv.Routes()
	log.Printf("VOID API listening on %s", addr)
	httpServer := &http.Server{
		Addr:              addr,
		Handler:           handler,
		ReadHeaderTimeout: 10 * time.Second,
	}
	return httpServer.ListenAndServe()
}

var _ = context.Background
