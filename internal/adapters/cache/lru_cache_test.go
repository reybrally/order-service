package cache

import (
	"github.com/reybrally/order-service/internal/domain/order"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"testing"
)

// мок структура для тестов
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
	c := NewCacheService(2)

	err := c.Set("a", mockOrder("a", 100))
	require.NoError(t, err)
	err = c.Set("b", mockOrder("b", 200))
	require.NoError(t, err)

	err = c.Set("c", mockOrder("c", 300))
	require.NoError(t, err)

	_, err = c.Get("a")
	assert.Error(t, err)

	err = c.Set("d", mockOrder("d", 400))
	require.NoError(t, err)

	_, err = c.Get("b")
	assert.Error(t, err)

	val, err := c.Get("c")
	require.NoError(t, err)
	assert.Equal(t, mockOrder("c", 300), val)

	val, err = c.Get("d")
	require.NoError(t, err)
	assert.Equal(t, mockOrder("d", 400), val)
}

// TestLRUCacheCapacity проверяет поведение кэша при достижении его емкости
func TestLRUCacheCapacity(t *testing.T) {
	c := NewCacheService(3)

	err := c.Set("a", mockOrder("a", 100))
	require.NoError(t, err)
	err = c.Set("b", mockOrder("b", 200))
	require.NoError(t, err)
	err = c.Set("c", mockOrder("c", 300))
	require.NoError(t, err)

	val, err := c.Get("a")
	require.NoError(t, err)
	assert.Equal(t, mockOrder("a", 100), val)

	val, err = c.Get("b")
	require.NoError(t, err)
	assert.Equal(t, mockOrder("b", 200), val)

	val, err = c.Get("c")
	require.NoError(t, err)
	assert.Equal(t, mockOrder("c", 300), val)

	err = c.Set("d", mockOrder("d", 400))
	require.NoError(t, err)

	_, err = c.Get("a")
	assert.Error(t, err)
}

// TestLRUCacheClearAll проверяет работу метода Clear
func TestLRUCacheClearAll(t *testing.T) {
	c := NewCacheService(3)

	err := c.Set("a", mockOrder("a", 100))
	require.NoError(t, err)
	err = c.Set("b", mockOrder("b", 200))
	require.NoError(t, err)

	c.Clear()

	_, err = c.Get("a")
	assert.Error(t, err)

	_, err = c.Get("b")
	assert.Error(t, err)
}

// TestLRUCacheUpdate проверяет обновление значения в кэше
func TestLRUCacheUpdate(t *testing.T) {
	c := NewCacheService(3)

	err := c.Set("a", mockOrder("a", 100))
	require.NoError(t, err)

	err = c.Set("a", mockOrder("a", 500))
	require.NoError(t, err)

	val, err := c.Get("a")
	require.NoError(t, err)
	assert.Equal(t, mockOrder("a", 500), val)
}

// TestLRUCacheEvictionWithOrder проверяет правильность удаления элементов при обновлении
func TestLRUCacheEvictionWithOrder(t *testing.T) {
	c := NewCacheService(3)

	err := c.Set("a", mockOrder("a", 100))
	require.NoError(t, err)
	err = c.Set("b", mockOrder("b", 200))
	require.NoError(t, err)
	err = c.Set("c", mockOrder("c", 300))
	require.NoError(t, err)

	_, err = c.Get("a")
	require.NoError(t, err)

	err = c.Set("d", mockOrder("d", 400))
	require.NoError(t, err)

	_, err = c.Get("b")
	assert.Error(t, err)

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
	c := NewCacheService(2)

	err := c.Set("a", mockOrder("a", 100))
	require.NoError(t, err)
	err = c.Set("b", mockOrder("b", 200))
	require.NoError(t, err)

	err = c.Set("a", mockOrder("a", 300))
	require.NoError(t, err)

	err = c.Set("c", mockOrder("c", 400))
	require.NoError(t, err)

	_, err = c.Get("b")
	assert.Error(t, err)

	val, err := c.Get("a")
	require.NoError(t, err)
	assert.Equal(t, mockOrder("a", 300), val)

	val, err = c.Get("c")
	require.NoError(t, err)
	assert.Equal(t, mockOrder("c", 400), val)
}

// TestLRUCacheSize проверяет правильность размера кэша
func TestLRUCacheSize(t *testing.T) {
	c := NewCacheService(2)

	err := c.Set("a", mockOrder("a", 100))
	require.NoError(t, err)
	err = c.Set("b", mockOrder("b", 200))
	require.NoError(t, err)

	val, err := c.Get("a")
	require.NoError(t, err)
	assert.Equal(t, mockOrder("a", 100), val)

	err = c.Set("c", mockOrder("c", 300))
	require.NoError(t, err)

	_, err = c.Get("b")
	assert.Error(t, err)

	assert.Equal(t, 2, len(c.cache))
}

// TestLRUCacheEvictionWhenReachingMaxCapacity проверяет удаление элемента при превышении емкости
func TestLRUCacheEvictionWhenReachingMaxCapacity(t *testing.T) {
	c := NewCacheService(2)

	err := c.Set("a", mockOrder("a", 100))
	require.NoError(t, err)
	err = c.Set("b", mockOrder("b", 200))
	require.NoError(t, err)

	err = c.Set("c", mockOrder("c", 300))
	require.NoError(t, err)

	_, err = c.Get("a")
	assert.Error(t, err)

	val, err := c.Get("b")
	require.NoError(t, err)
	assert.Equal(t, mockOrder("b", 200), val)

	val, err = c.Get("c")
	require.NoError(t, err)
	assert.Equal(t, mockOrder("c", 300), val)
}
