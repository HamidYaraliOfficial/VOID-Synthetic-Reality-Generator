package simulation

import (
	"fmt"
	"time"

	"void-platform/backend/internal/entity"
	"void-platform/backend/internal/storage"
)

// SnapshotData is the fully-serializable state of a Universe at one instant:
// enough to restore entity population, schemas and metrics, implementing
// Snapshot & Replay from the product spec. (Live network/behavior goroutine
// state is not resumable by design — only durable data is snapshotted.)
type SnapshotData struct {
	ID          string                        `json:"id"`
	UniverseID  string                        `json:"universeId"`
	Label       string                        `json:"label"`
	Seed        int64                         `json:"seed"`
	CreatedAt   time.Time                     `json:"createdAt"`
	Schemas     map[string]*entity.Schema     `json:"schemas"`
	Entities    map[string][]*entity.Entity   `json:"entities"`
	Metrics     map[string]float64            `json:"metrics"`
}

// SaveSnapshot serializes the Universe's current durable state to disk under
// ./data/snapshots/<label>.json and returns the metadata record.
func (eng *Engine) SaveSnapshot(label string) (*storage.SnapshotMeta, error) {
	u := eng.Universe
	u.mu.RLock()
	schemasCopy := map[string]*entity.Schema{}
	for k, v := range u.Schemas {
		schemasCopy[k] = v
	}
	u.mu.RUnlock()

	entitiesCopy := map[string][]*entity.Entity{}
	for typeName, coll := range u.Collections {
		entitiesCopy[typeName] = coll.All()
	}

	snap := SnapshotData{
		ID:         fmt.Sprintf("%s-%d", label, time.Now().Unix()),
		UniverseID: u.ID,
		Label:      label,
		Seed:       u.Seed,
		CreatedAt:  time.Now(),
		Schemas:    schemasCopy,
		Entities:   entitiesCopy,
		Metrics:    u.Metrics.Snapshot().Counters,
	}
	dir := "data/snapshots"
	path := dir + "/" + label + ".json"
	if err := storage.SaveJSON(path, snap); err != nil {
		return nil, err
	}
	u.Log("snapshot %q saved (%d entity types)", label, len(entitiesCopy))
	return &storage.SnapshotMeta{ID: snap.ID, Label: label, Path: path, CreatedAt: snap.CreatedAt}, nil
}

// LoadSnapshot restores entities+schemas from a previously saved snapshot
// file into the (already-constructed) Universe, letting a Simulation Run
// "continue from a Snapshot" as required by the product spec.
func (eng *Engine) LoadSnapshot(path string) error {
	var snap SnapshotData
	if err := storage.LoadJSON(path, &snap); err != nil {
		return err
	}
	u := eng.Universe
	for name, schema := range snap.Schemas {
		_ = u.AddSchema(schema)
		_ = name
	}
	for typeName, entities := range snap.Entities {
		coll := u.Collection(typeName)
		for _, e := range entities {
			coll.Add(e)
		}
	}
	u.Log("snapshot %q loaded (%d entity types restored)", snap.Label, len(snap.Entities))
	return nil
}

// DiffResult is the output of comparing two Simulation Runs' final metrics,
// implementing the Simulation Diff Engine / Experiment Manager comparison.
type DiffResult struct {
	MetricDeltas map[string]MetricDelta `json:"metricDeltas"`
	Summary      []string                `json:"summary"`
}

type MetricDelta struct {
	A     float64 `json:"a"`
	B     float64 `json:"b"`
	Delta float64 `json:"delta"`
	PctChange float64 `json:"pctChange"`
}

// DiffMetrics compares two metrics snapshots (e.g. from Experiment A vs B)
// and produces a per-metric delta plus a short natural-language summary of
// the most significant changes, used by the Experiment Manager UI.
func DiffMetrics(a, b map[string]float64) DiffResult {
	result := DiffResult{MetricDeltas: map[string]MetricDelta{}}
	names := map[string]bool{}
	for k := range a {
		names[k] = true
	}
	for k := range b {
		names[k] = true
	}
	for name := range names {
		av, bv := a[name], b[name]
		delta := bv - av
		pct := 0.0
		if av != 0 {
			pct = (delta / av) * 100
		}
		result.MetricDeltas[name] = MetricDelta{A: av, B: bv, Delta: delta, PctChange: pct}
	}
	// Surface the 3 largest absolute percentage swings as a human summary.
	type ranked struct {
		name string
		pct  float64
	}
	var rankedList []ranked
	for name, d := range result.MetricDeltas {
		abs := d.PctChange
		if abs < 0 {
			abs = -abs
		}
		rankedList = append(rankedList, ranked{name, abs})
	}
	for i := 0; i < len(rankedList); i++ {
		for j := i + 1; j < len(rankedList); j++ {
			if rankedList[j].pct > rankedList[i].pct {
				rankedList[i], rankedList[j] = rankedList[j], rankedList[i]
			}
		}
	}
	limit := 3
	if len(rankedList) < limit {
		limit = len(rankedList)
	}
	for i := 0; i < limit; i++ {
		n := rankedList[i].name
		d := result.MetricDeltas[n]
		result.Summary = append(result.Summary, fmt.Sprintf("%s changed by %.1f%% (%.2f -> %.2f)", n, d.PctChange, d.A, d.B))
	}
	return result
}
