package productos

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

// RegisterRoutes conecta las rutas de RF-12 (catálogo) sobre un router ya protegido por JWT.
func RegisterRoutes(r chi.Router, db *sql.DB) {
	h := NewHandler(NewService(NewRepository(db)))

	r.Get("/productos", h.ListarProductos)
	r.Get("/productos/categorias", h.ListarCategorias)
	r.Get("/productos/{id}", h.ObtenerProducto)

	r.Group(func(admin chi.Router) {
		admin.Use(middleware.RequireRoles("administrador"))
		admin.Post("/admin/productos", h.CrearProducto)
		admin.Put("/admin/productos/{id}", h.ActualizarProducto)
		admin.Patch("/admin/productos/{id}/stock", h.AjustarStock)
		admin.Post("/admin/productos/categorias", h.CrearCategoria)
		admin.Put("/admin/productos/categorias/{id}", h.ActualizarCategoria)
	})
}

func esAdmin(r *http.Request) bool {
	claims, ok := middleware.ClaimsFromContext(r.Context())
	return ok && strings.EqualFold(claims.Rol, "administrador")
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
	lista, err := h.service.ListarCategorias(r.Context(), !esAdmin(r))
	if err != nil {
		utils.WriteError(w, err)
		return
	}
	utils.JSON(w, http.StatusOK, lista)
}

func (h *Handler) ListarProductos(w http.ResponseWriter, r *http.Request) {
	admin := esAdmin(r)
	q := r.URL.Query()

	f := FiltroProductos{
		SoloActivos: !admin,
		Busqueda:    q.Get("q"),
		SoloBajos:   admin && q.Get("stock_bajo") == "true",
	}
	if c := q.Get("categoria"); c != "" {
		id, err := uuid.Parse(c)
		if err != nil {
			utils.ErrorJSON(w, http.StatusBadRequest, "ID de categoría inválido")
			return
		}
		f.IDCategoria = &id
	}

	lista, err := h.service.ListarProductos(r.Context(), f)
	if err != nil {
		utils.WriteError(w, err)
		return
	}
	utils.JSON(w, http.StatusOK, lista)
}

func (h *Handler) ObtenerProducto(w http.ResponseWriter, r *http.Request) {
	id, ok := parseID(w, r)
	if !ok {
		return
	}
	p, err := h.service.ObtenerProducto(r.Context(), id, !esAdmin(r))
	if err != nil {
		utils.WriteError(w, err)
		return
	}
	utils.JSON(w, http.StatusOK, p)
}

func (h *Handler) CrearProducto(w http.ResponseWriter, r *http.Request) {
	var in ProductoInput
	if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
		utils.ErrorJSON(w, http.StatusBadRequest, "Cuerpo JSON inválido")
		return
	}
	p, err := h.service.CrearProducto(r.Context(), in)
	if err != nil {
		utils.WriteError(w, err)
		return
	}
	utils.JSON(w, http.StatusCreated, p)
}

func (h *Handler) ActualizarProducto(w http.ResponseWriter, r *http.Request) {
	id, ok := parseID(w, r)
	if !ok {
		return
	}
	var in ProductoInput
	if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
		utils.ErrorJSON(w, http.StatusBadRequest, "Cuerpo JSON inválido")
		return
	}
	p, err := h.service.ActualizarProducto(r.Context(), id, in)
	if err != nil {
		utils.WriteError(w, err)
		return
	}
	utils.JSON(w, http.StatusOK, p)
}

func (h *Handler) AjustarStock(w http.ResponseWriter, r *http.Request) {
	id, ok := parseID(w, r)
	if !ok {
		return
	}
	var in AjusteStockInput
	if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
		utils.ErrorJSON(w, http.StatusBadRequest, "Cuerpo JSON inválido")
		return
	}
	p, err := h.service.AjustarStock(r.Context(), id, in.Cantidad)
	if err != nil {
		utils.WriteError(w, err)
		return
	}
	utils.JSON(w, http.StatusOK, p)
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
