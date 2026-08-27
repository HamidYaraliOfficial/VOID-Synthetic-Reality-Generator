package api

import (
	"fmt"
	"net/http"

	"void-platform/backend/internal/export"
	"void-platform/backend/internal/security"
	"void-platform/backend/internal/storage"
)

func (s *Server) handleSnapshot(w http.ResponseWriter, r *http.Request) {
	h, ok := s.getHandle(r.PathValue("id"))
	if !ok {
		writeError(w, http.StatusNotFound, fmt.Errorf("universe not found"))
		return
	}
	var body struct {
		Label string `json:"label"`
	}
	_ = decodeJSON(r, &body)
	if body.Label == "" {
		body.Label = h.Universe.ID
	}
	if err := security.ValidateInput(body.Label); err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	meta, err := h.Engine.SaveSnapshot(body.Label)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	s.Audit.Record(claimsFrom(r).Subject, "snapshot.save", h.Universe.ID, body.Label)
	writeJSON(w, http.StatusCreated, meta)
}

func (s *Server) handleSnapshotLoad(w http.ResponseWriter, r *http.Request) {
	h, ok := s.getHandle(r.PathValue("id"))
	if !ok {
		writeError(w, http.StatusNotFound, fmt.Errorf("universe not found"))
		return
	}
	var body struct {
		Path string `json:"path"`
	}
	if err := decodeJSON(r, &body); err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	if err := h.Engine.LoadSnapshot(body.Path); err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	s.Audit.Record(claimsFrom(r).Subject, "snapshot.load", h.Universe.ID, body.Path)
	writeJSON(w, http.StatusOK, universeView(h.Universe))
}

func (s *Server) handleListSnapshots(w http.ResponseWriter, r *http.Request) {
	metas, err := storage.ListSnapshots("data/snapshots")
	if err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	writeJSON(w, http.StatusOK, metas)
}

func (s *Server) handleExport(w http.ResponseWriter, r *http.Request) {
	h, ok := s.getHandle(r.PathValue("id"))
	if !ok {
		writeError(w, http.StatusNotFound, fmt.Errorf("universe not found"))
		return
	}
	var body struct {
		Type   string        `json:"type"`
		Format export.Format `json:"format"`
	}
	if err := decodeJSON(r, &body); err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	if err := security.ValidateInput(body.Type); err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	entities := h.Universe.Collection(body.Type).All()
	records := make([]export.Record, 0, len(entities))
	for _, e := range entities {
		rec := export.Record{"id": e.ID, "type": e.Type, "state": e.State, "createdAt": e.CreatedAt}
		for k, v := range e.Attributes {
			rec[k] = v
		}
		records = append(records, rec)
	}

	ext := string(body.Format)
	if ext == "" {
		ext = "json"
	}
	contentType := map[export.Format]string{
		export.FormatJSON: "application/json", export.FormatJSONL: "application/x-ndjson",
		export.FormatCSV: "text/csv", export.FormatYAML: "application/yaml",
		export.FormatXML: "application/xml", export.FormatSQL: "application/sql",
		export.FormatBinary: "application/octet-stream",
	}[body.Format]
	if contentType == "" {
		contentType = "application/octet-stream"
	}
	w.Header().Set("Content-Type", contentType)
	w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=%s.%s", body.Type, ext))
	if err := export.ToWriter(w, body.Format, body.Type, records); err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	s.Audit.Record(claimsFrom(r).Subject, "export.run", h.Universe.ID, fmt.Sprintf("%s as %s (%d records)", body.Type, ext, len(records)))
}

func snapshotList() ([]interface{}, error) {
	metas, err := storage.ListSnapshots("data/snapshots")
	if err != nil {
		return nil, err
	}
	out := make([]interface{}, len(metas))
	for i, m := range metas {
		out[i] = m
	}
	return out, nil
}
