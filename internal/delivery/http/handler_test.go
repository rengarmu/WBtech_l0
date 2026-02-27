package httpdelivery

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	"WBtech_l0/internal/domain"
)

// setupTestTemplate создаёт временную копию шаблона и переходит в эту директорию.
// Возвращает функцию для восстановления исходной директории.
func setupTestTemplate(t *testing.T) func() {
	// Содержимое минимального шаблона для тестов
	templateContent := `<!doctype html>
<html>
<head><title>Test</title></head>
<body>
    <h1>Order {{.Order.OrderUID}}</h1>
    {{if .Found}}Found{{else}}Not Found{{end}}
</body>
</html>`

	// Создаём временную директорию
	tmpDir := t.TempDir()

	// Создаём поддиректорию web
	webDir := filepath.Join(tmpDir, "web")
	err := os.Mkdir(webDir, 0755)
	require.NoError(t, err)

	// Записываем файл шаблона
	tmplPath := filepath.Join(webDir, "order_template.html")
	err = os.WriteFile(tmplPath, []byte(templateContent), 0644)
	require.NoError(t, err)

	// Сохраняем текущую рабочую директорию
	origDir, err := os.Getwd()
	require.NoError(t, err)

	// Переходим во временную директорию
	err = os.Chdir(tmpDir)
	require.NoError(t, err)

	// Возвращаем функцию для восстановления
	return func() {
		if err := os.Chdir(origDir); err != nil {
			t.Logf("failed to restore working directory: %v", err)
		}
	}
}

// MockUsecase реализует domain.OrderUsecase для тестов.
type MockUsecase struct {
	GetOrderFunc  func(ctx context.Context, orderUID string) (domain.Order, error)
	SaveOrderFunc func(ctx context.Context, order domain.Order) error
}

func (m *MockUsecase) GetOrder(ctx context.Context, orderUID string) (domain.Order, error) {
	return m.GetOrderFunc(ctx, orderUID)
}
func (m *MockUsecase) SaveOrder(ctx context.Context, order domain.Order) error {
	return m.SaveOrderFunc(ctx, order)
}

func TestMakeOrderHandler_Success(t *testing.T) {
	defer setupTestTemplate(t)()
	// given
	expectedOrder := domain.Order{OrderUID: "12345"}
	usecase := &MockUsecase{
		GetOrderFunc: func(_ context.Context, uid string) (domain.Order, error) {
			if uid == "12345" {
				return expectedOrder, nil
			}
			return domain.Order{}, errors.New("not found")
		},
	}
	handler := MakeOrderHandler(usecase)

	req := httptest.NewRequest("GET", "/order/12345", nil)
	w := httptest.NewRecorder()

	// when
	handler.ServeHTTP(w, req)

	// then
	if w.Code != http.StatusOK {
		t.Errorf("expected status OK, got %d", w.Code)
	}
	body := w.Body.String()
	if !strings.Contains(body, "12345") {
		t.Error("response body does not contain order UID")
	}
}

func TestMakeOrderHandler_NotFound(t *testing.T) {
	defer setupTestTemplate(t)()
	usecase := &MockUsecase{
		GetOrderFunc: func(_ context.Context, _ string) (domain.Order, error) {
			return domain.Order{}, errors.New("not found")
		},
	}
	handler := MakeOrderHandler(usecase)

	req := httptest.NewRequest("GET", "/order/unknown", nil)
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("expected status NotFound, got %d", w.Code)
	}
}

func TestMakeOrderHandler_InvalidUID(t *testing.T) {
	defer setupTestTemplate(t)()
	usecase := &MockUsecase{}
	handler := MakeOrderHandler(usecase)

	req := httptest.NewRequest("GET", "/order/", nil)
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest { // было http.StatusNotFound
		t.Errorf("expected BadRequest, got %d", w.Code)
	}
}
