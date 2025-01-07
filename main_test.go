package main

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gorilla/mux"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func initTestDB() error {
	var err error
	// Open SQLite database with GORM
	db, err = gorm.Open(sqlite.Open("url_shortener.db"), &gorm.Config{})
	tx := db.Begin()
	defer tx.Rollback()
	if err != nil {
		return err
	}

	// Auto migrate the schema
	err = tx.AutoMigrate(&URLShortener{})
	if err != nil {
		return err
	}

	return nil
}

func TestURLShortenerAndRedirect(t *testing.T) {
	if err := initDB(); err != nil {
		t.Fatalf("Failed to initialize database: %v", err)
	}

	r := mux.NewRouter()
	r.HandleFunc("/shorten", shortenHandler).Methods("POST")
	r.HandleFunc("/redirect", redirectHandler).Methods("GET")

	shortenReqPayload := map[string]string{"long_url": "https://example.com"}
	reqBody, _ := json.Marshal(shortenReqPayload)

	req := httptest.NewRequest(http.MethodPost, "/shorten", bytes.NewBuffer(reqBody))
	req.Header.Set("Content-Type", "application/json")
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

	if err := initDB(); err != nil {
		t.Fatalf("Failed to initialize database: %v", err)
	}

	// randomUrl := "http://www.example.com/test2"
	randomUrl := "http://www." + generateShortCode(7) + ".com/" + generateShortCode(4)

	r := mux.NewRouter()
	r.HandleFunc("/shorten", shortenHandler).Methods("POST")

	shortenReqPayload := map[string]string{"long_url": randomUrl}
	reqBody, _ := json.Marshal(shortenReqPayload)

	req := httptest.NewRequest(http.MethodPost, "/shorten", bytes.NewBuffer(reqBody))
	req.Header.Set("Content-Type", "application/json")
	resp := httptest.NewRecorder()

	r.ServeHTTP(resp, req)

	if resp.Code != http.StatusOK {
		t.Fatalf("Expected status code 200, got %d", resp.Code)
	}

	req2 := httptest.NewRequest(http.MethodPost, "/shorten", bytes.NewBuffer(reqBody))
	req2.Header.Set("Content-Type", "application/json")
	resp2 := httptest.NewRecorder()

	r.ServeHTTP(resp2, req2)

	if resp2.Code != http.StatusOK {
		t.Fatalf("Expected status code 200, got %d", resp2.Code)
	}
}

func TestShortCodeNotFound(t *testing.T) {
	if err := initDB(); err != nil {
		t.Fatalf("Failed to initialize database: %v", err)
	}
	var newShortCode string
	var uRLShortener URLShortener
	for {
		newShortCode = generateShortCode(7)
		result := db.Where("short_code = ?", newShortCode).First(&uRLShortener)
		if result.RowsAffected != 0 {
			continue
		} else {
			break
		}

	}
	r := mux.NewRouter()
	r.HandleFunc("/redirect", redirectHandler).Methods("GET")

	redirectURL := "/redirect?code=" + newShortCode
	req := httptest.NewRequest(http.MethodGet, redirectURL, nil)
	resp := httptest.NewRecorder()

	r.ServeHTTP(resp, req)

	if resp.Code != http.StatusBadRequest {
		t.Fatalf("Expected status code 400, got %d", resp.Code)
	}
}

func TestDeleteShortCode(t *testing.T) {
	if err := initDB(); err != nil {
		t.Fatalf("Failed to initialize database: %v", err)
	}
	var urlShortener URLShortener
	result := db.First(&urlShortener)
	if result.Error != nil {
		t.Fatalf("Error ")
	}

	r := mux.NewRouter()
	r.HandleFunc("/redirect", deleteShortenHandler).Methods("DELETE")

	redirectURL := "/redirect?code=" + urlShortener.ShortCode
	req := httptest.NewRequest(http.MethodDelete, redirectURL, nil)
	resp := httptest.NewRecorder()

	r.ServeHTTP(resp, req)

	if resp.Code != http.StatusOK {
		t.Fatalf("Expected status code 200, got %d", resp.Code)
	}
}

func TestInvalidUrl(t *testing.T) {
	if err := initDB(); err != nil {
		t.Fatalf("Failed to initialize database: %v", err)
	}
	r := mux.NewRouter()
	r.HandleFunc("/shorten", shortenHandler).Methods("POST")
	shortenReqPayload := map[string]string{"long_url": ""}
	reqBody, _ := json.Marshal(shortenReqPayload)
	req := httptest.NewRequest(http.MethodPost, "/shorten", bytes.NewBuffer(reqBody))
	req.Header.Set("Content-Type", "application/json")
	resp := httptest.NewRecorder()
	r.ServeHTTP(resp, req)
	if resp.Code != http.StatusBadRequest {
		t.Fatal("Expected an error: Invalid URL should not be saved, but it was successfully saved")
		t.Fatalf("Expected status code 404, got %d", resp.Code)
	}
}
