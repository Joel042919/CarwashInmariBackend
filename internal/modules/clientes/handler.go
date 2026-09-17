package clientes

import (
	"encoding/json"
	"net/http"

	"carwashinmaribackend/internal/middleware"
	"carwashinmaribackend/internal/utils"
)

type Handler struct {
	service Service
}

func NewHandler(service Service) *Handler {
	return &Handler{service: service}
}

// ObtenerMiPerfil maneja GET /api/v1/clientes/perfil
func (h *Handler) ObtenerMiPerfil(w http.ResponseWriter, r *http.Request) {
	claims, ok := r.Context().Value(middleware.UserContextKey).(*utils.CustomClaims)
	if !ok {
		utils.ErrorJSON(w, http.StatusUnauthorized, "Contexto de usuario inválido")
		return
	}

	perfil, err := h.service.ObtenerMiPerfil(r.Context(), claims.UserID)
	if err != nil {
		utils.ErrorJSON(w, http.StatusNotFound, err.Error())
		return
	}

	utils.JSON(w, http.StatusOK, perfil)
}

// ActualizarMiPerfil maneja PUT /api/v1/clientes/perfil
func (h *Handler) ActualizarMiPerfil(w http.ResponseWriter, r *http.Request) {
	claims, ok := r.Context().Value(middleware.UserContextKey).(*utils.CustomClaims)
	if !ok {
		utils.ErrorJSON(w, http.StatusUnauthorized, "Contexto de usuario inválido")
		return
	}

	var req UpdateClienteRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		utils.ErrorJSON(w, http.StatusBadRequest, "Formato de cuerpo JSON inválido")
		return
	}

	if err := h.service.ActualizarMiPerfil(r.Context(), claims.UserID, req); err != nil {
		utils.ErrorJSON(w, http.StatusBadRequest, err.Error())
		return
	}

	utils.JSON(w, http.StatusOK, map[string]string{"message": "Perfil actualizado exitosamente"})
}

// ObtenerHistorialCompleto maneja GET /api/v1/clientes/historial
func (h *Handler) ObtenerHistorialCompleto(w http.ResponseWriter, r *http.Request) {
	claims, ok := r.Context().Value(middleware.UserContextKey).(*utils.CustomClaims)
	if !ok {
		utils.ErrorJSON(w, http.StatusUnauthorized, "Contexto de usuario inválido")
		return
	}

	historial, err := h.service.ObtenerHistorialUnificado(r.Context(), claims.UserID)
	if err != nil {
		utils.ErrorJSON(w, http.StatusInternalServerError, err.Error())
		return
	}

	utils.JSON(w, http.StatusOK, historial)
}
