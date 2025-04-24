package main

import (
	"M2A1-URL-Shortner/cache"
	"M2A1-URL-Shortner/pubsub"
	"log"
	"os"
)

func main() {
	REDIS_URL := os.Getenv("REDIS_URL")
	REDIS_PASSWORD := os.Getenv("REDIS_PASSWORD")
	redisStore, err := cache.NewRedisStore(REDIS_URL, REDIS_PASSWORD, 0)
	if err != nil {
		log.Fatalf("Failed to initialize Redis cache: %v", err)
	}

	ps := pubsub.NewPubSub(redisStore)

	err = ps.Publish("image_uploaded", map[string]interface{}{
		"filename": "hello.jpg",
		"user":     "john123",
	})
	if err != nil {
		log.Println("Failed to publish:", err)
	} else {
		log.Println("✅ Event published")
	}

	select {}
}
