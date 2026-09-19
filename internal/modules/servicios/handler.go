package servicios

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

// RegisterRoutes conecta las rutas de RF-04 sobre un router ya protegido por JWT.
func RegisterRoutes(r chi.Router, db *sql.DB) {
	h := NewHandler(NewService(NewRepository(db)))

	r.Get("/servicios", h.ListarServicios)
	r.Get("/servicios/categorias", h.ListarCategorias)
	r.Get("/servicios/{id}", h.ObtenerServicio)

	r.Group(func(admin chi.Router) {
		admin.Use(middleware.RequireRoles("administrador"))
		admin.Post("/admin/servicios", h.CrearServicio)
		admin.Put("/admin/servicios/{id}", h.ActualizarServicio)
		admin.Patch("/admin/servicios/{id}/disponibilidad", h.CambiarDisponibilidad)
		admin.Post("/admin/servicios/categorias", h.CrearCategoria)
		admin.Put("/admin/servicios/categorias/{id}", h.ActualizarCategoria)
	})
}

// veSoloDisponibles: solo el administrador ve también los servicios no disponibles.
func veSoloDisponibles(r *http.Request) bool {
	claims, ok := middleware.ClaimsFromContext(r.Context())
	return !ok || !strings.EqualFold(claims.Rol, "administrador")
}

func parseID(w http.ResponseWriter, r *http.Request) (uuid.UUID, bool) {
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		utils.ErrorJSON(w, http.StatusBadRequest, "ID inválido")
		return uuid.Nil, false
	}
	return id, true
}

func (h *Handler) ListarCategorias(w http.ResponseWriter, r *http.Request) {
	lista, err := h.service.ListarCategorias(r.Context())
	if err != nil {
		utils.WriteError(w, err)
		return
	}
	utils.JSON(w, http.StatusOK, lista)
}

func (h *Handler) ListarServicios(w http.ResponseWriter, r *http.Request) {
	var idCategoria *uuid.UUID
	if q := r.URL.Query().Get("categoria"); q != "" {
		id, err := uuid.Parse(q)
		if err != nil {
			utils.ErrorJSON(w, http.StatusBadRequest, "ID de categoría inválido")
			return
		}
		idCategoria = &id
	}
	lista, err := h.service.ListarServicios(r.Context(), veSoloDisponibles(r), idCategoria)
	if err != nil {
		utils.WriteError(w, err)
		return
	}
	utils.JSON(w, http.StatusOK, lista)
}

func (h *Handler) ObtenerServicio(w http.ResponseWriter, r *http.Request) {
	id, ok := parseID(w, r)
	if !ok {
		return
	}
	srv, err := h.service.ObtenerServicio(r.Context(), id, veSoloDisponibles(r))
	if err != nil {
		utils.WriteError(w, err)
		return
	}
	utils.JSON(w, http.StatusOK, srv)
}

func (h *Handler) CrearServicio(w http.ResponseWriter, r *http.Request) {
	var in ServicioInput
	if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
		utils.ErrorJSON(w, http.StatusBadRequest, "Cuerpo JSON inválido")
		return
	}
	srv, err := h.service.CrearServicio(r.Context(), in)
	if err != nil {
		utils.WriteError(w, err)
		return
	}
	utils.JSON(w, http.StatusCreated, srv)
}

func (h *Handler) ActualizarServicio(w http.ResponseWriter, r *http.Request) {
	id, ok := parseID(w, r)
	if !ok {
		return
	}
	var in ServicioInput
	if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
		utils.ErrorJSON(w, http.StatusBadRequest, "Cuerpo JSON inválido")
		return
	}
	srv, err := h.service.ActualizarServicio(r.Context(), id, in)
	if err != nil {
		utils.WriteError(w, err)
		return
	}
	utils.JSON(w, http.StatusOK, srv)
}

func (h *Handler) CambiarDisponibilidad(w http.ResponseWriter, r *http.Request) {
	id, ok := parseID(w, r)
	if !ok {
		return
	}
	var in DisponibilidadInput
	if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
		utils.ErrorJSON(w, http.StatusBadRequest, "Cuerpo JSON inválido")
		return
	}
	srv, err := h.service.CambiarDisponibilidad(r.Context(), id, in.Disponible)
	if err != nil {
		utils.WriteError(w, err)
		return
	}
	utils.JSON(w, http.StatusOK, srv)
}

func (h *Handler) CrearCategoria(w http.ResponseWriter, r *http.Request) {
	var in CategoriaInput
	if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
		utils.ErrorJSON(w, http.StatusBadRequest, "Cuerpo JSON inválido")
		return
	}
	c, err := h.service.CrearCategoria(r.Context(), in)
	if err != nil {
		utils.WriteError(w, err)
		return
	}
	utils.JSON(w, http.StatusCreated, c)
}

func (h *Handler) ActualizarCategoria(w http.ResponseWriter, r *http.Request) {
	id, ok := parseID(w, r)
	if !ok {
		return
	}
	var in CategoriaInput
	if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
		utils.ErrorJSON(w, http.StatusBadRequest, "Cuerpo JSON inválido")
		return
	}
	c, err := h.service.ActualizarCategoria(r.Context(), id, in)
	if err != nil {
		utils.WriteError(w, err)
		return
	}
	utils.JSON(w, http.StatusOK, c)
}
