package handler

import (
	"encoding/json"
	"log"
	"net/http"
	"strings"

	"github.com/gorilla/mux"
)

// Handler обрабатывает HTTP запросы
type Handler struct {
	svc *service.Service // Сервис для обработки бизнес-логики
}

func NewHandler(svc *service.Service) *Handler {
	return &Handler{svc: svc}
}

func (h *Handler) GetOrderByID(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	orderUID := vars["id"]
	if orderUID == "" {
		http.Error(w, "Order UID is required", http.StatusBadRequest)
		return
	}
	order, err := h.svc.GetOrderByID(orderUID)
	if err != nil {
		log.Println("Error getting order:", err)
		if strings.Contains(err.Error(), "not found") {
			http.Error(w, "Order not found", http.StatusNotFound)
		} else {
			http.Error(w, "Internal server error", http.StatusInternalServerError)
		}
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("X-Content-Type-Options", "nosniff")
	if err := json.NewEncoder(w).Encode(order); err != nil {
		log.Println("Error encoding order:", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
	}
}

func (h *Handler) ServeStatic(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	http.ServeFile(w, r, "web/index.html")
}
