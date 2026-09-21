package reservas

import (
	"database/sql"
	"encoding/json"
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	"carwashinmaribackend/internal/middleware"
	"carwashinmaribackend/internal/modules/documentos"
	"carwashinmaribackend/internal/utils"
)

type Handler struct {
	service Service
}

func NewHandler(service Service) *Handler {
	return &Handler{service: service}
}

// RegisterRoutes conecta RF-06 (reservas del cliente) y RF-09 (programación y asignación).
//
// Contrato con RF-07: Programar no confirma sin documentos previos validados.
// Los cruces de horario se evitan con pg_advisory_xact_lock por espacio+fecha
// (y por trabajador+fecha al asignar) dentro de la transacción.
func RegisterRoutes(r chi.Router, db *sql.DB, docs documentos.Checker) {
	h := NewHandler(NewService(NewRepository(db), docs))

	// Rutas estáticas antes de /reservas/{id} para que chi no las capture como UUID.
	r.Get("/reservas/disponibilidad", h.Disponibilidad)

	r.Group(func(cliente chi.Router) {
		cliente.Use(middleware.RequireRoles("cliente"))
		cliente.Post("/reservas", h.Crear)
		cliente.Get("/reservas/mis-reservas", h.MisReservas)
		cliente.Patch("/reservas/{id}/reprogramar", h.Reprogramar)
		cliente.Patch("/reservas/{id}/cancelar", h.CancelarCliente)
	})

	r.Get("/reservas/{id}", h.Obtener)

	r.Group(func(admin chi.Router) {
		admin.Use(middleware.RequireRoles("administrador"))
		admin.Use(reservaDeSede(db))
		admin.Get("/admin/reservas/agenda", h.Agenda)
		admin.Get("/admin/reservas", h.ListarTodas)
		admin.Get("/admin/reservas/{id}/trabajadores", h.TrabajadoresDisponibles)
		admin.Post("/admin/reservas/{id}/programar", h.Programar)
		admin.Patch("/admin/reservas/{id}/cancelar", h.CancelarAdmin)
	})
}

func claims(w http.ResponseWriter, r *http.Request) (*utils.CustomClaims, bool) {
	c, ok := middleware.ClaimsFromContext(r.Context())
	if !ok {
		utils.ErrorJSON(w, http.StatusUnauthorized, "Contexto de usuario inválido")
	}
	return c, ok
}

func parseID(w http.ResponseWriter, r *http.Request) (uuid.UUID, bool) {
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		utils.ErrorJSON(w, http.StatusBadRequest, "ID de reserva inválido")
		return uuid.Nil, false
	}
	return id, true
}

func decode(w http.ResponseWriter, r *http.Request, dst any) bool {
	if err := json.NewDecoder(r.Body).Decode(dst); err != nil {
		utils.ErrorJSON(w, http.StatusBadRequest, "Cuerpo JSON inválido")
		return false
	}
	return true
}

// Disponibilidad: GET /reservas/disponibilidad?fecha=AAAA-MM-DD&servicios=id1,id2[&excluir_reserva=id]
func (h *Handler) Disponibilidad(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()

	var servicios []uuid.UUID
	if raw := strings.TrimSpace(q.Get("servicios")); raw != "" {
		for _, p := range strings.Split(raw, ",") {
			id, err := uuid.Parse(strings.TrimSpace(p))
			if err != nil {
				utils.ErrorJSON(w, http.StatusBadRequest, "ID de servicio inválido")
				return
			}
			servicios = append(servicios, id)
		}
	}
	var excluir *uuid.UUID
	if raw := q.Get("excluir_reserva"); raw != "" {
		id, err := uuid.Parse(raw)
		if err != nil {
			utils.ErrorJSON(w, http.StatusBadRequest, "ID de reserva inválido")
			return
		}
		excluir = &id
	}

	res, err := h.service.Disponibilidad(r.Context(), q.Get("fecha"), servicios, excluir)
	if err != nil {
		utils.WriteError(w, err)
		return
	}
	utils.JSON(w, http.StatusOK, res)
}

func (h *Handler) Crear(w http.ResponseWriter, r *http.Request) {
	c, ok := claims(w, r)
	if !ok {
		return
	}
	var in CrearReservaInput
	if !decode(w, r, &in) {
		return
	}
	res, err := h.service.Crear(r.Context(), c.UserID, in)
	if err != nil {
		utils.WriteError(w, err)
		return
	}
	utils.JSON(w, http.StatusCreated, res)
}

