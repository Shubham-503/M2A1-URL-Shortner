package batcher

import (
	"fmt"
	"sync"
	"time"
)

// TimeBatcher collects ViewEvents and flushes every flushInterval.
type TimeBatcher struct {
	mu            sync.Mutex
	events        []ViewEvent
	flushInterval time.Duration
	stopCh        chan struct{}
}

// NewTimeBatcher returns a TimeBatcher that flushes on the given interval.
// Pass flushInterval=0 to disable time‑based flushing.
func NewTimeBatcher(flushInterval time.Duration) *TimeBatcher {
	return &TimeBatcher{
		events:        []ViewEvent{},
		flushInterval: flushInterval,
		stopCh:        make(chan struct{}),
	}
}

// Start begins the background ticker.  Call Stop() to end it.
func (b *TimeBatcher) Start() {
	if b.flushInterval <= 0 {
		return
	}
	ticker := time.NewTicker(b.flushInterval)
	go func() {
		defer ticker.Stop()
		for {
			select {
			case <-ticker.C:
				b.mu.Lock()
				b.flushLocked()
				b.mu.Unlock()
			case <-b.stopCh:
				return
			}
		}
	}()
}

// Stop terminates the background ticker.
func (b *TimeBatcher) Stop() {
	close(b.stopCh)
}

// Enqueue adds an event; it will be included in the next flush.
func (b *TimeBatcher) Enqueue(evt ViewEvent) {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.events = append(b.events, evt)
}

// flushLocked assumes b.mu is held.
func (b *TimeBatcher) flushLocked() {
	if len(b.events) == 0 {
		return
	}
	agg := make(AggregatedCount)
	for _, e := range b.events {
		agg[e.ProductID]++
	}

	fmt.Printf("[%s] TimeBatcher flush: %d events → %v\n",
		time.Now().Format(time.RFC3339), len(b.events), agg,
	)
	b.events = b.events[:0]
}
