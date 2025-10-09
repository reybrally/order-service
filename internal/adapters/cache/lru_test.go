package cache_test

import (
	"github.com/reybrally/order-service/internal/adapters/cache"
	"github.com/reybrally/order-service/internal/domain/order"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"testing"
)

// Mocked Order structure for testing
func mockOrder(orderUID string, amount int64) order.Order {
	return order.Order{
		OrderUID: orderUID,
		Payment: order.Payment{
			Amount: amount,
		},
	}
}

// TestLRUCacheMultipleEvictions проверяет несколько удалений элементов из кэша
func TestLRUCacheMultipleEvictions(t *testing.T) {
	c := cache.NewLRUCache(2)

	// Добавляем несколько заказов
	err := c.Set("a", mockOrder("a", 100))
	require.NoError(t, err)
	err = c.Set("b", mockOrder("b", 200))
	require.NoError(t, err)

	// Добавляем третий заказ - должен удалить "a"
	err = c.Set("c", mockOrder("c", 300))
	require.NoError(t, err)

	// Убедимся, что "a" удален
	_, err = c.Get("a")
	assert.Error(t, err)

	// Добавляем новый заказ - должен удалить "b"
	err = c.Set("d", mockOrder("d", 400))
	require.NoError(t, err)

	// Убедимся, что "b" удален
	_, err = c.Get("b")
	assert.Error(t, err)

	// Остались "c" и "d"
	val, err := c.Get("c")
	require.NoError(t, err)
	assert.Equal(t, mockOrder("c", 300), val)

	val, err = c.Get("d")
	require.NoError(t, err)
	assert.Equal(t, mockOrder("d", 400), val)
}

// TestLRUCacheCapacity проверяет поведение кэша при достижении его емкости
func TestLRUCacheCapacity(t *testing.T) {
	c := cache.NewLRUCache(3)

	// Добавляем три элемента в кэш
	err := c.Set("a", mockOrder("a", 100))
	require.NoError(t, err)
	err = c.Set("b", mockOrder("b", 200))
	require.NoError(t, err)
	err = c.Set("c", mockOrder("c", 300))
	require.NoError(t, err)

	// Проверяем, что все три элемента находятся в кэше
	val, err := c.Get("a")
	require.NoError(t, err)
	assert.Equal(t, mockOrder("a", 100), val)

	val, err = c.Get("b")
	require.NoError(t, err)
	assert.Equal(t, mockOrder("b", 200), val)

	val, err = c.Get("c")
	require.NoError(t, err)
	assert.Equal(t, mockOrder("c", 300), val)

	// Добавляем четвертый элемент, "a" должен быть удален, так как он наименее использовался
	err = c.Set("d", mockOrder("d", 400))
	require.NoError(t, err)

	_, err = c.Get("a")
	assert.Error(t, err) // "a" должен быть удален
}

// TestLRUCacheClearAll проверяет работу метода Clear
func TestLRUCacheClearAll(t *testing.T) {
	c := cache.NewLRUCache(3)

	// Добавляем элементы в кэш
	err := c.Set("a", mockOrder("a", 100))
	require.NoError(t, err)
	err = c.Set("b", mockOrder("b", 200))
	require.NoError(t, err)

	// Очищаем кэш
	c.Clear()

	// Проверяем, что кэш пуст
	_, err = c.Get("a")
	assert.Error(t, err)

	_, err = c.Get("b")
	assert.Error(t, err)
}

// TestLRUCacheUpdate проверяет обновление значения в кэше
func TestLRUCacheUpdate(t *testing.T) {
	c := cache.NewLRUCache(3)

	// Добавляем элемент в кэш
	err := c.Set("a", mockOrder("a", 100))
	require.NoError(t, err)

	// Обновляем значение для ключа "a"
	err = c.Set("a", mockOrder("a", 500))
	require.NoError(t, err)

	// Проверяем, что значение обновилось
	val, err := c.Get("a")
	require.NoError(t, err)
	assert.Equal(t, mockOrder("a", 500), val)
}

