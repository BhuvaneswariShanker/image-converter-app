package websocket

import (
	"log"
	"net/http"
	"sync"

	"github.com/gorilla/websocket"
)

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		// Allow requests from any origin (for development)
		return true
	},
}

// jobId → client WebSocket connections
var clients = make(map[string]*websocket.Conn)
var clientsLock = sync.RWMutex{}

func WsHandlerGin(w http.ResponseWriter, r *http.Request, jobId string) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Println("WebSocket upgrade error:", err)
		return
	}
	defer conn.Close()

	log.Println("WebSocket connected for job:", jobId)

	clientsLock.Lock()
	clients[jobId] = conn
	clientsLock.Unlock()

	for {
		_, _, err := conn.ReadMessage()
		if err != nil {
			log.Println("WebSocket closed:", jobId)
			break
		}
	}

	clientsLock.Lock()
	delete(clients, jobId)
	clientsLock.Unlock()
}

func NotifyClient(jobId string, filename string) {
	clientsLock.RLock()
	conn, ok := clients[jobId]
	clientsLock.RUnlock()

	if !ok {
		log.Println("No WebSocket client for job:", jobId)
		return
	}

	msg := map[string]interface{}{
		"type":     "conversion-complete",
		"filename": filename,
	}

	err := conn.WriteJSON(msg)
	if err != nil {
		log.Println("WebSocket write error:", err)
	}
}
