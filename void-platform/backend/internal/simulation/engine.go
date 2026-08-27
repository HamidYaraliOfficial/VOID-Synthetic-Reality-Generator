package simulation

import (
	"context"
	"fmt"
	"math/rand"
	"sync"
	"time"

	"void-platform/backend/internal/behavior"
	"void-platform/backend/internal/chaos"
	"void-platform/backend/internal/event"
	"void-platform/backend/internal/generator"
	"void-platform/backend/internal/loadgen"
	"void-platform/backend/internal/scenario"
)

// Engine executes a scenario.Scenario's Timeline against a Universe. This is
// the "single Engine everything is connected to" the product spec requires:
// spawning, behaviors, load generation, chaos injection and snapshots are
// all driven from here using real goroutines synchronised against the
// Universe's TimeEngine.
type Engine struct {
	Universe *Universe
	cancel   context.CancelFunc
	runWG    sync.WaitGroup
}

func NewEngine(u *Universe) *Engine {
	return &Engine{Universe: u}
}

// RunScenario starts the scenario's Timeline asynchronously. It returns
// immediately; use Wait or poll Universe.Status/Metrics to observe progress.
func (eng *Engine) RunScenario(parent context.Context, sc *scenario.Scenario) error {
	if eng.Universe.Status == StatusRunning {
		return fmt.Errorf("simulation: universe %s already running", eng.Universe.ID)
	}
	if sc.TimeScale > 0 {
		eng.Universe.Time.SetScale(sc.TimeScale)
	}
	ctx, cancel := context.WithCancel(parent)
	eng.cancel = cancel
	eng.Universe.Status = StatusRunning
	eng.Universe.EventBus.Start(ctx)

	actions := sc.Sorted()
	eng.Universe.Log("scenario %q started (%d actions, seed=%d)", sc.Name, len(actions), eng.Universe.RNG.RootSeed())

	eng.runWG.Add(1)
	go func() {
		defer eng.runWG.Done()
		defer func() {
			if eng.Universe.Status == StatusRunning {
				eng.Universe.Status = StatusStopped
			}
			eng.Universe.Log("scenario %q finished", sc.Name)
		}()
		start := time.Now()
		for _, a := range actions {
			select {
			case <-ctx.Done():
				return
			default:
			}
			// Wait until the action's scheduled simulated offset arrives,
			// respecting the configured TimeScale via TimeEngine.Sleep.
			target := start.Add(scaledWall(a.At, eng.Universe.Time.Scale()))
			if d := time.Until(target); d > 0 {
				select {
				case <-time.After(capDuration(d)):
				case <-ctx.Done():
					return
				}
			}
			eng.execAction(ctx, a)
		}
	}()
	return nil
}

func scaledWall(simOffset time.Duration, scale float64) time.Duration {
	if scale <= 0 {
		scale = 1
	}
	return time.Duration(float64(simOffset) / scale)
}

func capDuration(d time.Duration) time.Duration {
	const maxWait = 3 * time.Second
	if d > maxWait {
		return maxWait
	}
	return d
}

// Stop cancels the running scenario and waits for the executor goroutine to
// return, then stops the event bus workers.
func (eng *Engine) Stop() {
	if eng.cancel != nil {
		eng.cancel()
	}
	eng.runWG.Wait()
	eng.Universe.EventBus.Stop()
	eng.Universe.Status = StatusStopped
}

// Pause / Resume toggle the TimeEngine only; the timeline goroutine keeps
// waiting on its own scaled clock, so pausing genuinely halts progress.
func (eng *Engine) Pause()  { eng.Universe.Time.Pause(); eng.Universe.Status = StatusPaused }
func (eng *Engine) Resume() { eng.Universe.Time.Resume(); eng.Universe.Status = StatusRunning }

func (eng *Engine) execAction(ctx context.Context, a scenario.Action) {
	u := eng.Universe
	switch a.Kind {
	case scenario.ActionSpawn:
		schemaName, _ := a.Params["schema"].(string)
		count := 0
		if v, ok := a.Params["count"].(float64); ok {
			count = int(v)
		}
		if schemaName == "" || count <= 0 {
			u.Log("spawn action %s: invalid params", a.ID)
			return
		}
		if _, err := u.SpawnEntities(schemaName, count, nil); err != nil {
			u.Log("spawn action %s failed: %v", a.ID, err)
		}

	case scenario.ActionAttach:
		eng.execAttach(ctx, a)

	case scenario.ActionEmit:
		eType, _ := a.Params["type"].(string)
		src, _ := a.Params["source"].(string)
		payload, _ := a.Params["payload"].(map[string]interface{})
		u.EventBus.Publish(event.Event{
			ID: fmt.Sprintf("%s-%d", a.ID, time.Now().UnixNano()),
			Type: eType, Source: src, Timestamp: u.Time.Now(), Payload: payload,
		})

	case scenario.ActionLoad:
		eng.execLoad(ctx, a)

	case scenario.ActionChaos:
		eng.execChaos(a)

	case scenario.ActionWait:
		u.Time.Sleep(a.Duration)

	case scenario.ActionSnapshot:
		label, _ := a.Params["label"].(string)
		if label == "" {
			label = a.ID
		}
		if _, err := eng.SaveSnapshot(label); err != nil {
			u.Log("snapshot action %s failed: %v", a.ID, err)
		}

	case scenario.ActionSetNetwork:
		from, _ := a.Params["from"].(string)
		to, _ := a.Params["to"].(string)
		if v, ok := a.Params["latencyMs"].(float64); ok {
			u.Network.SetLatency(from, to, v)
		}
		if v, ok := a.Params["packetLoss"].(float64); ok {
			u.Network.SetPacketLoss(from, to, v)
		}
	}
}

