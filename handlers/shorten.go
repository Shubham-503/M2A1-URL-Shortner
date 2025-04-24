package handlers

import (
	"M2A1-URL-Shortner/cache"
	"M2A1-URL-Shortner/middlewares"
	"M2A1-URL-Shortner/models"
	"M2A1-URL-Shortner/pubsub"
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"strconv"
	"time"

	"M2A1-URL-Shortner/config"

	"M2A1-URL-Shortner/utils"

	"gorm.io/gorm"
)

var URLCache cache.RedisURLCache
var PS *pubsub.PubSub

// Handler to shorten URLs
func ShortenHandler(w http.ResponseWriter, r *http.Request) {
	var request struct {
		LongURL    string     `json:"long_url"`
		ExpiredAt  *time.Time `json:"expired_at"`
		CustomCode string     `json:"custom_code"`
		Password   *string    `json:"password,omitempty"`
	}

	// var user models.User
	// Retrieve the user from the context
	user, ok := r.Context().Value(middlewares.UserContextKey).(*models.User)
	if !ok || user == nil {
		http.Error(w, "User not found in context", http.StatusInternalServerError)
		return
	}
	apiKey, ok := r.Context().Value(middlewares.APIContextKey).(string)
	if !ok || apiKey == "" {
		http.Error(w, "apiKey not found in context", http.StatusInternalServerError)
		return
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
		var urlShortenerExists models.URLShortener
		result := config.DB.Model(&models.URLShortener{}).Where("short_code = ?", request.CustomCode).First(&urlShortenerExists)
		if result.RowsAffected != 0 {
			http.Error(w, "code already exists please try different code", http.StatusConflict)
			return
		}
		shortCode = request.CustomCode
	} else {
		// Generate a unique short code for the provided URL
		shortCode = utils.GenerateShortCode(6)
	}

	// Create a new URLShortener record with the original URL, short code, and API key
	// TODO: Check if expired_at default value
	fmt.Printf("userId before url_shortner insertion:  %d\n", user.ID)
	fmt.Printf("request.Password before url_shortner insertion:  %v\n", request.Password)
	urlShortener := models.URLShortener{
		OriginalURL: request.LongURL,
		ShortCode:   shortCode,
		ApiKey:      apiKey,
		ExpiredAt:   request.ExpiredAt,
		UserID:      user.ID,
		Password:    request.Password,
	}

	// Save the URLShortener record to the database
	result := config.DB.Create(&urlShortener)
	if result.Error != nil {
		http.Error(w, "Error in saving", http.StatusInternalServerError)
		return
	}

	// Retrieve all existing records in the database with the same original URL
	var currentLongUrlList []models.URLShortener
	// check if long_url already exists
	result = config.DB.Model(&models.URLShortener{}).Find(&currentLongUrlList, "original_url = ?", urlShortener.OriginalURL)
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
		config.DB.Model(&models.URLShortener{}).Where("id IN ?", ids).Update("shorten_count", currentLongUrlList[0].ShortenCount+1)
	}

	// Send the generated short code back as the response
	response := map[string]string{"short_code": shortCode}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

