package documentos

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"path/filepath"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	"carwashinmaribackend/internal/middleware"
	"carwashinmaribackend/internal/utils"
)

const maxPDFBytes = 10 << 20 // 10MB

type Handler struct {
	service  Service
	r2Client *utils.R2Client
}

func NewHandler(service Service, r2Client *utils.R2Client) *Handler {
	return &Handler{service: service, r2Client: r2Client}
}

// RegisterRoutes conecta las rutas de RF-07 sobre un router ya protegido por JWT.
func RegisterRoutes(r chi.Router, db *sql.DB, r2Client *utils.R2Client) *Handler {
	h := NewHandler(NewService(NewRepository(db)), r2Client)

	r.Post("/documentos", h.SubirDocumento)
	r.Get("/documentos/mis-documentos", h.ListarMisDocumentos)
	r.Get("/reservas/{id}/requisitos-documentales", h.RequisitosReserva)

	r.Group(func(admin chi.Router) {
		admin.Use(middleware.RequireRoles("administrador"))
		admin.Get("/admin/documentos", h.ListarTodos)
		admin.Patch("/admin/documentos/{id}/resolver", h.Resolver)
	})
	return h
}

// SubirDocumento maneja POST /api/v1/documentos (multipart/form-data):
// campos id_reserva, id_servicio y archivo "pdf".
func (h *Handler) SubirDocumento(w http.ResponseWriter, r *http.Request) {
	claims, ok := middleware.ClaimsFromContext(r.Context())
	if !ok {
		utils.ErrorJSON(w, http.StatusUnauthorized, "Contexto de usuario inválido")
		return
	}

	r.Body = http.MaxBytesReader(w, r.Body, maxPDFBytes+(1<<20))
	if err := r.ParseMultipartForm(maxPDFBytes); err != nil {
		utils.ErrorJSON(w, http.StatusBadRequest, "El PDF excede el límite permitido de 10MB")
		return
	}

	idReserva, err := uuid.Parse(r.FormValue("id_reserva"))
	if err != nil {
		utils.ErrorJSON(w, http.StatusBadRequest, "id_reserva inválido")
		return
	}
	idServicio, err := uuid.Parse(r.FormValue("id_servicio"))
	if err != nil {
		utils.ErrorJSON(w, http.StatusBadRequest, "id_servicio inválido")
		return
	}

	file, header, err := r.FormFile("pdf")
	if err != nil {
		utils.ErrorJSON(w, http.StatusBadRequest, "Adjunta el documento PDF firmado (campo 'pdf')")
		return
	}
	defer file.Close()

	if strings.ToLower(filepath.Ext(header.Filename)) != ".pdf" {
		utils.ErrorJSON(w, http.StatusBadRequest, "Formato no válido. Solo se admite PDF")
		return
	}
	magic := make([]byte, 5)
	if _, err := io.ReadFull(file, magic); err != nil || !bytes.Equal(magic, []byte("%PDF-")) {
		utils.ErrorJSON(w, http.StatusBadRequest, "El archivo no es un PDF válido")
		return
	}
	if _, err := file.Seek(0, io.SeekStart); err != nil {
		utils.ErrorJSON(w, http.StatusInternalServerError, "Error al procesar el archivo")
		return
	}

	// Se valida la reserva antes de subir nada al almacenamiento.
	if _, err := h.service.RequisitosReserva(r.Context(), idReserva, &claims.UserID); err != nil {
		utils.WriteError(w, err)
		return
	}

	key := fmt.Sprintf("documentos/%d_%s.pdf", time.Now().UnixNano(), uuid.New().String()[:8])
	ruta, err := utils.GuardarArchivo(r.Context(), h.r2Client, file, key, "application/pdf")
	if err != nil {
		utils.WriteError(w, err)
		return
	}

	doc, err := h.service.SubirDocumento(r.Context(), SubirDocumentoInput{
		IDReserva:  idReserva,
		IDServicio: idServicio,
		IDCliente:  claims.UserID,
		RutaPDF:    ruta,
	})
	if err != nil {
		utils.WriteError(w, err)
		return
	}
	utils.JSON(w, http.StatusCreated, doc)
}

func (h *Handler) ListarMisDocumentos(w http.ResponseWriter, r *http.Request) {
	claims, ok := middleware.ClaimsFromContext(r.Context())
	if !ok {
		utils.ErrorJSON(w, http.StatusUnauthorized, "Contexto de usuario inválido")
		return
	}
	var idReserva *uuid.UUID
	if q := r.URL.Query().Get("id_reserva"); q != "" {
		id, err := uuid.Parse(q)
		if err != nil {
			utils.ErrorJSON(w, http.StatusBadRequest, "id_reserva inválido")
			return
		}
		idReserva = &id
	}
	lista, err := h.service.ListarMisDocumentos(r.Context(), claims.UserID, idReserva)
	if err != nil {
		utils.WriteError(w, err)
		return
	}
	utils.JSON(w, http.StatusOK, lista)
}

// RequisitosReserva maneja GET /api/v1/reservas/{id}/requisitos-documentales.
// Los clientes solo consultan sus propias reservas; el resto de roles cualquiera.
func (h *Handler) RequisitosReserva(w http.ResponseWriter, r *http.Request) {
	claims, ok := middleware.ClaimsFromContext(r.Context())
	if !ok {
		utils.ErrorJSON(w, http.StatusUnauthorized, "Contexto de usuario inválido")
		return
	}
	idReserva, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		utils.ErrorJSON(w, http.StatusBadRequest, "ID de reserva inválido")
		return
	}
	var idCliente *uuid.UUID
	if strings.EqualFold(claims.Rol, "cliente") {
		idCliente = &claims.UserID
	}
	res, err := h.service.RequisitosReserva(r.Context(), idReserva, idCliente)
	if err != nil {
		utils.WriteError(w, err)
		return
	}
	utils.JSON(w, http.StatusOK, res)
}

func (h *Handler) ListarTodos(w http.ResponseWriter, r *http.Request) {
	lista, err := h.service.ListarTodos(r.Context(), r.URL.Query().Get("estado"))
	if err != nil {
		utils.WriteError(w, err)
		return
	}
	utils.JSON(w, http.StatusOK, lista)
}

func (h *Handler) Resolver(w http.ResponseWriter, r *http.Request) {
	claims, ok := middleware.ClaimsFromContext(r.Context())
	if !ok {
		utils.ErrorJSON(w, http.StatusUnauthorized, "Contexto de usuario inválido")
		return
	}
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		utils.ErrorJSON(w, http.StatusBadRequest, "ID de documento inválido")
		return
	}
	var req ResolverDocumentoRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		utils.ErrorJSON(w, http.StatusBadRequest, "Cuerpo JSON inválido")
		return
	}
	doc, err := h.service.Resolver(r.Context(), id, claims.UserID, req)
	if err != nil {
		utils.WriteError(w, err)
		return
	}
	utils.JSON(w, http.StatusOK, doc)
}
