package app

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"os"
	"strconv"

	"orders/internal/dependencies"
	"orders/internal/generator"
	"orders/internal/repository"

	c "orders/internal/cache"
	k "orders/internal/kafka"

	_ "github.com/lib/pq"
)

type App struct {
	kafkaConsumer k.MessagesConsumer
	kafkaProducer k.MessagesProducer
	repo          repository.OrdersRepository
	cache         c.OrdersCache
}

func (a *App) HomeHandler(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/" {
		notFound, err := os.ReadFile("web/templates/404.html")
		if err != nil {
			log.Println("Error reading file:", err)
			http.Error(w, "Nothing's here...", http.StatusNotFound)
			return
		}
		if _, err := w.Write([]byte(notFound)); err != nil {
			log.Fatalln("Handler error: HomeHandler:", err)
		}
		return
	}

	html, err := os.ReadFile("web/templates/index.html")
	if err != nil {
		log.Println("Error reading file:", err)
		http.Error(w, "Nothing's here...", http.StatusNotFound)
		return
	}

	if _, err := w.Write([]byte(html)); err != nil {
		log.Fatalln("Handler error: HomeHandler:", err)
	}
}

func (a *App) GetOrderByIdHandler(w http.ResponseWriter, r *http.Request) {
	orderUID := r.PathValue("order_uid")
	ctx := context.Background()

	orderData, err := a.repo.GetOrderById(orderUID, ctx, true)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			http.Error(w, "Order not found", http.StatusNotFound)
		} else {
			http.Error(w, "Internal server error", http.StatusInternalServerError)
		}
	}

	orderJSON, err := json.MarshalIndent(orderData, "", "    ")
	if err != nil {
		log.Println("Error marshalling JSON:", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	if _, err := w.Write([]byte(orderJSON)); err != nil {
		log.Fatalln("Handler error: GetOrderByIdHandler:", err)
	}
}

func (a *App) ShowOrdersHandler(w http.ResponseWriter, r *http.Request) {
	ctx := context.Background()

	ordersList, err := a.repo.GetAllOrders(ctx)
	if err != nil {
		http.Error(w, "Internal server error", http.StatusInternalServerError)
	}

	ordersJSON, err := json.MarshalIndent(ordersList, "", "    ")
	if err != nil {
		log.Println("Error marshalling JSON:", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	if _, err := w.Write([]byte(ordersJSON)); err != nil {
		log.Fatalln("Handler error: ShowOrdersHandler:", err)
	}
}

func (a *App) RandomOrdersHandler(w http.ResponseWriter, r *http.Request) {
	value := r.PathValue("amount")
	amount, err := strconv.Atoi(value)

	badRequest := func() {
		result, err := os.ReadFile("web/templates/400.html")
		if err != nil {
			http.Error(w, "Nothing's here...", http.StatusNotFound)
			log.Fatalln("Error reading file:", err)
		}
		if _, err := w.Write([]byte(result)); err != nil {
			log.Fatalln("Handler error: RandomOrdersHandler:", err)
		}
	}

	if err != nil {
		log.Println("Error in internal/app/app.go: amount, err := strconv.Atoi(value):", err)
		badRequest()
	} else {
		if amount <= 0 {
			log.Println("Error creating orders: Value is equal or less than zero")
			badRequest()
			return
		}

		ctx := context.Background()
		orders := generator.MakeRandomOrder(amount)

		orderJSON, err := json.MarshalIndent(orders, "", "    ")
		if err != nil {
			log.Println("Error marshalling JSON:", err)
			http.Error(w, "Internal server error", http.StatusInternalServerError)
			return
		}

		err = k.WriteMessage(a.kafkaProducer, ctx, orderJSON)
		if err != nil {
			http.Error(w, "Internal server error", http.StatusInternalServerError)
			return
		}

		if _, err := w.Write([]byte(orderJSON)); err != nil {
			log.Fatalln("Handler error: RandomOrdersHandler:", err)
		}
	}
}

func NewApp(d *dependencies.Dependencies) *App {
	ctx := context.Background()

	latestOrders, err := d.Repo.GetLatestOrders(ctx, c.CacheCapacity)
	if err == nil {
		d.Cache.LoadInitialOrders(ctx, latestOrders, c.CacheCapacity)
	} else {
		log.Println("Cache is empty, running on redis:6379")
	}

	go k.StartConsuming(d.KafkaConsumer, d.Repo)

	return &App{
		kafkaConsumer: d.KafkaConsumer,
		kafkaProducer: d.KafkaProducer,
		repo:          d.Repo,
		cache:         d.Cache,
	}
}

func (a App) Close() error {
	log.Println("Closing service connections...")
	var errs []error

	err := a.repo.Close()
	if err != nil {
		errs = append(errs, err)
		log.Println("Database connection can't be closed:", err)
	}

	err = a.cache.Close()
	if err != nil {
		errs = append(errs, err)
		log.Println("Cache connection can't be closed:", err)
	}

	err = a.kafkaConsumer.Close()
	if err != nil {
		errs = append(errs, err)
		log.Println("Kafka stream can't be closed:", err)
	}

	err = a.kafkaProducer.Close()
	if err != nil {
		errs = append(errs, err)
		log.Println("Kafka producer can't be closed:", err)
	}
	log.Println("Done!")

	return errors.Join(errs...)
}
