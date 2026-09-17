package reclamos

import (
	"time"

	"github.com/google/uuid"
)

type EstadoReclamo string

const (
	EstadoRegistrado EstadoReclamo = "registrado"
	EstadoEnRevision EstadoReclamo = "en_revision"
	EstadoRespondido EstadoReclamo = "respondido"
	EstadoCerrado    EstadoReclamo = "cerrado"
)

type EvidenciaReclamo struct {
	IDEvidencia uuid.UUID `json:"id_evidencia"`
	IDReclamo   uuid.UUID `json:"id_reclamo"`
	RutaArchivo string    `json:"ruta_archivo"`
	Descripcion *string   `json:"descripcion,omitempty"`
	CreatedAt   time.Time `json:"created_at"`
}

type Reclamo struct {
	IDReclamo      uuid.UUID          `json:"id_reclamo"`
	IDCliente      uuid.UUID          `json:"id_cliente"`
	NombreCliente  string             `json:"nombre_cliente,omitempty"`
	CorreoCliente  string             `json:"correo_cliente,omitempty"`
	IDAtencion     *uuid.UUID         `json:"id_atencion,omitempty"`
	IDPago         *uuid.UUID         `json:"id_pago,omitempty"`
	IDPedido       *uuid.UUID         `json:"id_pedido,omitempty"`
	Asunto         string             `json:"asunto"`
	Descripcion    string             `json:"descripcion"`
	Estado         EstadoReclamo      `json:"estado"`
	RespuestaAdmin *string            `json:"respuesta_admin,omitempty"`
	RespondidoPor  *uuid.UUID         `json:"respondido_por,omitempty"`
	FechaRegistro  time.Time          `json:"fecha_registro"`
	FechaRespuesta *time.Time         `json:"fecha_respuesta,omitempty"`
	Evidencias     []EvidenciaReclamo `json:"evidencias"`
}

type EvidenciaInput struct {
	RutaArchivo string
	Descripcion *string
}

type CrearReclamoInput struct {
	IDCliente   uuid.UUID
	IDAtencion  *uuid.UUID
	IDPago      *uuid.UUID
	IDPedido    *uuid.UUID
	Asunto      string
	Descripcion string
	Evidencias  []EvidenciaInput
}

type ResponderReclamoRequest struct {
	RespuestaAdmin string        `json:"respuesta_admin"`
	NuevoEstado    EstadoReclamo `json:"nuevo_estado"`
}
