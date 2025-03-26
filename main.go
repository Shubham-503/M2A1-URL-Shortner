package main

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"time"

	"M2A1-URL-Shortner/cache"
	"M2A1-URL-Shortner/config"
	"M2A1-URL-Shortner/handlers"
	middleware "M2A1-URL-Shortner/middlewares"

	sentryhttp "github.com/getsentry/sentry-go/http"

	"github.com/getsentry/sentry-go"
	"github.com/gorilla/mux"
)

var URLCache *cache.URLCache

func main() {
	// To initialize Sentry's handler, you need to initialize Sentry itself beforehand
	if err := sentry.Init(sentry.ClientOptions{
		Dsn: "https://940806d8985f563b0def7c5b42ae03f8@o4508764232220672.ingest.us.sentry.io/4508764262301696",
		// Set TracesSampleRate to 1.0 to capture 100%
		// of transactions for tracing.
		// We recommend adjusting this value in production,
		TracesSampleRate: 1.0,
	}); err != nil {
		fmt.Printf("Sentry initialization failed: %v\n", err)
	}

	// sentry.CaptureMessage("It works!")
	sentryHandler := sentryhttp.New(sentryhttp.Options{})
	defer sentry.Flush(2 * time.Second)
	err := config.InitDB()
	// err := initDB()
	if err != nil {
		log.Fatalf("Failed to initialize the database: %v", err)
	}

	// var err error
	// URLCache, err := cache.NewBigCacheStore()
	// if err != nil {
	// 	panic("Failed to initialize cache: " + err.Error())
	// }
	// handlers.URLCache = URLCache

	// Initialize Redis cache.
	redisStore, err := cache.NewRedisStore("localhost:6379", "", 0)
	if err != nil {
		log.Fatalf("Failed to initialize Redis cache: %v", err)
	}
	handlers.URLCache = redisStore
	middleware.RateLimitRedisStore = redisStore

	// Initialize Redis Cache for rateLimiting
	// IpListRedisStore, err := cache.NewRedisStore("localhost:6379", "", 0)
	// if err != nil {
	// 	log.Fatalf("Failed to initialize Redis cache: %v", err)
	// }
	// middleware.
	// Serve static files using mux
	staticDir := "./static/"
	// fs := http.FileServer(http.Dir(staticDir))

	// Initialize the router
	r := mux.NewRouter()

	r.Use(middleware.LoggingMiddleware)
	r.Use(sentryHandler.Handle)
	r.Use(middleware.SentryAlertMiddleware)
	r.Use(middleware.ResponseTimeMiddleware)
	// r.Use(middleware.RateLimitMiddleware)
	r.Use(middleware.FreeTierMiddleware)
	// r.Use(middleware.LeakyBucketMiddleware(5, 0.005))
	var handler http.Handler = http.HandlerFunc(handlers.ShortenHandler)
	handler = middleware.AuthenticateAPIKey(handler)
	handler = middleware.BlacklistMiddleware(handler)
	handler = middleware.APIRateLimitMiddleware(2)(handler)
	// r.HandleFunc("/redirect", handlers.RedirectHandler).Methods("GET")
	// r.Handle("/shorten", middleware.LoggingMiddleware(http.HandlerFunc(handlers.ShortenHandler))).Methods("POST")
	// r.Handle("/shorten", middleware.AuthenticateAPIKey(http.HandlerFunc(handlers.ShortenHandler))).Methods("POST")
	r.Handle("/shorten", handler).Methods("POST")
	r.HandleFunc("/redirect", handlers.EditRedirectExpiryHandler).Methods("PATCH")
	r.Handle("/shorten-bulk", middleware.IsEnterprise(http.HandlerFunc(handlers.ShortenBulkHandler))).Methods("POST")
	r.HandleFunc("/redirect", handlers.DeleteShortenHandler).Methods("DELETE")
	r.Handle("/redirect", middleware.APIRateLimitMiddleware(50)(http.HandlerFunc(handlers.RedirectHandler))).Methods("GET")
	r.HandleFunc("/redirect", handlers.RedirectHandler).Methods("GET")
	r.HandleFunc("/users/url", handlers.GetUserUrlsHandler).Methods("GET")
	r.HandleFunc("/health", handlers.HealthHandler).Methods("GET")

	// static path
	r.PathPrefix("/").Handler(http.FileServer(http.Dir(staticDir)))
	// r.PathPrefix("/static/").Handler(http.StripPrefix("/static/", fs))

	// Start the server
	fmt.Println("Server is running on http://localhost:8080")
	port := os.Getenv("PORT") // Railway injects the PORT environment variable
	if port == "" {
		port = "8080"
	}
	log.Fatal(http.ListenAndServe(":"+port, r))
}

// func finalHandler(w http.ResponseWriter, r *http.Request) {
// 	w.Write([]byte("Final handler response\n"))
// }
