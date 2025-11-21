package repository

import (
	"context"
	"database/sql"
	"log"

	c "orders/internal/cache"
	db "orders/internal/database"
	g "orders/internal/generator"
)

type Repository struct {
	DB    *sql.DB
	Cache *c.Cache
}

func NewRepository(driverName, dataSourceName string, cache *c.Cache) (*Repository, error) {
	db, err := sql.Open(driverName, dataSourceName)
	if err != nil {
		return nil, err
	}

	err = db.Ping()
	if err != nil {
		mainErr := err

		if closeErr := db.Close(); closeErr != nil {
			log.Println("NewRepository: Database connection can't be closed:", closeErr)
		}
		return nil, mainErr
	}
	log.Println("Database connection opened on db:5432")

	return &Repository{DB: db, Cache: cache}, nil
}

func (r *Repository) SaveToDB(orders []*g.Order, ctx context.Context) error {
	queries := db.New(r.DB)

	for _, order := range orders {
		err := queries.CreateOrder(ctx, db.CreateOrderParams{
			OrderUid:    order.OrderUID,
			TrackNumber: order.TrackNumber,
			Entry:       order.Entry,
			Locale:      order.Locale,
			InternalSignature: sql.NullString{
				String: order.InternalSignature,
				Valid:  order.InternalSignature != "",
			},
			CustomerID:      order.CustomerID,
			DeliveryService: order.DeliveryService,
			Shardkey:        order.Shardkey,
			SmID:            int32(order.SmID),
			DateCreated:     order.DateCreated,
			OofShard:        order.OofShard,
		})
		if err != nil {
			log.Println("Error inserting order:", err)
			return err

		}

		err = queries.CreateDelivery(ctx, db.CreateDeliveryParams{
			OrderUid: order.OrderUID,
			Name:     order.Delivery.Name,
			Phone:    order.Delivery.Phone,
			Zip:      order.Delivery.Zip,
			City:     order.Delivery.City,
			Address:  order.Delivery.Address,
			Region:   order.Delivery.Region,
			Email:    order.Delivery.Email,
		})
		if err != nil {
			log.Println("Error inserting delivery:", err)
			return err

		}

		err = queries.CreatePayment(ctx, db.CreatePaymentParams{
			OrderUid:    order.OrderUID,
			Transaction: order.Payment.Transaction,
			RequestID: sql.NullString{
				String: order.Payment.RequestID,
				Valid:  order.Payment.RequestID != "",
			},
			Currency:     order.Payment.Currency,
			Provider:     order.Payment.Provider,
			Amount:       int32(order.Payment.Amount),
			PaymentDt:    int64(order.Payment.PaymentDT),
			Bank:         order.Payment.Bank,
			DeliveryCost: int32(order.Payment.DeliveryCost),
			GoodsTotal:   int32(order.Payment.GoodsTotal),
			CustomFee:    int32(order.Payment.CustomFee),
		})
		if err != nil {
			log.Println("Error inserting payment:", err)
			return err

		}

		for _, item := range order.Items {
			err = queries.CreateItem(ctx, db.CreateItemParams{
				OrderUid:    order.OrderUID,
				ChrtID:      int32(item.ChrtID),
				TrackNumber: item.TrackNumber,
				Price:       int32(item.Price),
				Rid:         item.Rid,
				Name:        item.Name,
				Sale:        int32(item.Sale),
				Size:        item.Size,
				TotalPrice:  int32(item.TotalPrice),
				NmID:        int32(item.NmID),
				Brand:       item.Brand,
				Status:      int32(item.Status),
			})
			if err != nil {
				log.Println("Error inserting item:", err)
				return err
			}
		}

		err = r.Cache.UpdateCache(ctx, order)
		if err != nil {
			return err
		}
	}
	return nil
}

func (r *Repository) GetOrderById(order_uid string, ctx context.Context, useCache bool) (*g.Order, error) {
	var orderData *g.Order

	if useCache {
		orderData, err := r.Cache.GetFromCache(ctx, order_uid)
		if err != nil {
			useCache = false
		} else {
			return orderData, nil
		}
	}

	if !useCache {
		queries := db.New(r.DB)

		order, err := queries.GetSpecificOrder(ctx, order_uid)
		if err != nil {
			log.Println("Error getting order:", err)
			return nil, err
		}

		delivery, err := queries.GetSpecificDelivery(ctx, order_uid)
		if err != nil {
			log.Println("Error getting delivery:", err)
			return nil, err
		}

		payments, err := queries.GetSpecificPayment(ctx, order_uid)
		if err != nil {
			log.Println("Error getting payment:", err)
			return nil, err
		}

		items, err := queries.GetSpecificItems(ctx, order_uid)
		if err != nil {
			log.Println("Error getting items:", err)
			return nil, err
		}

		var itemsList []g.Item
		for _, item := range items {
			itemsList = append(itemsList, g.Item{
				ChrtID:      int(item.ChrtID),
				TrackNumber: item.TrackNumber,
				Price:       int(item.Price),
				Rid:         item.Rid,
				Name:        item.Name,
				Sale:        int(item.Sale),
				Size:        item.Size,
				TotalPrice:  int(item.TotalPrice),
				NmID:        int(item.NmID),
				Brand:       item.Brand,
				Status:      int(item.Status),
			})
		}

		orderData = &g.Order{
			OrderUID:    order.OrderUid,
			TrackNumber: order.TrackNumber,
			Entry:       order.Entry,
			Delivery: g.Delivery{
				Name:    delivery.Name,
				Phone:   delivery.Phone,
				Zip:     delivery.Zip,
				City:    delivery.City,
				Address: delivery.Address,
				Region:  delivery.Region,
				Email:   delivery.Email,
			},
			Payment: g.Payment{
				Transaction:  payments.Transaction,
				RequestID:    payments.RequestID.String,
				Currency:     payments.Currency,
				Provider:     payments.Provider,
				Amount:       int(payments.Amount),
				PaymentDT:    int(payments.PaymentDt),
				Bank:         payments.Bank,
				DeliveryCost: int(payments.DeliveryCost),
				GoodsTotal:   int(payments.GoodsTotal),
				CustomFee:    int(payments.CustomFee),
			},
			Items:             itemsList,
			Locale:            order.Locale,
			InternalSignature: order.InternalSignature.String,
			CustomerID:        order.CustomerID,
			DeliveryService:   order.DeliveryService,
			Shardkey:          order.Shardkey,
			SmID:              int(order.SmID),
			DateCreated:       order.DateCreated,
			OofShard:          order.OofShard,
		}

		err = r.Cache.UpdateCache(ctx, orderData)
		if err != nil {
			return nil, err
		}
	}
	return orderData, nil
}

