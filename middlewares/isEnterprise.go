package middlewares

import (
	"M2A1-URL-Shortner/config"
	"M2A1-URL-Shortner/models"
	"fmt"
	"net/http"
	"time"
)

func IsEnterprise(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		fmt.Printf("IsEnterprise Called:")
		apiKey := r.Header.Get("api_key")
		if apiKey == "" {
			http.Error(w, "Please pass api_key", http.StatusUnauthorized)
			return
		}
		var user models.User
		result := config.DB.Model(&models.User{}).First(&user, "api_key = ?", apiKey)
		if result.RowsAffected == 0 {
			http.Error(w, "Please provide a valid api key", http.StatusUnauthorized)
			return
		}
		if result.Error != nil {
			http.Error(w, "DB Error", http.StatusInternalServerError)
			return
		}
		if user.Tier != "enterprise" {
			http.Error(w, "Access denied: bulk creation is only available for enterprise users", http.StatusForbidden)
			return
		}
		fmt.Printf("time before next handler: %s", start)
		next.ServeHTTP(w, r)
		end := time.Now()
		w.Header().Add("total-time", end.String())
		r.Header.Set("total-time", end.String())
		fmt.Printf("time after next : %s", end)

		AuditLogger.Printf("IsEnterprise Time Taken: %s", time.Since(start))

	})
}
