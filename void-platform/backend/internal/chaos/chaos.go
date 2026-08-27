// Package chaos implements the Failure & Chaos Simulation Engine. Every
// fault here acts ONLY on the synthetic network.Topology / in-memory
// simulation state that belongs to a Universe — never on a real host,
// process, or third-party network — so "chaos testing" in VOID means
// injecting controlled synthetic fault conditions to study how a *modelled*
// system reacts, not attacking anything real.
package chaos

import "time"

// FaultKind enumerates the controlled fault catalogue exposed to Scenario
// timeline "chaos" actions.
type FaultKind string

const (
	FaultServiceFailure   FaultKind = "service_failure"
	FaultTimeout          FaultKind = "timeout"
	FaultHighLatency      FaultKind = "high_latency"
	FaultDBUnavailable    FaultKind = "database_unavailable"
	FaultQueueBacklog     FaultKind = "queue_backlog"
	FaultMemoryPressure   FaultKind = "memory_pressure"
	FaultCPUPressure      FaultKind = "cpu_pressure"
	FaultPacketLoss       FaultKind = "packet_loss"
	FaultPartialFailure   FaultKind = "partial_failure"
)

// Target is the minimal surface a synthetic network/topology must expose so
// the chaos engine can apply/undo a fault without importing the network
// package directly (keeps package dependencies one-directional).
type Target interface {
	SetNodeAvailability(id string, available bool)
	SetLatency(from, to string, ms float64)
	SetPacketLoss(from, to string, ratio float64)
}

// Fault is one active, time-boxed chaos injection.
type Fault struct {
	ID        string                 `json:"id"`
	Kind      FaultKind              `json:"kind"`
	NodeID    string                 `json:"nodeId,omitempty"`
	LinkFrom  string                 `json:"linkFrom,omitempty"`
	LinkTo    string                 `json:"linkTo,omitempty"`
	Params    map[string]interface{} `json:"params,omitempty"`
	StartedAt time.Time              `json:"startedAt"`
	Duration  time.Duration          `json:"duration"`
	Active    bool                   `json:"active"`
}

// Engine tracks and applies/reverts chaos Faults against a Target.
type Engine struct {
	Target Target
	Active map[string]*Fault
}

func NewEngine(target Target) *Engine {
	return &Engine{Target: target, Active: map[string]*Fault{}}
}

// Inject applies a fault immediately and schedules its automatic reversal
// after Duration via a background goroutine (safe: only mutates the
// synthetic Target).
func (e *Engine) Inject(f *Fault) {
	f.StartedAt = time.Now()
	f.Active = true
	e.Active[f.ID] = f

	switch f.Kind {
	case FaultServiceFailure, FaultDBUnavailable:
		e.Target.SetNodeAvailability(f.NodeID, false)
	case FaultHighLatency, FaultTimeout:
		ms := 500.0
		if v, ok := f.Params["latencyMs"].(float64); ok {
			ms = v
		}
		e.Target.SetLatency(f.LinkFrom, f.LinkTo, ms)
	case FaultPacketLoss:
		ratio := 0.2
		if v, ok := f.Params["ratio"].(float64); ok {
			ratio = v
		}
		e.Target.SetPacketLoss(f.LinkFrom, f.LinkTo, ratio)
	case FaultPartialFailure:
		ratio := 0.3
		if v, ok := f.Params["ratio"].(float64); ok {
			ratio = v
		}
		e.Target.SetPacketLoss(f.LinkFrom, f.LinkTo, ratio)
	case FaultQueueBacklog, FaultMemoryPressure, FaultCPUPressure:
		// Resource-pressure faults are surfaced as metrics-level signals by
		// the simulation engine (see internal/simulation), not topology
		// mutations; recorded here for the UI status timeline regardless.
	}

	if f.Duration > 0 {
		go func() {
			time.Sleep(f.Duration)
			e.Revert(f.ID)
		}()
	}
}

// Revert undoes a fault's effect and marks it inactive.
func (e *Engine) Revert(id string) {
	f, ok := e.Active[id]
	if !ok || !f.Active {
		return
	}
	switch f.Kind {
	case FaultServiceFailure, FaultDBUnavailable:
		e.Target.SetNodeAvailability(f.NodeID, true)
	case FaultHighLatency, FaultTimeout:
		e.Target.SetLatency(f.LinkFrom, f.LinkTo, 0)
	case FaultPacketLoss, FaultPartialFailure:
		e.Target.SetPacketLoss(f.LinkFrom, f.LinkTo, 0)
	}
	f.Active = false
}

// List returns all faults injected so far (active and reverted) for the
// chaos timeline view in the dashboard.
func (e *Engine) List() []*Fault {
	out := make([]*Fault, 0, len(e.Active))
	for _, f := range e.Active {
		out = append(out, f)
	}
	return out
}
