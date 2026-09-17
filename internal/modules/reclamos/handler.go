package reclamos

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	"carwashinmaribackend/internal/middleware"
	"carwashinmaribackend/internal/utils"
)

type Handler struct {
	service  Service
	r2Client *utils.R2Client
}

func NewHandler(service Service, r2Client *utils.R2Client) *Handler {
	return &Handler{
		service:  service,
		r2Client: r2Client,
	}
}

// CrearReclamo maneja POST /api/v1/reclamos (multipart/form-data)
func (h *Handler) CrearReclamo(w http.ResponseWriter, r *http.Request) {
	claims, ok := r.Context().Value(middleware.UserContextKey).(*utils.CustomClaims)
	if !ok {
		utils.ErrorJSON(w, http.StatusUnauthorized, "Contexto de usuario inválido")
		return
	}

	if err := r.ParseMultipartForm(15 << 20); err != nil {
		utils.ErrorJSON(w, http.StatusBadRequest, "El payload excede el límite permitido de 15MB")
		return
	}

	asunto := r.FormValue("asunto")
	descripcion := r.FormValue("descripcion")
	idAtencionStr := r.FormValue("id_atencion")
	idPagoStr := r.FormValue("id_pago")
	idPedidoStr := r.FormValue("id_pedido")

	var idAtencion, idPago, idPedido *uuid.UUID
	if idAtencionStr != "" {
		if uid, err := uuid.Parse(idAtencionStr); err == nil {
			idAtencion = &uid
		}
	}
	if idPagoStr != "" {
		if uid, err := uuid.Parse(idPagoStr); err == nil {
			idPago = &uid
		}
	}
	if idPedidoStr != "" {
		if uid, err := uuid.Parse(idPedidoStr); err == nil {
			idPedido = &uid
		}
	}

	var evidencias []EvidenciaInput
	files := r.MultipartForm.File["evidencias"]

	for _, fileHeader := range files {
		ext := strings.ToLower(filepath.Ext(fileHeader.Filename))
		if ext != ".jpg" && ext != ".jpeg" && ext != ".png" && ext != ".webp" {
			utils.ErrorJSON(w, http.StatusBadRequest, "Formato no válido. Solo se admiten JPG, PNG y WEBP")
			return
		}

		srcFile, err := fileHeader.Open()
		if err != nil {
			utils.ErrorJSON(w, http.StatusInternalServerError, "Error al procesar archivo")
			return
		}
		defer srcFile.Close()

		// Key única: evidencias/<timestamp>_<uuid><ext>
		key := fmt.Sprintf("evidencias/%d_%s%s", time.Now().UnixNano(), uuid.New().String()[:8], ext)
		contentType := fileHeader.Header.Get("Content-Type")
		if contentType == "" {
			contentType = "application/octet-stream"
		}

		var urlFinal string
		// Intento 1: Cloudflare R2 si está inicializado
		if h.r2Client != nil {
			if urlR2, err := h.r2Client.SubirArchivo(r.Context(), srcFile, key, contentType); err == nil {
				urlFinal = urlR2
			}
		}

		// Fallback: Guardado local en disco si R2 no está disponible o falló
		if urlFinal == "" {
			_ = os.MkdirAll(filepath.Join(".", "uploads", "evidencias"), 0755)
			localFilePath := filepath.Join(".", "uploads", key)
			dstFile, err := os.Create(localFilePath)
			if err != nil {
				utils.ErrorJSON(w, http.StatusInternalServerError, "Error al guardar evidencia localmente: "+err.Error())
				return
			}
			_, _ = srcFile.Seek(0, io.SeekStart)
			if _, err := io.Copy(dstFile, srcFile); err != nil {
				dstFile.Close()
				utils.ErrorJSON(w, http.StatusInternalServerError, "Error al escribir archivo local: "+err.Error())
				return
			}
			dstFile.Close()
			urlFinal = fmt.Sprintf("/uploads/%s", key)
		}

		desc := r.FormValue("descripcion_evidencia")
		var descPtr *string
		if desc != "" {
			descPtr = &desc
		}

		evidencias = append(evidencias, EvidenciaInput{
			RutaArchivo: urlFinal,
			Descripcion: descPtr,
		})
	}

	input := CrearReclamoInput{
		IDCliente:   claims.UserID,
		IDAtencion:  idAtencion,
		IDPago:      idPago,
		IDPedido:    idPedido,
		Asunto:      asunto,
		Descripcion: descripcion,
		Evidencias:  evidencias,
	}

	reclamoCreado, err := h.service.CrearReclamo(r.Context(), input)
	if err != nil {
		utils.ErrorJSON(w, http.StatusBadRequest, err.Error())
		return
	}

	utils.JSON(w, http.StatusCreated, reclamoCreado)
}

func (h *Handler) ListarMisReclamos(w http.ResponseWriter, r *http.Request) {
	claims, ok := r.Context().Value(middleware.UserContextKey).(*utils.CustomClaims)
	if !ok {
		utils.ErrorJSON(w, http.StatusUnauthorized, "Contexto de usuario inválido")
		return
	}

	reclamos, err := h.service.ListarMisReclamos(r.Context(), claims.UserID)
	if err != nil {
		utils.ErrorJSON(w, http.StatusInternalServerError, err.Error())
		return
	}

	utils.JSON(w, http.StatusOK, reclamos)
}

func (h *Handler) ListarTodos(w http.ResponseWriter, r *http.Request) {
	reclamos, err := h.service.ListarTodos(r.Context())
	if err != nil {
		utils.ErrorJSON(w, http.StatusInternalServerError, err.Error())
		return
	}

	utils.JSON(w, http.StatusOK, reclamos)
}

func (h *Handler) ResponderReclamo(w http.ResponseWriter, r *http.Request) {
	claims, ok := r.Context().Value(middleware.UserContextKey).(*utils.CustomClaims)
	if !ok {
		utils.ErrorJSON(w, http.StatusUnauthorized, "Contexto de usuario inválido")
		return
	}

	idStr := chi.URLParam(r, "id")
	idReclamo, err := uuid.Parse(idStr)
	if err != nil {
		utils.ErrorJSON(w, http.StatusBadRequest, "ID de reclamo inválido")
		return
	}

	var req ResponderReclamoRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		utils.ErrorJSON(w, http.StatusBadRequest, "Cuerpo JSON inválido")
		return
	}

	if err := h.service.ResponderReclamo(r.Context(), idReclamo, claims.UserID, req); err != nil {
		utils.ErrorJSON(w, http.StatusBadRequest, err.Error())
		return
	}

	utils.JSON(w, http.StatusOK, map[string]string{"message": "Reclamo respondido exitosamente"})
}
