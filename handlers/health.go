package handlers

import (
	"M2A1-URL-Shortner/config"
	"encoding/json"
	"net/http"
)

func HealthHandler(w http.ResponseWriter, r *http.Request) {
	sqliteDB, dbErr := config.DB.DB()
	if dbErr != nil {
		response := map[string]interface{}{
			"status":  "unhealthy",
			"message": "Database connectivity failed",
			"error":   dbErr.Error(),
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(response)
		return
	}

	if err := sqliteDB.Ping(); err != nil {
		response := map[string]interface{}{
			"status":  "unhealthy",
			"message": "Database connectivity failed",
			"error":   err.Error(),
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(response)
		return
	}

	response := map[string]interface{}{
		"status":  "healthy",
		"message": "Server and database are up and running",
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}
