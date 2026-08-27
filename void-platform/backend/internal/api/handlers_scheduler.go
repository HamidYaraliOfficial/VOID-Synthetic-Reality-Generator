package api

import (
	"fmt"
	"net/http"
	"time"

	"void-platform/backend/internal/scheduler"
)

// handleGetHours returns the currently configured, fully user-editable
// weekly Business Hours schedule.
func (s *Server) handleGetHours(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, s.Scheduler.Hours)
}

// handleSetHours lets the user enter/replace opening hours for every day of
// the week (and the timezone they apply in) — nothing here is hard-coded,
// every window comes from the request body.
func (s *Server) handleSetHours(w http.ResponseWriter, r *http.Request) {
	var hours scheduler.BusinessHours
	if err := decodeJSON(r, &hours); err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	if hours.Timezone == "" {
		hours.Timezone = "UTC"
	}
	if _, err := time.LoadLocation(hours.Timezone); err != nil {
		writeError(w, http.StatusBadRequest, fmt.Errorf("invalid timezone %q", hours.Timezone))
		return
	}
	s.Scheduler.Hours = &hours
	s.Audit.Record(claimsFrom(r).Subject, "scheduler.hours.update", "", hours.Timezone)
	writeJSON(w, http.StatusOK, s.Scheduler.Hours)
}

// handleSchedulerStatus answers: right now, are we open, and — whichever way
// that goes — exactly how long until it next changes.
func (s *Server) handleSchedulerStatus(w http.ResponseWriter, r *http.Request) {
	status, err := s.Scheduler.Hours.StatusAt(time.Now())
	if err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	writeJSON(w, http.StatusOK, status)
}

func (s *Server) handleAddScheduledRun(w http.ResponseWriter, r *http.Request) {
	var run scheduler.ScheduledRun
	if err := decodeJSON(r, &run); err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	if run.ID == "" {
		run.ID = fmt.Sprintf("run-%d", time.Now().UnixNano())
	}
	s.Scheduler.Runs = append(s.Scheduler.Runs, &run)
	s.Audit.Record(claimsFrom(r).Subject, "scheduler.run.add", run.ID, run.ScenarioID)
	writeJSON(w, http.StatusCreated, run)
}

func (s *Server) handleListScheduledRuns(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, s.Scheduler.Runs)
}
