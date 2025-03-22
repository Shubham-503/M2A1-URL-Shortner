package utils

import (
	"M2A1-URL-Shortner/middlewares"
	"errors"
	"fmt"
	"math/rand"
	"net"
	"strings"
	"sync"
	"time"
)

func GenerateShortCode(length int) string {
	const chars = "ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789"
	rand.Seed(time.Now().UnixNano()) // Seed the random number generator with the current time

	shortCode := make([]byte, length)
	for i := 0; i < length; i++ {
		shortCode[i] = chars[rand.Intn(len(chars))]
	}

	return string(shortCode)
}

func isRecoverableError(err error) bool {
	fmt.Println("func isRecoverableError called")
	// return true
	// Check if the error is a network error that is temporary or due to a timeout.
	var netErr net.Error
	if errors.As(err, &netErr) {
		if netErr.Timeout() {
			return true
		}
	}

	msg := strings.ToLower(err.Error())
	if strings.Contains(msg, "timeout") || strings.Contains(msg, "temporarily unavailable") {
		return true
	}
	return false
}

func RetryWithExponentialBackoff(operation func() error, maxRetries int, initialDelay time.Duration) error {
	delay := initialDelay
	var err error

	rand.Seed(time.Now().UnixNano())

	for i := 0; i < maxRetries; i++ {
		err = operation()
		if err == nil {
			return nil
		}

		if !isRecoverableError(err) {
			return err
		}

		// Log the error and attempt number
		fmt.Printf("Attempt %d failed: %v. Retrying in %v...\n", i+1, err, delay)
		middlewares.AuditLogger.Printf("Attempt %d failed: %v. Retrying in %v...\n", i+1, err, delay)

		// Apply jitter: add a random duration between 0 and half the current delay.
		jitter := time.Duration(rand.Int63n(int64(delay / 2)))
		time.Sleep(delay + jitter)
		delay *= 2 // Exponential backoff.

	}
	// After exhausting retries, return an error wrapping the last failure.
	middlewares.AuditLogger.Printf("operation failed after %d attempts: %v", maxRetries, err)
	return fmt.Errorf("operation failed after %d attempts: %w", maxRetries, err)
}

// Retry with CircuitBreaker code start from here

// ErrCircuitOpen is returned when the circuit breaker is open.
var ErrCircuitOpen = errors.New("circuit breaker is open")

// CircuitBreaker implements a simple circuit breaker.
type CircuitBreaker struct {
	failureCount int
	threshold    int
	open         bool
	lastFailure  time.Time
	resetTimeout time.Duration
	lock         sync.Mutex
}

// NewCircuitBreaker creates a new circuit breaker with the given failure threshold and reset timeout.
func NewCircuitBreaker(threshold int, resetTimeout time.Duration) *CircuitBreaker {
	return &CircuitBreaker{
		threshold:    threshold,
		resetTimeout: resetTimeout,
	}
}

// Allow returns nil if the operation is allowed; otherwise, it returns ErrCircuitOpen.
func (cb *CircuitBreaker) Allow() error {
	cb.lock.Lock()
	defer cb.lock.Unlock()
	if cb.open {
		// If the circuit is open, check if resetTimeout has passed.
		if time.Since(cb.lastFailure) > cb.resetTimeout {
			// Transition to half-open state.
			cb.open = false
			cb.failureCount = 0
			return nil
		}
		return ErrCircuitOpen
	}
	return nil
}

// Success resets the circuit breaker after a successful operation.
func (cb *CircuitBreaker) Success() {
	cb.lock.Lock()
	defer cb.lock.Unlock()
	cb.failureCount = 0
	cb.open = false
}

// Failure increments the failure count and opens the circuit if the threshold is reached.
func (cb *CircuitBreaker) Failure() {
	cb.lock.Lock()
	defer cb.lock.Unlock()
	cb.failureCount++
	cb.lastFailure = time.Now()
	if cb.failureCount >= cb.threshold {
		cb.open = true
	}
}

// RetryWithCircuitBreaker retries the provided operation using exponential backoff with jitter.
// It uses a circuit breaker that opens after a threshold of consecutive failures and resets after a timeout.
func RetryWithCircuitBreaker(cb *CircuitBreaker, operation func() error, maxRetries int, initialDelay time.Duration) error {
	delay := initialDelay
	var err error
	rand.Seed(time.Now().UnixNano())

	for i := 0; i < maxRetries; i++ {
		// Check circuit breaker status.
		if err = cb.Allow(); err != nil {
			return err
		}

		err = operation()
		if err == nil {
			cb.Success()
			return nil
		}

		// If the error is not recoverable stop retrying.
		if !isRecoverableError(err) {
			return err
		}

		// Record failure in the circuit breaker.
		cb.Failure()

		// Log the error and attempt number
		fmt.Printf("Attempt %d failed: %v. Retrying in %v...\n", i+1, err, delay)
		middlewares.AuditLogger.Printf("Attempt %d failed: %v. Retrying in %v...\n", i+1, err, delay)

		// Apply jitter: add a random duration between 0 and half the current delay.
		jitter := time.Duration(rand.Int63n(int64(delay / 2)))
		time.Sleep(delay + jitter)
		delay *= 2 // Exponential backoff.

	}
	// After exhausting retries, return an error wrapping the last failure.
	middlewares.AuditLogger.Printf("operation failed after %d attempts: %v", maxRetries, err)
	return fmt.Errorf("operation failed after %d attempts: %w", maxRetries, err)
}
