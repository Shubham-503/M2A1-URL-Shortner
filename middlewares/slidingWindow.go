package middlewares

import (
	"net/http"
	"strconv"
	"time"

	"github.com/redis/go-redis/v9"
)

func SlidingWindowMiddleware(maxRequests int, windowDuration time.Duration) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ip := getIPAddress(r)
			key := "rate:sliding:" + ip + ":" + r.URL.Path

			ctx := RateLimitRedisStore.Ctx

			now := time.Now().UnixNano() / 1e6
			windowStart := now - int64(windowDuration/time.Millisecond)

			// Add current timestamp to the sorted set.
			_, err := RateLimitRedisStore.Client.ZAdd(ctx, key, redis.Z{
				Score:  float64(now),
				Member: strconv.FormatInt(now, 10),
			}).Result()
			if err != nil {
				next.ServeHTTP(w, r)
				return
			}
			// Remove old entries.
			RateLimitRedisStore.Client.ZRemRangeByScore(ctx, key, "0", strconv.FormatInt(windowStart, 10))

			// Get the current count.
			count, err := RateLimitRedisStore.Client.ZCard(ctx, key).Result()
			if err != nil {
				next.ServeHTTP(w, r)
				return
			}
			// Optionally, set an expiry on the key.
			RateLimitRedisStore.Client.Expire(ctx, key, windowDuration*2)

			w.Header().Set("X-RateLimit-Limit", strconv.Itoa(maxRequests))
			w.Header().Set("X-RateLimit-Remaining", strconv.Itoa(maxRequests-int(count)))
			// For sliding window, you can set reset header as the window duration.
			w.Header().Set("X-RateLimit-Reset", strconv.Itoa(int(windowDuration.Seconds())))

			if count > int64(maxRequests) {
				w.WriteHeader(http.StatusTooManyRequests)
				w.Write([]byte("Rate limit exceeded. Try again later."))
				return
			}

			next.ServeHTTP(w, r)

		})
	}
}
