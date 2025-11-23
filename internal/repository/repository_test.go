package repository

import (
	"context"
	"testing"

	"orders/internal/generator"
	"orders/internal/mocks"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"

	_ "github.com/lib/pq"
)

// Строка подключения к тестовой бд
const testConnStr = "postgres://orders_user:12345@localhost:5432/orders_test_db?sslmode=disable"

// newTestRepo создает экземпляр репозитория с подключением к тестовой бд
func newTestRepo(t *testing.T, ctrl *gomock.Controller) *Repository {
	// Мокируем кэш, настоящий нам не нужен
	mockCache := mocks.NewMockOrdersCache(ctrl)

	// Создаем экземпляр репозитория
	testRepo, err := NewRepository("postgres", testConnStr, mockCache)
	// Небольшая подсказка для тех, кто не будет читать обновленный ридми :) [1]
	require.NoError(t, err, "Run: docker exec -it orders-microservice-db-1 psql -U orders_user -d orders_db -c \"CREATE DATABASE orders_test_db;\"")

	// Очищаем тестовую бд от имеющихся в ней данных
	_, err = testRepo.DB.Exec("TRUNCATE TABLE orders, delivery, payments, items RESTART IDENTITY")
	// Небольшая подсказка для тех, кто не будет читать обновленный ридми :) [2]
	require.NoError(t, err, "Run: docker exec -it orders-microservice-db-1 psql -U orders_user -d orders_test_db -f /docker-entrypoint-initdb.d/init.sql")

	return testRepo
}

// Тестирует сохранение и извлечение заказов в/из бд
func TestSaveAndGet(t *testing.T) {
	// Создаем контроллер
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	// Получаем экземпляр репозитория
	testRepo := newTestRepo(t, ctrl)
	defer testRepo.Close()

	ctx := context.Background()
	// Генерируем заказ
	ordersList := generator.MakeRandomOrder(1)
	testOrder := ordersList[0]

	// Получаем экземпляр мок кэша
	mockCache := testRepo.cache.(*mocks.MockOrdersCache)
	// Мокируем поведение реального кэша: нужно успешно кэшировать заказ после добавления в бд
	mockCache.EXPECT().UpdateCache(ctx, testOrder).Return(nil).Times(1)

	// Добавляем заказ в бд
	err := testRepo.SaveToDB(ordersList, ctx)
	require.NoError(t, err, "Failed to save test order to DB")
	t.Log("Test order saved to orders_test_db, order uid:", testOrder.OrderUID)

	// Снова мокируем поведение кэша, так как при извлечении заказа из бд он обновляется
	mockCache.EXPECT().UpdateCache(ctx, gomock.Any()).Return(nil).Times(1)

	t.Log("Preparing for retrieving order from DB...")
	// Получаем заказ именно из бд
	retrievedOrder, err := testRepo.GetOrderById(testOrder.OrderUID, ctx, false)
	require.NoError(t, err, "Failed to retrieve order from DB")

	t.Log("Order retrieved successfully")

	// Сравниваем результаты
	t.Logf("Expected OrderUID: %s, got: %s", testOrder.OrderUID, retrievedOrder.OrderUID)
	assert.Equal(t, testOrder.OrderUID, retrievedOrder.OrderUID, "OrderUID should match")

	t.Logf("Expected TrackNumber: %s, got: %s", testOrder.TrackNumber, retrievedOrder.TrackNumber)
	assert.Equal(t, testOrder.TrackNumber, retrievedOrder.TrackNumber, "TrackNumber should match")

	t.Logf("Expected CustomerID: %s, got: %s", testOrder.CustomerID, retrievedOrder.CustomerID)
	assert.Equal(t, testOrder.CustomerID, retrievedOrder.CustomerID, "CustomerID should match")
}

// generateOrdersAndSave является вспомогательной функцией для
// генерации и сохранения указанного количества заказов в бд
func generateOrdersAndSave(t *testing.T, ctrl *gomock.Controller, ctx context.Context, ordersAmount int) (*Repository, *mocks.MockOrdersCache) {
	testRepo := newTestRepo(t, ctrl)

	t.Logf("Preparing to generate %d random orders total...", ordersAmount)
	ordersList := generator.MakeRandomOrder(ordersAmount)

	mockCache := testRepo.cache.(*mocks.MockOrdersCache)
	mockCache.EXPECT().UpdateCache(ctx, gomock.Any()).Return(nil).Times(ordersAmount)

	err := testRepo.SaveToDB(ordersList, ctx)
	require.NoError(t, err, "Failed to save test orders to DB")
	t.Logf("Successfully saved %d orders to DB", ordersAmount)

	return testRepo, mockCache
}

// Тестирует извлечение ВСЕХ существующих заказов из бд
func TestGetAllOrders(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	ctx := context.Background()
	ordersAmount := 19
	// Используем вспомогательную функцию
	testRepo, _ := generateOrdersAndSave(t, ctrl, ctx, ordersAmount)
	defer testRepo.Close()

	// Получаем слайс со всеми заказами
	retrievedOrders, err := testRepo.GetAllOrders(ctx)
	require.NoError(t, err, "Failed to retrieve orders from DB")

	// Проверяем, что извлекли именно столько заказов, сколько сохраняли
	require.Equal(t, ordersAmount, len(retrievedOrders), "Retrieved orders count should match saved amount")
	t.Logf("Successfully retrieved orders list from DB. Expected: %d, got: %d", ordersAmount, len(retrievedOrders))
}

// Тестирует извлечение X самых новых заказов в бд, эта функия используется
// для наполнения кэша на старте и по умолчанию принимает 200 последних заказов
func TestGetLatestOrders(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	ctx := context.Background()
	ordersAmount := 83
	// Сгенерируем и сохраним указанное количество заказов
	testRepo, mockCache := generateOrdersAndSave(t, ctrl, ctx, ordersAmount)
	defer testRepo.Close()

	// Извлекать будем произвольное количество последних заказов
	latestAmount := 50
	// В тесте заполнять реальный кэш не нужно, поэтому мокируем обновление
	mockCache.EXPECT().UpdateCache(ctx, gomock.Any()).Return(nil).Times(latestAmount)

	t.Logf("Preparing to retrieve %d latest orders...", latestAmount)
	// Получим список извлеченных заказов
	latstOrders, err := testRepo.GetLatestOrders(ctx, int32(latestAmount))
	require.NoError(t, err, "Failed to retrieve latest orders")
	// И сравним что извлекли именно столько, сколько хотели
	t.Logf("Successfully retrieved latest orders. Expected: %d/%d, got: %d", latestAmount, ordersAmount, len(latstOrders))
}
