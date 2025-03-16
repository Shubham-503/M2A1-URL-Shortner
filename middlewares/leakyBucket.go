package middlewares

import (
	"net/http"
	"strconv"
	"time"
)

// LeakyBucketScript is a Lua script for leaky bucket rate limiting.
const LeakyBucketScript = `
local key = KEYS[1]
local capacity = tonumber(ARGV[1])
local leakRate = tonumber(ARGV[2])  -- units per millisecond
local now = tonumber(ARGV[3])
local requestCost = tonumber(ARGV[4])

local bucket = redis.call("HMGET", key, "level", "last_update")
local level = tonumber(bucket[1])
local lastUpdate = tonumber(bucket[2])
if level == nil then
  level = 0
  lastUpdate = now
end

local delta = now - lastUpdate
local leaked = delta * leakRate
level = math.max(0, level - leaked)

if level + requestCost > capacity then
  return -1
else
  level = level + requestCost
  redis.call("HMSET", key, "level", level, "last_update", now)
  redis.call("EXPIRE", key, 3600)
  return level
end
`

// LeakyBucketMiddleware returns a middleware using the leaky bucket algorithm.
// capacity: maximum allowed "level" in the bucket, leakRate: leak per millisecond.
func LeakyBucketMiddleware(capacity int, leakRate float64) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ip := getIPAddress(r)
			key := "rate:leaky:" + ip + ":" + r.URL.Path
			ctx := RateLimitRedisStore.Ctx
			now := time.Now().UnixNano() / 1e6
			// Assume each request adds a cost of 1 unit.
			res, err := RateLimitRedisStore.Client.Eval(ctx, LeakyBucketScript, []string{key},
				capacity, leakRate, now, 1).Result()
			if err != nil {
				next.ServeHTTP(w, r)
				return
			}
			level, ok := res.(float64)
			if !ok || level < 0 {
				w.WriteHeader(http.StatusTooManyRequests)
				w.Write([]byte("Rate limit exceeded. Try again later."))
				return
			}
			w.Header().Set("X-RateLimit-Limit", strconv.Itoa(capacity))
			// In leaky bucket, the remaining capacity is (capacity - level).
			w.Header().Set("X-RateLimit-Remaining", strconv.Itoa(capacity-int(level)))
			// For simplicity, we set a fixed reset value.
			w.Header().Set("X-RateLimit-Reset", "1")
			next.ServeHTTP(w, r)
		})
	}
}
