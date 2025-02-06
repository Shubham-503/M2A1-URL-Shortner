package middlewares

import (
	"M2A1-URL-Shortner/config"
	"M2A1-URL-Shortner/models"
	"context"
	"fmt"
	"net/http"
	"time"
)

type contextKey string

const UserContextKey contextKey = "user"
const APIContextKey contextKey = "api_key"

func AuthenticateAPIKey(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		// Retrieve the API key from the request headers
		apiKey := r.Header.Get("api_key")
		fmt.Println("AuthenticateAPIKey called with api_key:", apiKey)
		if apiKey == "" {
			http.Error(w, "Please pass api_key", http.StatusUnauthorized)
			return
		}

		var user models.User
		result := config.DB.Model(&models.User{}).First(&user, "api_key = ?", apiKey)
		if result.Error != nil {
			http.Error(w, "DB Error", http.StatusInternalServerError)
			return
		}
		if result.RowsAffected == 0 {
			http.Error(w, "Please provide a valid api key", http.StatusUnauthorized)
			return
		}

		ctx := context.WithValue(r.Context(), UserContextKey, &user)
		ctx = context.WithValue(ctx, APIContextKey, apiKey)
		r = r.WithContext(ctx)

		next.ServeHTTP(w, r)
		AuditLogger.Printf("AuthenticateAPIKey Time Taken: %s", time.Since(start))

	})
}
