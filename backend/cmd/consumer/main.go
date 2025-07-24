// cmd/consumer/main.go
package main

import (
	"log"
	"net/http"
	"os"
	"time"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"

	"github.com/BhuvaneswariShanker/image-converter-backend/config"
	"github.com/BhuvaneswariShanker/image-converter-backend/internal/converter"
	"github.com/BhuvaneswariShanker/image-converter-backend/internal/jwt"
	"github.com/BhuvaneswariShanker/image-converter-backend/internal/kafka"
	"github.com/BhuvaneswariShanker/image-converter-backend/internal/ratelimiter"
	"github.com/BhuvaneswariShanker/image-converter-backend/internal/s3"
	"github.com/BhuvaneswariShanker/image-converter-backend/internal/websocket"
)

func main() {
	initEnv()
	kafka.StartKafkaConsumer(getEnv("KAFKA_BROKER", "localhost:9092"), converter.ConvertAndStoreImage)

	r := gin.Default()

	r.Use(cors.New(cors.Config{
		AllowOrigins:     []string{getEnv("FRONTEND_URL", "http://localhost:4200")},
		AllowMethods:     []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowHeaders:     []string{"Origin", "Content-Type", "Authorization"},
		ExposeHeaders:    []string{"Content-Length"},
		AllowCredentials: true,
		MaxAge:           12 * time.Hour,
	}))

	r.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "up"})
	})

	r.GET("/ws/:jobId", func(c *gin.Context) {
		jobId := c.Param("jobId")
		tokenString := c.Query("token")

		// Validate the token
		claims, err := jwt.ValidateJWT(tokenString)
		if err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid or expired token"})
			return
		}

		// Check rate limit
		userID := claims["sub"].(string)
		limiter := ratelimiter.GetLimiter(userID)
		if !limiter.Allow() {
			c.AbortWithStatusJSON(http.StatusTooManyRequests, gin.H{
				"error": "Rate limit exceeded. Try again later.",
			})
			return
		}

		// Upgrade to WebSocket after validation
		websocket.WsHandlerGin(c.Writer, c.Request, jobId)
	})

	port := os.Getenv("PORT")
	log.Printf("Service starting on port %s", port)
	addr := ":" + port
	log.Fatal(r.Run(addr))
}

func initEnv() {
	// Setup ENV
	os.Setenv("ROLE", "consumer")
	config.LoadEnv()
	s3.InitS3()
}

func getEnv(key, fallback string) string {
	if val := os.Getenv(key); val != "" {
		return val
	}
	return fallback
}
