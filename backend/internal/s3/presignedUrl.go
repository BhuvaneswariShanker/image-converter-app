package s3

import (
	"context"
	"net/url"
	"time"

	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"

	"github.com/BhuvaneswariShanker/image-converter-backend/internal/config"
)

func GeneratePresignedURL(object string, expiryMinutes int) (string, error) {
	// Initialize MinIO client
	minioClient, err := minio.New("localhost:9000", &minio.Options{
		Creds:  credentials.NewStaticV4(config.GetEnv("MINIO_ACCESS_KEY", "minioadmin"), config.GetEnv("MINIO_SECRET_KEY", "minioadmin"), ""),
		Secure: false, // true if using https
	})
	if err != nil {
		return "", err
	}

	// Optional: Custom query parameters
	reqParams := make(url.Values)

	// Generate presigned URL
	presignedURL, err := minioClient.PresignedGetObject(context.Background(), config.GetEnv("CONVERTED_BUCKET_NAME", "converted"), object, time.Duration(expiryMinutes)*time.Minute, reqParams)
	if err != nil {
		return "", err
	}

	return presignedURL.String(), nil
}
