package reservas

import (
	"time"

	"github.com/google/uuid"
)

const (
	EstadoPendiente       = "pendiente"
	EstadoConfirmada      = "confirmada"
	EstadoReprogramada    = "reprogramada"
	EstadoCancelada       = "cancelada"
	EstadoCompletada      = "completada"
	estadoAtencionInicial = "programada"
)

type ReservaServicio struct {
	IDServicio        uuid.UUID `json:"id_servicio"`
	Nombre            string    `json:"nombre"`
	PrecioUnitario    float64   `json:"precio_unitario"`
	Cantidad          int       `json:"cantidad"`
	DuracionMin       int       `json:"duracion_min"`
	RequiereDocumento bool      `json:"requiere_documento"`
}

type TrabajadorAsignado struct {
	IDTrabajador uuid.UUID `json:"id_trabajador"`
	Nombre       string    `json:"nombre"`
}

type Reserva struct {
	IDReserva         uuid.UUID            `json:"id_reserva"`
	IDCliente         uuid.UUID            `json:"id_cliente"`
	NombreCliente     string               `json:"nombre_cliente,omitempty"`
	CorreoCliente     string               `json:"correo_cliente,omitempty"`
	IDVehiculo        uuid.UUID            `json:"id_vehiculo"`
	Placa             string               `json:"placa"`
	Vehiculo          string               `json:"vehiculo"`
	IDEspacio         uuid.UUID            `json:"id_espacio"`
	CodigoEspacio     string               `json:"codigo_espacio"`
	FechaReserva      string               `json:"fecha_reserva"` // AAAA-MM-DD
	HoraInicio        string               `json:"hora_inicio"`   // HH:MM
	HoraFin           string               `json:"hora_fin"`      // HH:MM
	Estado            string               `json:"estado"`
	TotalEstimado     float64              `json:"total_estimado"`
	Observaciones     *string              `json:"observaciones,omitempty"`
	MotivoCancelacion *string              `json:"motivo_cancelacion,omitempty"`
	FechaCreacion     time.Time            `json:"fecha_creacion"`
	IDAtencion        *uuid.UUID           `json:"id_atencion,omitempty"`
	EstadoAtencion    *string              `json:"estado_atencion,omitempty"`
	RequiereDocumento bool                 `json:"requiere_documento"`
	Servicios         []ReservaServicio    `json:"servicios"`
	Trabajadores      []TrabajadorAsignado `json:"trabajadores"`
}

// ServicioInfo son los datos de un servicio que se usan al reservar.
type ServicioInfo struct {
	ID                uuid.UUID
	Nombre            string
	Precio            float64
	DuracionMin       int
	Disponible        bool
	RequiereDocumento bool
}

// ---- Peticiones

type CrearReservaInput struct {
	IDVehiculo    uuid.UUID   `json:"id_vehiculo"`
	IDEspacio     uuid.UUID   `json:"id_espacio"`
	Fecha         string      `json:"fecha"`       // AAAA-MM-DD
	HoraInicio    string      `json:"hora_inicio"` // HH:MM
	Servicios     []uuid.UUID `json:"servicios"`
	Observaciones string      `json:"observaciones"`
}

type ReprogramarInput struct {
	IDEspacio  uuid.UUID `json:"id_espacio"`
	Fecha      string    `json:"fecha"`
	HoraInicio string    `json:"hora_inicio"`
}

type CancelarInput struct {
	Motivo string `json:"motivo"`
}

type ProgramarInput struct {
	Trabajadores []uuid.UUID `json:"trabajadores"`
}

// ---- Disponibilidad

type Slot struct {
	HoraInicio string `json:"hora_inicio"`
	HoraFin    string `json:"hora_fin"`
}

type EspacioDisponible struct {
	IDEspacio uuid.UUID `json:"id_espacio"`
	Codigo    string    `json:"codigo"`
	Slots     []Slot    `json:"slots"`
}

type Disponibilidad struct {
	Fecha       string              `json:"fecha"`
	DuracionMin int                 `json:"duracion_min"`
	Espacios    []EspacioDisponible `json:"espacios"`
}

// TrabajadorDisponible indica si un trabajador puede asignarse al horario de una reserva.
type TrabajadorDisponible struct {
	IDTrabajador uuid.UUID `json:"id_trabajador"`
	Nombre       string    `json:"nombre"`
	Disponible   bool      `json:"disponible"` // marcado como disponible en su ficha
	Ocupado      bool      `json:"ocupado"`    // ya tiene otra atención que se cruza con este horario
}
