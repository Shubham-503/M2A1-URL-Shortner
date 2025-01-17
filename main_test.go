package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"M2A1-URL-Shortner/config"
	"M2A1-URL-Shortner/handlers"
	"M2A1-URL-Shortner/models"
	"M2A1-URL-Shortner/utils"

	"github.com/gorilla/mux"
)

// func initTestDB() error {
// 	var err error
// 	// Open SQLite database with GORM
// 	db, err = gorm.Open(sqlite.Open("url_shortener.db"), &gorm.Config{})
// 	tx := db.Begin()
// 	defer tx.Rollback()
// 	if err != nil {
// 		return err
// 	}

// 	// Auto migrate the schema
// 	err = tx.AutoMigrate(&URLShortener{})
// 	if err != nil {
// 		return err
// 	}

// 	return nil
// }

func TestURLShortenerAndRedirect(t *testing.T) {
	if err := config.InitDB(); err != nil {
		t.Fatalf("Failed to initialize database: %v", err)
	}

	r := mux.NewRouter()
	r.HandleFunc("/shorten", handlers.ShortenHandler).Methods("POST")
	r.HandleFunc("/redirect", handlers.RedirectHandler).Methods("GET")

	shortenReqPayload := map[string]string{"long_url": "https://example.com"}
	reqBody, _ := json.Marshal(shortenReqPayload)

	req := httptest.NewRequest(http.MethodPost, "/shorten", bytes.NewBuffer(reqBody))
	req.Header.Set("Content-Type", "application/json")
	apiKey := fmt.Sprint(time.Time.Nanosecond(time.Now()))
	req.Header.Set("api_key", apiKey)
	resp := httptest.NewRecorder()

	r.ServeHTTP(resp, req)

	if resp.Code != http.StatusOK {
		t.Fatalf("Expected status code 200, got %d", resp.Code)
	}

	var shortenResp map[string]string
	if err := json.Unmarshal(resp.Body.Bytes(), &shortenResp); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	shortCode, exists := shortenResp["short_code"]
	if !exists || shortCode == "" {
		t.Fatalf("Short code not found in response")
	}

	redirectURL := "/redirect?code=" + shortCode
	req = httptest.NewRequest(http.MethodGet, redirectURL, nil)
	req.Header.Set("api_key", "test12345")
	resp = httptest.NewRecorder()

	r.ServeHTTP(resp, req)

	if resp.Code != http.StatusOK {
		t.Fatalf("Expected status code 200, got %d", resp.Code)
	}

	var redirectResp map[string]string
	if err := json.Unmarshal(resp.Body.Bytes(), &redirectResp); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	longURL, exists := redirectResp["long_url"]
	if !exists || longURL != "https://example.com" {
		t.Fatalf("Expected long URL 'https://example.com', got '%s'", longURL)
	}
}

func TestDuplcateUrl(t *testing.T) {

	if err := config.InitDB(); err != nil {
		t.Fatalf("Failed to initialize database: %v", err)
	}

	// randomUrl := "http://www.example.com/test2"
	randomUrl := "http://www." + utils.GenerateShortCode(7) + ".com/" + utils.GenerateShortCode(4)

	r := mux.NewRouter()
	r.HandleFunc("/shorten", handlers.ShortenHandler).Methods("POST")

	shortenReqPayload := map[string]string{"long_url": randomUrl}
	reqBody, _ := json.Marshal(shortenReqPayload)
	apiKey1 := fmt.Sprint(time.Time.Nanosecond(time.Now()))
	req := httptest.NewRequest(http.MethodPost, "/shorten", bytes.NewBuffer(reqBody))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("api_key", apiKey1)
	resp := httptest.NewRecorder()

	r.ServeHTTP(resp, req)

	if resp.Code != http.StatusOK {
		t.Fatalf("Expected status code 200, got %d", resp.Code)
	}

	req2 := httptest.NewRequest(http.MethodPost, "/shorten", bytes.NewBuffer(reqBody))
	apiKey2 := fmt.Sprint(time.Time.Nanosecond(time.Now()))
	req2.Header.Set("Content-Type", "application/json")
	req2.Header.Set("api_key", apiKey2)
	// req2.Header.Set("api_key", "test12345")

	resp2 := httptest.NewRecorder()

	r.ServeHTTP(resp2, req2)

	if resp2.Code != http.StatusOK {
		t.Fatalf("Expected status code 200, got %d", resp2.Code)
	}
}

