package queue

// TaskQueue is a buffered in-memory queue of functions to run.
var TaskQueue = make(chan func(), 100)

// StartWorker launches a background goroutine that processes queued tasks.
func StartWorker() {
	go func() {
		for task := range TaskQueue {
			task()
		}
	}()
}
