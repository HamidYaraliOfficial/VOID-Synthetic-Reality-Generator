// Package metrics implements the built-in Metrics & Observability Engine:
// an in-memory counter/gauge registry plus a Prometheus-text-format exporter
// so external monitoring stacks (or the bundled Dashboard Builder) can poll
// /api/metrics without any third-party client library.
package metrics

import (
	"fmt"
	"runtime"
	"sort"
	"strings"
	"sync"
	"time"
)

// Collector is a concurrency-safe registry of named counters and gauges.
type Collector struct {
	mu       sync.RWMutex
	counters map[string]float64
	gauges   map[string]float64
	started  time.Time
}

func NewCollector() *Collector {
	return &Collector{
		counters: map[string]float64{},
		gauges:   map[string]float64{},
		started:  time.Now(),
	}
}

// Inc increments a named counter (event rate, transaction rate, error rate, ...).
func (c *Collector) Inc(name string, delta float64) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.counters[name] += delta
}

// Set overwrites a named gauge (queue length, goroutine count, cpu%, ...).
func (c *Collector) Set(name string, value float64) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.gauges[name] = value
}

// Snapshot is a point-in-time read of every tracked metric plus Go runtime
// stats (goroutine count, memory) requested by the product spec.
type Snapshot struct {
	Counters      map[string]float64 `json:"counters"`
	Gauges        map[string]float64 `json:"gauges"`
	GoroutineCount int               `json:"goroutineCount"`
	MemoryAllocMB  float64           `json:"memoryAllocMB"`
	UptimeSeconds  float64           `json:"uptimeSeconds"`
	Timestamp      time.Time         `json:"timestamp"`
}

func (c *Collector) Snapshot() Snapshot {
	c.mu.RLock()
	defer c.mu.RUnlock()
	var mem runtime.MemStats
	runtime.ReadMemStats(&mem)

	counters := make(map[string]float64, len(c.counters))
	for k, v := range c.counters {
		counters[k] = v
	}
	gauges := make(map[string]float64, len(c.gauges))
	for k, v := range c.gauges {
		gauges[k] = v
	}
	return Snapshot{
		Counters:       counters,
		Gauges:         gauges,
		GoroutineCount: runtime.NumGoroutine(),
		MemoryAllocMB:  float64(mem.Alloc) / (1024 * 1024),
		UptimeSeconds:  time.Since(c.started).Seconds(),
		Timestamp:      time.Now(),
	}
}

// PrometheusText renders every counter/gauge in Prometheus text exposition
// format for scraping by an external Prometheus-compatible collector.
func (c *Collector) PrometheusText() string {
	snap := c.Snapshot()
	var sb strings.Builder
	writeSorted := func(prefix string, kind string, m map[string]float64) {
		names := make([]string, 0, len(m))
		for k := range m {
			names = append(names, k)
		}
		sort.Strings(names)
		for _, name := range names {
			metricName := "void_" + sanitize(name)
			fmt.Fprintf(&sb, "# TYPE %s %s\n%s %v\n", metricName, kind, metricName, m[name])
		}
		_ = prefix
	}
	writeSorted("void_", "counter", snap.Counters)
	writeSorted("void_", "gauge", snap.Gauges)
	fmt.Fprintf(&sb, "# TYPE void_goroutines gauge\nvoid_goroutines %d\n", snap.GoroutineCount)
	fmt.Fprintf(&sb, "# TYPE void_memory_alloc_mb gauge\nvoid_memory_alloc_mb %v\n", snap.MemoryAllocMB)
	fmt.Fprintf(&sb, "# TYPE void_uptime_seconds counter\nvoid_uptime_seconds %v\n", snap.UptimeSeconds)
	return sb.String()
}

func sanitize(name string) string {
	r := strings.NewReplacer(".", "_", "-", "_", " ", "_", ":", "_")
	return r.Replace(name)
}

// LatencyTracker keeps a bounded rolling window of latency samples (ms) to
// compute P95/P99/throughput for the Load Generator's live dashboard.
type LatencyTracker struct {
	mu      sync.Mutex
	samples []float64
	cap     int
}

func NewLatencyTracker(capacity int) *LatencyTracker {
	if capacity <= 0 {
		capacity = 5000
	}
	return &LatencyTracker{cap: capacity}
}

func (lt *LatencyTracker) Add(ms float64) {
	lt.mu.Lock()
	defer lt.mu.Unlock()
	lt.samples = append(lt.samples, ms)
	if len(lt.samples) > lt.cap {
		lt.samples = lt.samples[len(lt.samples)-lt.cap:]
	}
}

type LatencyStats struct {
	Count int     `json:"count"`
	P50   float64 `json:"p50"`
	P95   float64 `json:"p95"`
	P99   float64 `json:"p99"`
	Max   float64 `json:"max"`
	Mean  float64 `json:"mean"`
}

func (lt *LatencyTracker) Stats() LatencyStats {
	lt.mu.Lock()
	defer lt.mu.Unlock()
	n := len(lt.samples)
	if n == 0 {
		return LatencyStats{}
	}
	sorted := append([]float64{}, lt.samples...)
	sort.Float64s(sorted)
	sum := 0.0
	for _, v := range sorted {
		sum += v
	}
	pick := func(p float64) float64 {
		idx := int(p * float64(n-1))
		if idx < 0 {
			idx = 0
		}
		if idx >= n {
			idx = n - 1
		}
		return sorted[idx]
	}
	return LatencyStats{
		Count: n,
		P50:   pick(0.50),
		P95:   pick(0.95),
		P99:   pick(0.99),
		Max:   sorted[n-1],
		Mean:  sum / float64(n),
	}
}
