package s3

import (
	"bytes"
	"context"
	"fmt"

	"github.com/minio/minio-go/v7"

	"github.com/BhuvaneswariShanker/image-converter-backend/internal/config"
)

func UploadRawFile(filename string, content []byte, contentType string) error {
	reader := bytes.NewReader(content)
	_, err := minioClient.PutObject(context.Background(), config.GetEnv("CONVERTED_BUCKET_NAME", "converted"), filename, reader, int64(len(content)), minio.PutObjectOptions{
		ContentType: contentType,
	})
	if err != nil {
		return fmt.Errorf("failed to upload to MinIO: %v", err)
	}
	return nil
}
