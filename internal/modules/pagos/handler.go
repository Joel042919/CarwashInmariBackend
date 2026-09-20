package pagos

import (
	"database/sql"
	"encoding/json"
	"net/http"
	"strings"

	"carwashinmaribackend/internal/middleware"
	"carwashinmaribackend/internal/utils"
	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
)

type Handler struct{ service *Service }

func RegisterRoutes(r chi.Router, db *sql.DB) {
	h := &Handler{service: NewService(NewRepository(db))}
	r.With(middleware.RequireRoles("administrador")).Get("/admin/pagos", h.Listar)
	r.With(middleware.RequireRoles("administrador")).Get("/admin/pagos/pendientes", h.ListarPendientes)
	r.With(middleware.RequireRoles("administrador")).Post("/admin/pagos", h.Registrar)
	r.With(middleware.RequireRoles("administrador")).Post("/admin/pagos/{id}/reversar", h.Revertir)
	r.With(middleware.RequireRoles("cliente")).Get("/pagos/mis-pagos", h.MisPagos)
	r.With(middleware.RequireRoles("administrador", "cliente")).Get("/pagos/{id}/comprobante", h.Comprobante)
}

func (h *Handler) MisPagos(w http.ResponseWriter, r *http.Request) {
	c, ok := middleware.ClaimsFromContext(r.Context())
	if !ok {
		utils.ErrorJSON(w, 401, "Sesión requerida")
		return
	}
	lista, err := h.service.ListarPorCliente(r.Context(), c.UserID)
	if err != nil {
		utils.WriteError(w, err)
		return
	}
	utils.JSON(w, 200, lista)
}

func (h *Handler) Listar(w http.ResponseWriter, r *http.Request) {
	c, ok := middleware.ClaimsFromContext(r.Context())
	if !ok {
		utils.ErrorJSON(w, 401, "Sesión requerida")
		return
	}
	lista, err := h.service.Listar(r.Context(), c.SedeID, strings.TrimSpace(r.URL.Query().Get("estado")))
	if err != nil {
		utils.WriteError(w, err)
		return
	}
	utils.JSON(w, 200, lista)
}

func (h *Handler) ListarPendientes(w http.ResponseWriter, r *http.Request) {
	c, ok := middleware.ClaimsFromContext(r.Context())
	if !ok {
		utils.ErrorJSON(w, 401, "Sesión requerida")
		return
	}
	lista, err := h.service.ListarPendientes(r.Context(), c.SedeID, strings.TrimSpace(r.URL.Query().Get("tipo")))
	if err != nil {
		utils.WriteError(w, err)
		return
	}
	utils.JSON(w, 200, lista)
}

func (h *Handler) Registrar(w http.ResponseWriter, r *http.Request) {
	c, ok := middleware.ClaimsFromContext(r.Context())
	if !ok {
		utils.ErrorJSON(w, 401, "Sesión requerida")
		return
	}
	var in RegistrarPagoInput
	if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
		utils.ErrorJSON(w, 400, "Cuerpo JSON inválido")
		return
	}
	pago, err := h.service.Registrar(r.Context(), c.SedeID, c.UserID, in)
	if err != nil {
		utils.WriteError(w, err)
		return
	}
	utils.JSON(w, http.StatusCreated, pago)
}

func (h *Handler) Comprobante(w http.ResponseWriter, r *http.Request) {
	c, ok := middleware.ClaimsFromContext(r.Context())
	if !ok {
		utils.ErrorJSON(w, 401, "Sesión requerida")
		return
	}
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		utils.ErrorJSON(w, 400, "ID inválido")
		return
	}
	pago, err := h.service.ObtenerComprobante(r.Context(), *c, id)
	if err != nil {
		utils.WriteError(w, err)
		return
	}
	utils.JSON(w, 200, pago)
}

func (h *Handler) Revertir(w http.ResponseWriter, r *http.Request) {
	c, ok := middleware.ClaimsFromContext(r.Context())
	if !ok {
		utils.ErrorJSON(w, 401, "Sesión requerida")
		return
	}
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		utils.ErrorJSON(w, 400, "ID inválido")
		return
	}
	var in RevertirPagoInput
	if err = json.NewDecoder(r.Body).Decode(&in); err != nil {
		utils.ErrorJSON(w, 400, "Cuerpo JSON inválido")
		return
	}
	pago, err := h.service.Revertir(r.Context(), c.SedeID, c.UserID, id, in)
	if err != nil {
		utils.WriteError(w, err)
		return
	}
	utils.JSON(w, 200, pago)
}
