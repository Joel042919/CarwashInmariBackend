package reportes

import (
	"database/sql"
	"net/http"

	"carwashinmaribackend/internal/middleware"
	"carwashinmaribackend/internal/utils"
	"github.com/go-chi/chi/v5"
)

type Handler struct{ service *Service }

func RegisterRoutes(r chi.Router, db *sql.DB) {
	h := &Handler{service: NewService(NewRepository(db))}
	r.With(middleware.RequireRoles("administrador")).Get("/admin/reportes", h.Consultar)
}

func (h *Handler) Consultar(w http.ResponseWriter, r *http.Request) {
	actor, ok := middleware.ClaimsFromContext(r.Context())
	if !ok {
		utils.ErrorJSON(w, 401, "Sesión requerida")
		return
	}
	rango, err := utils.ParseRangoFechas(r.URL.Query().Get("desde"), r.URL.Query().Get("hasta"))
	if err != nil {
		utils.WriteError(w, err)
		return
	}
	datos, err := h.service.Consultar(r.Context(), actor.SedeID, rango)
	if err != nil {
		utils.WriteError(w, err)
		return
	}
	utils.JSON(w, http.StatusOK, datos)
}