func TestShortCodeNotFound(t *testing.T) {
	if err := config.InitDB(); err != nil {
		t.Fatalf("Failed to initialize database: %v", err)
	}
	var newShortCode string
	var uRLShortener models.URLShortener
	for {
		newShortCode = utils.GenerateShortCode(7)
		result := config.DB.Model(&models.URLShortener{}).Where("short_code = ? AND deleted_at IS NULL", newShortCode).First(&uRLShortener)
		if result.RowsAffected != 0 {
			continue
		} else {
			break
		}

	}
	r := mux.NewRouter()
	r.HandleFunc("/redirect", handlers.RedirectHandler).Methods("GET")

	redirectURL := "/redirect?code=" + newShortCode
	req := httptest.NewRequest(http.MethodGet, redirectURL, nil)
	resp := httptest.NewRecorder()

	r.ServeHTTP(resp, req)

	if resp.Code != http.StatusNotFound {
		t.Fatalf("Expected status code 400, got %d", resp.Code)
	}
}

func TestDeleteShortCode(t *testing.T) {
	if err := config.InitDB(); err != nil {
		t.Fatalf("Failed to initialize database: %v", err)
	}
	var urlShortener models.URLShortener
	result := config.DB.Model(&models.URLShortener{}).First(&urlShortener, "api_key IS NOT NULL AND deleted_at IS NULL")
	if result.Error != nil {
		t.Fatalf("Error ")
	}

	r := mux.NewRouter()
	r.HandleFunc("/redirect", handlers.DeleteShortenHandler).Methods("DELETE")

	redirectURL := "/redirect?code=" + urlShortener.ShortCode
	req := httptest.NewRequest(http.MethodDelete, redirectURL, nil)
	resp := httptest.NewRecorder()
	req.Header.Set("api_key", urlShortener.ApiKey)

	r.ServeHTTP(resp, req)

	if resp.Code != http.StatusOK {
		t.Fatalf("Expected status code 200, got %d", resp.Code)
	}
}

func TestInvalidUrl(t *testing.T) {
	if err := config.InitDB(); err != nil {
		t.Fatalf("Failed to initialize database: %v", err)
	}
	r := mux.NewRouter()
	r.HandleFunc("/shorten", handlers.ShortenHandler).Methods("POST")
	shortenReqPayload := map[string]string{"long_url": ""}
	reqBody, _ := json.Marshal(shortenReqPayload)
	req := httptest.NewRequest(http.MethodPost, "/shorten", bytes.NewBuffer(reqBody))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("api_key", "test12345")
	resp := httptest.NewRecorder()
	r.ServeHTTP(resp, req)
	if resp.Code != http.StatusBadRequest {
		t.Fatal("Expected an error: Invalid URL should not be saved, but it was successfully saved")
		t.Fatalf("Expected status code 404, got %d", resp.Code)
	}
}

func TestPreventUnathorizedDeleteShortCode(t *testing.T) {
	if err := config.InitDB(); err != nil {
		t.Fatalf("Failed to initialize database: %v", err)
	}
	r := mux.NewRouter()
	r.HandleFunc("/redirect", handlers.DeleteShortenHandler).Methods("DELETE")

	var urlShortener models.URLShortener
	result := config.DB.Model(&models.URLShortener{}).First(&urlShortener, "api_key IS NOT NULL")
	if result.Error != nil {
		t.Fatalf("Error ")
	}
	redirectURL := "/redirect?code=" + urlShortener.ShortCode
	req := httptest.NewRequest(http.MethodDelete, redirectURL, nil)
	resp := httptest.NewRecorder()
	req.Header.Set("api_key", urlShortener.ApiKey+urlShortener.ApiKey)

	r.ServeHTTP(resp, req)

	if resp.Code != http.StatusNotFound {
		t.Fatalf("Expected status code 404, got %d", resp.Code)
	}

}

