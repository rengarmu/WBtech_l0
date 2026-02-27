package usecase

import (
	"context"
	"errors"
	"testing"
	"time"

	"WBtech_l0/internal/domain"
)

// MockRepository — мок доменного репозитория.
type MockRepository struct {
	SaveOrderFunc     func(ctx context.Context, order domain.Order) error
	GetOrderFunc      func(ctx context.Context, orderUID string) (domain.Order, error)
	LoadAllOrdersFunc func(ctx context.Context) ([]domain.Order, error)
	ClearAllFunc      func(ctx context.Context) error
}

func (m *MockRepository) SaveOrder(ctx context.Context, order domain.Order) error {
	return m.SaveOrderFunc(ctx, order)
}
func (m *MockRepository) GetOrder(ctx context.Context, orderUID string) (domain.Order, error) {
	return m.GetOrderFunc(ctx, orderUID)
}
func (m *MockRepository) LoadAllOrders(ctx context.Context) ([]domain.Order, error) {
	return m.LoadAllOrdersFunc(ctx)
}
func (m *MockRepository) ClearAll(ctx context.Context) error {
	return m.ClearAllFunc(ctx)
}

// MockCache — мок доменного кэша.
type MockCache struct {
	GetFunc        func(orderUID string) (domain.Order, bool)
	SetFunc        func(order domain.Order)
	SetWithTTLFunc func(order domain.Order, ttl time.Duration)
	DeleteFunc     func(orderUID string)
	ClearFunc      func()
	GetAllFunc     func() map[string]domain.Order
	GetStatsFunc   func() domain.CacheStats
}

func (m *MockCache) Get(orderUID string) (domain.Order, bool) {
	return m.GetFunc(orderUID)
}
func (m *MockCache) Set(order domain.Order) {
	m.SetFunc(order)
}
func (m *MockCache) SetWithTTL(order domain.Order, ttl time.Duration) {
	m.SetWithTTLFunc(order, ttl)
}
func (m *MockCache) Delete(orderUID string) {
	m.DeleteFunc(orderUID)
}
func (m *MockCache) Clear() {
	m.ClearFunc()
}
func (m *MockCache) GetAll() map[string]domain.Order {
	return m.GetAllFunc()
}
func (m *MockCache) GetStats() domain.CacheStats {
	return m.GetStatsFunc()
}

func TestOrderUsecase_GetOrder_CacheHit(t *testing.T) {
	// given
	expectedOrder := domain.Order{OrderUID: "test-uid"}
	cache := &MockCache{
		GetFunc: func(uid string) (domain.Order, bool) {
			if uid == "test-uid" {
				return expectedOrder, true
			}
			return domain.Order{}, false
		},
	}
	repo := &MockRepository{
		GetOrderFunc: func(_ context.Context, _ string) (domain.Order, error) {
			t.Error("repo.GetOrder should not be called on cache hit")
			return domain.Order{}, nil
		},
	}
	usecase := NewOrderUsecase(repo, cache)

	// when
	order, err := usecase.GetOrder(context.Background(), "test-uid")

	// then
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if order.OrderUID != "test-uid" {
		t.Errorf("expected order UID test-uid, got %s", order.OrderUID)
	}
}

func TestOrderUsecase_GetOrder_CacheMiss_RepoSuccess(t *testing.T) {
	// given
	expectedOrder := domain.Order{OrderUID: "test-uid"}
	cacheCalled := false
	cache := &MockCache{
		GetFunc: func(_ string) (domain.Order, bool) {
			return domain.Order{}, false
		},
		SetFunc: func(order domain.Order) {
			if order.OrderUID != "test-uid" {
				t.Errorf("Set called with wrong UID: %s", order.OrderUID)
			}
			cacheCalled = true
		},
	}
	repo := &MockRepository{
		GetOrderFunc: func(_ context.Context, uid string) (domain.Order, error) {
			if uid == "test-uid" {
				return expectedOrder, nil
			}
			return domain.Order{}, errors.New("not found")
		},
	}
	usecase := NewOrderUsecase(repo, cache)

	// when
	order, err := usecase.GetOrder(context.Background(), "test-uid")

	// then
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if order.OrderUID != "test-uid" {
		t.Errorf("expected order UID test-uid, got %s", order.OrderUID)
	}
	if !cacheCalled {
		t.Error("expected cache.Set to be called")
	}
}

func TestOrderUsecase_GetOrder_RepoError(t *testing.T) {
	// given
	cache := &MockCache{
		GetFunc: func(_ string) (domain.Order, bool) {
			return domain.Order{}, false
		},
	}
	repoErr := errors.New("db error")
	repo := &MockRepository{
		GetOrderFunc: func(_ context.Context, _ string) (domain.Order, error) {
			return domain.Order{}, repoErr
		},
	}
	usecase := NewOrderUsecase(repo, cache)

	// when
	_, err := usecase.GetOrder(context.Background(), "test-uid")

	// then
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !errors.Is(err, repoErr) && err.Error() != "repo.GetOrder: db error" {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestOrderUsecase_SaveOrder(t *testing.T) {
	// given
	order := domain.Order{OrderUID: "test-uid"}
	repoCalled := false
	cacheCalled := false
	repo := &MockRepository{
		SaveOrderFunc: func(_ context.Context, o domain.Order) error {
			if o.OrderUID != "test-uid" {
				t.Errorf("SaveOrder called with wrong UID: %s", o.OrderUID)
			}
			repoCalled = true
			return nil
		},
	}
	cache := &MockCache{
		SetFunc: func(o domain.Order) {
			if o.OrderUID != "test-uid" {
				t.Errorf("Set called with wrong UID: %s", o.OrderUID)
			}
			cacheCalled = true
		},
	}
	usecase := NewOrderUsecase(repo, cache)

	// when
	err := usecase.SaveOrder(context.Background(), order)

	// then
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !repoCalled {
		t.Error("expected repo.SaveOrder to be called")
	}
	if !cacheCalled {
		t.Error("expected cache.Set to be called")
	}
}

func TestOrderUsecase_SaveOrder_RepoError(t *testing.T) {
	// given
	repoErr := errors.New("save failed")
	repo := &MockRepository{
		SaveOrderFunc: func(_ context.Context, _ domain.Order) error {
			return repoErr
		},
	}
	cache := &MockCache{
		SetFunc: func(_ domain.Order) {
			t.Error("cache.Set should not be called on repo error")
		},
	}
	usecase := NewOrderUsecase(repo, cache)

	// when
	err := usecase.SaveOrder(context.Background(), domain.Order{OrderUID: "test"})

	// then
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !errors.Is(err, repoErr) {
		t.Errorf("expected repoErr, got %v", err)
	}
}
