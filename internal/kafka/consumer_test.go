package kafka

import (
	"testing"

	"orders/internal/generator"

	"github.com/stretchr/testify/require"
)

// Тестирует корректность обработки невалидных данных в заказах функцией validateOrders
func TestValidateOrders(t *testing.T) {
	// Тестируем при корректности всех заказов
	t.Run("All orders are valid", func(t *testing.T) {
		// Генерируем 10 заказов
		validOrdersAmount := 10
		t.Logf("Preparing to generate %d valid orders...", validOrdersAmount)
		orders := generator.MakeRandomOrder(validOrdersAmount)

		// Проверяем итоговый список заказов для сохранения в бд и последующего коммита
		validOrders := validateOrders(orders)
		require.Len(t, validOrders, len(orders), "Should return all orders if they are valid")
		t.Logf("All %d orders were validated. Returned %d/%d as valid", validOrdersAmount, len(validOrders), validOrdersAmount)
	})

	// Тестируем с наличием некорректных заказов
	t.Run("Having invalid orders", func(t *testing.T) {
		t.Log("Generating order with invalid OrderUID...")
		invalidOrderUID := generator.MakeRandomOrder(1)[0]
		// Заказ с некорректным OrderUID
		invalidOrderUID.OrderUID = ""

		t.Log("Generating order with invalid TrackNumber...")
		invalidOrderTrackNumber := generator.MakeRandomOrder(1)[0]
		// Заказ с некорректным TrackNumber
		invalidOrderTrackNumber.TrackNumber = ""

		t.Log("Generating order with invalid CustomerID...")
		invalidOrderCustomerID := generator.MakeRandomOrder(1)[0]
		// Заказ с некорректным CustomerID
		invalidOrderCustomerID.CustomerID = ""

		t.Log("Generating order with invalid Phone number...")
		invalidOrderPhoneNumber := generator.MakeRandomOrder(1)[0]
		// Заказ с некорректным номером телефона
		invalidOrderPhoneNumber.Delivery.Phone = "012345678"

		t.Log("Generating valid order...")
		// Какой-нибудь валидный заказ
		someValidOrder := generator.MakeRandomOrder(1)[0]

		// Объединим все заказы в одно сообщение
		ordersBatch := []*generator.Order{
			invalidOrderUID,
			invalidOrderTrackNumber,
			invalidOrderCustomerID,
			invalidOrderPhoneNumber,
			someValidOrder,
		}
		inputLen := len(ordersBatch) // Количество заказов на вход
		t.Log("Total orders in one message:", inputLen)

		expectedLen := inputLen - 4 // Ожидаемое количество заказов на выходе
		t.Log("Expected valid orders in one message:", expectedLen)

		// Сравниваем ожидание с реальностью
		validOrders := validateOrders(ordersBatch)
		require.Equal(t, expectedLen, len(validOrders), "Valid orders amount should match expected value")
		t.Logf("All %d orders were validated. Returned %d/%d as valid", inputLen, len(validOrders), expectedLen)
	})
}
