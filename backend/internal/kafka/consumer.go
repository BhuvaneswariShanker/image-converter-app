package kafka

import (
	"context"
	"fmt"
	"log"

	"github.com/segmentio/kafka-go"
)

func StartKafkaConsumer(brokerAddress string, processFunc func(string)) {
	reader := kafka.NewReader(kafka.ReaderConfig{
		Brokers: []string{brokerAddress},
		Topic:   "file-events",
		GroupID: "image-converter-group",
	})

	go func() {
		fmt.Println("👂 Kafka consumer started...")
		for {
			msg, err := reader.ReadMessage(context.Background())
			if err != nil {
				log.Printf("❌ Error reading Kafka message: %v", err)
				continue
			}
			log.Printf("📨 Received message: %s", string(msg.Value))
			processFunc(string(msg.Value))

		}
	}()
}
