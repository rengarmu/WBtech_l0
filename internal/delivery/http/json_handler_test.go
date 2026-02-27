package httpdelivery

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"WBtech_l0/internal/domain"
)

func TestMakeJSONOrderHandler_Success(t *testing.T) {
	expectedOrder := domain.Order{OrderUID: "json123"}
	usecase := &MockUsecase{
		GetOrderFunc: func(_ context.Context, uid string) (domain.Order, error) {
			if uid == "json123" {
				return expectedOrder, nil
			}
			return domain.Order{}, errors.New("not found")
		},
	}
	handler := MakeJSONOrderHandler(usecase)

	req := httptest.NewRequest("GET", "/api/order/json123", nil)
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status OK, got %d", w.Code)
	}
	var resp JSONResponse
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if !resp.Success {
		t.Error("expected success true")
	}
	data, ok := resp.Data.(map[string]interface{})
	if !ok {
		t.Fatalf("data is not map: %v", resp.Data)
	}
	if data["order_uid"] != "json123" {
		t.Errorf("expected order_uid json123, got %v", data["order_uid"])
	}
}

func TestMakeJSONOrderHandler_NotFound(t *testing.T) {
	usecase := &MockUsecase{
		GetOrderFunc: func(_ context.Context, _ string) (domain.Order, error) {
			return domain.Order{}, errors.New("not found")
		},
	}
	handler := MakeJSONOrderHandler(usecase)

	req := httptest.NewRequest("GET", "/api/order/unknown", nil)
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("expected status NotFound, got %d", w.Code)
	}
	var resp JSONResponse
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if resp.Success {
		t.Error("expected success false")
	}
	if resp.Error == "" {
		t.Error("expected error message")
	}
}

func TestMakeJSONOrderHandler_InvalidUID(t *testing.T) {
	usecase := &MockUsecase{}
	handler := MakeJSONOrderHandler(usecase)

	// слишком длинный UID (более 255 символов) – но в isValidOrderUID есть проверка длины, проще проверить пустой сегмент
	req := httptest.NewRequest("GET", "/api/order/", nil) // missing UID
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected BadRequest, got %d", w.Code)
	}
}
