package main

import (
	"context"
	"errors"
	"log"
	"net/http"
	"orders/internal/app"
	"orders/internal/dependencies"
	"os"
	"os/signal"
	"syscall"

	"github.com/joho/godotenv"
	"github.com/swaggo/http-swagger"
)

func main() {
	godotenv.Load()

	dbURL := os.Getenv("DB_CONN_STRING")
	if dbURL == "" {
		log.Fatalln("DB_CONN_STRING is not found")
	}

	driver := os.Getenv("DRIVER")
	if driver == "" {
		log.Fatalln("DRIVER is not found")
	}

	redisURL := os.Getenv("REDIS_CONN_STRING")
	if redisURL == "" {
		log.Fatalln("Redis URL is not found")
	}

	// Создаем внешние зависимости сервиса
	deps, err := dependencies.InitDependencies(driver, dbURL, redisURL)
	if err != nil {
		log.Fatalf("Failed to init dependencies: %s", err)
	}

	// Передаем зависимости и инициализируем приложение
	myApp := app.NewApp(deps)

	// Создаем контекст для остановки сервиса при получении сигнала
	sigCtx, cancel := signal.NotifyContext(context.Background(), syscall.SIGTERM, syscall.SIGINT)
	defer cancel()

	// Отдаем статику
	staticFileServer := http.FileServer(http.Dir("web/static"))
	http.Handle("/static/", http.StripPrefix("/static/", staticFileServer))

	// Основные эндпоинты
	http.HandleFunc("/", myApp.HomeHandler)
	http.HandleFunc("/orders", myApp.ShowOrdersHandler)
	http.HandleFunc("/orders/{order_uid}", myApp.GetOrderByIdHandler)
	http.HandleFunc("/random/{amount}", myApp.RandomOrdersHandler)

	// Отдаем файл с документацией и рендерим его по эндпоинту /docs
	http.HandleFunc("/swagger.yaml", func(w http.ResponseWriter, r *http.Request) {
		http.ServeFile(w, r, "./docs/swagger.yaml")
	})
	http.Handle("/docs/", httpSwagger.Handler(httpSwagger.URL("/swagger.yaml")))

	// Создаем сервер
	server := &http.Server{
		Addr: ":8080",
	}
	// Запускаем сервер фоном, ListenAndServe - блокирующая функция
	go func() {
		log.Println("Server is running on http://localhost:8080")
		if err := server.ListenAndServe(); !errors.Is(err, http.ErrServerClosed) {
			log.Fatalln("Server error:", err)
		}
	}()

	// Блокируем завершение главной горутины, ожидая сигнал
	<-sigCtx.Done()
	// При получении сигнала останавливаем все процессы далее

	log.Println("Service stopped by a signal: shutting down HTTP server...")
	if err := server.Shutdown(context.Background()); err != nil {
		log.Printf("HTTP server shutdown error: %v", err)
	}

	if err := myApp.Close(); err != nil {
		log.Println("Service resources close error:", err)
	}
}
