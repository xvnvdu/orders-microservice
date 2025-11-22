package cache

import (
	"context"
	"orders/internal/generator"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Тестирует добавление и извлечение заказов в/из кэша
func TestCachePutAndGet(t *testing.T) {
	// Используется отдельная бд под номером 1, в проде используется нулевая
	testCache, err := NewCache("redis://localhost:6379/1")
	require.NoError(t, err, "NewCache function should not return error if successful")

	// Очищаем кэш
	err = testCache.redisClient.FlushDB(context.Background()).Err()
	require.NoError(t, err, "Failed to flush Redis")

	// Генерируем случайный заказ
	randOrder := generator.MakeRandomOrder(1)[0]
	ctx := context.Background()

	// Пытаемся добавить заказ в кэш
	err = testCache.UpdateCache(ctx, randOrder)
	require.NoError(t, err, "UpdateCache should not return error if successful")
	t.Log("Cache updated successfully")

	// При успешном добавлении пытаемся извлечь добавленный заказ
	retrievedOrder, err := testCache.GetFromCache(ctx, randOrder.OrderUID)
	require.NoError(t, err, "GetFromCache should not return error if successful")
	t.Log("Order retrieved successfully")

	// Проверим, что заказ был извлечен успешно
	require.NotNil(t, retrievedOrder, "Retrieved order should not be nil")
	t.Log("Retrieved order is not nil")

	// Сравним некоторые параметры исходного и извлеченного заказов: они должны совпадать
	assert.Equal(t, randOrder.OrderUID, retrievedOrder.OrderUID)
	t.Logf("Expected: %s | Got: %s\n", randOrder.OrderUID, retrievedOrder.OrderUID)

	assert.Equal(t, randOrder.CustomerID, retrievedOrder.CustomerID)
	t.Logf("Expected: %s | Got: %s\n", randOrder.CustomerID, retrievedOrder.CustomerID)

	assert.Equal(t, randOrder.TrackNumber, retrievedOrder.TrackNumber)
	t.Logf("Expected: %s | Got: %s\n", randOrder.TrackNumber, retrievedOrder.TrackNumber)
}

// Тестирует вытеснение заазов из кэша при превышении лимита
func TestCacheLRU(t *testing.T) {
	// Используется отдельная бд под номером 2 в проде используется нулевая
	testCache, err := NewCache("redis://localhost:6379/2")
	require.NoError(t, err, "NewCache function should not return error if successful")

	// Очищаем кэш
	err = testCache.redisClient.FlushDB(context.Background()).Err()
	require.NoError(t, err, "Failed to flush Redis")

	// Создаем первый заказ, который впоследствии должен быть вытеснен из кэша
	firstOrder := generator.MakeRandomOrder(1)[0]
	ctx := context.Background()

	// Добавим первый заказ в кэш
	err = testCache.UpdateCache(ctx, firstOrder)
	require.NoError(t, err, "UpdateCache should not return error if successful")
	t.Logf("Cache filled with first order: %s\n", firstOrder.OrderUID)

	t.Logf("Filling cache with %d more orders...", CacheCapacity)
	// Добавляем сверху еще Х заказов, где Х - максимальная вместимость кэша
	fullCapOrders := generator.MakeRandomOrder(int(CacheCapacity))
	count := 0
	for i := range CacheCapacity {
		err = testCache.UpdateCache(ctx, fullCapOrders[i])
		require.NoError(t, err, "UpdateCache should not return error if successful")
		count++
	}
	t.Logf("Cache filled with %d/%d orders\n", count, CacheCapacity)

	// Проверим текущую заполненость кэша
	currentCap, err := testCache.redisClient.ZCard(ctx, "LRU-orders").Result()
	require.NoError(t, err, "Failed to get current cache capacity")
	t.Logf("Current cache capacity: %d\n", currentCap)
	t.Logf("Total orders count: %d\n", currentCap+1)

	// Первый заказ должен быть вытеснен - попытка его извлечь
	// должна вернуть nil значение заказа и ненулевую ошибку
	getFirstOrder, err := testCache.GetFromCache(ctx, firstOrder.OrderUID)
	if assert.Error(t, err, "Attempting to get first order should return error") {
		t.Log("First order is no longer cached: was displaced by newcomers")
	}
	assert.Nil(t, getFirstOrder, "First order should be nil, as it was displaced")
}

// Тестирует заполнение кэша заказами на старте сервиса
func TestLoadInitialOrders(t *testing.T) {
	// Используется отдельная бд под номером 3 в проде используется нулевая
	testCache, err := NewCache("redis://localhost:6379/3")
	require.NoError(t, err, "NewCache function should not return error if successful")

	// Очищаем кэш
	err = testCache.redisClient.FlushDB(context.Background()).Err()
	require.NoError(t, err, "Failed to flush Redis")

	ctx := context.Background()
	// Задаем произвольное количество заказов для заполнения на старте
	ordersToCache := 67
	// Генерируем эти заказы
	latestOrders := generator.MakeRandomOrder(ordersToCache)

	// На данном этапе кэш должен быть пустым
	currentCap, err := testCache.redisClient.DBSize(ctx).Result()
	require.NoError(t, err, "DBSize should not return error if successful")
	t.Log("Cache capacity before loading initial orders:", currentCap)

	t.Log("Filling cache with orders...")
	// После заполнения в кэше должно быть ровно столько заказов,
	// сколько их было на момент старта сервиса (если не превысили лимит)
	testCache.LoadInitialOrders(ctx, latestOrders, CacheCapacity)
	currentCap, err = testCache.redisClient.ZCard(ctx, "LRU-orders").Result()
	require.NoError(t, err, "DBSize should not return error if successful")
	t.Logf("Expected %d/%d orders, got %d\n", ordersToCache, CacheCapacity, currentCap)
}