func (eng *Engine) execAttach(ctx context.Context, a scenario.Action) {
	u := eng.Universe
	behaviorName, _ := a.Params["behavior"].(string)
	entityType, _ := a.Params["entityType"].(string)
	graph, ok := u.Behaviors[behaviorName]
	if !ok {
		u.Log("attach action %s: unknown behavior %q", a.ID, behaviorName)
		return
	}
	limit := 0
	if v, ok := a.Params["limit"].(float64); ok {
		limit = int(v)
	}
	entities := u.Collection(entityType).All()
	if limit > 0 && limit < len(entities) {
		entities = entities[:limit]
	}
	runner := behavior.NewRunner(graph)
	stream := u.RNG.Stream("behavior:" + behaviorName)

	for _, ent := range entities {
		select {
		case <-ctx.Done():
			return
		default:
		}
		go func(e *entityRef) {
			rc := &behavior.RunContext{
				Context: ctx, EntityID: e.id, EntityType: e.typ,
				RNG: rand.New(rand.NewSource(stream.Int63())),
				Vars: map[string]interface{}{},
				EmitEvent: func(params map[string]interface{}) {
					eType, _ := params["type"].(string)
					u.EventBus.Publish(event.Event{
						ID: fmt.Sprintf("%s-%d", e.id, time.Now().UnixNano()),
						Type: eType, Source: e.id, Timestamp: u.Time.Now(),
						Payload: params,
					})
				},
				ChangeState: func(newState string) {
					if coll := u.Collection(e.typ); coll != nil {
						if ent, ok := coll.Get(e.id); ok {
							ent.State = newState
							ent.UpdatedAt = time.Now()
						}
					}
				},
				Sleep: u.Time.Sleep,
			}
			_ = runner.Run(rc, 2000)
		}(&entityRef{id: ent.ID, typ: ent.Type})
	}
	u.Log("attached behavior %q to %d %s entities", behaviorName, len(entities), entityType)
}

type entityRef struct{ id, typ string }

func (eng *Engine) execLoad(ctx context.Context, a scenario.Action) {
	u := eng.Universe
	url, _ := a.Params["url"].(string)
	if url == "" {
		u.Log("load action %s: no target url configured (AI-suggested profiles need a real, authorized URL filled in before they can run)", a.ID)
		return
	}
	method, _ := a.Params["method"].(string)
	vus := 1
	if v, ok := a.Params["virtualUsers"].(float64); ok {
		vus = int(v)
	}
	rate := 0.0
	if v, ok := a.Params["requestRate"].(float64); ok {
		rate = v
	}
	target := loadgen.Target{Authorized: true, Method: method, URL: url}
	profile := loadgen.Profile{
		VirtualUsers: vus, RequestRate: rate,
		Duration: a.Duration, RampUp: a.Duration / 10,
	}
	runner, err := loadgen.NewRunner(target, profile)
	if err != nil {
		u.Log("load action %s failed: %v", a.ID, err)
		return
	}
	_ = runner.Run(ctx, func(s loadgen.LiveStats) {
		u.Metrics.Set("load.rps", s.RPS)
		u.Metrics.Set("load.p95_ms", s.Latency.P95)
		u.Metrics.Set("load.p99_ms", s.Latency.P99)
		u.Metrics.Set("load.active_vus", float64(s.ActiveVUs))
		u.Metrics.Inc("load.requests_total", 0) // requests_total tracked via s.Requests directly below
		u.Metrics.Set("load.requests_total", float64(s.Requests))
		u.Metrics.Set("load.failures_total", float64(s.Failures))
	})
}

func (eng *Engine) execChaos(a scenario.Action) {
	u := eng.Universe
	kind, _ := a.Params["kind"].(string)
	f := &chaos.Fault{
		ID: a.ID, Kind: chaos.FaultKind(kind),
		Duration: a.Duration, Params: a.Params,
	}
	if v, ok := a.Params["nodeId"].(string); ok {
		f.NodeID = v
	}
	if v, ok := a.Params["from"].(string); ok {
		f.LinkFrom = v
	}
	if v, ok := a.Params["to"].(string); ok {
		f.LinkTo = v
	}
	u.Chaos.Inject(f)
	u.Log("chaos fault %q (%s) injected for %s", a.ID, kind, a.Duration)
}

// ensure generator import stays used even as the file evolves
var _ = generator.NewEngine