func (r *Repository) GetAllOrders(ctx context.Context) ([]*g.Order, error) {
	queries := db.New(r.DB)

	orders, err := queries.GetOrders(ctx)
	if err != nil {
		log.Fatalln("Error getting orders:", err)
		return nil, err
	}

	deliveries, err := queries.GetDelivery(ctx)
	if err != nil {
		log.Fatalln("Error getting deliveries:", err)
		return nil, err
	}

	payments, err := queries.GetPayment(ctx)
	if err != nil {
		log.Fatalln("Error getting payments:", err)
		return nil, err
	}

	items, err := queries.GetItems(ctx)
	if err != nil {
		log.Fatalln("Error getting items:", err)
		return nil, err

	}

	deliveriesMap := make(map[string]g.Delivery)
	paymentsMap := make(map[string]g.Payment)
	itemsMap := make(map[string][]g.Item)

	for _, delivery := range deliveries {
		deliveriesMap[delivery.OrderUid] = g.Delivery{
			Name:    delivery.Name,
			Phone:   delivery.Phone,
			Zip:     delivery.Zip,
			City:    delivery.City,
			Address: delivery.Address,
			Region:  delivery.Region,
			Email:   delivery.Email,
		}
	}

	for _, payment := range payments {
		paymentsMap[payment.OrderUid] = g.Payment{
			Transaction:  payment.Transaction,
			RequestID:    payment.RequestID.String,
			Currency:     payment.Currency,
			Provider:     payment.Provider,
			Amount:       int(payment.Amount),
			PaymentDT:    int(payment.PaymentDt),
			Bank:         payment.Bank,
			DeliveryCost: int(payment.DeliveryCost),
			GoodsTotal:   int(payment.GoodsTotal),
			CustomFee:    int(payment.CustomFee),
		}
	}

	for _, item := range items {
		if itemsMap[item.OrderUid] == nil {
			itemsMap[item.OrderUid] = []g.Item{}
		}
		itemsMap[item.OrderUid] = append(itemsMap[item.OrderUid], g.Item{
			ChrtID:      int(item.ChrtID),
			TrackNumber: item.TrackNumber,
			Price:       int(item.Price),
			Rid:         item.Rid,
			Name:        item.Name,
			Sale:        int(item.Sale),
			Size:        item.Size,
			TotalPrice:  int(item.TotalPrice),
			NmID:        int(item.NmID),
			Brand:       item.Brand,
			Status:      int(item.Status),
		})
	}

	var ordersList []*g.Order
	for _, order := range orders {
		ordersList = append(ordersList, &g.Order{
			OrderUID:          order.OrderUid,
			TrackNumber:       order.TrackNumber,
			Entry:             order.Entry,
			Delivery:          deliveriesMap[order.OrderUid],
			Payment:           paymentsMap[order.OrderUid],
			Items:             itemsMap[order.OrderUid],
			Locale:            order.Locale,
			InternalSignature: order.InternalSignature.String,
			CustomerID:        order.CustomerID,
			DeliveryService:   order.DeliveryService,
			Shardkey:          order.Shardkey,
			SmID:              int(order.SmID),
			DateCreated:       order.DateCreated,
			OofShard:          order.OofShard,
		})
	}

	return ordersList, nil
}

func (r *Repository) GetLatestOrders(ctx context.Context, limit int32) ([]*g.Order, error) {
	queries := db.New(r.DB)

	latestOrders, err := queries.GetLatestOrders(ctx, limit)
	if err != nil {
		log.Println("Error getting latest orders:", err)
		return nil, err
	}

	var ordersList []*g.Order
	for _, orderUID := range latestOrders {
		orderData, err := r.GetOrderById(orderUID, ctx, false)
		if err != nil {
			log.Println("Error getting order data by id:", err)
			continue
		}
		ordersList = append(ordersList, orderData)
	}

	return ordersList, nil
}

func (r *Repository) Close() error {
	err := r.DB.Close()
	if err != nil {
		return err
	}
	return nil
}
