package main

import (
	"M2A1-URL-Shortner/handlers"
	"fmt"
	"log"
	"net/http"

	"M2A1-URL-Shortner/config"

	"github.com/gorilla/mux"
)

func main() {
	err := config.InitDB()
	// err := initDB()
	if err != nil {
		log.Fatalf("Failed to initialize the database: %v", err)
	}

	// Serve static files using mux
	staticDir := "./static/"
	// fs := http.FileServer(http.Dir(staticDir))

	// Initialize the router
	r := mux.NewRouter()
	// r.HandleFunc("/redirect", handlers.RedirectHandler).Methods("GET")
	r.HandleFunc("/shorten", handlers.ShortenHandler).Methods("POST")
	r.HandleFunc("/redirect", handlers.EditRedirectExpiryHandler).Methods("PATCH")
	r.HandleFunc("/shorten-bulk", handlers.ShortenBulkHandler).Methods("POST")
	r.HandleFunc("/redirect", handlers.DeleteShortenHandler).Methods("DELETE")
	r.HandleFunc("/redirect", handlers.RedirectHandler).Methods("GET")
	r.HandleFunc("/users/url", handlers.GetUserUrlsHandler).Methods("GET")
	r.HandleFunc("/health", handlers.HealthHandler).Methods("GET")

	// static path
	r.PathPrefix("/").Handler(http.FileServer(http.Dir(staticDir)))
	// r.PathPrefix("/static/").Handler(http.StripPrefix("/static/", fs))

	// Start the server
	fmt.Println("Server is running on http://localhost:8080")
	log.Fatal(http.ListenAndServe(":8080", r))
}
