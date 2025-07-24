// cmd/producer/main.go
package main

import (
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"

	"github.com/BhuvaneswariShanker/image-converter-backend/config"
	"github.com/BhuvaneswariShanker/image-converter-backend/internal/jwt"
	"github.com/BhuvaneswariShanker/image-converter-backend/internal/ratelimiter"
	"github.com/BhuvaneswariShanker/image-converter-backend/internal/s3"
)

func main() {
	initEnv()
	s3.InitS3()

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

	r.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "up"})
	})

	protected := r.Group("/api")
	protected.Use(jwt.JWTAuthMiddleware())
	protected.Use(ratelimiter.RateLimiterMiddlewareByUserID())

	protected.GET("/download/:jobId/:name", func(c *gin.Context) {
		name := c.Param("name")
		jobId := c.Param("jobId")
		filename := jobId + "/" + name

		tempZip := filepath.Join("data", filepath.Base(filename))
		log.Printf("filename: %v", filename)
		log.Printf("local filename: %v", tempZip)

		err1 := os.MkdirAll("data", 0755)
		if err1 != nil {
			log.Printf("❌ Failed to create directory: %v", err1)
			return
		}
		err := s3.DownloadFileIntoLocal(filename, getEnv("CONVERTED_BUCKET_NAME", "converted"), tempZip)
		if err != nil {
			log.Printf("❌ Failed to download from MinIO: %v", err)
			c.String(http.StatusInternalServerError, "Failed to download file")
			return
		}

		// Open the file
		file, err := os.Open(tempZip)
		if err != nil {
			log.Printf("❌ Failed to open file: %v", err)
			c.String(http.StatusInternalServerError, "Failed to open file")
			return
		}
		defer file.Close()

		// Get file info for headers
		stat, err := file.Stat()
		if err != nil {
			log.Printf("❌ Failed to stat file: %v", err)
			c.String(http.StatusInternalServerError, "Failed to stat file")
			return
		}

		// Set response headers
		ext := strings.ToLower(filepath.Ext(stat.Name()))
		var contentType string

		switch ext {
		case ".zip":
			contentType = "application/zip"
		case ".jpg", ".jpeg":
			contentType = "image/jpeg"
		default:
			contentType = "application/octet-stream" // fallback
		}
		c.Header("Content-Disposition", fmt.Sprintf("attachment; filename=\"%s\"", stat.Name()))
		c.Header("Content-Type", contentType) // or application/octet-stream
		c.Header("Content-Length", fmt.Sprintf("%d", stat.Size()))
		c.Status(http.StatusOK)

		// Stream the file content to response
		_, err = io.Copy(c.Writer, file)
		if err != nil {
			log.Printf("❌ Failed to stream file: %v", err)
		}

		// Delete all uploaded files from minio - cleanup
		go func() {
			uploadKey := strings.TrimSuffix(filename, filepath.Ext(filename)) + ".pdf"
			convertedKey := filename

			err1 := s3.DeleteFile(getEnv("UPLOAD_BUCKET_NAME", "uploads"), uploadKey)
			err2 := s3.DeleteFile(getEnv("CONVERTED_BUCKET_NAME", "converted"), convertedKey)

			if err1 != nil || err2 != nil {
				log.Printf("⚠️ Post-download cleanup error: upload=%v, converted=%v", err1, err2)
			} else {
				log.Printf("✅ Post-download cleanup done for job: %s", filename)
			}
		}()
	})

	addr := ":" + port
	log.Fatal(r.Run(addr))
}

func initEnv() {
	// Setup ENV
	os.Setenv("ROLE", "downloader")
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
