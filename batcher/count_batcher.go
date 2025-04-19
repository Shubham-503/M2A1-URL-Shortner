package batcher

import (
	"fmt"
	"sync"
	"time"
)

// CountBatcher collects ViewEvents and flushes whenever count ≥ threshold.
type CountBatcher struct {
	mu        sync.Mutex
	events    []ViewEvent
	threshold int
}

// NewCountBatcher returns a CountBatcher that flushes when
// len(events) >= threshold.  Pass threshold=0 to disable.
func NewCountBatcher(threshold int) *CountBatcher {
	return &CountBatcher{
		events:    make([]ViewEvent, 0, threshold),
		threshold: threshold,
	}
}

// Enqueue adds an event and triggers flush if the threshold is reached.
func (b *CountBatcher) Enqueue(evt ViewEvent) {
	b.mu.Lock()
	defer b.mu.Unlock()

	b.events = append(b.events, evt)
	if b.threshold > 0 && len(b.events) >= b.threshold {
		b.flushLocked()
	}
}

// flushLocked assumes b.mu is held.
func (b *CountBatcher) flushLocked() {
	if len(b.events) == 0 {
		return
	}
	agg := make(AggregatedCount)
	for _, e := range b.events {
		agg[e.ProductID]++
	}

	fmt.Printf("[%s] CountBatcher flush: %d events → %v\n",
		time.Now().Format(time.RFC3339), len(b.events), agg,
	)
	b.events = b.events[:0]
}
