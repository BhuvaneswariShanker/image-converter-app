// cmd/producer/main.go
package main

import (
	"log"
	"net/http"
	"os"
	"time"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"

	"github.com/BhuvaneswariShanker/image-converter-backend/config"
	"github.com/BhuvaneswariShanker/image-converter-backend/internal/jwt"
	"github.com/BhuvaneswariShanker/image-converter-backend/internal/kafka"
	"github.com/BhuvaneswariShanker/image-converter-backend/internal/ratelimiter"
	"github.com/BhuvaneswariShanker/image-converter-backend/internal/s3"
)

func main() {
	initEnv()
	s3.InitS3()
	kafka.InitProducer(getEnv("KAFKA_BROKER", "localhost:9092"))

	port := os.Getenv("PORT")

	r := gin.Default()

	r.Use(cors.New(cors.Config{
		AllowOrigins:     []string{getEnv("FRONTEND_URL", "http://localhost:4200")},
		AllowMethods:     []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowHeaders:     []string{"Origin", "Content-Type", "Authorization"},
		ExposeHeaders:    []string{"Content-Length"},
		AllowCredentials: true,
		MaxAge:           12 * time.Hour,
	}))

	// Public route to get JWT token
	r.GET("/token/:userID", func(c *gin.Context) {
		userID := c.Param("userID")
		token, err := jwt.GenerateJWT(userID)
		if err != nil {
			c.JSON(500, gin.H{"error": "Failed to generate token"})
			return
		}
		c.JSON(200, gin.H{"token": token})
	})

	r.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "up"})
	})

	protected := r.Group("/api")
	protected.Use(jwt.JWTAuthMiddleware())
	protected.Use(ratelimiter.RateLimiterMiddlewareByUserID())

	protected.POST("/upload", func(c *gin.Context) {
		file, header, err := c.Request.FormFile("file")
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "No file uploaded"})
			return
		}
		defer file.Close()

		jobId := c.PostForm("jobId")
		if jobId == "" {
			c.String(http.StatusBadRequest, "jobId is required")
			return
		}

		filename := header.Filename
		uniqueFilename := jobId + "/" + filename

		log.Printf("Filename %s", uniqueFilename)
		err = s3.UploadFile(uniqueFilename, file, header)
		if err != nil {
			log.Printf("Error on uploading into minIO %s", err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		kafka.PublishMessage(uniqueFilename) // Only send Kafka message

		c.JSON(http.StatusOK, gin.H{"message": "File uploaded", "file": filename})
	})

	addr := ":" + port
	log.Fatal(r.Run(addr))
}

func initEnv() {
	// Setup ENV
	os.Setenv("ROLE", "producer")
	config.LoadEnv()

	port := os.Getenv("PORT")
	log.Printf("Service starting on port %s", port)
}

func getEnv(key, fallback string) string {
	if val := os.Getenv(key); val != "" {
		return val
	}
	return fallback
}
