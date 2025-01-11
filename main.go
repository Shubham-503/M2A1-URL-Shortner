package main

import (
	"encoding/json"
	"fmt"
	"log"
	"math/rand"
	"net/http"
	"time"

	"github.com/gorilla/mux"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

var db *gorm.DB

// Define the URLShortener model
type URLShortener struct {
	ID             uint   `gorm:"primaryKey"`
	OriginalURL    string `gorm:"size:2083;not null"`
	ShortCode      string `gorm:"unique;not null"`
	HitCount       uint   `gorm:"default:0"`
	ShortenCount   uint   `gorm:"default:1"`
	CreatedAt      time.Time
	ApiKey         string
	LastAccessedAt *time.Time
	DeletedAt      *time.Time
	UserID         uint
	User           User `gorm:"foreignKey:UserID;constraint:OnUpdate:CASCADE,OnDelete:CASCADE;"`
}

type User struct {
	ID        uint   `gorm:"primaryKey;autoIncrement"`
	Email     string `gorm:"unique"`
	Name      string
	ApiKey    string
	CreatedAt time.Time
}

func initDB() error {
	var err error
	// Open SQLite database with GORM
	db, err = gorm.Open(sqlite.Open("url_shortener.db"), &gorm.Config{})
	if err != nil {
		return err
	}

	// Auto migrate the schema
	err = db.AutoMigrate(&URLShortener{}, &User{})
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

	// Retrieve the API key from the request headers
	apiKey := r.Header.Get("api_key")
	if apiKey == "" {
		http.Error(w, "Please pass api_key", http.StatusUnauthorized)
		return
	}

	// var user User
	// result := db.Model(&User{}).First(&user, "api_key = ?", apiKey)
	// if result.Error != nil {
	// 	user := User{
	// 		ApiKey: apiKey,
	// 	}
	// 	result = db.Model(&User{}).Create(&user)
	// 	if result.Error != nil {
	// 		http.Error(w, "Error occured in creating user", http.StatusInternalServerError)
	// 		return
	// 	}
	// }

	// Decode the JSON request body into the request struct
	err := json.NewDecoder(r.Body).Decode(&request)
	if err != nil || request.LongURL == "" {
		http.Error(w, "Invalid request payload", http.StatusBadRequest)
		return
	}

	// Generate a unique short code for the provided URL
	shortCode := generateShortCode(6)

	// Create a new URLShortener record with the original URL, short code, and API key
	urlShortener := URLShortener{
		OriginalURL: request.LongURL,
		ShortCode:   shortCode,
		ApiKey:      apiKey,
		// UserID:      user.ID,
	}

	// Save the URLShortener record to the database
	result := db.Create(&urlShortener)
	if result.Error != nil {
		http.Error(w, "Error in saving", http.StatusInternalServerError)
		return
	}

	// Retrieve all existing records in the database with the same original URL
	var currentLongUrlList []URLShortener
	// check if long_url already exists
	result = db.Model(&URLShortener{}).Find(&currentLongUrlList, "original_url = ?", urlShortener.OriginalURL)
	if result.Error != nil {
		http.Error(w, "Error in saving", http.StatusInternalServerError)
		return
	}
	// if url exists increment count for each row
	var ids []uint
	for _, record := range currentLongUrlList {
		ids = append(ids, record.ID)
	}
	if len(ids) > 1 {
		db.Model(&URLShortener{}).Where("id IN ?", ids).Update("shorten_count", gorm.Expr("shorten_count + ?", 1))
	}

	// Send the generated short code back as the response
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
	result := db.Where("short_code = ? AND deleted_at IS NULL", shortCode).First(&urlShortener)

	if result.Error != nil {
		http.Error(w, "Short code not found", http.StatusBadRequest)
		return
	}

	// increment hit_count and update last_accessed_at column
	// TODO: Try to use single db query
	result = db.Model(&URLShortener{}).Where("short_code = ? AND deleted_at IS NULL", shortCode).Update("last_accessed_at", time.Now()).Update("hit_count", urlShortener.HitCount+1)
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

	// var urlShortener URLShortener

	// Retrieve the API key from the request headers
	apiKey := r.Header.Get("api_key")
	if apiKey == "" {
		http.Error(w, "Please pass api_key", http.StatusUnauthorized)
		return
	}

	result := db.Model(&URLShortener{}).Where("short_code = ? AND api_key = ? AND deleted_at IS NULL", shortCode, apiKey).Update("deleted_at", time.Now())
	fmt.Printf("short_code and api_key is: %s,  %s\n", shortCode, apiKey)
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

	// Serve static files using mux
	staticDir := "./static/"
	// fs := http.FileServer(http.Dir(staticDir))

	// Initialize the router
	r := mux.NewRouter()
	r.HandleFunc("/shorten", shortenHandler).Methods("POST")
	r.HandleFunc("/redirect", deleteShortenHandler).Methods("DELETE")
	r.HandleFunc("/redirect", redirectHandler).Methods("GET")

	// static path
	r.PathPrefix("/").Handler(http.FileServer(http.Dir(staticDir)))
	// r.PathPrefix("/static/").Handler(http.StripPrefix("/static/", fs))

	// Start the server
	fmt.Println("Server is running on http://localhost:8080")
	log.Fatal(http.ListenAndServe(":8080", r))
}
