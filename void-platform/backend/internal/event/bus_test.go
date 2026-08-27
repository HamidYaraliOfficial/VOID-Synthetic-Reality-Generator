package event

import (
	"context"
	"sync/atomic"
	"testing"
	"time"
)

func TestBusDeliversToSubscribersConcurrently(t *testing.T) {
	bus := NewBus(1000, 4)
	var count atomic.Int64
	bus.Subscribe("login", func(ctx context.Context, e Event) error {
		count.Add(1)
		return nil
	})
	ctx, cancel := context.WithCancel(context.Background())
	bus.Start(ctx)
	for i := 0; i < 500; i++ {
		bus.Publish(Event{ID: "e", Type: "login", Timestamp: time.Now()})
	}
	deadline := time.Now().Add(2 * time.Second)
	for count.Load() < 500 && time.Now().Before(deadline) {
		time.Sleep(5 * time.Millisecond)
	}
	cancel()
	bus.Stop()
	if count.Load() != 500 {
		t.Fatalf("expected 500 delivered events, got %d", count.Load())
	}
}

func TestBusBackpressureDropsWhenFull(t *testing.T) {
	bus := NewBus(2, 0) // capacity 2, no workers draining -> queue fills fast
	ok1 := bus.Publish(Event{ID: "1"})
	ok2 := bus.Publish(Event{ID: "2"})
	ok3 := bus.Publish(Event{ID: "3"})
	if !ok1 || !ok2 {
		t.Fatalf("expected first two publishes to succeed")
	}
	if ok3 {
		t.Fatalf("expected third publish to be dropped under backpressure")
	}
	if bus.Stats().Dropped != 1 {
		t.Fatalf("expected 1 dropped event, got %d", bus.Stats().Dropped)
	}
}
