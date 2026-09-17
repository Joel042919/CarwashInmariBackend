package clientes

import (
	"time"

	"github.com/google/uuid"
)

type ClientePerfil struct {
	IDUsuario       uuid.UUID  `json:"id_usuario"`
	IDSede          uuid.UUID  `json:"id_sede"`
	Nombre          string     `json:"nombre"`
	Apellido        string     `json:"apellido"`
	Correo          string     `json:"correo"`
	Telefono        *string    `json:"telefono,omitempty"`
	DNI             *string    `json:"dni,omitempty"`
	FechaNacimiento *time.Time `json:"fecha_nacimiento,omitempty"`
	Direccion       *string    `json:"direccion,omitempty"`
	Notas           *string    `json:"notas,omitempty"`
	CreatedAt       time.Time  `json:"created_at"`
}

type UpdateClienteRequest struct {
	Nombre          string     `json:"nombre"`
	Apellido        string     `json:"apellido"`
	Telefono        *string    `json:"telefono"`
	Direccion       *string    `json:"direccion"`
	FechaNacimiento *time.Time `json:"fecha_nacimiento"`
}

// Estructuras para el historial unificado

type ReservaHistorial struct {
	IDReserva     uuid.UUID `json:"id_reserva"`
	PlacaVehiculo string    `json:"placa_vehiculo"`
	Modelo        string    `json:"modelo"`
	CodigoEspacio string    `json:"codigo_espacio"`
	FechaReserva  time.Time `json:"fecha_reserva"`
	HoraInicio    string    `json:"hora_inicio"`
	HoraFin       string    `json:"hora_fin"`
	Estado        string    `json:"estado"`
	TotalEstimado float64   `json:"total_estimado"`
}

type AtencionHistorial struct {
	IDAtencion      uuid.UUID  `json:"id_atencion"`
	IDReserva       uuid.UUID  `json:"id_reserva"`
	PlacaVehiculo   string     `json:"placa_vehiculo"`
	Estado          string     `json:"estado"`
	FechaInicioReal *time.Time `json:"fecha_inicio_real,omitempty"`
	FechaFinReal    *time.Time `json:"fecha_fin_real,omitempty"`
}

type PagoHistorial struct {
	IDPago             uuid.UUID `json:"id_pago"`
	Monto              float64   `json:"monto"`
	Metodo             string    `json:"metodo"`
	Estado             string    `json:"estado"`
	ComprobanteInterno string    `json:"comprobante_interno"`
	FechaPago          time.Time `json:"fecha_pago"`
}

type PedidoHistorial struct {
	IDPedido      uuid.UUID  `json:"id_pedido"`
	Estado        string     `json:"estado"`
	Total         float64    `json:"total"`
	Observaciones *string    `json:"observaciones,omitempty"`
	FechaRegistro time.Time  `json:"fecha_registro"`
	FechaEntrega  *time.Time `json:"fecha_entrega,omitempty"`
}

type HistorialClienteUnificado struct {
	Reservas   []ReservaHistorial  `json:"reservas"`
	Atenciones []AtencionHistorial `json:"atenciones"`
	Pagos      []PagoHistorial     `json:"pagos"`
	Pedidos    []PedidoHistorial   `json:"pedidos"`
}