func TestExpiredUrl(t *testing.T) {
	if err := config.InitDB(); err != nil {
		t.Fatalf("Failed to initialize database: %v", err)
	}
	r := mux.NewRouter()
	r.HandleFunc("/redirect", handlers.RedirectHandler).Methods("GET")
	var urlShortener models.URLShortener
	result := config.DB.Model(&models.URLShortener{}).First(&urlShortener, "api_key IS NOT NULL AND expired_at < ? AND deleted_at IS NULL", time.Now())
	if result.Error != nil {
		t.Fatalf("Error in db ")
	}
	fmt.Print(urlShortener)

	redirectURL := "/redirect?code=" + urlShortener.ShortCode
	req := httptest.NewRequest(http.MethodGet, redirectURL, nil)
	resp := httptest.NewRecorder()

	r.ServeHTTP(resp, req)

	if resp.Code != http.StatusGone {
		t.Fatalf("Expected status code 410, got %d", resp.Code)
	}

}

func TestCustomCodeExists(t *testing.T) {
	if err := config.InitDB(); err != nil {
		t.Fatalf("Failed to initialize database: %v", err)
	}
	randomUrl := "http://www." + utils.GenerateShortCode(7) + ".com/" + utils.GenerateShortCode(4)
	var urlShortner models.URLShortener
	result := config.DB.Model(&models.URLShortener{}).First(&urlShortner, "deleted_at IS NULL")
	if result.Error != nil {
		t.Fatal("DB error")
	}
	fmt.Printf("short code %s", urlShortner.ShortCode)
	r := mux.NewRouter()
	r.HandleFunc("/shorten", handlers.ShortenHandler).Methods("POST")

	shortenReqPayload := map[string]string{"long_url": randomUrl, "custom_code": urlShortner.ShortCode}
	reqBody, _ := json.Marshal(shortenReqPayload)
	apiKey1 := fmt.Sprint(time.Time.Nanosecond(time.Now()))
	req := httptest.NewRequest(http.MethodPost, "/shorten", bytes.NewBuffer(reqBody))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("api_key", apiKey1)
	resp := httptest.NewRecorder()

	r.ServeHTTP(resp, req)

	if resp.Code != http.StatusConflict {
		t.Fatalf("Expected status code 409, got %d", resp.Code)
	}
}

func TestBulkShorten(t *testing.T) {
	if err := config.InitDB(); err != nil {
		t.Fatalf("Failed to initialize database: %v", err)
	}

	r := mux.NewRouter()
	r.HandleFunc("/shorten-bulk", handlers.ShortenBulkHandler).Methods("POST")
	jsonPayload := `{
    "urls": [
        {
            "long_url": "https://example.com",
            "expired_at": "2025-02-10T13:06:30.521Z",
            "custom_code": "example1"
        },
        {
            "long_url": "https://anotherexample.com",
            "custom_code": "example2"
        },
        {
            "long_url": "https://yetanotherexample.com"
        }
    ]
	}`

	// err := json.Unmarshal([]byte(jsonPayload), &shortenReqPayload)
	// if err != nil {
	// 	t.Fatalf("Error unmarshaling JSON: %v", err)
	// }

	// reqBody, _ := json.Marshal(jsonPayload)
	var user models.User
	result := config.DB.Model(&models.User{}).First(&user, "tier = 'enterprise'")
	if result.Error != nil {
		t.Fatal("DB error")
	}

	req := httptest.NewRequest(http.MethodPost, "/shorten-bulk", bytes.NewBuffer([]byte(jsonPayload)))
	req.Header.Set("Content-Type", "application/json")
	// apiKey := fmt.Sprint(time.Time.Nanosecond(time.Now()))
	req.Header.Set("api_key", user.ApiKey)
	resp := httptest.NewRecorder()

	r.ServeHTTP(resp, req)

	if resp.Code != http.StatusOK {
		t.Fatalf("Expected status code 200, got %d", resp.Code)
	}

}