func RedirectHandler(w http.ResponseWriter, r *http.Request) {
	fmt.Println("redirect handler called")
	queryParams := r.URL.Query()
	shortCode := queryParams.Get("code")
	password := queryParams.Get("password")

	var urlShortener models.URLShortener
	// Set up a circuit breaker that opens after 3 failures and resets after 10 seconds.
	cb := utils.NewCircuitBreaker(3, 10*time.Second)

	// 1. Check Cache First
	if data, err := URLCache.Get(shortCode); err == nil {
		// Cache hit: Decode JSON into struct
		if data.Password != nil && *data.Password != password {
			http.Error(w, "Please pass password", http.StatusUnauthorized)
			return
		}
		if data.ExpiredAt != nil && data.ExpiredAt.Before(time.Now()) {
			http.Error(w, "Short code has expired", http.StatusGone)
			return
		}

		response := map[string]string{"long_url": data.OriginalURL}
		w.Header().Set("Content-Type", "application/json")
		// Set header to indicate a cache hit.
		w.Header().Set("X-Cache", "HIT")
		w.Header().Set("Cache-Control", "public, max-age=86400")
		// w.Header().Set("Cache-Control", "no-store, no-cache, must-revalidate, proxy-revalidate")
		// w.Header().Set("Pragma", "no-cache")
		// w.Header().Set("Expires", "0")
		json.NewEncoder(w).Encode(response)
	} else {
		// Use GORM to query the original URL based on the short code
		// result := config.DB.Model(&models.URLShortener{}).Where("short_code = ?  AND deleted_at IS NULL", shortCode).First(&urlShortener)
		// if result.Error != nil {
		// 	fmt.Println("errors ::")
		// 	fmt.Print(result.Error.Error())
		// }
		var result *gorm.DB
		fetchOp := func() error {
			result := config.DB.
				Model(&models.URLShortener{}).
				Where("short_code = ? AND deleted_at IS NULL", shortCode).
				First(&urlShortener)
			return result.Error
		}

		maxRetries := 3
		initialDelay := 100 * time.Millisecond
		if err := utils.RetryWithCircuitBreaker(cb, fetchOp, maxRetries, initialDelay); err != nil {
			fmt.Printf("Error fetching record from DB: %v\n", err)
			http.Error(w, "DB error", http.StatusInternalServerError)
			return
		}

		URLCache.Set(shortCode, urlShortener)

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
		// TODO: Try to use single config.DB query
		// TODO: Update last_accessed_at and hit-count for cache hit too
		result = config.DB.Model(&models.URLShortener{}).Where("short_code = ? AND deleted_at IS NULL", shortCode).Update("last_accessed_at", time.Now()).Update("hit_count", urlShortener.HitCount+1)
		if result.Error != nil {
			fmt.Printf("Error in update: %s", result.Error.Error())
			return
		}

		updateOp := func() error {
			result := config.DB.Model(&models.URLShortener{}).Where("short_code = ? AND deleted_at IS NULL", shortCode).Update("last_accessed_at", time.Now()).Update("hit_count", urlShortener.HitCount+1)
			return result.Error
		}
		if err := utils.RetryWithCircuitBreaker(cb, updateOp, maxRetries, initialDelay); err != nil {
			fmt.Printf("Error updating record in DB: %v\n", err)
			http.Error(w, "DB update error", http.StatusInternalServerError)
			return
		}

		// Redirect the user to the original URL
		response := map[string]string{"long_url": urlShortener.OriginalURL}
		w.Header().Set("Content-Type", "application/json")
		// Set header to indicate a cache miss.
		w.Header().Set("X-Cache", "MISS")
		w.Header().Set("Cache-Control", "public, max-age=86400")
		// w.Header().Set("Cache-Control", "no-store, no-cache, must-revalidate, proxy-revalidate")
		// w.Header().Set("Pragma", "no-cache")
		// w.Header().Set("Expires", "0")
		json.NewEncoder(w).Encode(response)
		// http.Redirect(w, r, urlShortener.OriginalURL, http.StatusFound)
	}
}

func EditRedirectExpiryHandler(w http.ResponseWriter, r *http.Request) {
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
		result := config.DB.Model(&models.URLShortener{}).
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
		var urlShortener models.URLShortener
		err := config.DB.Model(&models.URLShortener{}).Where("short_code = ? AND api_key = ? AND deleted_at IS NULL", shortCode, apiKey).First(&urlShortener)
		if err != nil {
			http.Error(w, "Error in db", http.StatusInternalServerError)
			return
		}
		URLCache.Set(shortCode, urlShortener)
	}

	response := map[string]string{"message": "Update Successfull"}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)

}

