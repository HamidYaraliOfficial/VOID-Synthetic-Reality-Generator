package scheduler

import (
	"testing"
	"time"
)

func TestStatusAtOpenWindow(t *testing.T) {
	bh := &BusinessHours{
		Timezone: "UTC",
		Days: map[string][]Window{
			"monday": {{Start: "09:00", End: "18:00"}},
		},
	}
	// Monday 2024-01-01 12:00 UTC
	now := time.Date(2024, 1, 1, 12, 0, 0, 0, time.UTC)
	status, err := bh.StatusAt(now)
	if err != nil {
		t.Fatal(err)
	}
	if !status.IsOpen {
		t.Fatalf("expected open at 12:00 within 09:00-18:00 window")
	}
	if status.TimeUntilNext != 6*time.Hour {
		t.Fatalf("expected 6h until close, got %s", status.TimeUntilNext)
	}
}

func TestStatusAtClosedFindsNextWindow(t *testing.T) {
	bh := &BusinessHours{
		Timezone: "UTC",
		Days: map[string][]Window{
			"tuesday": {{Start: "09:00", End: "17:00"}},
		},
	}
	// Monday 2024-01-01 08:00 UTC (before Tuesday's window)
	now := time.Date(2024, 1, 1, 8, 0, 0, 0, time.UTC)
	status, err := bh.StatusAt(now)
	if err != nil {
		t.Fatal(err)
	}
	if status.IsOpen {
		t.Fatalf("expected closed on Monday with only Tuesday configured")
	}
	wantNext := time.Date(2024, 1, 2, 9, 0, 0, 0, time.UTC)
	if !status.NextChangeAt.Equal(wantNext) {
		t.Fatalf("expected next change at %s, got %s", wantNext, status.NextChangeAt)
	}
}

func TestOvernightWindow(t *testing.T) {
	bh := &BusinessHours{
		Timezone: "UTC",
		Days: map[string][]Window{
			"friday": {{Start: "22:00", End: "02:00"}},
		},
	}
	now := time.Date(2024, 1, 5, 23, 0, 0, 0, time.UTC) // Friday 23:00
	status, err := bh.StatusAt(now)
	if err != nil {
		t.Fatal(err)
	}
	if !status.IsOpen {
		t.Fatalf("expected open during overnight window at 23:00")
	}
}
