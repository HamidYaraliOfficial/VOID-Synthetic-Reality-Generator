// Package randomx provides deterministic, seed-based randomness for the whole
// VOID simulation stack. Every generator, behavior probability and load
// pattern in the platform pulls its entropy from a randomx.Manager so that an
// entire Simulation Run can be reproduced bit-for-bit from a single seed.
package randomx

import (
	"encoding/binary"
	"hash/fnv"
	"math"
	"math/rand"
	"sync"
)

// Manager owns a tree of named, independently-seeded streams derived from a
// single root seed. Using named sub-streams (one per subsystem: "entity",
// "event", "loadgen", ...) means adding/removing consumers of randomness in
// one subsystem never perturbs the sequence seen by another subsystem, which
// keeps Simulation Diff / Reproducibility guarantees stable across versions.
type Manager struct {
	mu      sync.Mutex
	rootSeed int64
	streams  map[string]*rand.Rand
}

// NewManager creates a seed manager. If seed == 0 a random root seed is
// generated so ad-hoc runs still work, but it is echoed back via RootSeed()
// so the caller can persist it for later reproduction.
func NewManager(seed int64) *Manager {
	if seed == 0 {
		seed = int64(fnvHash("void-default-seed"))
	}
	return &Manager{
		rootSeed: seed,
		streams:  make(map[string]*rand.Rand),
	}
}

// RootSeed returns the seed the whole universe was created with.
func (m *Manager) RootSeed() int64 { return m.rootSeed }

// Stream returns (creating if necessary) a *rand.Rand deterministically
// derived from the root seed and the given name. Two managers built with the
// same root seed always produce identical streams for the same name.
func (m *Manager) Stream(name string) *rand.Rand {
	m.mu.Lock()
	defer m.mu.Unlock()
	if r, ok := m.streams[name]; ok {
		return r
	}
	derived := deriveSeed(m.rootSeed, name)
	r := rand.New(rand.NewSource(derived))
	m.streams[name] = r
	return r
}

func deriveSeed(root int64, name string) int64 {
	h := fnv.New64a()
	buf := make([]byte, 8)
	binary.LittleEndian.PutUint64(buf, uint64(root))
	h.Write(buf)
	h.Write([]byte(name))
	return int64(h.Sum64())
}

func fnvHash(s string) uint64 {
	h := fnv.New64a()
	h.Write([]byte(s))
	return h.Sum64()
}

// --- Distribution sampling -------------------------------------------------

// Distribution is a named probability distribution used by field generators,
// load-pattern shaping and behavior timing.
type Distribution struct {
	Kind   string  `json:"kind"` // uniform|normal|lognormal|poisson|exponential|pareto|weighted
	Min    float64 `json:"min,omitempty"`
	Max    float64 `json:"max,omitempty"`
	Mean   float64 `json:"mean,omitempty"`
	StdDev float64 `json:"stddev,omitempty"`
	Lambda float64 `json:"lambda,omitempty"` // poisson / exponential rate
	Alpha  float64 `json:"alpha,omitempty"`  // pareto shape
	XMin   float64 `json:"xmin,omitempty"`   // pareto scale
	// Weighted distribution: parallel arrays of discrete values and weights.
	Values  []float64 `json:"values,omitempty"`
	Weights []float64 `json:"weights,omitempty"`
}

// Sample draws one value from the configured distribution using r.
func (d Distribution) Sample(r *rand.Rand) float64 {
	switch d.Kind {
	case "normal", "gaussian":
		mean, std := d.Mean, d.StdDev
		if std == 0 {
			std = 1
		}
		return r.NormFloat64()*std + mean
	case "lognormal":
		mean, std := d.Mean, d.StdDev
		if std == 0 {
			std = 1
		}
		return math.Exp(r.NormFloat64()*std + mean)
	case "exponential":
		lambda := d.Lambda
		if lambda == 0 {
			lambda = 1
		}
		return r.ExpFloat64() / lambda
	case "poisson":
		return samplePoisson(r, d.Lambda)
	case "pareto":
		alpha, xmin := d.Alpha, d.XMin
		if alpha == 0 {
			alpha = 1
		}
		if xmin == 0 {
			xmin = 1
		}
		u := r.Float64()
		return xmin / math.Pow(1-u, 1/alpha)
	case "weighted":
		return sampleWeighted(r, d.Values, d.Weights)
	default: // uniform
		lo, hi := d.Min, d.Max
		if hi <= lo {
			hi = lo + 1
		}
		return lo + r.Float64()*(hi-lo)
	}
}

func samplePoisson(r *rand.Rand, lambda float64) float64 {
	if lambda <= 0 {
		lambda = 1
	}
	L := math.Exp(-lambda)
	k := 0.0
	p := 1.0
	for {
		k++
		p *= r.Float64()
		if p <= L {
			break
		}
	}
	return k - 1
}

func sampleWeighted(r *rand.Rand, values, weights []float64) float64 {
	if len(values) == 0 {
		return 0
	}
	if len(weights) != len(values) {
		return values[r.Intn(len(values))]
	}
	total := 0.0
	for _, w := range weights {
		total += w
	}
	if total <= 0 {
		return values[r.Intn(len(values))]
	}
	target := r.Float64() * total
	acc := 0.0
	for i, w := range weights {
		acc += w
		if target <= acc {
			return values[i]
		}
	}
	return values[len(values)-1]
}
