# image-converter-app
🚀 Event-driven PDF to JPG/ZIP converter with Angular frontend and Go microservices using Kafka, MinIO, and Docker. Converts PDFs, notifies via WebSocket, and auto-deletes files post-download.

# PDF to JPG/ZIP Converter (Local, Event-Driven Microservice)

A lightweight, local-first PDF to JPG converter that generates:
- A single `.jpg` for 1-page PDFs
- A `.zip` file containing one `.jpg` per page for multi-page PDFs

Built using **Angular 18** frontend and **Go microservices** on the backend with **Kafka**, **MinIO**, and **Docker**.

---

## 🧱 Tech Stack

### Frontend:
- Angular 18
- WebSocket support for real-time conversion status

### Backend (Microservices in Go):
- **Producer**: Auth + Upload + Kafka Publish
- **Consumer**: File Conversion + WebSocket Notify
- **Downloader**: Download + Cleanup

### Infrastructure:
- Kafka (Confluent image)
- Zookeeper
- MinIO (S3-compatible)
- Docker + Docker Compose
- JWT (3 min expiry)
- Rate Limiting (5 requests / 10 seconds)

---

## ⚙️ Architecture Overview

```text
Frontend (Angular 18)
  └── Calls Producer APIs
       ├── POST /token         => JWT (expires in 3 mins)
       └── POST /upload        => Upload PDF to MinIO + Kafka message
       
Producer (Go)
  ├── Auth & Upload handler
  └── Publishes to Kafka Topic

Consumer (Go)
  ├── Consumes Kafka message
  ├── Downloads PDF from MinIO
  ├── Converts PDF to JPG or ZIP
  ├── Uploads result to MinIO
  └── Sends WebSocket event to frontend

Downloader (Go)
  ├── GET /download           => Fetch result from MinIO
  └── On successful download => Deletes original + converted files from MinIO
