package cache

import (
	"context"
	g "orders/internal/generator"
)

// OrdersCache описывает поведение структуры Cache
type OrdersCache interface {
	LoadInitialOrders(ctx context.Context, latestOrders []*g.Order, limit int32)
	GetFromCache(ctx context.Context, uid string) (*g.Order, error)
	UpdateCache(ctx context.Context, order *g.Order) error
	Close() error
}
