package vehiculos

import (
	"database/sql"
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"

	"carwashinmaribackend/internal/middleware"
	"carwashinmaribackend/internal/utils"
)

type Handler struct {
	service Service
}

func NewHandler(service Service) *Handler {
	return &Handler{service: service}
}

// RegisterRoutes conecta el registro mínimo de vehículos que necesitan las reservas.
// Es solo lo necesario para RF-06; RF-03 (Mego) puedes ampliarlo o reemplazarlo.
//
// TODO(Mego): completar RF-03 (editar, dar de baja, foto del vehículo, consulta para el
// administrador). RF-08 (evidencias) y RF-10 (trazabilidad) siguen sin implementar:
// reservas.Programar ya deja la atención en estado "programada" con su primer registro
// en historial_estados_atencion; los siguientes cambios de estado parten de ahí.
func RegisterRoutes(r chi.Router, db *sql.DB) {
	h := NewHandler(NewService(NewRepository(db)))

	r.Group(func(cliente chi.Router) {
		cliente.Use(middleware.RequireRoles("cliente"))
		cliente.Post("/vehiculos", h.Registrar)
		cliente.Get("/vehiculos/mis-vehiculos", h.MisVehiculos)
	})
}

func (h *Handler) Registrar(w http.ResponseWriter, r *http.Request) {
	claims, ok := middleware.ClaimsFromContext(r.Context())
	if !ok {
		utils.ErrorJSON(w, http.StatusUnauthorized, "Contexto de usuario inválido")
		return
	}
	var in VehiculoInput
	if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
		utils.ErrorJSON(w, http.StatusBadRequest, "Cuerpo JSON inválido")
		return
	}
	v, err := h.service.Registrar(r.Context(), claims.UserID, in)
	if err != nil {
		utils.WriteError(w, err)
		return
	}
	utils.JSON(w, http.StatusCreated, v)
}

func (h *Handler) MisVehiculos(w http.ResponseWriter, r *http.Request) {
	claims, ok := middleware.ClaimsFromContext(r.Context())
	if !ok {
		utils.ErrorJSON(w, http.StatusUnauthorized, "Contexto de usuario inválido")
		return
	}
	lista, err := h.service.MisVehiculos(r.Context(), claims.UserID)
	if err != nil {
		utils.WriteError(w, err)
		return
	}
	utils.JSON(w, http.StatusOK, lista)
}
