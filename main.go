package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
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
	Password       *string `json:"password,omitempty"`
	ExpiredAt      *time.Time
	LastAccessedAt *time.Time
	DeletedAt      *time.Time
	UserID         uint
	User           User `gorm:"foreignKey:UserID;constraint:OnUpdate:CASCADE,OnDelete:CASCADE;"`
}

type User struct {
	ID        uint `gorm:"primaryKey;autoIncrement"`
	Email     string
	Name      string
	ApiKey    string `gorm:"unique"`
	Tier      string `gorm:"default:'hobby';check: tier IN ('hobby', 'enterprise')"`
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
		LongURL    string     `json:"long_url"`
		ExpiredAt  *time.Time `json:"expired_at"`
		CustomCode string     `json:"custom_code"`
		Password   *string    `json:"password,omitempty"`
	}

	// Retrieve the API key from the request headers
	apiKey := r.Header.Get("api_key")
	if apiKey == "" {
		http.Error(w, "Please pass api_key", http.StatusUnauthorized)
		return
	}

	var user User
	result := db.Model(&User{}).First(&user, "api_key = ?", apiKey)
	if result.RowsAffected == 0 {
		newUser := User{
			ApiKey: apiKey,
		}
		result = db.Model(&User{}).Create(&newUser)
		user = newUser
		fmt.Printf("userId:  %d\n", user.ID)
		if result.Error != nil {
			http.Error(w, "Error occured in creating user", http.StatusInternalServerError)
			return
		}
	}

	// Decode the JSON request body into the request struct
	err := json.NewDecoder(r.Body).Decode(&request)
	if err != nil || request.LongURL == "" {
		http.Error(w, "Invalid request payload", http.StatusBadRequest)
		return
	}

	// Check whether customCode is aval for not
	var shortCode string
	if request.CustomCode != "" {
		var urlShortenerExists URLShortener
		result := db.Model(&URLShortener{}).Where("short_code = ?", request.CustomCode).First(&urlShortenerExists)
		if result.RowsAffected != 0 {
			http.Error(w, "code already exists please try different code", http.StatusConflict)
			return
		}
		shortCode = request.CustomCode
	} else {
		// Generate a unique short code for the provided URL
		shortCode = generateShortCode(6)
	}

	// Create a new URLShortener record with the original URL, short code, and API key
	// TODO: Check if expired_at default value
	fmt.Printf("userId before url_shortner insertion:  %d\n", user.ID)
	fmt.Printf("userId before url_shortner insertion:  %v\n", request.Password)
	urlShortener := URLShortener{
		OriginalURL: request.LongURL,
		ShortCode:   shortCode,
		ApiKey:      apiKey,
		ExpiredAt:   request.ExpiredAt,
		UserID:      user.ID,
		Password:    request.Password,
	}

	// Save the URLShortener record to the database
	result = db.Create(&urlShortener)
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
		db.Model(&URLShortener{}).Where("id IN ?", ids).Update("shorten_count", currentLongUrlList[0].ShortenCount+1)
	}

	// Send the generated short code back as the response
	response := map[string]string{"short_code": shortCode}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

