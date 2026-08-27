package event

import (
	"context"
	"sync"
	"sync/atomic"
)

// Handler processes one Event. Returning an error only logs/increments a
// counter; it never stops the Bus, matching the "chaos-tolerant" nature of a
// simulation engine that must keep running through injected failures.
type Handler func(ctx context.Context, e Event) error

// Bus is a fan-out, worker-pool backed event bus with bounded backpressure.
// Producers call Publish; a fixed pool of goroutines drains the channel and
// invokes every subscribed Handler for the event's Type (plus any "*"
// wildcard subscribers used by metrics/logging).
type Bus struct {
	queue       chan Event
	workers     int
	mu          sync.RWMutex
	subscribers map[string][]Handler
	wildcard    []Handler

	published atomic.Int64
	processed atomic.Int64
	dropped   atomic.Int64
	errors    atomic.Int64

	wg     sync.WaitGroup
	cancel context.CancelFunc
}

// NewBus builds a Bus with the given channel capacity (backpressure buffer)
// and worker pool size (parallel Goroutines consuming events).
func NewBus(capacity, workers int) *Bus {
	if capacity <= 0 {
		capacity = 10000
	}
	if workers <= 0 {
		workers = 8
	}
	return &Bus{
		queue:       make(chan Event, capacity),
		workers:     workers,
		subscribers: make(map[string][]Handler),
	}
}

// Subscribe registers a Handler for a specific event Type, or "*" for all.
func (b *Bus) Subscribe(eventType string, h Handler) {
	b.mu.Lock()
	defer b.mu.Unlock()
	if eventType == "*" {
		b.wildcard = append(b.wildcard, h)
		return
	}
	b.subscribers[eventType] = append(b.subscribers[eventType], h)
}

// Start launches the worker pool. Safe to call once per Bus lifetime.
func (b *Bus) Start(ctx context.Context) {
	ctx, cancel := context.WithCancel(ctx)
	b.cancel = cancel
	for i := 0; i < b.workers; i++ {
		b.wg.Add(1)
		go b.worker(ctx)
	}
}

// Stop signals all workers to exit and waits for in-flight events to drain.
func (b *Bus) Stop() {
	if b.cancel != nil {
		b.cancel()
	}
	b.wg.Wait()
}

func (b *Bus) worker(ctx context.Context) {
	defer b.wg.Done()
	for {
		select {
		case <-ctx.Done():
			return
		case e, ok := <-b.queue:
			if !ok {
				return
			}
			b.dispatch(ctx, e)
		}
	}
}

func (b *Bus) dispatch(ctx context.Context, e Event) {
	b.mu.RLock()
	handlers := append([]Handler{}, b.subscribers[e.Type]...)
	wildcard := append([]Handler{}, b.wildcard...)
	b.mu.RUnlock()

	for _, h := range wildcard {
		if err := h(ctx, e); err != nil {
			b.errors.Add(1)
		}
	}
	for _, h := range handlers {
		if err := h(ctx, e); err != nil {
			b.errors.Add(1)
		}
	}
	b.processed.Add(1)
}

// Publish enqueues an event. If the internal buffer is full the event is
// dropped and counted (documented backpressure behaviour) rather than
// blocking the producer goroutine indefinitely.
func (b *Bus) Publish(e Event) bool {
	b.published.Add(1)
	select {
	case b.queue <- e:
		return true
	default:
		b.dropped.Add(1)
		return false
	}
}

// PublishBlocking enqueues an event, blocking until there is room or ctx is
// done. Useful for callers that would rather slow down than drop events.
func (b *Bus) PublishBlocking(ctx context.Context, e Event) bool {
	b.published.Add(1)
	select {
	case b.queue <- e:
		return true
	case <-ctx.Done():
		b.dropped.Add(1)
		return false
	}
}

// Stats is a point-in-time snapshot of Bus throughput counters.
type Stats struct {
	Published int64 `json:"published"`
	Processed int64 `json:"processed"`
	Dropped   int64 `json:"dropped"`
	Errors    int64 `json:"errors"`
	QueueLen  int   `json:"queueLen"`
	QueueCap  int   `json:"queueCap"`
	Workers   int   `json:"workers"`
}

func (b *Bus) Stats() Stats {
	return Stats{
		Published: b.published.Load(),
		Processed: b.processed.Load(),
		Dropped:   b.dropped.Load(),
		Errors:    b.errors.Load(),
		QueueLen:  len(b.queue),
		QueueCap:  cap(b.queue),
		Workers:   b.workers,
	}
}
