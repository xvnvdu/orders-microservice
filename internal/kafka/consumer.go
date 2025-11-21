package kafka

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net"
	"orders/internal/generator"
	"orders/internal/repository"
	"strconv"
	"time"

	"github.com/segmentio/kafka-go"
)

const (
	topic   string = "orders"
	address string = "kafka:9092"
)

func CreateReader() *kafka.Reader {
	r := kafka.NewReader(kafka.ReaderConfig{
		Brokers:   []string{address},
		Topic:     topic,
		GroupID:   "orders-group",
		Partition: 0,
	})
	return r
}

func CreateTopic() error {
	var conn *kafka.Conn
	var err error
	maxRetries := 10

	for i := 0; i < maxRetries; i++ {
		conn, err = kafka.Dial("tcp", address)
		if err == nil {
			break
		}
		log.Printf("Error creating Kafka connection (attempt %d/%d): %v", i+1, maxRetries, err)

		if i == maxRetries-1 {
			return fmt.Errorf("Failed to connect to Kafka after all attempts.")
		}

		time.Sleep(time.Second * 5)
	}
	defer conn.Close()

	controller, err := conn.Controller()
	if err != nil {
		return fmt.Errorf("Error creating kafka controller: %w", err)
	}

	var controllerConn *kafka.Conn
	controllerConn, err = kafka.Dial("tcp", net.JoinHostPort(controller.Host, strconv.Itoa(controller.Port)))
	if err != nil {
		return fmt.Errorf("Error creating controlerConn: %w", err)
	}
	defer controllerConn.Close()

	topicConfigs := []kafka.TopicConfig{
		{
			Topic:             topic,
			NumPartitions:     1,
			ReplicationFactor: 1,
		},
	}

	err = controllerConn.CreateTopics(topicConfigs...)
	if err != nil {
		return fmt.Errorf("%w", err)
	}
	log.Printf("Topic %s created successfuly on %s", topic, address)

	return nil
}

func StartConsuming(c MessagesConsumer, repo repository.OrdersRepository) {
	ctx := context.Background()
	for {
		m, err := c.FetchMessage(context.Background())
		if err != nil {
			// При graceful shutdown ридер закрывается,
			// эту ошибку пропускаем
			if errors.Is(err, io.EOF) {
				break
			}
			log.Println("Error reading message:", err)
			break
		}
		log.Printf("New message at topic/partition/offset %v/%v/%v: %s = %s\n",
			m.Topic, m.Partition, m.Offset, string(m.Key), string(m.Value))

		var orders []*generator.Order
		err = json.Unmarshal(m.Value, &orders)
		if err != nil {
			log.Println("Error unmarshalling orders data:", err)
			continue
		}

		err = repo.SaveToDB(orders, ctx)
		if err != nil {
			log.Printf("Failed to save orders from Kafka message: %v\n", err)
			continue
		}

		if err := c.CommitMessages(ctx, m); err != nil {
			log.Fatalln("Error committing message:", err)
		}
		log.Printf("Committed message at topic/partition/offset %v/%v/%v\n",
			m.Topic, m.Partition, m.Offset)
	}
}