// TestLRUCacheEvictionWithOrder проверяет правильность удаления элементов при обновлении
func TestLRUCacheEvictionWithOrder(t *testing.T) {
	c := cache.NewLRUCache(3)

	// Добавляем элементы в кэш
	err := c.Set("a", mockOrder("a", 100))
	require.NoError(t, err)
	err = c.Set("b", mockOrder("b", 200))
	require.NoError(t, err)
	err = c.Set("c", mockOrder("c", 300))
	require.NoError(t, err)

	// Получаем "a", чтобы оно стало самым недавно использованным
	_, err = c.Get("a")
	require.NoError(t, err)

	// Добавляем "d", "b" должно быть удалено, так как оно наименее использовалось
	err = c.Set("d", mockOrder("d", 400))
	require.NoError(t, err)

	_, err = c.Get("b")
	assert.Error(t, err) // "b" должно быть удалено

	// Проверяем оставшиеся элементы
	val, err := c.Get("a")
	require.NoError(t, err)
	assert.Equal(t, mockOrder("a", 100), val)

	val, err = c.Get("c")
	require.NoError(t, err)
	assert.Equal(t, mockOrder("c", 300), val)

	val, err = c.Get("d")
	require.NoError(t, err)
	assert.Equal(t, mockOrder("d", 400), val)
}

// TestLRUCacheMultipleEvictionsForMultipleSets проверяет работу кэша при множественном обновлении значений
func TestLRUCacheMultipleEvictionsForMultipleSets(t *testing.T) {
	c := cache.NewLRUCache(2)

	// Добавляем элементы в кэш
	err := c.Set("a", mockOrder("a", 100))
	require.NoError(t, err)
	err = c.Set("b", mockOrder("b", 200))
	require.NoError(t, err)

	// Обновляем элементы, "a" должен стать самым недавно использованным
	err = c.Set("a", mockOrder("a", 300))
	require.NoError(t, err)

	// Добавляем новый элемент, который удалит "b"
	err = c.Set("c", mockOrder("c", 400))
	require.NoError(t, err)

	// Проверяем, что "b" был удален
	_, err = c.Get("b")
	assert.Error(t, err)

	// Проверяем оставшиеся элементы
	val, err := c.Get("a")
	require.NoError(t, err)
	assert.Equal(t, mockOrder("a", 300), val)

	val, err = c.Get("c")
	require.NoError(t, err)
	assert.Equal(t, mockOrder("c", 400), val)
}

// TestLRUCacheSize проверяет правильность размера кэша
func TestLRUCacheSize(t *testing.T) {
	c := cache.NewLRUCache(2)

	// Добавляем несколько элементов
	err := c.Set("a", mockOrder("a", 100))
	require.NoError(t, err)
	err = c.Set("b", mockOrder("b", 200))
	require.NoError(t, err)

	// Проверяем, что кэш не переполнен
	val, err := c.Get("a")
	require.NoError(t, err)
	assert.Equal(t, mockOrder("a", 100), val)

	// Добавляем третий элемент, который должен удалить "a"
	err = c.Set("c", mockOrder("c", 300))
	require.NoError(t, err)

	_, err = c.Get("a")
	assert.Error(t, err) // "a" должно быть удалено

	// Проверяем размер кэша
	assert.Equal(t, 2, len(c.cache)) // Ожидаем, что размер кэша равен 2
}

// TestLRUCacheEvictionWhenReachingMaxCapacity проверяет удаление элемента при превышении емкости
func TestLRUCacheEvictionWhenReachingMaxCapacity(t *testing.T) {
	c := cache.NewLRUCache(2)

	// Добавляем элементы
	err := c.Set("a", mockOrder("a", 100))
	require.NoError(t, err)
	err = c.Set("b", mockOrder("b", 200))
	require.NoError(t, err)

	// Добавляем новый элемент
	err = c.Set("c", mockOrder("c", 300))
	require.NoError(t, err)

	// Проверяем, что "a" удален
	_, err = c.Get("a")
	assert.Error(t, err)

	// Проверяем оставшиеся элементы
	val, err := c.Get("b")
	require.NoError(t, err)
	assert.Equal(t, mockOrder("b", 200), val)

	val, err = c.Get("c")
	require.NoError(t, err)
	assert.Equal(t, mockOrder("c", 300), val)
}
