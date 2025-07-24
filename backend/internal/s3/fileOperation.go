package s3

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"log"
	"mime/multipart"
	"os"
	"time"

	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"

	"github.com/BhuvaneswariShanker/image-converter-backend/internal/config"
)

var minioClient *minio.Client

func InitS3() {
	var err error
	minioClient, err = minio.New(config.GetEnv("MINIO_ENDPOINT", "http://localhost:9000"), &minio.Options{
		Creds:  credentials.NewStaticV4(config.GetEnv("MINIO_ACCESS_KEY", "minioadmin"), config.GetEnv("MINIO_SECRET_KEY", "minioadmin"), ""),
		Secure: false, // Keep false for HTTP
	})
	if err != nil {
		log.Fatalf("Failed to initialize MinIO client: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	err1 := minioClient.MakeBucket(ctx, config.GetEnv("UPLOAD_BUCKET_NAME", "uploads"), minio.MakeBucketOptions{})
	if err1 != nil {
		exists, errBucketExists := minioClient.BucketExists(ctx, config.GetEnv("UPLOAD_BUCKET_NAME", "uploads"))
		if errBucketExists == nil && exists {
			log.Printf("Bucket already exists.")
		} else {
			log.Fatalf("Failed to create bucket: %v", err)
		}
	}

	err2 := minioClient.MakeBucket(ctx, config.GetEnv("CONVERTED_BUCKET_NAME", "converted"), minio.MakeBucketOptions{})
	if err2 != nil {
		exists, errBucketExists := minioClient.BucketExists(ctx, config.GetEnv("CONVERTED_BUCKET_NAME", "converted"))
		if errBucketExists == nil && exists {
			log.Printf("Bucket already exists.")
		} else {
			log.Fatalf("Failed to create bucket: %v", err)
		}
	}

}

func UploadFile(filename string, file multipart.File, fileHeader *multipart.FileHeader) error {
	buf := new(bytes.Buffer)
	_, err := buf.ReadFrom(file)
	if err != nil {
		return fmt.Errorf("failed to read file: %v", err)
	}

	_, err = minioClient.PutObject(context.Background(), config.GetEnv("UPLOAD_BUCKET_NAME", "uploads"), filename, bytes.NewReader(buf.Bytes()), fileHeader.Size, minio.PutObjectOptions{
		ContentType: fileHeader.Header.Get("Content-Type"),
	})
	if err != nil {
		return fmt.Errorf("failed to upload to MinIO: %v", err)
	}
	return nil
}

func DownloadFileIntoLocal(objectKey string, bucket string, localPath string) error {
	obj, err := minioClient.GetObject(context.Background(), bucket, objectKey, minio.GetObjectOptions{})
	if err != nil {
		return err
	}
	defer obj.Close()

	file, err := os.Create(localPath)
	if err != nil {
		return err
	}
	defer file.Close()

	_, err = io.Copy(file, obj)
	return err
}

func DeleteFile(bucketName string, objectName string) error {
	err := minioClient.RemoveObject(context.Background(), bucketName, objectName, minio.RemoveObjectOptions{})
	if err != nil {
		log.Printf("❌ Failed to delete object %s/%s: %v", bucketName, objectName, err)
		return err
	}

	log.Printf("🗑️ Deleted object: %s/%s", bucketName, objectName)
	return nil
}
