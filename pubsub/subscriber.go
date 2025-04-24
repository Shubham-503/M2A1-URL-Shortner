package pubsub

import (
	"M2A1-URL-Shortner/cache"
	"encoding/json"
	"fmt"
)

// type HandlerFunc func(data map[string]interface{})

func SubscribeToEvent(redisStore *cache.RedisStore, eventName string, handler HandlerFunc) {
	go func() {
		sub := redisStore.Client.Subscribe(redisStore.Ctx, "events")
		ch := sub.Channel()

		for msg := range ch {
			var payload map[string]interface{}
			if err := json.Unmarshal([]byte(msg.Payload), &payload); err != nil {
				fmt.Println("Error decoding:", err)
				continue
			}

			event, ok := payload["event"].(string)
			if !ok || event != eventName {
				continue
			}

			data, ok := payload["data"].(map[string]interface{})
			if ok {
				handler(data)
			}
		}
	}()
}
