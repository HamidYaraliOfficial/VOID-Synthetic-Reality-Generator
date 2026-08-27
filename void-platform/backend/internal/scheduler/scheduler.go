// Package scheduler implements simulation scheduling AND the user-configurable
// "business hours" feature: the user enters opening/closing times for each
// day of the week (fully editable, nothing hard-coded), and the platform
// works out, at any given moment, whether it is currently "open", and if
// not, exactly how long until the next status change — used both for
// gating scheduled Simulation Runs to an operating window and for the
// SchedulerPanel status widget in the UI.
package scheduler

import (
	"fmt"
	"sort"
	"time"
)

// Window is one open interval within a single day, expressed as "HH:MM" in
// 24h format. A day can have more than one Window (e.g. split shifts).
type Window struct {
	Start string `json:"start"` // "09:00"
	End   string `json:"end"`   // "18:00"
}

// BusinessHours is a fully user-editable weekly schedule: zero windows on a
// weekday means "closed all day"; multiple windows are allowed per day.
type BusinessHours struct {
	Timezone string             `json:"timezone"` // IANA name, e.g. "Europe/Berlin"
	Days     map[string][]Window `json:"days"`     // "monday".."sunday" -> windows
}

// DefaultBusinessHours returns an empty (fully closed) schedule; the caller
// (user, via the UI or API) fills in every day/window themselves — VOID
// never assumes a default operating schedule on the user's behalf.
func DefaultBusinessHours() *BusinessHours {
	return &BusinessHours{
		Timezone: "UTC",
		Days:     map[string][]Window{},
	}
}

var weekdayNames = []string{"sunday", "monday", "tuesday", "wednesday", "thursday", "friday", "saturday"}

// Status is the computed open/closed state at a point in time, including how
// long remains until the next transition (open->closed or closed->open).
type Status struct {
	Now            time.Time     `json:"now"`
	IsOpen         bool          `json:"isOpen"`
	NextChangeAt   time.Time     `json:"nextChangeAt"`
	TimeUntilNext  time.Duration `json:"timeUntilNext"`
	TimeUntilNextH string        `json:"timeUntilNextHuman"`
	CurrentWindow  *Window       `json:"currentWindow,omitempty"`
}

// StatusAt computes open/closed + time-to-next-change for the given instant,
// scanning up to 8 days forward so it always finds the next transition even
// for a schedule that is closed for several consecutive days.
func (bh *BusinessHours) StatusAt(now time.Time) (Status, error) {
	loc := time.UTC
	if bh.Timezone != "" {
		l, err := time.LoadLocation(bh.Timezone)
		if err != nil {
			return Status{}, fmt.Errorf("scheduler: invalid timezone %q: %w", bh.Timezone, err)
		}
		loc = l
	}
	local := now.In(loc)

	// Check "is open right now" against today's windows.
	todayName := weekdayNames[int(local.Weekday())]
	dayStart := time.Date(local.Year(), local.Month(), local.Day(), 0, 0, 0, 0, loc)

	type absWindow struct {
		start, end time.Time
	}
	var todaysAbsWindows []absWindow
	for _, w := range bh.Days[todayName] {
		s, e, err := absoluteWindow(dayStart, w)
		if err != nil {
			continue
		}
		todaysAbsWindows = append(todaysAbsWindows, absWindow{s, e})
	}
	sort.Slice(todaysAbsWindows, func(i, j int) bool { return todaysAbsWindows[i].start.Before(todaysAbsWindows[j].start) })

	for _, aw := range todaysAbsWindows {
		if !local.Before(aw.start) && local.Before(aw.end) {
			until := aw.end.Sub(local)
			return Status{
				Now: now, IsOpen: true, NextChangeAt: aw.end,
				TimeUntilNext: until, TimeUntilNextH: humanDuration(until),
				CurrentWindow: &Window{Start: aw.start.Format("15:04"), End: aw.end.Format("15:04")},
			}, nil
		}
	}

	// Not open now: find the next future window start within the next 8 days.
	for offset := 0; offset < 8; offset++ {
		d := dayStart.AddDate(0, 0, offset)
		name := weekdayNames[int(d.Weekday())]
		var windows []absWindow
		for _, w := range bh.Days[name] {
			s, e, err := absoluteWindow(d, w)
			if err != nil {
				continue
			}
			windows = append(windows, absWindow{s, e})
		}
		sort.Slice(windows, func(i, j int) bool { return windows[i].start.Before(windows[j].start) })
		for _, aw := range windows {
			if aw.start.After(local) {
				until := aw.start.Sub(local)
				return Status{
					Now: now, IsOpen: false, NextChangeAt: aw.start,
					TimeUntilNext: until, TimeUntilNextH: humanDuration(until),
				}, nil
			}
		}
	}
	// No windows configured at all in the lookahead: report "closed indefinitely".
	return Status{Now: now, IsOpen: false}, nil
}

