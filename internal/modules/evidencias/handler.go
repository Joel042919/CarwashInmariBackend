package evidencias

import (
	"carwashinmaribackend/internal/middleware"
	"carwashinmaribackend/internal/utils"
	"database/sql"
	"fmt"
	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"net/http"
	"path/filepath"
	"strings"
	"time"
)

type Handler struct {
	repo Repository
	r2   *utils.R2Client
}

func RegisterRoutes(r chi.Router, db *sql.DB, r2 *utils.R2Client) {
	h := &Handler{repo: NewRepository(db), r2: r2}
	r.Get("/atenciones/{id}/evidencias", h.Listar)
	r.With(middleware.RequireRoles("administrador", "trabajador")).Post("/atenciones/{id}/evidencias", h.Crear)
}
func (h *Handler) id(w http.ResponseWriter, r *http.Request) (uuid.UUID, *utils.CustomClaims, bool) {
	c, ok := middleware.ClaimsFromContext(r.Context())
	if !ok {
		utils.ErrorJSON(w, 401, "Sesión requerida")
		return uuid.Nil, nil, false
	}
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		utils.ErrorJSON(w, 400, "ID de atención inválido")
		return uuid.Nil, nil, false
	}
	return id, c, true
}
func (h *Handler) Listar(w http.ResponseWriter, r *http.Request) {
	id, c, ok := h.id(w, r)
	if !ok {
		return
	}
	items, err := h.repo.Listar(r.Context(), *c, id)
	if err != nil {
		utils.WriteError(w, err)
		return
	}
	utils.JSON(w, 200, items)
}
func (h *Handler) Crear(w http.ResponseWriter, r *http.Request) {
	id, c, ok := h.id(w, r)
	if !ok {
		return
	}
	if err := r.ParseMultipartForm(10 << 20); err != nil {
		utils.ErrorJSON(w, 400, "La evidencia no puede superar 10 MB")
		return
	}
	vid, err := uuid.Parse(r.FormValue("id_vehiculo"))
	if err != nil {
		utils.ErrorJSON(w, 400, "Selecciona el vehículo de la atención")
		return
	}
	desc := strings.TrimSpace(r.FormValue("descripcion"))
	if len(desc) < 3 || len(desc) > 500 {
		utils.ErrorJSON(w, 400, "Describe el daño o condición (3 a 500 caracteres)")
		return
	}
	files := r.MultipartForm.File["foto"]
	if len(files) != 1 {
		utils.ErrorJSON(w, 400, "Adjunta una fotografía")
		return
	}
	fh := files[0]
	ext := strings.ToLower(filepath.Ext(fh.Filename))
	if ext != ".jpg" && ext != ".jpeg" && ext != ".png" && ext != ".webp" {
		utils.ErrorJSON(w, 400, "Solo se admiten JPG, PNG o WEBP")
		return
	}
	src, err := fh.Open()
	if err != nil {
		utils.ErrorJSON(w, 500, "No se pudo leer la fotografía")
		return
	}
	defer src.Close()
	key := fmt.Sprintf("evidencias/%s/%d_%s%s", id, time.Now().UnixNano(), uuid.New().String()[:8], ext)
	ruta, err := utils.GuardarArchivo(r.Context(), h.r2, src, key, fh.Header.Get("Content-Type"))
	if err != nil {
		utils.ErrorJSON(w, 500, "No se pudo guardar la fotografía")
		return
	}
	e := &Evidencia{IDEvidencia: uuid.New(), IDAtencion: id, IDVehiculo: vid, RegistradoPor: c.UserID, Tipo: "preexistente", Descripcion: &desc, RutaFoto: ruta, CreatedAt: time.Now()}
	if err = h.repo.Crear(r.Context(), *c, e); err != nil {
		utils.WriteError(w, err)
		return
	}
	utils.JSON(w, http.StatusCreated, e)
}
