package ratelimiter

import (
	"net/http"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"golang.org/x/time/rate"
)

// Global limiter map and mutex
var (
	rateLimit  = rate.Every(2 * time.Second) // 5 requests per 10 seconds
	burstLimit = 5
	users      = make(map[string]*rate.Limiter)
	mu         sync.Mutex
)

// GetLimiter returns rate limiter for a given user ID
func GetLimiter(userID string) *rate.Limiter {
	mu.Lock()
	defer mu.Unlock()

	limiter, exists := users[userID]
	if !exists {
		limiter = rate.NewLimiter(rateLimit, burstLimit)
		users[userID] = limiter
	}
	return limiter
}

// RateLimiterMiddlewareByUserID limits based on user ID
func RateLimiterMiddlewareByUserID() gin.HandlerFunc {
	return func(c *gin.Context) {
		// Assuming userID is stored in context (e.g., from JWT middleware)
		userID := c.GetString("userID")

		if userID == "" {
			// User not authenticated; optionally reject or skip rate limiting
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
				"error": "User ID missing in context",
			})
			return
		}

		limiter := GetLimiter(userID)
		if !limiter.Allow() {
			c.AbortWithStatusJSON(http.StatusTooManyRequests, gin.H{
				"error": "Rate limit exceeded. Try again later.",
			})
			return
		}

		c.Next()
	}
}
