package middlewares

import (
	"M2A1-URL-Shortner/cache"
	"net/http"
	"strconv"
	"time"
)

var RateLimitRedisStore *cache.RedisStore

func RateLimitMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ip := getIPAddress(r)
		key := "rate:" + ip

		ctx := RateLimitRedisStore.Ctx

		count, err := RateLimitRedisStore.Client.Incr(ctx, key).Result()
		w.Header().Set("X-RateLimit-Limit", "100")
		w.Header().Set("X-RateLimit-Remaining", strconv.Itoa(int(100-count)))
		w.Header().Set("X-RateLimit-Reset", strconv.Itoa(int(time.Minute.Seconds())))
		if err != nil {
			// In case of error, let the request pass.
			next.ServeHTTP(w, r)
			return
		}
		// If this is the first request, set an expiry of 1 minute.
		if count == 1 {
			RateLimitRedisStore.Client.Expire(ctx, key, time.Minute)
		}

		// Retrieve the TTL for this key.
		ttl, err := RateLimitRedisStore.Client.TTL(ctx, key).Result()
		if err == nil && ttl > 0 {
			// Convert the TTL (a time.Duration) to seconds.
			w.Header().Set("X-RateLimit-Reset", strconv.Itoa(int(ttl.Seconds())))
		} else {
			// Fallback if TTL is not available.
			w.Header().Set("X-RateLimit-Reset", "60")
		}

		// If the number of requests exceeds 100, throttle the request.
		if count > 100 {
			w.WriteHeader(http.StatusTooManyRequests)
			w.Write([]byte("Rate limit exceeded. Try again later."))
			return
		}

		next.ServeHTTP(w, r)
	})
}
func APIRateLimitMiddleware(maxRequest int64) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ip := getIPAddress(r)
			endpoint := r.URL.Path
			key := "rate:" + ip + ":" + endpoint

			ctx := RateLimitRedisStore.Ctx

			count, err := RateLimitRedisStore.Client.Incr(ctx, key).Result()
			w.Header().Set("X-RateLimit-Limit", strconv.Itoa(int(maxRequest)))
			w.Header().Set("X-RateLimit-Remaining", strconv.Itoa(int(maxRequest-count)))
			// w.Header().Set("X-RateLimit-Reset", strconv.Itoa(int(time.Minute.Seconds())))
			if err != nil {
				// In case of error, let the request pass.
				next.ServeHTTP(w, r)
				return
			}
			// If this is the first request, set an expiry of 1 minute.
			if count == 1 {
				RateLimitRedisStore.Client.Expire(ctx, key, time.Minute)
			}

			// Retrieve the TTL for this key.
			ttl, err := RateLimitRedisStore.Client.TTL(ctx, key).Result()
			if err == nil && ttl > 0 {
				// Convert the TTL (a time.Duration) to seconds.
				w.Header().Set("X-RateLimit-Reset", strconv.Itoa(int(ttl.Seconds())))
			} else {
				// Fallback if TTL is not available.
				w.Header().Set("X-RateLimit-Reset", "60")
			}

			// If the number of requests exceeds maxrequest, throttle the request.
			if count >= int64(maxRequest) {
				w.WriteHeader(http.StatusTooManyRequests)
				w.Write([]byte("Rate limit exceeded. Try again later."))
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

func FreeTierMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		apiKey := r.Header.Get("api_key")
		if apiKey == "" || apiKey == "free" {

			ip := getIPAddress(r)
			key := "rate:free: " + ip
			ctx := RateLimitRedisStore.Ctx

			count, err := RateLimitRedisStore.Client.Incr(ctx, key).Result()
			w.Header().Set("X-RateLimit-Limit", "5")
			w.Header().Set("X-RateLimit-Remaining", strconv.Itoa(int(5-count)))
			// w.Header().Set("X-RateLimit-Reset", strconv.Itoa(int(time.Minute.Seconds())))
			if err != nil {
				// In case of error, let the request pass.
				next.ServeHTTP(w, r)
				return
			}
			// If this is the first request, set an expiry of 1 minute.
			if count == 1 {
				RateLimitRedisStore.Client.Expire(ctx, key, time.Minute)
			}

			// Retrieve the TTL for this key.
			ttl, err := RateLimitRedisStore.Client.TTL(ctx, key).Result()
			if err == nil && ttl > 0 {
				// Convert the TTL (a time.Duration) to seconds.
				w.Header().Set("X-RateLimit-Reset", strconv.Itoa(int(ttl.Seconds())))
			} else {
				// Fallback if TTL is not available.
				w.Header().Set("X-RateLimit-Reset", "60")
			}

			// If the number of requests exceeds 100, throttle the request.
			if count >= 5 {
				w.WriteHeader(http.StatusTooManyRequests)
				w.Write([]byte("Rate limit exceeded for free tier. Please upgrade your plan or try again later."))
				return
			}
		}
		next.ServeHTTP(w, r)
	})
}
