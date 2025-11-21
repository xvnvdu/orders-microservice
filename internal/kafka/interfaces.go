package kafka

import (
	"context"

	"github.com/segmentio/kafka-go"
)

// MessagesConsumer описывает поведение kafka.Reader
type MessagesConsumer interface {
	FetchMessage(ctx context.Context) (kafka.Message, error)
	CommitMessages(ctx context.Context, msgs ...kafka.Message) error
	Close() error
}

// MessagesProducer описывает поведение kafka.Writer
type MessagesProducer interface {
	WriteMessages(ctx context.Context, msgs ...kafka.Message) error
	Close() error
}
