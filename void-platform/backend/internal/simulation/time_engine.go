package simulation

import (
	"sync"
	"sync/atomic"
	"time"
)

// TimeEngine implements the Time Simulation Engine: simulation time can run
// Real-Time (scale=1), Accelerated (scale>1, e.g. compress 30 days into a
// few minutes), Slowed (0<scale<1), Paused (scale=0) or fully Deterministic
// (never sleeps at all — used by tests that need bit-for-bit reproducible
// runs regardless of wall-clock speed).
type TimeEngine struct {
	mu          sync.RWMutex
	scale       float64
	deterministic bool
	simStart    time.Time
	wallStart   time.Time
	paused      atomic.Bool
}

func NewTimeEngine(scale float64) *TimeEngine {
	now := time.Now()
	return &TimeEngine{scale: scale, simStart: now, wallStart: now}
}

func (te *TimeEngine) SetScale(scale float64) {
	te.mu.Lock()
	defer te.mu.Unlock()
	// Re-anchor so changing scale mid-run doesn't jump simulated time.
	te.simStart = te.simNowLocked()
	te.wallStart = time.Now()
	te.scale = scale
}

func (te *TimeEngine) SetDeterministic(d bool) {
	te.mu.Lock()
	defer te.mu.Unlock()
	te.deterministic = d
}

func (te *TimeEngine) Pause()  { te.paused.Store(true) }
func (te *TimeEngine) Resume() { te.paused.Store(false) }
func (te *TimeEngine) IsPaused() bool { return te.paused.Load() }

func (te *TimeEngine) simNowLocked() time.Time {
	if te.paused.Load() {
		return te.simStart
	}
	elapsedWall := time.Since(te.wallStart)
	return te.simStart.Add(time.Duration(float64(elapsedWall) * te.scale))
}

// Now returns the current simulated timestamp.
func (te *TimeEngine) Now() time.Time {
	te.mu.RLock()
	defer te.mu.RUnlock()
	return te.simNowLocked()
}

// Sleep blocks for a simulated duration `d`, honouring the current scale:
// in Deterministic mode it returns immediately (no wall-clock cost at all),
// letting a 30-day scenario execute its full event sequence in milliseconds
// while every timestamp recorded on emitted events still reflects the
// correct simulated instant.
func (te *TimeEngine) Sleep(d time.Duration) {
	te.mu.RLock()
	scale := te.scale
	deterministic := te.deterministic
	te.mu.RUnlock()
	if deterministic || d <= 0 {
		return
	}
	if scale <= 0 {
		scale = 1
	}
	wallDelay := time.Duration(float64(d) / scale)
	// Cap any single sleep so pathological configs (very small scale) can't
	// stall a worker goroutine for an unreasonable amount of wall time.
	const maxSleep = 5 * time.Second
	if wallDelay > maxSleep {
		wallDelay = maxSleep
	}
	time.Sleep(wallDelay)
}

// Scale returns the current time multiplier.
func (te *TimeEngine) Scale() float64 {
	te.mu.RLock()
	defer te.mu.RUnlock()
	return te.scale
}
