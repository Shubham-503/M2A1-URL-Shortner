package main

import (
	"encoding/json"
	"fmt"
	"log"
	"math/rand"
	"net/http"
	"strings"
	"time"

	"github.com/gorilla/mux"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

var db *gorm.DB

// Define the URLShortener model
type URLShortener struct {
	ID             uint   `gorm:"primaryKey"`
	OriginalURL    string `gorm:"unique;size:2083;not null"`
	ShortCode      string `gorm:"unique;not null"`
	HitCount       uint   `gorm:"default:0"`
	CreatedAt      time.Time
	LastAccessedAt *time.Time
}

func initDB() error {
	var err error
	// Open SQLite database with GORM
	db, err = gorm.Open(sqlite.Open("url_shortener.db"), &gorm.Config{})
	if err != nil {
		return err
	}

	// Auto migrate the schema
	err = db.AutoMigrate(&URLShortener{})
	if err != nil {
		return err
	}

	return nil
}

// Handler to shorten URLs
func shortenHandler(w http.ResponseWriter, r *http.Request) {
	var request struct {
		LongURL string `json:"long_url"`
	}

	err := json.NewDecoder(r.Body).Decode(&request)
	if err != nil || request.LongURL == "" {
		http.Error(w, "Invalid request payload", http.StatusBadRequest)
		return
	}

	shortCode := generateShortCode(6)

	// Use GORM to insert the URL and short code into the database
	urlShortener := URLShortener{
		OriginalURL: request.LongURL,
		ShortCode:   shortCode,
	}
	result := db.Create(&urlShortener)
	// fmt.Print("result.Error.Error()", result.Error.Error())

	if result.Error != nil && strings.Contains(result.Error.Error(), "UNIQUE constraint failed") {
		fmt.Println("unique values")
		// Use GORM to query the original URL based on the short code
		result := db.Where("original_url = ?", urlShortener.OriginalURL).First(&urlShortener)
		if result.Error != nil {
			http.Error(w, "Short code not found", http.StatusNotFound)
			return

		}

		// Redirect the user to the original URL
		response := map[string]string{"short_code": urlShortener.ShortCode}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
		return
	}
	response := map[string]string{"short_code": shortCode}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

func generateShortCode(length int) string {
	const chars = "ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789"
	rand.Seed(time.Now().UnixNano()) // Seed the random number generator with the current time

	shortCode := make([]byte, length)
	for i := 0; i < length; i++ {
		shortCode[i] = chars[rand.Intn(len(chars))]
	}

	return string(shortCode)
}

func redirectHandler(w http.ResponseWriter, r *http.Request) {
	queryParams := r.URL.Query()
	shortCode := queryParams.Get("code")

	var urlShortener URLShortener
	// Use GORM to query the original URL based on the short code
	result := db.Where("short_code = ?", shortCode).First(&urlShortener)

	if result.Error != nil {
		http.Error(w, "Short code not found", http.StatusBadRequest)
		return
	}

	// increment hit_count and update last_accessed_at column
	// TODO: Try to use single db query
	result = db.Model(&URLShortener{}).Where("short_code = ?", shortCode).Update("last_accessed_at", time.Now()).Update("hit_count", urlShortener.HitCount+1)
	if result.Error != nil {
		fmt.Printf("Error in update: %s", result.Error.Error())
		return
	}

	// Redirect the user to the original URL
	response := map[string]string{"long_url": urlShortener.OriginalURL}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
	// http.Redirect(w, r, urlShortener.OriginalURL, http.StatusFound)
}

func deleteShortenHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	queryParams := r.URL.Query()
	shortCode := queryParams.Get("code")

	var urlShortener URLShortener

	result := db.Where("short_code = ?", shortCode).Delete(&urlShortener)

	if result.RowsAffected == 0 {
		response := map[string]string{"error": "short code not found"}
		w.WriteHeader(http.StatusNotFound)
		json.NewEncoder(w).Encode(response)
		return
	}

	if result.Error != nil {
		response := map[string]string{"error": result.Error.Error()}
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(response)
		return
	}

	response := map[string]string{"message": "short code deleted successfully"}
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(response)
}

func main() {
	err := initDB()
	if err != nil {
		log.Fatalf("Failed to initialize the database: %v", err)
	}

	// Initialize the router
	r := mux.NewRouter()
	r.HandleFunc("/shorten", shortenHandler).Methods("POST")
	r.HandleFunc("/redirect", deleteShortenHandler).Methods("DELETE")
	r.HandleFunc("/redirect", redirectHandler).Methods("GET")

	// Start the server
	fmt.Println("Server is running on http://localhost:8080")
	log.Fatal(http.ListenAndServe(":8080", r))
}
