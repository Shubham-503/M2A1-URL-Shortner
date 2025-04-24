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
	"M2A1-URL-Shortner/pubsub"
	"M2A1-URL-Shortner/utils"

	sentryhttp "github.com/getsentry/sentry-go/http"
	"github.com/robfig/cron/v3"

	"github.com/getsentry/sentry-go"
	"github.com/gorilla/mux"
)

var URLCache *cache.URLCache
var PS *pubsub.PubSub

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
	REDIS_URL := os.Getenv("REDIS_URL")
	REDIS_PASSWORD := os.Getenv("REDIS_PASSWORD")

	redisStore, err := cache.NewRedisStore(REDIS_URL, REDIS_PASSWORD, 0)
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

	// cronjob setup
	c := cron.New()

	// Schedule job to run every minute
	c.AddFunc("@every 1m", func() {
		log.Println("Running CheckThumbnail job at", time.Now())
		profileImgBytes, err := os.ReadFile("assets/image/profileImg.jpg")
		if err != nil {
			fmt.Printf("Error in profilepc read %v", err.Error())
		}
		data := map[string]interface{}{
			"image": profileImgBytes,
		}
		utils.CheckThumbnail(data)
	})

	// c.Start()
	// queue.StartEventWorker()
	// queue.StartWorker("worker 2")
	// queue.StartLogUploadWorker()
	// queue.StartNotifyAdminWorker()

	// For demo, flush every 10s or every 5 events
	// cb := batcher.NewCountBatcher(5)
	// tb := batcher.NewTimeBatcher(10 * time.Second)
	// tb.Start()
	// defer tb.Stop()

	// ticker := time.NewTicker(1 * time.Second)
	// for i := 0; i < 12; i++ {
	// 	<-ticker.C
	// 	evt := batcher.ViewEvent{ProductID: rand.Intn(3) + 1, Timestamp: time.Now()}
	// 	fmt.Printf("Enqueue event #%d for product %d\n", i+1, evt.ProductID)
	// 	cb.Enqueue(evt)
	// 	tb.Enqueue(evt)
	// }

	// ps := pubsub.NewPubSub()

	// ps.Subscribe("image_uploaded", func(data interface{}) {
	// 	fmt.Println("Resize:", data)
	// })

	// ps.Subscribe("image_uploaded", func(data interface{}) {
	// 	fmt.Println("Notify Slack:", data)
	// })

	// ps.Publish("image_uploaded", map[string]string{
	// 	"filename": "photo.jpg",
	// })

	// Create PubSub using RedisStore
	PS := pubsub.NewPubSub(redisStore)
	PS.Subscribe("image_uploaded", utils.CheckThumbnail)
	PS.Subscribe("image_uploaded", utils.LogUpload)
	PS.Subscribe("image_uploaded", utils.NotifyAdmin)
	handlers.PS = PS
	// pubsub.SubscribeToEvent(redisStore,"image_uploaded", utils.CheckThumbnail("s"))

	// Register subscribers
	// ps.Subscribe("image_uploaded", func(data map[string]interface{}) {
	// 	fmt.Println("Thumbnail generation for:")
	// 	utils.CheckThumbnail(data["filename"])
	// })

	// ps.Subscribe("image_uploaded", func(data map[string]interface{}) {
	// 	fmt.Println("Notify admin:")
	// })

	// Start PubSub worker
	// ps.StartWorker()

	// Simulate publishing an event
	profileImgBytes, err := os.ReadFile("assets/image/profileImg.jpg")
	if err != nil {
		fmt.Printf("Error in profilepc read %v", err.Error())
	}
	err = PS.Publish("image_uploaded", map[string]interface{}{
		"filename": profileImgBytes,
	})
	if err != nil {
		fmt.Println("Publish error:", err)
	}

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
	// r.Use(middleware.FreeTierMiddleware)
	// r.Use(middleware.LeakyBucketMiddleware(5, 0.005))
	var handler http.Handler = http.HandlerFunc(handlers.ShortenHandler)
	handler = middleware.AuthenticateAPIKey(handler)
	handler = middleware.BlacklistMiddleware(handler)
	// handler = middleware.APIRateLimitMiddleware(2)(handler)
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

	r.HandleFunc("/sync", handlers.SyncHandler).Methods("GET")
	r.HandleFunc("/async", handlers.AsyncHandler).Methods("GET")
	r.HandleFunc("/enqueue", handlers.EnqueueHandler).Methods("GET")

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
