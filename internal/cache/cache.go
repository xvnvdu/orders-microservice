package cache

import (
	"context"
	"encoding/json"
	"log"
	"time"

	g "orders/internal/generator"

	"github.com/redis/go-redis/v9"
)

type Cache struct {
	redisClient *redis.Client
	Capacity    int32
}

const CacheCapacity int32 = 200

func NewCache(redisURL string) (*Cache, error) {
	opt, err := redis.ParseURL(redisURL)
	if err != nil {
		log.Fatalln("Error connecting to redis:", err)
	}

	rdb := redis.NewClient(opt)
	return &Cache{redisClient: rdb, Capacity: CacheCapacity}, nil
}

func (c *Cache) LoadInitialOrders(ctx context.Context, latestOrders []*g.Order, limit int32) {
	var successfulOrders int

	for _, order := range latestOrders {
		orderJSON, err := json.Marshal(order)
		if err != nil {
			log.Println("Error marshalling order to JSON:", err)
			continue
		}

		redisKey := order.OrderUID
		err = c.addToCache(ctx, redisKey, orderJSON)
		if err != nil {
			log.Printf("Error adding order with uid %s\n", redisKey)
			continue
		}

		err = c.updateLRU(ctx, redisKey)
		if err != nil {
			log.Printf("Error updating LRU with order with uid %s\n", redisKey)
			continue
		}

		successfulOrders++
	}
	log.Printf("Cache filled with %d/%d orders, running on redis:6379\n", successfulOrders, CacheCapacity)
}

func (c *Cache) addToCache(ctx context.Context, redisKey string, orderJSON []byte) error {
	err := c.redisClient.Set(ctx, redisKey, orderJSON, 0).Err()
	if err != nil {
		log.Println("Error adding order to Redis:", err)
		return err
	}
	return nil
}

func (c *Cache) updateLRU(ctx context.Context, uid string) error {
	zKey := "LRU-orders"

	now := float64(time.Now().UnixMilli())
	err := c.redisClient.ZAdd(ctx, zKey, redis.Z{
		Member: uid,
		Score:  now,
	}).Err()
	if err != nil {
		log.Println("Error adding to ZSET:")
		return err
	}

	currentCapacity, err := c.redisClient.ZCard(ctx, zKey).Result()
	if err != nil {
		log.Println("Error getting cache capacity:")
		return err
	}

	for currentCapacity > int64(CacheCapacity) {
		members, err := c.redisClient.ZPopMin(ctx, zKey, 1).Result()
		if err != nil {
			log.Println("Error removing order with lowest score:", err)
			return err
		}
		currentCapacity--

		lowestScoreUid := members[0].Member.(string)
		err = c.removeFromCache(ctx, lowestScoreUid)
		if err != nil {
			return err
		}
	}
	return nil
}

func (c *Cache) removeFromCache(ctx context.Context, uid string) error {
	err := c.redisClient.Del(ctx, uid).Err()
	if err != nil {
		log.Printf("Error removing order with uid %s from cache: %d\n", uid, err)
		return err
	}
	return nil
}

func (c *Cache) GetFromCache(ctx context.Context, uid string) (*g.Order, error) {
	cmd := c.redisClient.Get(ctx, uid)
	if cmd.Err() != nil {
		log.Println("Can't find cached data for", uid)
		return nil, cmd.Err()
	}

	orderJSON, err := cmd.Bytes()
	if err != nil {
		log.Println("Error marshalling cached data for", uid)
		return nil, err
	}

	var order g.Order
	err = json.Unmarshal(orderJSON, &order)
	if err != nil {
		log.Println("Error marshalling cached data for", uid)
		return nil, err
	}

	err = c.updateLRU(ctx, uid)
	if err != nil {
		return nil, err
	}
	return &order, nil
}

func (c *Cache) UpdateCache(ctx context.Context, order *g.Order) error {
	orderJSON, err := json.Marshal(order)
	if err != nil {
		log.Println("Error marshalling order before adding to cache:", err)
		return err
	}

	redisKey := order.OrderUID
	err = c.addToCache(ctx, redisKey, orderJSON)
	if err != nil {
		return err
	}

	err = c.updateLRU(ctx, redisKey)
	if err != nil {
		return err
	}
	return nil
}

func (c *Cache) Close() error {
	err := c.redisClient.Close()
	if err != nil {
		return err
	}
	return nil
}
