// Package scenario implements the Scenario Builder + Timeline Editor model:
// a reproducible, time-ordered plan of Spawn / Behavior / Traffic / Chaos /
// Wait actions that the simulation engine executes against a Universe.
package scenario

import "time"

// ActionKind enumerates the step types a Timeline can contain.
type ActionKind string

const (
	ActionSpawn      ActionKind = "spawn"       // create N entities of a type
	ActionAttach     ActionKind = "attach"      // attach a behavior graph to entities
	ActionEmit       ActionKind = "emit"        // emit a one-off event
	ActionLoad       ActionKind = "load"        // start/stop a load generation profile
	ActionChaos      ActionKind = "chaos"       // inject a chaos fault
	ActionWait       ActionKind = "wait"        // pause the timeline
	ActionSnapshot   ActionKind = "snapshot"    // take a state snapshot
	ActionSetNetwork ActionKind = "set_network" // mutate topology latency/loss
)

// Action is one entry on the Timeline, offset from scenario start by At.
type Action struct {
	ID       string                 `json:"id"`
	At       time.Duration          `json:"at"`       // offset from scenario start
	Kind     ActionKind             `json:"kind"`
	Duration time.Duration          `json:"duration,omitempty"` // for load/chaos windows
	Params   map[string]interface{} `json:"params,omitempty"`
}

// Scenario is a full, named, reproducible simulation plan.
type Scenario struct {
	ID          string   `json:"id"`
	Name        string   `json:"name"`
	Description string   `json:"description,omitempty"`
	Seed        int64    `json:"seed"`
	Timeline    []Action `json:"timeline"`
	TimeScale   float64  `json:"timeScale"` // 1 = real time, >1 = accelerated, 0 = deterministic (no sleeping)
}

// Sorted returns the timeline ordered by offset, stable for equal offsets.
func (s *Scenario) Sorted() []Action {
	out := make([]Action, len(s.Timeline))
	copy(out, s.Timeline)
	for i := 1; i < len(out); i++ {
		j := i
		for j > 0 && out[j-1].At > out[j].At {
			out[j-1], out[j] = out[j], out[j-1]
			j--
		}
	}
	return out
}

// TotalDuration returns the offset (plus duration) of the last action, i.e.
// how long the scenario spans end-to-end.
func (s *Scenario) TotalDuration() time.Duration {
	max := time.Duration(0)
	for _, a := range s.Timeline {
		end := a.At + a.Duration
		if end > max {
			max = end
		}
	}
	return max
}
