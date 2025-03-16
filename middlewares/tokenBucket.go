package middlewares

import (
	"net/http"
	"strconv"
	"time"
)

// TokenBucketScript is a Lua script for token bucket rate limiting.
const TokenBucketScript = `
local key = KEYS[1]
local capacity = tonumber(ARGV[1])
local refillRate = tonumber(ARGV[2])
local now = tonumber(ARGV[3])
local requested = tonumber(ARGV[4])

local bucket = redis.call("HMGET", key, "tokens", "last_refill")
local tokens = tonumber(bucket[1])
local last_refill = tonumber(bucket[2])
if tokens == nil then
  tokens = capacity
  last_refill = now
end

local delta = now - last_refill
local refill = delta * refillRate
tokens = math.min(capacity, tokens + refill)
if tokens < requested then
  return -1
else
  tokens = tokens - requested
  redis.call("HMSET", key, "tokens", tokens, "last_refill", now)
  redis.call("EXPIRE", key, 3600)
  return tokens
end
`

// TokenBucketMiddleware returns a middleware using token bucket algorithm.
// capacity: maximum tokens, refillRate: tokens per millisecond.
func TokenBucketMiddleware(capacity int, refillRate float64) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ip := getIPAddress(r)
			key := "rate:token:" + ip + ":" + r.URL.Path
			ctx := RateLimitRedisStore.Ctx
			now := time.Now().UnixNano() / 1e6 // current time in milliseconds
			// requested tokens: 1 per request.
			res, err := RateLimitRedisStore.Client.Eval(ctx, TokenBucketScript, []string{key},
				capacity, refillRate, now, 1).Result()
			if err != nil {
				next.ServeHTTP(w, r)
				return
			}
			tokensLeft, ok := res.(int64)
			if !ok || tokensLeft < 0 {
				w.WriteHeader(http.StatusTooManyRequests)
				w.Write([]byte("Rate limit exceeded. Try again later."))
				return
			}
			w.Header().Set("X-RateLimit-Limit", strconv.Itoa(capacity))
			w.Header().Set("X-RateLimit-Remaining", strconv.FormatInt(tokensLeft, 10))
			// For token bucket, reset can be computed roughly from tokens refilled per second.
			// For simplicity, we can send a fixed value.
			w.Header().Set("X-RateLimit-Reset", "1")
			next.ServeHTTP(w, r)
		})
	}
}
