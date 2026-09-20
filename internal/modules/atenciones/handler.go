package atenciones

import (
	"carwashinmaribackend/internal/middleware"
	"carwashinmaribackend/internal/utils"
	"database/sql"
	"encoding/json"
	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"net/http"
)

type Handler struct{ service *Service }

func RegisterRoutes(r chi.Router, db *sql.DB) {
	h := &Handler{service: NewService(NewRepository(db))}
	r.Get("/atenciones", h.Listar)
	r.Get("/atenciones/{id}/historial", h.Historial)
	r.With(middleware.RequireRoles("administrador", "trabajador")).Patch("/atenciones/{id}/estado", h.CambiarEstado)
	r.With(middleware.RequireRoles("administrador")).Get("/admin/trabajadores/{id}/rendimiento", h.Rendimiento)
	r.With(middleware.RequireRoles("trabajador")).Get("/trabajadores/mi-rendimiento", h.Rendimiento)
}
func actor(w http.ResponseWriter, r *http.Request) (utils.CustomClaims, bool) {
	c, ok := middleware.ClaimsFromContext(r.Context())
	if !ok {
		utils.ErrorJSON(w, 401, "Sesión requerida")
		return utils.CustomClaims{}, false
	}
	return *c, true
}
func paramID(w http.ResponseWriter, r *http.Request) (uuid.UUID, bool) {
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		utils.ErrorJSON(w, 400, "ID inválido")
		return uuid.Nil, false
	}
	return id, true
}
func (h *Handler) Listar(w http.ResponseWriter, r *http.Request) {
	c, ok := actor(w, r)
	if !ok {
		return
	}
	q := r.URL.Query()
	rango, err := utils.ParseRangoFechas(q.Get("desde"), q.Get("hasta"))
	if err != nil {
		utils.WriteError(w, err)
		return
	}
	f := Filtro{Estado: q.Get("estado"), Rango: rango}
	if raw := q.Get("trabajador"); raw != "" {
		id, err := uuid.Parse(raw)
		if err != nil {
			utils.ErrorJSON(w, 400, "Trabajador inválido")
			return
		}
		if c.Rol != "administrador" {
			utils.ErrorJSON(w, 403, "Filtro reservado al administrador")
			return
		}
		f.Trabajador = &id
	}
	lista, err := h.service.Listar(r.Context(), c, f)
	if err != nil {
		utils.WriteError(w, err)
		return
	}
	utils.JSON(w, 200, lista)
}
func (h *Handler) Historial(w http.ResponseWriter, r *http.Request) {
	c, ok := actor(w, r)
	if !ok {
		return
	}
	id, ok := paramID(w, r)
	if !ok {
		return
	}
	lista, err := h.service.Historial(r.Context(), c, id)
	if err != nil {
		utils.WriteError(w, err)
		return
	}
	utils.JSON(w, 200, lista)
}
func (h *Handler) CambiarEstado(w http.ResponseWriter, r *http.Request) {
	c, ok := actor(w, r)
	if !ok {
		return
	}
	id, ok := paramID(w, r)
	if !ok {
		return
	}
	var in struct {
		Estado string `json:"estado"`
	}
	if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
		utils.ErrorJSON(w, 400, "Cuerpo JSON inválido")
		return
	}
	if err := h.service.CambiarEstado(r.Context(), c, id, in.Estado); err != nil {
		utils.WriteError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
func (h *Handler) Rendimiento(w http.ResponseWriter, r *http.Request) {
	c, ok := actor(w, r)
	if !ok {
		return
	}
	id := c.UserID
	if c.Rol == "administrador" {
		id, ok = paramID(w, r)
		if !ok {
			return
		}
	}
	rango, err := utils.ParseRangoFechas(r.URL.Query().Get("desde"), r.URL.Query().Get("hasta"))
	if err != nil {
		utils.WriteError(w, err)
		return
	}
	data, err := h.service.Rendimiento(r.Context(), c, id, rango)
	if err != nil {
		utils.WriteError(w, err)
		return
	}
	utils.JSON(w, 200, data)
}
