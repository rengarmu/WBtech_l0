package cache

import (
	"testing"
	"time"

	"WBtech_l0/internal/domain"
)

func TestOrderCache_SetAndGet(t *testing.T) {
	cache := NewOrderCache(10*time.Minute, 10)
	order := domain.Order{OrderUID: "123"}

	cache.Set(order)

	got, found := cache.Get("123")
	if !found {
		t.Error("expected order to be found")
	}
	if got.OrderUID != "123" {
		t.Errorf("expected UID 123, got %s", got.OrderUID)
	}
}

func TestOrderCache_Get_Expired(t *testing.T) {
	cache := NewOrderCache(1*time.Millisecond, 10)
	order := domain.Order{OrderUID: "123"}
	cache.Set(order)

	time.Sleep(2 * time.Millisecond)

	_, found := cache.Get("123")
	if found {
		t.Error("expected order to be expired and not found")
	}
}

func TestOrderCache_Delete(t *testing.T) {
	cache := NewOrderCache(10*time.Minute, 10)
	order := domain.Order{OrderUID: "123"}
	cache.Set(order)

	cache.Delete("123")

	_, found := cache.Get("123")
	if found {
		t.Error("expected order to be deleted")
	}
}

func TestOrderCache_MaxSizeEviction(t *testing.T) {
	cache := NewOrderCache(10*time.Minute, 2)
	order1 := domain.Order{OrderUID: "1"}
	order2 := domain.Order{OrderUID: "2"}
	order3 := domain.Order{OrderUID: "3"}

	cache.Set(order1)
	time.Sleep(1 * time.Millisecond) // ensure different timestamps
	cache.Set(order2)
	cache.Set(order3) // should evict oldest (order1)

	// order1 should be gone
	if _, found := cache.Get("1"); found {
		t.Error("expected order1 to be evicted")
	}
	// order2 and order3 should be present
	if _, found := cache.Get("2"); !found {
		t.Error("expected order2 to be present")
	}
	if _, found := cache.Get("3"); !found {
		t.Error("expected order3 to be present")
	}
}

func TestOrderCache_Stats(t *testing.T) {
	cache := NewOrderCache(10*time.Minute, 10)
	order := domain.Order{OrderUID: "stats-test"}

	// miss
	cache.Get("nonexistent")
	stats := cache.GetStats()
	if stats.Misses != 1 {
		t.Errorf("expected misses=1, got %d", stats.Misses)
	}

	// hit
	cache.Set(order)
	cache.Get(order.OrderUID)
	stats = cache.GetStats()
	if stats.Hits != 1 {
		t.Errorf("expected hits=1, got %d", stats.Hits)
	}
	if stats.Size != 1 {
		t.Errorf("expected size=1, got %d", stats.Size)
	}
}

func TestOrderCache_Clear(t *testing.T) {
	cache := NewOrderCache(10*time.Minute, 10)
	cache.Set(domain.Order{OrderUID: "1"})
	cache.Set(domain.Order{OrderUID: "2"})

	cache.Clear()

	if len(cache.GetAll()) != 0 {
		t.Error("expected cache to be empty after Clear")
	}
	stats := cache.GetStats()
	if stats.Size != 0 || stats.Hits != 0 || stats.Misses != 0 {
		t.Errorf("stats not reset: %+v", stats)
	}
}
