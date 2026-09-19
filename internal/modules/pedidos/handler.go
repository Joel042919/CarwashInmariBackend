package pedidos

import (
	"database/sql"
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	"carwashinmaribackend/internal/middleware"
	"carwashinmaribackend/internal/utils"
)

type Handler struct {
	service Service
}

func NewHandler(service Service) *Handler {
	return &Handler{service: service}
}

// RegisterRoutes conecta las rutas de RF-12 (pedidos) sobre un router ya protegido por JWT.
func RegisterRoutes(r chi.Router, db *sql.DB) {
	h := NewHandler(NewService(NewRepository(db)))

	r.Post("/pedidos", h.CrearPedido)
	r.Get("/pedidos/mis-pedidos", h.ListarMisPedidos)
	r.Patch("/pedidos/{id}/cancelar", h.CancelarMiPedido)

	r.Group(func(admin chi.Router) {
		admin.Use(middleware.RequireRoles("administrador"))
		admin.Get("/admin/pedidos", h.ListarTodos)
		admin.Patch("/admin/pedidos/{id}/estado", h.CambiarEstado)
	})
}

func (h *Handler) CrearPedido(w http.ResponseWriter, r *http.Request) {
	claims, ok := middleware.ClaimsFromContext(r.Context())
	if !ok {
		utils.ErrorJSON(w, http.StatusUnauthorized, "Contexto de usuario inválido")
		return
	}
	var req CrearPedidoRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		utils.ErrorJSON(w, http.StatusBadRequest, "Cuerpo JSON inválido")
		return
	}
	pedido, err := h.service.CrearPedido(r.Context(), claims.UserID, req)
	if err != nil {
		utils.WriteError(w, err)
		return
	}
	utils.JSON(w, http.StatusCreated, pedido)
}

func (h *Handler) ListarMisPedidos(w http.ResponseWriter, r *http.Request) {
	claims, ok := middleware.ClaimsFromContext(r.Context())
	if !ok {
		utils.ErrorJSON(w, http.StatusUnauthorized, "Contexto de usuario inválido")
		return
	}
	lista, err := h.service.ListarMisPedidos(r.Context(), claims.UserID)
	if err != nil {
		utils.WriteError(w, err)
		return
	}
	utils.JSON(w, http.StatusOK, lista)
}

func (h *Handler) CancelarMiPedido(w http.ResponseWriter, r *http.Request) {
	claims, ok := middleware.ClaimsFromContext(r.Context())
	if !ok {
		utils.ErrorJSON(w, http.StatusUnauthorized, "Contexto de usuario inválido")
		return
	}
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		utils.ErrorJSON(w, http.StatusBadRequest, "ID de pedido inválido")
		return
	}
	pedido, err := h.service.CancelarMiPedido(r.Context(), claims.UserID, id)
	if err != nil {
		utils.WriteError(w, err)
		return
	}
	utils.JSON(w, http.StatusOK, pedido)
}

func (h *Handler) ListarTodos(w http.ResponseWriter, r *http.Request) {
	lista, err := h.service.ListarTodos(r.Context(), r.URL.Query().Get("estado"))
	if err != nil {
		utils.WriteError(w, err)
		return
	}
	utils.JSON(w, http.StatusOK, lista)
}

func (h *Handler) CambiarEstado(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		utils.ErrorJSON(w, http.StatusBadRequest, "ID de pedido inválido")
		return
	}
	var req CambiarEstadoRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		utils.ErrorJSON(w, http.StatusBadRequest, "Cuerpo JSON inválido")
		return
	}
	pedido, err := h.service.CambiarEstado(r.Context(), id, req.Estado)
	if err != nil {
		utils.WriteError(w, err)
		return
	}
	utils.JSON(w, http.StatusOK, pedido)
}