func absoluteWindow(dayStart time.Time, w Window) (time.Time, time.Time, error) {
	sh, sm, err := parseHHMM(w.Start)
	if err != nil {
		return time.Time{}, time.Time{}, err
	}
	eh, em, err := parseHHMM(w.End)
	if err != nil {
		return time.Time{}, time.Time{}, err
	}
	start := dayStart.Add(time.Duration(sh)*time.Hour + time.Duration(sm)*time.Minute)
	end := dayStart.Add(time.Duration(eh)*time.Hour + time.Duration(em)*time.Minute)
	if !end.After(start) {
		end = end.Add(24 * time.Hour) // overnight window, e.g. 22:00 -> 02:00
	}
	return start, end, nil
}

func parseHHMM(s string) (int, int, error) {
	var h, m int
	_, err := fmt.Sscanf(s, "%d:%d", &h, &m)
	if err != nil || h < 0 || h > 23 || m < 0 || m > 59 {
		return 0, 0, fmt.Errorf("scheduler: invalid time %q, expected HH:MM", s)
	}
	return h, m, nil
}

func humanDuration(d time.Duration) string {
	if d < 0 {
		d = 0
	}
	h := int(d.Hours())
	m := int(d.Minutes()) % 60
	sSec := int(d.Seconds()) % 60
	if h > 0 {
		return fmt.Sprintf("%dh %dm", h, m)
	}
	if m > 0 {
		return fmt.Sprintf("%dm %ds", m, sSec)
	}
	return fmt.Sprintf("%ds", sSec)
}

// --- Simulation run scheduling ---------------------------------------------

// ScheduledRun is a Batch Experiment / recurring Simulation entry: run a
// named Scenario either once at RunAt, or repeatedly every Every, optionally
// gated to only fire while BusinessHours reports IsOpen.
type ScheduledRun struct {
	ID           string        `json:"id"`
	ScenarioID   string        `json:"scenarioId"`
	RunAt        *time.Time    `json:"runAt,omitempty"`
	Every        time.Duration `json:"every,omitempty"`
	OnlyInHours  bool          `json:"onlyInHours"`
	LastRunAt    *time.Time    `json:"lastRunAt,omitempty"`
}

// Scheduler holds business hours plus a list of scheduled runs and decides,
// on each Tick, which runs are due.
type Scheduler struct {
	Hours *BusinessHours
	Runs  []*ScheduledRun
}

func New() *Scheduler {
	return &Scheduler{Hours: DefaultBusinessHours()}
}

// DueRuns returns every ScheduledRun that should fire at instant `now`,
// respecting BusinessHours gating and Every-interval bookkeeping.
func (s *Scheduler) DueRuns(now time.Time) []*ScheduledRun {
	var due []*ScheduledRun
	var status Status
	if s.Hours != nil {
		status, _ = s.Hours.StatusAt(now)
	}
	for _, r := range s.Runs {
		if r.OnlyInHours && !status.IsOpen {
			continue
		}
		if r.RunAt != nil && r.LastRunAt == nil && !now.Before(*r.RunAt) {
			due = append(due, r)
			continue
		}
		if r.Every > 0 {
			if r.LastRunAt == nil || now.Sub(*r.LastRunAt) >= r.Every {
				due = append(due, r)
			}
		}
	}
	return due
}