func TestPreventUnauthorisedBulkShorten(t *testing.T) {
	if err := config.InitDB(); err != nil {
		t.Fatalf("Failed to initialize database: %v", err)
	}

	r := mux.NewRouter()
	r.HandleFunc("/shorten-bulk", handlers.ShortenBulkHandler).Methods("POST")
	jsonPayload := `{
    "urls": [
        {
            "long_url": "https://example.com",
            "expired_at": "2025-02-10T13:06:30.521Z",
            "custom_code": "example1"
        },
        {
            "long_url": "https://anotherexample.com",
            "custom_code": "example2"
        },
        {
            "long_url": "https://yetanotherexample.com"
        }
    ]
	}`

	// err := json.Unmarshal([]byte(jsonPayload), &shortenReqPayload)
	// if err != nil {
	// 	t.Fatalf("Error unmarshaling JSON: %v", err)
	// }

	// reqBody, _ := json.Marshal(jsonPayload)
	var user models.User
	result := config.DB.Model(&models.User{}).First(&user, "tier = 'hobby'")
	if result.Error != nil {
		t.Fatal("DB error")
	}

	req := httptest.NewRequest(http.MethodPost, "/shorten-bulk", bytes.NewBuffer([]byte(jsonPayload)))
	req.Header.Set("Content-Type", "application/json")
	// apiKey := fmt.Sprint(time.Time.Nanosecond(time.Now()))
	req.Header.Set("api_key", user.ApiKey)
	resp := httptest.NewRecorder()

	r.ServeHTTP(resp, req)

	if resp.Code != http.StatusForbidden {
		t.Fatalf("Expected status code 200, got %d", resp.Code)
	}

}

func TestRedirectExpiry(t *testing.T) {
	if err := config.InitDB(); err != nil {
		t.Fatalf("Failed to initialize database: %v", err)
	}

	now := time.Now()
	yesterday := now.Add(-24 * time.Hour)
	yesterdayISO := yesterday.Format(time.RFC3339)

	var urlShortner models.URLShortener
	config.DB.Model(&models.URLShortener{}).First(&urlShortner, "expired_at is null and api_key is not null and deleted_at is null")
	fmt.Printf("date : %s\n", urlShortner.ApiKey)
	fmt.Printf("date : %s\n", urlShortner.ShortCode)
	fmt.Printf("date : %s\n", urlShortner.OriginalURL)
	fmt.Printf("date : %s\n", yesterdayISO)

	r := mux.NewRouter()
	r.HandleFunc("/redirect", handlers.EditRedirectExpiryHandler).Methods("PATCH")
	redirectURL := "/redirect?code=" + urlShortner.ShortCode
	ReqPayload := map[string]string{"expired_at": yesterdayISO}
	reqBody, _ := json.Marshal(ReqPayload)

	req := httptest.NewRequest(http.MethodPatch, redirectURL, bytes.NewBuffer(reqBody))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("api_key", urlShortner.ApiKey)
	resp := httptest.NewRecorder()

	r.ServeHTTP(resp, req)

	if resp.Code != http.StatusOK {
		t.Fatalf("Expected status code 200, got %d", resp.Code)
	}

}

func TestPasswordProtectedCode(t *testing.T) {
	if err := config.InitDB(); err != nil {
		t.Fatalf("Failed to initialize database: %v", err)
	}
	var urlShortner models.URLShortener
	config.DB.Model(&models.URLShortener{}).First(&urlShortner, "expired_at is null and api_key is not null and deleted_at is null and password is not null")

	r := mux.NewRouter()
	r.HandleFunc("/redirect", handlers.RedirectHandler).Methods("GET")
	redirectURL := "/redirect?code=" + urlShortner.ShortCode
	ReqPayload := map[string]string{"password": *urlShortner.Password + *urlShortner.Password}
	reqBody, _ := json.Marshal(ReqPayload)

	req := httptest.NewRequest(http.MethodGet, redirectURL, bytes.NewBuffer(reqBody))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("api_key", urlShortner.ApiKey)
	resp := httptest.NewRecorder()

	r.ServeHTTP(resp, req)

	if resp.Code != http.StatusUnauthorized {
		t.Fatalf("Expected status code 401, got %d", resp.Code)
	}

}

func TestGetUserUrls(t *testing.T) {
	if err := config.InitDB(); err != nil {
		t.Fatalf("Failed to initialize database: %v", err)
	}
	r := mux.NewRouter()
	r.HandleFunc("/users/url", handlers.GetUserUrlsHandler).Methods("GET")
	req := httptest.NewRequest(http.MethodGet, "/users/url", nil)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("api_key", "234786100")

	resp := httptest.NewRecorder()

	r.ServeHTTP(resp, req)

	if resp.Code != http.StatusOK {
		t.Fatalf("Expected status code 200, got %d", resp.Code)
	}

}
