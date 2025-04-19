package queue

import (
	"testing"
	"time"
)

func TestTaskQueueProcessing(t *testing.T) {
	// Replace global queue for test
	TaskQueue = make(chan func(), 1)
	processed := false

	StartWorker()
	TaskQueue <- func() { processed = true }

	// allow some time for the worker to pick up the task
	time.Sleep(10 * time.Millisecond)

	if !processed {
		t.Error("expected task to be processed")
	}
}
