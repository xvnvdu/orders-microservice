package dependencies

//go:generate mockgen -source=../repository/interfaces.go -destination=../mocks/repository_mock.go -package=mocks
//go:generate mockgen -source=../cache/interfaces.go -destination=../mocks/cache_mock.go -package=mocks
//go:generate mockgen -source=../kafka/interfaces.go -destination=../mocks/kafka_mock.go -package=mocks

import (
	"fmt"

	c "orders/internal/cache"
	k "orders/internal/kafka"
	r "orders/internal/repository"
)

type Dependencies struct {
	KafkaConsumer k.MessagesConsumer
	KafkaProducer k.MessagesProducer
	Repo          r.OrdersRepository
	Cache         c.OrdersCache
}

func InitDependencies(driverName, dataSourceName, redisURL string) (*Dependencies, error) {
	cache, err := c.NewCache(redisURL)
	if err != nil {
		return nil, fmt.Errorf("Error creating new cache: %w", err)
	}

	repo, err := r.NewRepository(driverName, dataSourceName, cache)
	if err != nil {
		return nil, fmt.Errorf("Error creating new repository: %w", err)
	}

	err = k.CreateTopic()
	if err != nil {
		return nil, fmt.Errorf("Error creating Kafka topic: %w", err)
	}

	reader := k.CreateReader()
	writer := k.CreateWriter()

	return &Dependencies{
		KafkaConsumer: reader,
		KafkaProducer: writer,
		Repo:          repo,
		Cache:         cache,
	}, nil
}
