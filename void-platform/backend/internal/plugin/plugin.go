// Package plugin defines VOID's extension surface: Versioned Go interfaces
// that let new Generators, Entity Types, Behaviors, Connectors and Exporters
// be registered into the running engine without modifying core code. This
// ships as an in-process registry (safe, no dynamic .so loading) — a plugin
// is any Go package that calls Register* during its init().
package plugin

import (
	"context"
	"fmt"
	"sync"
)

// APIVersion is bumped whenever a breaking change is made to any interface
// below; plugins should assert compatibility against it at registration time.
const APIVersion = "1.0"

// FieldGenerator lets a plugin supply a custom_function field generator.
type FieldGenerator interface {
	Name() string
	Generate(ctx context.Context, params map[string]interface{}) (interface{}, error)
}

// EntityTypeProvider lets a plugin ship a ready-made entity.Schema template
// (e.g. a domain-specific entity not in the built-in Template Library).
type EntityTypeProvider interface {
	Name() string
	SchemaJSON() []byte
}

// BehaviorProvider lets a plugin ship a reusable behavior.Graph template.
type BehaviorProvider interface {
	Name() string
	GraphJSON() []byte
}

// Connector lets a plugin add a new external system integration (e.g. a
// message queue, a new database dialect) surfaced to Scenarios by name.
type Connector interface {
	Name() string
	Connect(ctx context.Context, config map[string]interface{}) (Session, error)
}

// Session is a live handle returned by a Connector, closed when a Simulation
// Run ends.
type Session interface {
	Send(ctx context.Context, payload map[string]interface{}) error
	Close() error
}

// Exporter lets a plugin add a new dataset export target/format.
type Exporter interface {
	Name() string
	Export(ctx context.Context, table string, records []map[string]interface{}) ([]byte, error)
}

// Registry is the process-wide plugin registry.
type Registry struct {
	mu         sync.RWMutex
	generators map[string]FieldGenerator
	entities   map[string]EntityTypeProvider
	behaviors  map[string]BehaviorProvider
	connectors map[string]Connector
	exporters  map[string]Exporter
}

var global = &Registry{
	generators: map[string]FieldGenerator{},
	entities:   map[string]EntityTypeProvider{},
	behaviors:  map[string]BehaviorProvider{},
	connectors: map[string]Connector{},
	exporters:  map[string]Exporter{},
}

// Global returns the process-wide plugin registry.
func Global() *Registry { return global }

func (r *Registry) RegisterGenerator(g FieldGenerator) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if _, exists := r.generators[g.Name()]; exists {
		return fmt.Errorf("plugin: generator %q already registered", g.Name())
	}
	r.generators[g.Name()] = g
	return nil
}

func (r *Registry) Generator(name string) (FieldGenerator, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	g, ok := r.generators[name]
	return g, ok
}

func (r *Registry) RegisterEntityType(e EntityTypeProvider) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if _, exists := r.entities[e.Name()]; exists {
		return fmt.Errorf("plugin: entity type %q already registered", e.Name())
	}
	r.entities[e.Name()] = e
	return nil
}

func (r *Registry) RegisterBehavior(b BehaviorProvider) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if _, exists := r.behaviors[b.Name()]; exists {
		return fmt.Errorf("plugin: behavior %q already registered", b.Name())
	}
	r.behaviors[b.Name()] = b
	return nil
}

func (r *Registry) RegisterConnector(c Connector) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if _, exists := r.connectors[c.Name()]; exists {
		return fmt.Errorf("plugin: connector %q already registered", c.Name())
	}
	r.connectors[c.Name()] = c
	return nil
}

func (r *Registry) RegisterExporter(e Exporter) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if _, exists := r.exporters[e.Name()]; exists {
		return fmt.Errorf("plugin: exporter %q already registered", e.Name())
	}
	r.exporters[e.Name()] = e
	return nil
}

// List returns the names of every registered extension, grouped by kind —
// used by the API's /api/plugins endpoint and the frontend Plugin Manager.
func (r *Registry) List() map[string][]string {
	r.mu.RLock()
	defer r.mu.RUnlock()
	out := map[string][]string{}
	for k := range r.generators {
		out["generators"] = append(out["generators"], k)
	}
	for k := range r.entities {
		out["entityTypes"] = append(out["entityTypes"], k)
	}
	for k := range r.behaviors {
		out["behaviors"] = append(out["behaviors"], k)
	}
	for k := range r.connectors {
		out["connectors"] = append(out["connectors"], k)
	}
	for k := range r.exporters {
		out["exporters"] = append(out["exporters"], k)
	}
	return out
}
