// Package database implements the Database Simulation Layer. It defines a
// small driver-agnostic interface (Target) so the Seeder can push generated
// entity.Entity records into PostgreSQL, MySQL or SQLite via database/sql —
// bring your own driver import in cmd/api or cmd/cli — while shipping with a
// zero-dependency in-memory Target so the whole platform runs standalone
// with no external services required.
package database

import (
	"fmt"
	"sync"

	"void-platform/backend/internal/entity"
)

// Target is anything that can receive seeded rows/documents. A real SQL
// target is expected to translate Insert/Update/Delete into parameterised
// statements against database/sql; see docs/DATABASE_DRIVERS.md.
type Target interface {
	Insert(table string, e *entity.Entity) error
	Update(table string, e *entity.Entity) error
	Delete(table string, id string) error
	Count(table string) int
	Snapshot(table string) []*entity.Entity
	Rollback(table string) error // restore to last Snapshot() taken via BeginRollbackPoint
	BeginRollbackPoint(table string)
	Clear(table string) error
}

// MemoryTarget is the default, dependency-free simulation target: an
// in-memory relational-ish table store good enough for QA/dev-scale seeding
// and for driving Query Load simulation numbers without a live database.
type MemoryTarget struct {
	mu           sync.RWMutex
	tables       map[string]map[string]*entity.Entity
	rollbackSnap map[string]map[string]*entity.Entity
	queryCount   map[string]int64
}

func NewMemoryTarget() *MemoryTarget {
	return &MemoryTarget{
		tables:       map[string]map[string]*entity.Entity{},
		rollbackSnap: map[string]map[string]*entity.Entity{},
		queryCount:   map[string]int64{},
	}
}

func (m *MemoryTarget) table(name string) map[string]*entity.Entity {
	if _, ok := m.tables[name]; !ok {
		m.tables[name] = map[string]*entity.Entity{}
	}
	return m.tables[name]
}

func (m *MemoryTarget) Insert(table string, e *entity.Entity) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.table(table)[e.ID] = e
	return nil
}

func (m *MemoryTarget) Update(table string, e *entity.Entity) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	t := m.table(table)
	if _, ok := t[e.ID]; !ok {
		return fmt.Errorf("row %s not found in table %s", e.ID, table)
	}
	t[e.ID] = e
	return nil
}

func (m *MemoryTarget) Delete(table string, id string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	delete(m.table(table), id)
	return nil
}

func (m *MemoryTarget) Count(table string) int {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return len(m.tables[table])
}

func (m *MemoryTarget) Snapshot(table string) []*entity.Entity {
	m.mu.RLock()
	defer m.mu.RUnlock()
	out := make([]*entity.Entity, 0, len(m.tables[table]))
	for _, e := range m.tables[table] {
		out = append(out, e)
	}
	return out
}

func (m *MemoryTarget) BeginRollbackPoint(table string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	snap := map[string]*entity.Entity{}
	for k, v := range m.table(table) {
		cp := *v
		snap[k] = &cp
	}
	m.rollbackSnap[table] = snap
}

func (m *MemoryTarget) Rollback(table string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	snap, ok := m.rollbackSnap[table]
	if !ok {
		return fmt.Errorf("no rollback point for table %s", table)
	}
	restored := map[string]*entity.Entity{}
	for k, v := range snap {
		cp := *v
		restored[k] = &cp
	}
	m.tables[table] = restored
	return nil
}

func (m *MemoryTarget) Clear(table string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.tables[table] = map[string]*entity.Entity{}
	return nil
}

// Seeder drives batches of generated entities into a Target using parallel
// workers, so very large synthetic datasets (millions of rows) seed quickly
// without blocking the caller on a single connection/goroutine.
type Seeder struct {
	Target  Target
	Workers int
}

func NewSeeder(target Target, workers int) *Seeder {
	if workers <= 0 {
		workers = 4
	}
	return &Seeder{Target: target, Workers: workers}
}

// SeedBatch inserts entities into table using a worker pool, returning the
// count successfully inserted and the first error encountered (if any).
func (s *Seeder) SeedBatch(table string, entities []*entity.Entity) (int, error) {
	jobs := make(chan *entity.Entity, len(entities))
	for _, e := range entities {
		jobs <- e
	}
	close(jobs)

	var wg sync.WaitGroup
	var mu sync.Mutex
	inserted := 0
	var firstErr error

	for i := 0; i < s.Workers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for e := range jobs {
				if err := s.Target.Insert(table, e); err != nil {
					mu.Lock()
					if firstErr == nil {
						firstErr = err
					}
					mu.Unlock()
					continue
				}
				mu.Lock()
				inserted++
				mu.Unlock()
			}
		}()
	}
	wg.Wait()
	return inserted, firstErr
}
