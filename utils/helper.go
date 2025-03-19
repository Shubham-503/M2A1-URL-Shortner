package utils

import (
	"M2A1-URL-Shortner/middlewares"
	"errors"
	"fmt"
	"math/rand"
	"net"
	"strings"
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
