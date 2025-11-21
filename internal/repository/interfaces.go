package repository

import (
	"context"
	g "orders/internal/generator"
)

// OrdersRepository описывает поведение структуры Repository
type OrdersRepository interface {
	SaveToDB(orders []*g.Order, ctx context.Context) error
	GetOrderById(order_uid string, ctx context.Context, useCache bool) (*g.Order, error)
	GetAllOrders(ctx context.Context) ([]*g.Order, error)
	GetLatestOrders(ctx context.Context, limit int32) ([]*g.Order, error)
	Close() error
}
