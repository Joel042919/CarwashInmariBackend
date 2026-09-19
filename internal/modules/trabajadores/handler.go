package trabajadores

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

// RegisterRoutes conecta el alta mínima de trabajadores que necesita la programación (RF-09).
// RF-13 (Ingrid) puedes ampliarlo con rendimiento y reportes.
//
// TODO(Ingrid): completar RF-13 (editar y dar de baja, especialidades, consulta de
// asignaciones y medición de rendimiento) y definir qué ve un trabajador al iniciar
// sesión: hoy el rol "trabajador" existe pero la app aún no tiene vista para él.
func RegisterRoutes(r chi.Router, db *sql.DB) {
	h := NewHandler(NewService(NewRepository(db)))

	r.Group(func(admin chi.Router) {
		admin.Use(middleware.RequireRoles("administrador"))
		admin.Get("/admin/trabajadores", h.Listar)
		admin.Post("/admin/trabajadores", h.Crear)
		admin.Patch("/admin/trabajadores/{id}/disponibilidad", h.CambiarDisponibilidad)
	})
}

func (h *Handler) Listar(w http.ResponseWriter, r *http.Request) {
	lista, err := h.service.Listar(r.Context())
	if err != nil {
		utils.WriteError(w, err)
		return
	}
	utils.JSON(w, http.StatusOK, lista)
}

func (h *Handler) Crear(w http.ResponseWriter, r *http.Request) {
	claims, ok := middleware.ClaimsFromContext(r.Context())
	if !ok {
		utils.ErrorJSON(w, http.StatusUnauthorized, "Contexto de usuario inválido")
		return
	}
	var in CrearTrabajadorInput
	if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
		utils.ErrorJSON(w, http.StatusBadRequest, "Cuerpo JSON inválido")
		return
	}
	t, err := h.service.Crear(r.Context(), claims.SedeID, in)
	if err != nil {
		utils.WriteError(w, err)
		return
	}
	utils.JSON(w, http.StatusCreated, t)
}

func (h *Handler) CambiarDisponibilidad(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		utils.ErrorJSON(w, http.StatusBadRequest, "ID inválido")
		return
	}
	var in DisponibilidadInput
	if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
		utils.ErrorJSON(w, http.StatusBadRequest, "Cuerpo JSON inválido")
		return
	}
	if err := h.service.CambiarDisponibilidad(r.Context(), id, in.Disponible); err != nil {
		utils.WriteError(w, err)
		return
	}
	utils.JSON(w, http.StatusOK, map[string]bool{"disponible": in.Disponible})
}