func ShortenBulkHandler(w http.ResponseWriter, r *http.Request) {
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

	var user models.User
	result := config.DB.Model(&models.User{}).First(&user, "api_key = ?", apiKey)
	fmt.Print(result)
	if result.RowsAffected == 0 {
		http.Error(w, "Please provide a valid api key", http.StatusUnauthorized)
		return
	}
	if result.Error != nil {
		http.Error(w, "DB Error", http.StatusInternalServerError)
		return
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
			var urlShortenerExists models.URLShortener
			result := config.DB.Model(&models.URLShortener{}).Where("short_code = ?", urlRequest.CustomCode).First(&urlShortenerExists)
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
			shortCode = utils.GenerateShortCode(6)
		}

		// Create a new URLShortener record with the original URL, short code, and API key
		// TODO: Check if expired_at default value
		urlShortener := models.URLShortener{
			OriginalURL: urlRequest.LongURL,
			ShortCode:   shortCode,
			ApiKey:      apiKey,
			ExpiredAt:   urlRequest.ExpiredAt,
			UserID:      user.ID,
			Password:    urlRequest.Password,
		}

		// Save the URLShortener record to the database
		result := config.DB.Create(&urlShortener)
		if result.Error != nil {
			errors = append(errors, map[string]string{
				"long_url": urlRequest.LongURL,
				"error":    "Failed to save short code",
			})
			continue
		}

		// Retrieve all existing records in the database with the same original URL
		var currentLongUrlList []models.URLShortener
		// check if long_url already exists
		result = config.DB.Model(&models.URLShortener{}).Find(&currentLongUrlList, "original_url = ?", urlShortener.OriginalURL)
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
			config.DB.Model(&models.URLShortener{}).Where("id IN ?", ids).Update("shorten_count", gorm.Expr("shorten_count + ?", 1))
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

func DeleteShortenHandler(w http.ResponseWriter, r *http.Request) {
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

	result := config.DB.Model(&models.URLShortener{}).Where("short_code = ? AND api_key = ? AND deleted_at IS NULL", shortCode, apiKey).Update("deleted_at", time.Now())
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

func GetUserUrlsHandler(w http.ResponseWriter, r *http.Request) {
	apiKey := r.Header.Get("api_key")
	if apiKey == "" {
		http.Error(w, "Please pass api_key", http.StatusUnauthorized)
		return
	}
	var user models.User
	result := config.DB.Model(&models.User{}).First(&user, "api_key = ?", apiKey)
	if result.Error != nil {
		http.Error(w, "Invalid API Key", http.StatusUnauthorized)
		return
	}

	pageStr := r.URL.Query().Get("page")
	limitStr := r.URL.Query().Get("limit")
	page, err := strconv.Atoi(pageStr)
	if err != nil || page < 1 {
		page = 1
	}
	limit, err := strconv.Atoi(limitStr)
	if err != nil || limit < 1 {
		limit = 10
	}

	offset := (page - 1) * limit

	var urls []models.URLShortener
	result = config.DB.Model(&models.URLShortener{}).Where("user_id = ?", user.ID).Limit(limit).Offset(offset).Find(&urls)
	if result.Error != nil {
		http.Error(w, "Error fetching URLs", http.StatusInternalServerError)
		return
	}

	response := map[string]interface{}{
		"page":  page,
		"limit": limit,
		"urls":  urls,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)

}

func SyncHandler(w http.ResponseWriter, r *http.Request) {
	start := time.Now()
	utils.SimulateSlowOperation()

	response := map[string]string{"message": "Done"}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
	middlewares.AuditLogger.Printf("SyncHandler time take: %s", time.Since(start))
}

func AsyncHandler(w http.ResponseWriter, r *http.Request) {
	start := time.Now()
	go utils.SimulateSlowOperation()
	profileImgBytes, err := os.ReadFile("assets/image/profileImg.jpg")
	if err != nil {
		fmt.Printf("Error in profilepc read %v", err.Error())
	}
	data := map[string]interface{}{
		"image": profileImgBytes,
	}
	utils.CheckThumbnail(data)
	response := map[string]string{"message": "Accepted"}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
	middlewares.AuditLogger.Printf("AsyncHandler time take: %s", time.Since(start))

}

func EnqueueHandler(w http.ResponseWriter, r *http.Request) {
	start := time.Now()

	// queue.TaskQueue <- utils.CheckThumbnail("image_upload")

	// queue.TaskQueue <- types.Task{
	// 	Event: "image_uploaded",
	// 	Data:  nil,
	// }

	// queue.EventQueue <- types.Task{
	// 	Event: "image_uploaded",
	// 	Data:  nil,
	// }

	PS.Publish("image_uploaded", map[string]interface{}{
		"image":  "original_Image",
		"userID": "123",
	})

	log.Printf("[%.3f] Enqueued CheckThumbnail task\n", time.Since(start).Seconds())
	w.WriteHeader(http.StatusAccepted)
	fmt.Fprintln(w, "Task enqueued: CheckThumbnail")
}
