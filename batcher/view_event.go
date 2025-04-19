package batcher

import "time"

// ViewEvent represents a “view” event on a product.
type ViewEvent struct {
	ProductID int
	Timestamp time.Time
}

// AggregatedCount maps product IDs to total view counts.
type AggregatedCount map[int]int
