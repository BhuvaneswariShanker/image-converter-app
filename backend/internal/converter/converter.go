package converter

import (
	"archive/zip"
	"bytes"
	"io/ioutil"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/BhuvaneswariShanker/image-converter-backend/internal/s3"
	"github.com/BhuvaneswariShanker/image-converter-backend/internal/websocket"
)

func ConvertAndStoreImage(message string) {
	jobId, fileName := splitByFirstSlash(message)
	log.Printf("🔄 Starting conversion for: %s", fileName)

	// Step 1: Download PDF from MinIO
	err1 := os.MkdirAll("data", 0755)
	if err1 != nil {
		log.Printf("❌ Failed to create directory: %v", err1)
		return
	}
	tempPDF := filepath.Join("data", filepath.Base(fileName))
	err := s3.DownloadFileIntoLocal(message, os.Getenv("UPLOAD_BUCKET_NAME"), tempPDF)
	if err != nil {
		log.Printf("❌ Failed to download from MinIO: %v", err)
		return
	}

	// Step 2: Convert PDF to JPG using pdftoppm
	outputPrefix := strings.TrimSuffix(tempPDF, ".pdf")
	cmd := exec.Command("pdftoppm", "-jpeg", tempPDF, outputPrefix)
	var stderr bytes.Buffer
	cmd.Stderr = &stderr

	err = cmd.Run()
	if err != nil {
		log.Printf("❌ Conversion error: %v\nDetails: %s", err, stderr.String())
		return
	}

	// Step 3: Upload all converted images to MinIO
	files, err := filepath.Glob(outputPrefix + "-*.jpg")
	if err != nil || len(files) == 0 {
		log.Printf("❌ No output images found for: %s", fileName)
		return
	}

	if len(files) == 1 {
		// Single image - no need to zip
		singleImagePath := files[0]
		log.Printf("✅ Single image found: %s, skipping zip", singleImagePath)

		// Read image into memory
		imgData, err := ioutil.ReadFile(singleImagePath)
		if err != nil {
			log.Printf("❌ Failed to read single image: %v", err)
			return
		}

		// Set image name with .jpg extension
		imgName := strings.TrimSuffix(message, filepath.Ext(message)) + ".jpg"

		// Upload image to MinIO with correct content type
		err = s3.UploadRawFile(imgName, imgData, "image/jpeg")
		if err != nil {
			log.Printf("❌ Failed to upload image to MinIO: %v", err)
		} else {
			log.Printf("✅ Uploaded %s to MinIO", imgName)
		}

		sendWSNotification(jobId, imgName, tempPDF, files)

		return

	}

	var buf bytes.Buffer
	zipWriter := zip.NewWriter(&buf)

	for _, imgPath := range files {
		imgData, err := ioutil.ReadFile(imgPath)
		if err != nil {
			log.Printf("❌ Error reading image file %s: %v", imgPath, err)
			continue
		}

		fileName := filepath.Base(imgPath)
		writer, err := zipWriter.Create(fileName)
		if err != nil {
			log.Printf("❌ Error adding file %s to zip: %v", fileName, err)
			continue
		}

		_, err = writer.Write(imgData)
		if err != nil {
			log.Printf("❌ Error writing file %s to zip: %v", fileName, err)
			continue
		}
	}

	err3 := zipWriter.Close()
	if err3 != nil {
		log.Printf("❌ Error closing zip writer: %v", err3)
		return
	}

	// Upload compressed archive to MinIO
	zipName := strings.TrimSuffix(message, filepath.Ext(message)) + ".zip"
	err = s3.UploadRawFile(zipName, buf.Bytes(), "application/zip")
	if err != nil {
		log.Printf("❌ Failed to upload zip to MinIO: %v", err)
	} else {
		log.Printf("✅ Uploaded %s to MinIO", zipName)
	}

	sendWSNotification(jobId, zipName, tempPDF, files)

}

func sendWSNotification(jobId string, filename string, tempPDF string, files []string) {
	// inform frontend that data is available
	websocket.NotifyClient(jobId, filename)

	// Cleanup
	_ = os.Remove(tempPDF)
	for _, f := range files {
		_ = os.Remove(f)
	}
}

func splitByFirstSlash(s string) (string, string) {
	index := strings.Index(s, "/")
	if index == -1 {
		return s, "" // No slash found
	}
	return s[:index], s[index+1:]
}
