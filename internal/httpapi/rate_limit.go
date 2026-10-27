package httpapi

import (
	"net/http"
	"strconv"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
)

// RateLimit limits each client IP using a sliding window. State is local to this
// middleware instance and process; multiple server instances need shared storage.
func RateLimit(maxRequests int, window time.Duration) gin.HandlerFunc {
	return newRateLimiter(maxRequests, window, time.Now)
}

func newRateLimiter(maxRequests int, window time.Duration, now func() time.Time) gin.HandlerFunc {
	if maxRequests <= 0 || window <= 0 {
		panic("rate limit and window must be positive")
	}

	var mu sync.Mutex
	clients := make(map[string][]time.Time)
	var lastCleanup time.Time

	return func(c *gin.Context) {
		key := c.ClientIP()
		mu.Lock()
		current := now()
		cutoff := current.Add(-window)

		// Lazy cleanup bounds retention of inactive clients without a background goroutine.
		if lastCleanup.IsZero() || current.Sub(lastCleanup) >= window {
			for client, requests := range clients {
				if !requests[len(requests)-1].After(cutoff) {
					delete(clients, client)
				}
			}
			lastCleanup = current
		}

		requests := clients[key]
		first := 0
		for first < len(requests) && !requests[first].After(cutoff) {
			first++
		}
		requests = requests[first:]

		if len(requests) >= maxRequests {
			retryAfter := requests[0].Add(window).Sub(current)
			clients[key] = requests
			mu.Unlock()

			seconds := int64((retryAfter + time.Second - 1) / time.Second)
			c.Header("Retry-After", strconv.FormatInt(seconds, 10))
			fail(c, http.StatusTooManyRequests, "rate_limit_exceeded", "too many requests; try again later")
			c.Abort()
			return
		}

		clients[key] = append(requests, current)
		mu.Unlock()
		c.Next()
	}
}