func shortenBulkHandler(w http.ResponseWriter, r *http.Request) {
	var request struct {
		URLs []struct {
			LongURL    string     `json:"long_url"`
			ExpiredAt  *time.Time `json:"expired_at"`
			CustomCode string     `json:"custom_code"`
			Password   *string    `json:"password,omitempty"`
		} `json:"urls"`
	}

	// Retrieve the API key from the request headers
	apiKey := r.Header.Get("api_key")
	if apiKey == "" {
		http.Error(w, "Please pass api_key", http.StatusUnauthorized)
		return
	}

	// Debug payload
	body, _ := io.ReadAll(r.Body)
	// fmt.Println("Raw Payload:", string(body))
	// Decode the JSON request body into the request struct
	// err := json.NewDecoder(r.Body).Decode(&request)
	err := json.NewDecoder(bytes.NewReader(body)).Decode(&request)
	if err != nil {
		fmt.Println("JSON Decoding Error:", err)
		http.Error(w, "Invalid request payload", http.StatusBadRequest)
		return
	}
	if len(request.URLs) == 0 {
		http.Error(w, "No URLs provided", http.StatusBadRequest)
		return
	}

	var user User
	result := db.Model(&User{}).First(&user, "api_key = ?", apiKey)
	if user.Tier != "enterprise" {
		http.Error(w, "Access denied: bulk creation is only available for enterprise users", http.StatusForbidden)
		return
	}
	if result.RowsAffected == 0 {
		newUser := User{
			ApiKey: apiKey,
		}
		result = db.Model(&User{}).Create(&newUser)
		user = newUser
		fmt.Printf("userId:  %d\n", user.ID)
		if result.Error != nil {
			http.Error(w, "Error occured in creating user", http.StatusInternalServerError)
			return
		}
	}

	var successes []map[string]string
	var errors []map[string]string

	for _, urlRequest := range request.URLs {
		if urlRequest.LongURL == "" {
			errors = append(errors, map[string]string{
				"long_url": urlRequest.LongURL,
				"error":    "Long URL is required",
			})
			continue
		}

		// Check whether customCode is aval for not
		var shortCode string
		if urlRequest.CustomCode != "" {
			var urlShortenerExists URLShortener
			result := db.Model(&URLShortener{}).Where("short_code = ?", urlRequest.CustomCode).First(&urlShortenerExists)
			if result.RowsAffected != 0 {
				errors = append(errors, map[string]string{
					"long_url": urlRequest.LongURL,
					"error":    "Short code already exists",
				})
				continue
			}
			shortCode = urlRequest.CustomCode
		} else {
			// Generate a unique short code for the provided URL
			shortCode = generateShortCode(6)
		}

		// Create a new URLShortener record with the original URL, short code, and API key
		// TODO: Check if expired_at default value
		urlShortener := URLShortener{
			OriginalURL: urlRequest.LongURL,
			ShortCode:   shortCode,
			ApiKey:      apiKey,
			ExpiredAt:   urlRequest.ExpiredAt,
			UserID:      user.ID,
			Password:    urlRequest.Password,
		}

		// Save the URLShortener record to the database
		result := db.Create(&urlShortener)
		if result.Error != nil {
			errors = append(errors, map[string]string{
				"long_url": urlRequest.LongURL,
				"error":    "Failed to save short code",
			})
			continue
		}

		// Retrieve all existing records in the database with the same original URL
		var currentLongUrlList []URLShortener
		// check if long_url already exists
		result = db.Model(&URLShortener{}).Find(&currentLongUrlList, "original_url = ?", urlShortener.OriginalURL)
		if result.Error != nil {
			errors = append(errors, map[string]string{
				"long_url": urlRequest.LongURL,
				"error":    "DB Error",
			})
			continue
		}
		// if url exists increment count for each row
		var ids []uint
		for _, record := range currentLongUrlList {
			ids = append(ids, record.ID)
		}
		if len(ids) > 1 {
			db.Model(&URLShortener{}).Where("id IN ?", ids).Update("shorten_count", gorm.Expr("shorten_count + ?", 1))
		}

		successes = append(successes, map[string]string{
			"long_url":   urlRequest.LongURL,
			"short_code": shortCode,
		})
	}
	response := map[string]interface{}{
		"success": successes,
		"errors":  errors,
	}
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
	password := queryParams.Get("password")

	var urlShortener URLShortener
	// Use GORM to query the original URL based on the short code
	result := db.Where("short_code = ?  AND deleted_at IS NULL", shortCode).First(&urlShortener)

	if urlShortener.Password != nil && *urlShortener.Password != password {
		http.Error(w, "Please pass password", http.StatusUnauthorized)
		return
	}

	if result.RowsAffected == 0 {
		http.Error(w, "Short code not found", http.StatusNotFound)
		return
	}

	if urlShortener.ExpiredAt != nil && urlShortener.ExpiredAt.Before(time.Now()) {
		http.Error(w, "Short code has expired", http.StatusGone)
		return
	}

	if result.Error != nil {
		fmt.Printf("error %s", result.Error.Error())
		http.Error(w, "DB error", http.StatusInternalServerError)
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

func editRedirectExpiryHandler(w http.ResponseWriter, r *http.Request) {
	var request struct {
		ExpiredAt *time.Time `json:"expired_at"`
		Password  *string    `json:"password,omitempty"`
	}

	w.Header().Set("Content-Type", "application/json")
	queryParams := r.URL.Query()
	shortCode := queryParams.Get("code")

	// Retrieve the API key from the request headers
	apiKey := r.Header.Get("api_key")
	if apiKey == "" {
		http.Error(w, "Please pass api_key", http.StatusUnauthorized)
		return
	}

	// Decode the JSON request body into the request struct
	err := json.NewDecoder(r.Body).Decode(&request)
	if err != nil || shortCode == "" || (request.ExpiredAt == nil && request.Password == nil) {
		http.Error(w, "Invalid request payload", http.StatusBadRequest)
		return
	}
	fmt.Printf("date : %s\n", request.ExpiredAt)
	fmt.Printf("apiKey : %s\n", apiKey)
	fmt.Printf("shortCode : %s\n", shortCode)

	updates := map[string]interface{}{}

	if request.ExpiredAt != nil {
		updates["expired_at"] = request.ExpiredAt
	}

	if request.Password != nil {
		updates["password"] = request.Password
	}

	if len(updates) > 0 {
		result := db.Model(&URLShortener{}).
			Where("short_code = ? AND api_key = ? AND deleted_at IS NULL", shortCode, apiKey).
			Updates(updates)

		if result.RowsAffected == 0 {
			http.Error(w, "No rows updated, check short code and API key", http.StatusNotFound)
			return
		}

		if result.Error != nil {
			http.Error(w, "Error in db", http.StatusInternalServerError)
			return
		}
	}

	// result := db.Model(&URLShortener{}).Where("short_code = ? AND api_key = ? AND deleted_at IS NULL", shortCode, apiKey).Update("expired_at", request.ExpiredAt).Update("password", request.Password)
	// if result.RowsAffected == 0 {
	// 	http.Error(w, "Error in db", http.StatusInternalServerError)
	// }
	// if result.Error != nil {
	// 	http.Error(w, "Error in db", http.StatusInternalServerError)
	// 	return
	// }

	response := map[string]string{"message": "Update Successfull"}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)

}

func getUserUrlsHandler(w http.ResponseWriter, r *http.Request) {
	apiKey := r.Header.Get("api_key")
	if apiKey == "" {
		http.Error(w, "Please pass api_key", http.StatusUnauthorized)
		return
	}
	var user User
	result := db.Model(&User{}).First(&user, "api_key = ?", apiKey)
	if result.Error != nil {
		http.Error(w, "Invalid API Key", http.StatusUnauthorized)
		return
	}

	var urls []URLShortener
	result = db.Model(&URLShortener{}).Where("user_id = ?", user.ID).Find(&urls)
	if result.Error != nil {
		http.Error(w, "Error fetching URLs", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(urls)

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
	r.HandleFunc("/redirect", editRedirectExpiryHandler).Methods("PATCH")
	r.HandleFunc("/shorten-bulk", shortenBulkHandler).Methods("POST")
	r.HandleFunc("/redirect", deleteShortenHandler).Methods("DELETE")
	r.HandleFunc("/redirect", redirectHandler).Methods("GET")
	r.HandleFunc("/users/url", getUserUrlsHandler).Methods("GET")

	// static path
	r.PathPrefix("/").Handler(http.FileServer(http.Dir(staticDir)))
	// r.PathPrefix("/static/").Handler(http.StripPrefix("/static/", fs))

	// Start the server
	fmt.Println("Server is running on http://localhost:8080")
	log.Fatal(http.ListenAndServe(":8080", r))
}
