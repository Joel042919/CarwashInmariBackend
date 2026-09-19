package espacios

import (
	"database/sql"
	"encoding/json"
	"net/http"
	"strings"

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

// RegisterRoutes conecta las rutas de RF-05 (espacios y horarios de atención).
//
// TODO(Erick): este módulo es una primera versión COMPLETA de RF-05 (espacios y horario
// semanal), hecha para poder probar las reservas. Es tuyo: puedes modificarla o
// reemplazarla por completo. No usa el tipo de lavadero (espacios_lavado.id_tipo_lavadero
// está definido como texto pero es clave foránea de un uuid: corregir el esquema con Joel
// de ser necesario). Ideas pendientes: días festivos o bloqueos puntuales de un espacio.
func RegisterRoutes(r chi.Router, db *sql.DB) {
	h := NewHandler(NewService(NewRepository(db)))

	r.Get("/espacios", h.Listar)

	r.Group(func(admin chi.Router) {
		admin.Use(middleware.RequireRoles("administrador"))
		admin.Post("/admin/espacios", h.Crear)
		admin.Put("/admin/espacios/{id}", h.Actualizar)
		admin.Put("/admin/espacios/{id}/horarios", h.ReemplazarHorarios)
	})
}

func parseID(w http.ResponseWriter, r *http.Request) (uuid.UUID, bool) {
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		utils.ErrorJSON(w, http.StatusBadRequest, "ID inválido")
		return uuid.Nil, false
	}
	return id, true
}

func (h *Handler) Listar(w http.ResponseWriter, r *http.Request) {
	claims, ok := middleware.ClaimsFromContext(r.Context())
	soloActivos := !ok || !strings.EqualFold(claims.Rol, "administrador")
	lista, err := h.service.Listar(r.Context(), soloActivos)
	if err != nil {
		utils.WriteError(w, err)
		return
	}
	utils.JSON(w, http.StatusOK, lista)
}

func (h *Handler) Crear(w http.ResponseWriter, r *http.Request) {
	var in EspacioInput
	if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
		utils.ErrorJSON(w, http.StatusBadRequest, "Cuerpo JSON inválido")
		return
	}
	e, err := h.service.Crear(r.Context(), in)
	if err != nil {
		utils.WriteError(w, err)
		return
	}
	utils.JSON(w, http.StatusCreated, e)
}

func (h *Handler) Actualizar(w http.ResponseWriter, r *http.Request) {
	id, ok := parseID(w, r)
	if !ok {
		return
	}
	var in EspacioInput
	if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
		utils.ErrorJSON(w, http.StatusBadRequest, "Cuerpo JSON inválido")
		return
	}
	e, err := h.service.Actualizar(r.Context(), id, in)
	if err != nil {
		utils.WriteError(w, err)
		return
	}
	utils.JSON(w, http.StatusOK, e)
}

func (h *Handler) ReemplazarHorarios(w http.ResponseWriter, r *http.Request) {
	id, ok := parseID(w, r)
	if !ok {
		return
	}
	var in HorariosInput
	if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
		utils.ErrorJSON(w, http.StatusBadRequest, "Cuerpo JSON inválido")
		return
	}
	e, err := h.service.ReemplazarHorarios(r.Context(), id, in)
	if err != nil {
		utils.WriteError(w, err)
		return
	}
	utils.JSON(w, http.StatusOK, e)
}
