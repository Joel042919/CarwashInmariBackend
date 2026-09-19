package documentos

import (
	"time"

	"github.com/google/uuid"
)

const (
	EstadoAdjuntado = "adjuntado"
	EstadoValidado  = "validado"
	EstadoRechazado = "rechazado"
)

type Documento struct {
	IDDocumento     uuid.UUID  `json:"id_documento"`
	IDReserva       uuid.UUID  `json:"id_reserva"`
	IDServicio      uuid.UUID  `json:"id_servicio"`
	NombreServicio  string     `json:"nombre_servicio,omitempty"`
	IDCliente       uuid.UUID  `json:"id_cliente"`
	NombreCliente   string     `json:"nombre_cliente,omitempty"`
	CorreoCliente   string     `json:"correo_cliente,omitempty"`
	RutaPDF         string     `json:"ruta_pdf"`
	Estado          string     `json:"estado"`
	AdminRespuesta  *string    `json:"admin_respuesta,omitempty"`
	ValidadoPor     *uuid.UUID `json:"validado_por,omitempty"`
	FechaValidacion *time.Time `json:"fecha_validacion,omitempty"`
	CreatedAt       time.Time  `json:"created_at"`
}

type SubirDocumentoInput struct {
	IDReserva  uuid.UUID
	IDServicio uuid.UUID
	IDCliente  uuid.UUID
	RutaPDF    string
}

type ResolverDocumentoRequest struct {
	Estado         string `json:"estado"`
	AdminRespuesta string `json:"admin_respuesta"`
}

// RequisitoDocumento describe el estado documental de un servicio de la reserva
// que exige PDF firmado.
type RequisitoDocumento struct {
	IDServicio      uuid.UUID  `json:"id_servicio"`
	NombreServicio  string     `json:"nombre_servicio"`
	IDDocumento     *uuid.UUID `json:"id_documento,omitempty"`
	EstadoDocumento *string    `json:"estado_documento,omitempty"`
	Cumplido        bool       `json:"cumplido"`
}

// RequisitosReserva permite al módulo de reservas saber si una reserva ya puede
// pasar a "confirmada" (RF-07).
type RequisitosReserva struct {
	IDReserva        uuid.UUID            `json:"id_reserva"`
	Requisitos       []RequisitoDocumento `json:"requisitos"`
	PuedeConfirmarse bool                 `json:"puede_confirmarse"`
}
