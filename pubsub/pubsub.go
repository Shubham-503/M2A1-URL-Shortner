package pubsub

import (
	"M2A1-URL-Shortner/cache"
	"context"
	"encoding/json"
	"fmt"
)

type HandlerFunc func(data map[string]interface{})

type PubSub struct {
	redisStore *cache.RedisStore
}

func NewPubSub(redisStore *cache.RedisStore) *PubSub {
	return &PubSub{redisStore: redisStore}
}

// Subscribe to an event
func (ps *PubSub) Subscribe(event string, handler HandlerFunc) {
	go func() {
		sub := ps.redisStore.Client.Subscribe(ps.redisStore.Ctx, "events")
		ch := sub.Channel()

		for msg := range ch {
			var payload map[string]interface{}
			if err := json.Unmarshal([]byte(msg.Payload), &payload); err != nil {
				fmt.Println("Decode error:", err)
				continue
			}

			if evt, ok := payload["event"].(string); ok && evt == event {
				if data, ok := payload["data"].(map[string]interface{}); ok {
					handler(data)
				}
			}
		}
	}()
}

// Publish an event
func (ps *PubSub) Publish(event string, data map[string]interface{}) error {
	payload := map[string]interface{}{
		"event": event,
		"data":  data,
	}

	bytes, err := json.Marshal(payload)
	if err != nil {
		return err
	}

	return ps.redisStore.Client.Publish(context.Background(), "events", bytes).Err()
}

// func (ps *PubSub) StartWorker() {
// 	sub := ps.store.Client.Subscribe(ps.store.Ctx, "events")
// 	ch := sub.Channel()

// 	go func() {
// 		for msg := range ch {
// 			var payload map[string]interface{}
// 			err := json.Unmarshal([]byte(msg.Payload), &payload)
// 			if err != nil {
// 				fmt.Println("Error decoding payload:", err)
// 				continue
// 			}

// 			event := payload["event"].(string)
// 			data := payload["data"].(map[string]interface{})

// 			if handlers, ok := ps.subscribers[event]; ok {
// 				for _, handler := range handlers {
// 					go handler(data) // async call
// 				}
// 			}
// 		}
// 	}()
// }

// // HandlerFunc defines the type for subscriber functions.
// type HandlerFunc func(data interface{})

// // PubSub struct manages subscribers and event publishing.
// type PubSub struct {
// 	mu          sync.RWMutex
// 	subscribers map[string][]HandlerFunc
// }

// // NewPubSub initializes and returns a new PubSub instance.
// func NewPubSub() *PubSub {
// 	return &PubSub{
// 		subscribers: make(map[string][]HandlerFunc),
// 	}
// }

// // Subscribe registers a function to an event name.
// func (ps *PubSub) Subscribe(event string, fn HandlerFunc) {
// 	ps.mu.Lock()
// 	defer ps.mu.Unlock()
// 	ps.subscribers[event] = append(ps.subscribers[event], fn)
// }

// // Publish sends data to all subscribers of an event.
// func (ps *PubSub) Publish(event string, data interface{}) {
// 	ps.mu.RLock()
// 	defer ps.mu.RUnlock()
// 	if fns, ok := ps.subscribers[event]; ok {
// 		for _, fn := range fns {
// 			go fn(data) // run asynchronously
// 		}
// 	}
// }
