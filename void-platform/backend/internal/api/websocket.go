package api

import (
	"encoding/json"
	"net/http"
	"time"

	"void-platform/backend/internal/wsutil"
)

// handleWSMetrics upgrades to a WebSocket and pushes a JSON metrics snapshot
// for the requested universe (?universeId=...) twice a second, powering the
// Dashboard Builder's real-time widgets without any client-side polling.
func (s *Server) handleWSMetrics(w http.ResponseWriter, r *http.Request) {
	if !wsutil.SafeUpgradeCheckOrigin(r, s.AllowedOrigins) {
		writeError(w, http.StatusForbidden, errAuth("origin not allowed"))
		return
	}
	universeID := r.URL.Query().Get("universeId")
	h, ok := s.getHandle(universeID)
	if !ok {
		writeError(w, http.StatusNotFound, errAuth("universe not found"))
		return
	}

	conn, err := wsutil.Upgrade(w, r)
	if err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	defer conn.Close()

	// Drain client control frames (e.g. pings) in the background so the
	// connection doesn't stall; exit the handler once the client disconnects.
	closed := make(chan struct{})
	go func() {
		for {
			if _, err := conn.ReadMessage(); err != nil {
				close(closed)
				return
			}
		}
	}()

	ticker := time.NewTicker(500 * time.Millisecond)
	defer ticker.Stop()
	for {
		select {
		case <-closed:
			return
		case <-r.Context().Done():
			return
		case <-ticker.C:
			payload, _ := json.Marshal(map[string]interface{}{
				"universeId": universeID,
				"status":     h.Universe.Status,
				"metrics":    h.Universe.Metrics.Snapshot(),
				"eventBus":   h.Universe.EventBus.Stats(),
				"entityCounts": h.Universe.EntityCounts(),
			})
			if err := conn.WriteText(payload); err != nil {
				return
			}
		}
	}
}
