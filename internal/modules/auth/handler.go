package auth

import (
	"encoding/json"
	"net/http"

	"carwashinmaribackend/internal/utils"
)

type Handler struct {
	service Service
}

func NewHandler(service Service) *Handler {
	return &Handler{service: service}
}

// Login maneja POST /api/v1/auth/login
func (h *Handler) Login(w http.ResponseWriter, r *http.Request) {
	var req LoginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		utils.ErrorJSON(w, http.StatusBadRequest, "Formato JSON inválido")
		return
	}

	res, err := h.service.Login(r.Context(), req)
	if err != nil {
		utils.ErrorJSON(w, http.StatusUnauthorized, err.Error())
		return
	}

	utils.JSON(w, http.StatusOK, res)
}

// RegistrarCliente maneja POST /api/v1/auth/registro-cliente
func (h *Handler) RegistrarCliente(w http.ResponseWriter, r *http.Request) {
	var req RegisterRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		utils.ErrorJSON(w, http.StatusBadRequest, "Formato JSON inválido")
		return
	}

	res, err := h.service.RegistrarCliente(r.Context(), req)
	if err != nil {
		utils.ErrorJSON(w, http.StatusBadRequest, err.Error())
		return
	}

	utils.JSON(w, http.StatusCreated, res)
}
