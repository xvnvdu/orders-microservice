package kafka

import (
	"context"
	"log"

	"github.com/segmentio/kafka-go"
)

func CreateWriter() *kafka.Writer {
	w := &kafka.Writer{
		Addr:     kafka.TCP(address),
		Topic:    topic,
		Balancer: &kafka.LeastBytes{},
	}
	return w
}

func WriteMessage(p MessagesProducer, ctx context.Context, msg []byte) error {
	err := p.WriteMessages(ctx,
		kafka.Message{
			Key:   nil,
			Value: []byte(msg),
		},
	)
	if err != nil {
		log.Println("Failed to write message:", err)
		return err
	}

	return nil
}
