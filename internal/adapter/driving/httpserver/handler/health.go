package handler

import (
	"encoding/json"
	"net/http"
)

type HealthHandler struct{}

func NewHealthHandler() *HealthHandler {
	return &HealthHandler{}
}

type healthResponse struct {
	Status string `json:"status"`
}

func (h *HealthHandler) Handle(w http.ResponseWriter, r *http.Request) {
	response := healthResponse{
		Status: "ok",
	}

	data, err := json.Marshal(response)
	if err != nil {
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	w.Write(data)
}
