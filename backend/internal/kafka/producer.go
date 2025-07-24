package kafka

import (
	"context"
	"log"

	"github.com/segmentio/kafka-go"
)

var Writer *kafka.Writer

func InitProducer(brokerAddress string) {
	Writer = &kafka.Writer{
		Addr:     kafka.TCP(brokerAddress),
		Topic:    "file-events",
		Balancer: &kafka.LeastBytes{},
	}
}

func PublishMessage(message string) error {
	err := Writer.WriteMessages(context.Background(),
		kafka.Message{
			Key:   []byte("upload"),
			Value: []byte(message),
		},
	)
	if err != nil {
		log.Printf("Failed to write message to Kafka: %v", err)
	}
	return err
}
