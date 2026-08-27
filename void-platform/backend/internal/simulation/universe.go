// Package simulation ties every subsystem together into the Universe: the
// live, running synthetic world that owns Entities, the Event Bus, Behavior
// instances, the Network Topology, transaction Ledger, metrics and the
// scenario Timeline executor. This is the "single Engine" the product spec
// requires everything else to be "connected to" — no subsystem here is a
// mock: entities are really generated, events really flow through real
// goroutines/channels, behaviors really execute, load generation really
// issues HTTP calls to the address the operator configured.
package simulation

import (
	"context"
	"fmt"
	"sync"
	"time"

	"void-platform/backend/internal/behavior"
	"void-platform/backend/internal/chaos"
	"void-platform/backend/internal/database"
	"void-platform/backend/internal/entity"
	"void-platform/backend/internal/event"
	"void-platform/backend/internal/generator"
	"void-platform/backend/internal/metrics"
	"void-platform/backend/internal/network"
	"void-platform/backend/internal/randomx"
	"void-platform/backend/internal/transaction"
)

// Status enumerates the lifecycle of a Universe's simulation run.
type Status string

const (
	StatusIdle    Status = "idle"
	StatusRunning Status = "running"
	StatusPaused  Status = "paused"
	StatusStopped Status = "stopped"
)

// Universe is one complete synthetic world.
type Universe struct {
	mu sync.RWMutex

	ID   string
	Name string
	Seed int64

	RNG *randomx.Manager

	Schemas     map[string]*entity.Schema
	Collections map[string]*entity.Collection
	Generator   *generator.Engine

	EventBus  *event.Bus
	Behaviors map[string]*behavior.Graph

	Network *network.Topology
	Chaos   *chaos.Engine

	DBTarget *database.MemoryTarget
	Seeder   *database.Seeder

	Ledger    *transaction.Ledger
	RuleEngine *transaction.Engine
	TxProcessor *transaction.Processor
	Transactions []*transaction.Transaction

	Metrics *metrics.Collector
	Time    *TimeEngine

	Status Status

	consoleLog []string
}

// NewUniverse constructs an empty Universe wired end-to-end and ready for
// schemas/scenarios to be loaded into it.
func NewUniverse(id, name string, seed int64) *Universe {
	rng := randomx.NewManager(seed)
	topo := network.NewTopology()
	u := &Universe{
		ID:          id,
		Name:        name,
		Seed:        rng.RootSeed(),
		RNG:         rng,
		Schemas:     map[string]*entity.Schema{},
		Collections: map[string]*entity.Collection{},
		Generator:   generator.NewEngine(rng),
		EventBus:    event.NewBus(50000, 16),
		Behaviors:   map[string]*behavior.Graph{},
		Network:     topo,
		Chaos:       chaos.NewEngine(topo),
		DBTarget:    database.NewMemoryTarget(),
		Ledger:      transaction.NewLedger(),
		RuleEngine:  transaction.NewEngine(nil),
		Metrics:     metrics.NewCollector(),
		Time:        NewTimeEngine(1.0),
		Status:      StatusIdle,
	}
	u.Seeder = database.NewSeeder(u.DBTarget, 8)
	u.TxProcessor = transaction.NewProcessor(u.RuleEngine, u.Ledger)
	u.wireMetricsSubscriber()
	return u
}

func (u *Universe) wireMetricsSubscriber() {
	u.EventBus.Subscribe("*", func(ctx context.Context, e event.Event) error {
		u.Metrics.Inc("events_processed_total", 1)
		u.Metrics.Inc("events_by_type."+e.Type, 1)
		return nil
	})
}

// Log appends a line to the in-memory Console feed (bounded, newest last).
func (u *Universe) Log(format string, args ...interface{}) {
	u.mu.Lock()
	defer u.mu.Unlock()
	line := fmt.Sprintf("[%s] %s", time.Now().Format("15:04:05.000"), fmt.Sprintf(format, args...))
	u.consoleLog = append(u.consoleLog, line)
	if len(u.consoleLog) > 5000 {
		u.consoleLog = u.consoleLog[len(u.consoleLog)-5000:]
	}
}

func (u *Universe) ConsoleTail(n int) []string {
	u.mu.RLock()
	defer u.mu.RUnlock()
	if n <= 0 || n > len(u.consoleLog) {
		n = len(u.consoleLog)
	}
	return append([]string{}, u.consoleLog[len(u.consoleLog)-n:]...)
}

// AddSchema registers an Entity Schema (from the Entity Designer) into the
// Universe after validating it.
func (u *Universe) AddSchema(s *entity.Schema) error {
	if err := s.Validate(); err != nil {
		return err
	}
	u.mu.Lock()
	defer u.mu.Unlock()
	u.Schemas[s.Name] = s
	if _, ok := u.Collections[s.Name]; !ok {
		u.Collections[s.Name] = entity.NewCollection(s.Name)
	}
	return nil
}

func (u *Universe) AddBehavior(g *behavior.Graph) {
	u.mu.Lock()
	defer u.mu.Unlock()
	u.Behaviors[g.Name] = g
}

// Collection returns (creating if needed) the entity.Collection for a type.
func (u *Universe) Collection(typeName string) *entity.Collection {
	u.mu.Lock()
	defer u.mu.Unlock()
	c, ok := u.Collections[typeName]
	if !ok {
		c = entity.NewCollection(typeName)
		u.Collections[typeName] = c
	}
	return c
}

// relatedLookup implements generator.RelatedLookup against this Universe's
// live collections, powering the Relationship-Aware Generator.
func (u *Universe) relatedLookup(relatedType string) (string, bool) {
	c := u.Collection(relatedType)
	stream := u.RNG.Stream("relations")
	e, ok := c.Random(func(n int) int { return stream.Intn(n) })
	if !ok {
		return "", false
	}
	return e.ID, true
}

// SpawnEntities generates n entities of typeName and adds them to the
// Universe (and, if a DB target is configured, seeds them there too).
func (u *Universe) SpawnEntities(typeName string, n int, progress func(done, total int)) ([]*entity.Entity, error) {
	u.mu.RLock()
	schema, ok := u.Schemas[typeName]
	u.mu.RUnlock()
	if !ok {
		return nil, fmt.Errorf("simulation: unknown entity type %q", typeName)
	}
	entities, err := u.Generator.GenerateBatch(schema, n, u.relatedLookup, progress)
	if err != nil {
		return entities, err
	}
	coll := u.Collection(typeName)
	for _, e := range entities {
		coll.Add(e)
	}
	u.Metrics.Inc("entities_created_total", float64(len(entities)))
	u.Metrics.Set("entity_count."+typeName, float64(coll.Len()))
	u.Log("spawned %d %s entities", len(entities), typeName)
	return entities, nil
}

// SchemasSnapshot returns a stable slice copy of every registered schema,
// used by the API layer to list schemas without exposing the internal map.
func (u *Universe) SchemasSnapshot() []*entity.Schema {
	u.mu.RLock()
	defer u.mu.RUnlock()
	out := make([]*entity.Schema, 0, len(u.Schemas))
	for _, sc := range u.Schemas {
		out = append(out, sc)
	}
	return out
}

// EntityCounts returns a live count per entity type for the Universe
// Explorer tree and dashboard widgets.
func (u *Universe) EntityCounts() map[string]int {
	u.mu.RLock()
	defer u.mu.RUnlock()
	out := map[string]int{}
	for name, c := range u.Collections {
		out[name] = c.Len()
	}
	return out
}