func (h *Handler) MisReservas(w http.ResponseWriter, r *http.Request) {
	c, ok := claims(w, r)
	if !ok {
		return
	}
	lista, err := h.service.MisReservas(r.Context(), c.UserID)
	if err != nil {
		utils.WriteError(w, err)
		return
	}
	utils.JSON(w, http.StatusOK, lista)
}

// Obtener: el cliente solo ve sus reservas; administrador y trabajador ven cualquiera.
func (h *Handler) Obtener(w http.ResponseWriter, r *http.Request) {
	c, ok := claims(w, r)
	if !ok {
		return
	}
	id, ok := parseID(w, r)
	if !ok {
		return
	}
	var idCliente *uuid.UUID
	if strings.EqualFold(c.Rol, "cliente") {
		idCliente = &c.UserID
	}
	res, err := h.service.Obtener(r.Context(), id, idCliente)
	if err != nil {
		utils.WriteError(w, err)
		return
	}
	utils.JSON(w, http.StatusOK, res)
}

func (h *Handler) Reprogramar(w http.ResponseWriter, r *http.Request) {
	c, ok := claims(w, r)
	if !ok {
		return
	}
	id, ok := parseID(w, r)
	if !ok {
		return
	}
	var in ReprogramarInput
	if !decode(w, r, &in) {
		return
	}
	res, err := h.service.Reprogramar(r.Context(), c.UserID, id, in)
	if err != nil {
		utils.WriteError(w, err)
		return
	}
	utils.JSON(w, http.StatusOK, res)
}

func (h *Handler) cancelar(w http.ResponseWriter, r *http.Request, idCliente *uuid.UUID) {
	id, ok := parseID(w, r)
	if !ok {
		return
	}
	var in CancelarInput
	// El cuerpo es opcional: sin motivo se usa uno por defecto.
	_ = json.NewDecoder(r.Body).Decode(&in)

	res, err := h.service.Cancelar(r.Context(), idCliente, id, in.Motivo)
	if err != nil {
		utils.WriteError(w, err)
		return
	}
	utils.JSON(w, http.StatusOK, res)
}

func (h *Handler) CancelarCliente(w http.ResponseWriter, r *http.Request) {
	c, ok := claims(w, r)
	if !ok {
		return
	}
	h.cancelar(w, r, &c.UserID)
}

func (h *Handler) CancelarAdmin(w http.ResponseWriter, r *http.Request) {
	h.cancelar(w, r, nil)
}

func (h *Handler) ListarTodas(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	c, ok := claims(w, r)
	if !ok {
		return
	}
	lista, err := h.service.ListarTodas(r.Context(), c.SedeID, q.Get("estado"), q.Get("fecha"))
	if err != nil {
		utils.WriteError(w, err)
		return
	}
	utils.JSON(w, http.StatusOK, lista)
}

// Agenda: GET /admin/reservas/agenda?fecha=AAAA-MM-DD
// Vista del día agrupada por espacio (RF-09 planificación).
func (h *Handler) Agenda(w http.ResponseWriter, r *http.Request) {
	res, err := h.service.Agenda(r.Context(), r.URL.Query().Get("fecha"))
	if err != nil {
		utils.WriteError(w, err)
		return
	}
	utils.JSON(w, http.StatusOK, res)
}

func (h *Handler) TrabajadoresDisponibles(w http.ResponseWriter, r *http.Request) {
	id, ok := parseID(w, r)
	if !ok {
		return
	}
	lista, err := h.service.TrabajadoresDisponibles(r.Context(), id)
	if err != nil {
		utils.WriteError(w, err)
		return
	}
	utils.JSON(w, http.StatusOK, lista)
}

func (h *Handler) Programar(w http.ResponseWriter, r *http.Request) {
	c, ok := claims(w, r)
	if !ok {
		return
	}
	id, ok := parseID(w, r)
	if !ok {
		return
	}
	var in ProgramarInput
	if !decode(w, r, &in) {
		return
	}
	res, err := h.service.Programar(r.Context(), id, c.UserID, in)
	if err != nil {
		utils.WriteError(w, err)
		return
	}
	utils.JSON(w, http.StatusOK, res)
}
