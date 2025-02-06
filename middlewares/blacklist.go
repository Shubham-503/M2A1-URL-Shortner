package middlewares

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"sync"
	"time"
)

var (
	blacklistedKeys map[string]bool
	blacklistLock   sync.RWMutex
)

func LoadBlacklist(filePath string) error {
	file, err := os.Open(filePath)
	if err != nil {
		return err
	}
	defer file.Close()

	var data struct {
		Blacklisted []string `json:"blacklisted_api_keys"`
	}

	if err := json.NewDecoder(file).Decode(&data); err != nil {
		return err
	}

	// Create a map for fast lookups.
	temp := make(map[string]bool)
	for _, key := range data.Blacklisted {
		temp[key] = true
	}

	// Use a write lock to update the global blacklist.
	blacklistLock.Lock()
	blacklistedKeys = temp
	blacklistLock.Unlock()

	return nil
}

func BlacklistMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		fmt.Println("Blacklisted middleware called")
		apiKey := r.Header.Get("api_key")
		if apiKey == "" {
			http.Error(w, "Missing API key", http.StatusUnauthorized)
			return
		}

		// Load the API key blacklist from the config file.
		if err := LoadBlacklist("config/blacklist.json"); err != nil {
			log.Printf("Error loading blacklist: %v", err)
		}

		// Use a read lock to safely access the blacklist.
		blacklistLock.RLock()
		isBlacklisted := blacklistedKeys[apiKey]
		blacklistLock.RUnlock()

		if isBlacklisted {
			http.Error(w, "Forbidden: API key is blacklisted", http.StatusForbidden)
			return
		}

		// Proceed with the next handler if the API key is not blacklisted.
		next.ServeHTTP(w, r)

		AuditLogger.Printf("BlacklistMiddleware Time Taken: %s", time.Since(start))
	})
}
