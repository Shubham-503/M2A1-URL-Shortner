package main

import (
	"M2A1-URL-Shortner/cache"
	"M2A1-URL-Shortner/pubsub"
	"M2A1-URL-Shortner/utils"
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
	// profileImgBytes, err := os.ReadFile("assets/image/profileImg.jpg")
	// if err != nil {
	// 	fmt.Printf("Error in profilepc read %v", err.Error())
	// }
	ps := pubsub.NewPubSub(redisStore)

	ps.Subscribe("image_uploaded", utils.CheckThumbnail)
	ps.Subscribe("image_uploaded", utils.LogUpload)
	// or
	ps.Subscribe("image_uploaded", utils.NotifyAdmin)
	select {}
}
