package middleware

import (
	"sync"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/UN-Self/ResumeGenius/backend/internal/shared/response"
)

type rateEntry struct {
	count    int
	windowStart time.Time
}

// RateLimit returns a Gin middleware that limits requests per IP+key.
// maxRequests: max requests allowed within the window.
// window: time window duration.
func RateLimit(maxRequests int, window time.Duration) gin.HandlerFunc {
	var mu sync.Mutex
	buckets := make(map[string]*rateEntry)

	// Purge expired entries every 5 minutes.
	go func() {
		for {
			time.Sleep(5 * time.Minute)
			mu.Lock()
			for k, v := range buckets {
				if time.Since(v.windowStart) > window {
					delete(buckets, k)
				}
			}
			mu.Unlock()
		}
	}()

	return func(c *gin.Context) {
		ip := c.ClientIP()
		key := ip + ":" + c.Request.URL.Path

		mu.Lock()
		entry, exists := buckets[key]
		now := time.Now()

		if !exists || now.Sub(entry.windowStart) > window {
			buckets[key] = &rateEntry{count: 1, windowStart: now}
			mu.Unlock()
			c.Next()
			return
		}

		entry.count++
		if entry.count > maxRequests {
			mu.Unlock()
			response.ErrorWithStatus(c, 429, 42900, "请求过于频繁，请稍后再试")
			c.Abort()
			return
		}
		mu.Unlock()

		c.Next()
	}
}
