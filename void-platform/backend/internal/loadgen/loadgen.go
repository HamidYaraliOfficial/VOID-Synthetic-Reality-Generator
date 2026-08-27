// Package loadgen implements the Traffic & Load Generator / Synthetic API
// Traffic Studio. It ONLY ever sends requests to a Target explicitly
// configured by the operator (Authorized must be true) and never spoofs,
// amplifies, or targets endpoints it wasn't pointed at — this is a load and
// performance testing tool for systems the user owns or is authorized to
// test, not an attack tool.
package loadgen

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net/http"
	"sync"
	"sync/atomic"
	"time"

	"void-platform/backend/internal/metrics"
)

// Target fully describes one authorized synthetic-traffic destination.
type Target struct {
	Authorized bool              `json:"authorized"` // MUST be true; set explicitly by the operator
	Method     string            `json:"method"`
	URL        string            `json:"url"`
	Headers    map[string]string `json:"headers,omitempty"`
	BodyTemplate string          `json:"bodyTemplate,omitempty"`
}

// Profile configures a load-generation run: virtual users, request rate,
// ramp-up/down and total duration.
type Profile struct {
	VirtualUsers int           `json:"virtualUsers"`
	RequestRate  float64       `json:"requestRatePerSecond"` // 0 = as-fast-as-VUs-allow, capped by VirtualUsers
	RampUp       time.Duration `json:"rampUp"`
	RampDown     time.Duration `json:"rampDown"`
	Duration     time.Duration `json:"duration"`
	Timeout      time.Duration `json:"timeout"`
}

// LiveStats is a real-time snapshot published while a run is active.
type LiveStats struct {
	Requests    int64              `json:"requests"`
	Successes   int64              `json:"successes"`
	Failures    int64              `json:"failures"`
	ActiveVUs   int32              `json:"activeVUs"`
	RPS         float64            `json:"rps"`
	Latency     metrics.LatencyStats `json:"latency"`
	ElapsedSec  float64            `json:"elapsedSeconds"`
}

// Runner executes one load-generation Profile against one Target.
type Runner struct {
	Target  Target
	Profile Profile
	client  *http.Client
	tracker *metrics.LatencyTracker

	requests  atomic.Int64
	successes atomic.Int64
	failures  atomic.Int64
	activeVUs atomic.Int32
	started   time.Time
}

func NewRunner(target Target, profile Profile) (*Runner, error) {
	if !target.Authorized {
		return nil, fmt.Errorf("loadgen: target not marked Authorized — refusing to generate traffic against %q", target.URL)
	}
	if target.URL == "" {
		return nil, fmt.Errorf("loadgen: target URL is required")
	}
	if profile.Timeout <= 0 {
		profile.Timeout = 10 * time.Second
	}
	return &Runner{
		Target:  target,
		Profile: profile,
		client:  &http.Client{Timeout: profile.Timeout},
		tracker: metrics.NewLatencyTracker(20000),
	}, nil
}

// Run drives virtual users for Profile.Duration, honestly ramping up/down,
// and calls onTick periodically with a LiveStats snapshot for the UI.
func (r *Runner) Run(ctx context.Context, onTick func(LiveStats)) error {
	r.started = time.Now()
	runCtx, cancel := context.WithTimeout(ctx, r.Profile.Duration+r.Profile.RampUp+r.Profile.RampDown)
	defer cancel()

	var wg sync.WaitGroup
	vus := r.Profile.VirtualUsers
	if vus <= 0 {
		vus = 1
	}
	rampStep := time.Duration(0)
	if vus > 0 && r.Profile.RampUp > 0 {
		rampStep = r.Profile.RampUp / time.Duration(vus)
	}

	var minInterval time.Duration
	if r.Profile.RequestRate > 0 {
		minInterval = time.Duration(float64(time.Second) / r.Profile.RequestRate)
	}

	for i := 0; i < vus; i++ {
		wg.Add(1)
		delay := time.Duration(i) * rampStep
		go func(startDelay time.Duration) {
			defer wg.Done()
			timer := time.NewTimer(startDelay)
			defer timer.Stop()
			select {
			case <-runCtx.Done():
				return
			case <-timer.C:
			}
			r.activeVUs.Add(1)
			defer r.activeVUs.Add(-1)
			r.vuLoop(runCtx, minInterval)
		}(delay)
	}

	ticker := time.NewTicker(500 * time.Millisecond)
	defer ticker.Stop()
	done := make(chan struct{})
	go func() { wg.Wait(); close(done) }()

	for {
		select {
		case <-ticker.C:
			if onTick != nil {
				onTick(r.snapshot())
			}
		case <-done:
			if onTick != nil {
				onTick(r.snapshot())
			}
			return nil
		case <-ctx.Done():
			return ctx.Err()
		}
	}
}

func (r *Runner) vuLoop(ctx context.Context, minInterval time.Duration) {
	for {
		select {
		case <-ctx.Done():
			return
		default:
		}
		start := time.Now()
		r.doRequest(ctx)
		r.requests.Add(1)
		elapsed := time.Since(start)
		if minInterval > elapsed {
			select {
			case <-time.After(minInterval - elapsed):
			case <-ctx.Done():
				return
			}
		}
	}
}

func (r *Runner) doRequest(ctx context.Context) {
	method := r.Target.Method
	if method == "" {
		method = http.MethodGet
	}
	var body io.Reader
	if r.Target.BodyTemplate != "" {
		body = bytes.NewBufferString(r.Target.BodyTemplate)
	}
	req, err := http.NewRequestWithContext(ctx, method, r.Target.URL, body)
	if err != nil {
		r.failures.Add(1)
		return
	}
	for k, v := range r.Target.Headers {
		req.Header.Set(k, v)
	}
	start := time.Now()
	resp, err := r.client.Do(req)
	latencyMs := float64(time.Since(start).Microseconds()) / 1000.0
	r.tracker.Add(latencyMs)
	if err != nil {
		r.failures.Add(1)
		return
	}
	defer resp.Body.Close()
	_, _ = io.Copy(io.Discard, resp.Body)
	if resp.StatusCode >= 200 && resp.StatusCode < 400 {
		r.successes.Add(1)
	} else {
		r.failures.Add(1)
	}
}

func (r *Runner) snapshot() LiveStats {
	elapsed := time.Since(r.started).Seconds()
	reqs := r.requests.Load()
	rps := 0.0
	if elapsed > 0 {
		rps = float64(reqs) / elapsed
	}
	return LiveStats{
		Requests:   reqs,
		Successes:  r.successes.Load(),
		Failures:   r.failures.Load(),
		ActiveVUs:  r.activeVUs.Load(),
		RPS:        rps,
		Latency:    r.tracker.Stats(),
		ElapsedSec: elapsed,
	}
}
